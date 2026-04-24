package ra

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/bytedance/sonic/decoder"
	"github.com/e11it/ra/helpers"
	"github.com/e11it/ra/pkg/auth"
	"github.com/gin-gonic/gin"
)

// readAndRestoreBody читает тело запроса целиком и восстанавливает его,
// чтобы последующие хендлеры (в т.ч. reverse proxy) получили нетронутый поток.
func readAndRestoreBody(c *gin.Context) ([]byte, error) {
	if c.Request.Body == nil {
		return nil, nil
	}
	buf, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(buf))
	return buf, nil
}

// Создает объект auth.AuthRequest из git.Context
func (ra *Ra) GetAuthRequest(c *gin.Context, proxy bool) (authRequest *auth.AuthRequest) {
	authRequest = new(auth.AuthRequest)
	authRequest.ContentType = c.ContentType()
	if proxy {
		authRequest.AuthURL = strings.TrimPrefix(c.Request.RequestURI, ra.config.TrimURLPrefix)
		authRequest.IP = c.ClientIP()
		authRequest.Method = c.Request.Method
	} else {
		// Не очень удачное место для обрезания URL
		authRequest.AuthURL = strings.TrimPrefix(c.GetHeader(ra.config.Headers.AuthURL), ra.config.TrimURLPrefix)
		authRequest.IP = c.GetHeader(ra.config.Headers.IP)
		authRequest.Method = c.GetHeader(ra.config.Headers.Method)
	}
	// Нам так же важно знать имя пользователя
	// TODO: это может делать другой middleware(проверять и выставлять)
	username, _, authOK := c.Request.BasicAuth()
	if !authOK {
		username = "anon"
	}
	authRequest.AuthUser = username

	return
}

func (ra *Ra) GetAuthMiddlerware(proxy bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		authRequest := ra.GetAuthRequest(c, proxy)
		err := ra.auth.Validate(authRequest)
		if err != nil {
			log.Warn().
				Err(err).
				Str("request_uri", c.Request.RequestURI).
				Str("auth_url", authRequest.AuthURL).
				Str("method", authRequest.Method).
				Str("user", authRequest.AuthUser).
				Str("content_type", authRequest.ContentType).
				Msg("auth rejected")
			WriteJSONErrorGin(
				c,
				http.StatusForbidden,
				ErrorCodeAuthDenied,
				"Ra: auth denied",
				err.Error(),
				DetailsWithReason(GinTraceID(c), err),
			)
			return
		}

		if ra.bodyValidator != nil && c.Request.Method == http.MethodPost {
			body, err := readAndRestoreBody(c)
			if err != nil {
				log.Warn().Err(err).Str("request_uri", c.Request.RequestURI).Msg("read request body failed")
				WriteJSONErrorGin(
					c,
					http.StatusBadRequest,
					ErrorCodeMalformedBody,
					"Ra: cannot read request body",
					err.Error(),
					DetailsWithReason(GinTraceID(c), err),
				)
				return
			}
			rep := ra.bodyValidator.Validate(body)
			if rep.HasErrors() {
				log.Warn().
					Str("request_uri", c.Request.RequestURI).
					Str("summary", rep.SummaryLine()).
					Msg("body validation failed")
				WriteJSONErrorGin(
					c,
					http.StatusUnprocessableEntity,
					ErrorCodePayloadValidate,
					"Ra: payload validation errors",
					rep.SummaryLine(),
					BuildValidationDetails(rep, GinTraceID(c)),
				)
				return
			}
		}

		c.Set("username", authRequest.AuthUser)
		c.Next()
	}
}

func GetUUIDMiddlerware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var rid string
		// TODO: получить x-request-id от клиента
		rid = c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.Must(uuid.NewRandom()).String()
		}

		c.Header("X-Request-ID", rid)
		c.Set("x-request-id", rid)
		c.Next()
	}
}

func (ra *Ra) GetProxyHandler() gin.HandlerFunc {
	remote, err := url.Parse(ra.config.Proxy.ProxyHost)
	if err != nil {
		log.Panic().
			Err(err).Msg("failed to parse ProxyHost to URL")
	}
	err = helpers.IsAlive(remote)
	if err != nil {
		log.Panic().
			Err(err).Msg("ProxyHost connection check failed")
	}
	log.Info().Msgf("Enable proxy to %s", remote.String())

	return func(c *gin.Context) {
		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.ErrorLog = ReverseProxyErrorLog()
		proxy.Director = func(req *http.Request) {
			req.Header = c.Request.Header
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
		}
		proxy.ModifyResponse = func(resp *http.Response) error {
			type ErrorResp struct {
				ErrorCode int    `json:"error_code"`
				Message   string `json:"message"`
			}
			if resp.StatusCode != http.StatusOK {
				body, readErr := io.ReadAll(resp.Body)
				if readErr != nil {
					log.Err(readErr).Msg("cant read proxy response")
					// Keep proxy path best-effort: return partial data if any.
					resp.Body = io.NopCloser(bytes.NewReader(body))
					return nil
				}
				resp.Body = io.NopCloser(bytes.NewReader(body))

				er := new(ErrorResp)
				dec := decoder.NewStreamDecoder(bytes.NewReader(body))
				err := dec.Decode(er)
				if err == nil {
					log.Info().
						Str("x-request-id", c.MustGet("x-request-id").(string)).
						Str("request_uri", c.Request.RequestURI).
						Str("remote_user", c.MustGet("username").(string)).
						Msgf("error: %d msg: %s", er.ErrorCode, er.Message)
					resp.Header.Set("X-Error-Code", fmt.Sprintf("%d", er.ErrorCode))
				} else {
					// Upstream may return HTML/plain (e.g. python -m http.server 501) — not Confluent JSON.
					log.Debug().
						Err(err).
						Int("status", resp.StatusCode).
						Str("content_type", resp.Header.Get("Content-Type")).
						Msg("proxy upstream non-JSON error body (skip decode)")
				}
			}
			return nil
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

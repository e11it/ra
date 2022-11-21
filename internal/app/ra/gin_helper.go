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
			c.Header("X-RA-ERROR", err.Error())
			_ = c.AbortWithError(http.StatusForbidden, err)
		}
		c.Set("username", authRequest.AuthUser)
		c.Next()
	}
}

func GetUUIDMiddlerware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: получить x-request-id от клиента
		_uuid := uuid.Must(uuid.NewRandom())
		c.Header("X-Request-ID", _uuid.String())
		c.Set("x-request-id", _uuid.String())
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
		proxy.Director = func(req *http.Request) {
			req.Header = c.Request.Header
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
		}
		proxy.ModifyResponse = func(resp *http.Response) error {
			var buf bytes.Buffer

			type ErrorResp struct {
				ErrorCode int    `json:"error_code"`
				Message   string `json:"message"`
			}
			if resp.StatusCode != http.StatusOK {
				tee := io.TeeReader(resp.Body, &buf)
				er := new(ErrorResp)
				dec := decoder.NewStreamDecoder(tee)
				err := dec.Decode(er)
				if err == nil {
					log.Info().
						Str("x-request-id", c.MustGet("x-request-id").(string)).
						Str("request_uri", c.Request.RequestURI).
						Str("remote_user", c.MustGet("username").(string)).
						Msgf("error: %d msg: %s", er.ErrorCode, er.Message)
					resp.Body = io.NopCloser(&buf)
					resp.Header.Set("X-Error-Code", fmt.Sprintf("%d", er.ErrorCode))
				} else {
					log.Err(err).Msg("cant decode proxy response")
				}
			}
			return nil
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

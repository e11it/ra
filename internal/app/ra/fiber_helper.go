package ra

import (
	"bytes"
	"encoding/base64"
	"strings"

	"github.com/e11it/ra/pkg/auth"
	"github.com/gofiber/fiber/v3"
)

func (ra *Ra) GetFiberAuthRequest(c fiber.Ctx) (authRequest *auth.AuthRequest) {
	authRequest = new(auth.AuthRequest)

	// Не очень удачное место для обрезания URL
	authRequest.AuthURL = strings.TrimPrefix(c.Get(ra.config.Headers.AuthURL), ra.config.TrimURLPrefix)
	authRequest.ContentType = c.Get("Content-Type")
	authRequest.IP = c.Get(ra.config.Headers.IP)
	authRequest.Method = c.Get(ra.config.Headers.Method)
	// Нам так же важно знать имя пользователя
	// TODO: это может делать другой middleware(проверять и выставлять)
	//
	username, _, authOK := FiberBasicAuth(c)
	if !authOK {
		username = "anon"
	}
	authRequest.AuthUser = username

	return
}

func (ra *Ra) GetFiberAuthMiddlerware() fiber.Handler {
	return func(c fiber.Ctx) error {
		authRequest := ra.GetFiberAuthRequest(c)
		err := ra.auth.Validate(authRequest)
		if err != nil {
			c.Set(fiber.HeaderWWWAuthenticate, err.Error())
			return WriteJSONErrorFiber(
				c,
				fiber.StatusForbidden,
				ErrorCodeAuthDenied,
				"Ra: auth denied",
				err.Error(),
				DetailsWithReason(FiberTraceID(c), err),
			)
		}

		if ra.bodyValidator != nil && c.Method() == fiber.MethodPost {
			rep := ra.bodyValidator.Validate(c.Body())
			if rep.HasErrors() {
				return WriteJSONErrorFiber(
					c,
					fiber.StatusUnprocessableEntity,
					ErrorCodePayloadValidate,
					"Ra: payload validation errors",
					rep.SummaryLine(),
					BuildValidationDetails(rep, FiberTraceID(c)),
				)
			}
		}

		return c.Next()
	}
}

func FiberBasicAuth(c fiber.Ctx) (username, password string, ok bool) {
	// Get authorization header
	auth := c.Get(fiber.HeaderAuthorization)

	// Check if the header contains content besides "basic".
	if len(auth) <= 6 || !strings.EqualFold(auth[:5], "basic") {
		return "", "", false
	}

	// Decode the header contents
	raw, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		return "", "", false
	}

	// Check if the credentials are in the correct form
	// which is "username:password".
	index := bytes.IndexByte(raw, ':')
	if index == -1 {
		return "", "", false
	}

	// Get the username and password
	creds := string(raw)
	return creds[:index], creds[index+1:], true
}

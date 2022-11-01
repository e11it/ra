package ra

import (
	"encoding/base64"
	"strings"

	"github.com/e11it/ra/pkg/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
)

func (ra *Ra) GetFiberAuthRequest(c *fiber.Ctx) (authRequest *auth.AuthRequest) {
	authRequest = new(auth.AuthRequest)

	// Не очень удачное место для обрезания URL
	authRequest.AuthURL = strings.TrimPrefix(c.Get(ra.config.Headers.AuthURL), ra.config.TrimUrlPrefix)
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
	return func(c *fiber.Ctx) error {
		authRequest := ra.GetFiberAuthRequest(c)
		err := ra.auth.Validate(authRequest)
		if err != nil {
			c.Set(fiber.HeaderWWWAuthenticate, err.Error())
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		return c.Next()
	}
}

func FiberBasicAuth(c *fiber.Ctx) (username, password string, ok bool) {
	// Get authorization header
	auth := c.Get(fiber.HeaderAuthorization)

	// Check if the header contains content besides "basic".
	if len(auth) <= 6 || strings.ToLower(auth[:5]) != "basic" {
		return "", "", false
	}

	// Decode the header contents
	raw, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		return "", "", false
	}

	// Get the credentials
	creds := utils.UnsafeString(raw)

	// Check if the credentials are in the correct form
	// which is "username:password".
	index := strings.Index(creds, ":")
	if index == -1 {
		return "", "", false
	}

	// Get the username and password
	// username := creds[:index]
	// password := creds[index+1:]
	return creds[:index], creds[index+1:], true
}

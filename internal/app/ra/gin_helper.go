package ra

import (
	"net/http"
	"strings"

	"github.com/e11it/ra/pkg/auth"
	"github.com/gin-gonic/gin"
)

// Создает объект auth.AuthRequest из git.Context
func (ra *Ra) GetAuthRequest(c *gin.Context) (authRequest *auth.AuthRequest) {
	authRequest = new(auth.AuthRequest)
	// Не очень удачное место для обрезания URL
	authRequest.AuthURL = strings.TrimPrefix(c.GetHeader(ra.config.Headers.AuthURL), ra.config.TrimUrlPrefix)
	authRequest.ContentType = c.ContentType()
	authRequest.IP = c.GetHeader(ra.config.Headers.IP)
	authRequest.Method = c.GetHeader(ra.config.Headers.Method)
	// Нам так же важно знать имя пользователя
	// TODO: это может делать другой middleware(проверять и выставлять)
	username, _, authOK := c.Request.BasicAuth()
	if !authOK {
		username = "anon"
	}
	authRequest.AuthUser = username

	return
}

func (ra *Ra) GetAuthMiddlerware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authRequest := ra.GetAuthRequest(c)
		err := ra.auth.Validate(authRequest)
		if err != nil {
			c.AbortWithError(http.StatusForbidden, err)
		}
		c.Next()
	}
}

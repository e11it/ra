package auth

import "github.com/gin-gonic/gin"

type cacheKey struct {
	AuthURL     interface{}
	AuthUser    interface{}
	IP          interface{}
	Method      interface{}
	ContentType interface{}
}

func getCacheKey(c *gin.Context) *cacheKey {
	return &cacheKey{
		AuthURL:     c.MustGet("AuthURL"),
		AuthUser:    c.MustGet("AuthUser"),
		IP:          c.MustGet("IP"),
		Method:      c.MustGet("Method"),
		ContentType: c.MustGet("ContentType"),
	}
}

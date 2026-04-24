package ra

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrAbortHandlerSilenceMiddleware must sit between [gin.Recovery] (outer) and
// [AccessLogMiddleware] (inner). The stdlib reverse proxy ends with
// [http.ErrAbortHandler] after a copy error; [AccessLogMiddleware] re-raises
// that panic so the access line is still written. [gin.Recovery] would otherwise
// log non-JSON text (the error string + request dump) for a condition that is
// expected, not a bug, and is already covered by the access and proxy warn logs.
func ErrAbortHandlerSilenceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			err, ok := r.(error)
			if ok && errors.Is(err, http.ErrAbortHandler) {
				c.Abort()
				return
			}
			panic(r)
		}()
		c.Next()
	}
}

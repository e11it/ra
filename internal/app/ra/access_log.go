package ra

import (
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// buildAccessLogExcludeSet нормализует пути (через [path.Clean], ведущий /) в множество для O(1) поиска.
func buildAccessLogExcludeSet(paths []string) map[string]struct{} {
	if len(paths) == 0 {
		return nil
	}
	m := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = path.Clean(p)
		if p != "/" && !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		m[p] = struct{}{}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// AccessLogMiddleware пишет в лог завершение каждого запроса: статус, длительность, путь.
// В stdlib и Gin такого нет — это стандартный шаблон обёртки (аналог access_log nginx).
// excludePaths — [http.Request.URL.Path] без query; совпадение с теми, что в excludePaths, пропускает access-лог.
func AccessLogMiddleware(excludePaths []string) gin.HandlerFunc {
	excluded := buildAccessLogExcludeSet(excludePaths)
	return func(c *gin.Context) {
		lookup := path.Clean(c.Request.URL.Path)
		if _, skip := excluded[lookup]; skip {
			c.Next()
			return
		}
		start := time.Now()
		reqPath := c.Request.URL.Path
		if q := c.Request.URL.RawQuery; q != "" {
			reqPath = reqPath + "?" + q
		}
		// stdlib httputil.ReverseProxy on response copy error panics with [http.ErrAbortHandler]
		// after the status line may already be sent (e.g. 200 from upstream). That skips any
		// code after c.Next() unless we log from defer and re-raise the panic.
		defer func() {
			r := recover()
			lat := time.Since(start)
			status := c.Writer.Status()
			if r != nil {
				// Unusual: panic before status was set (e.g. round-trip error).
				if status == 0 {
					status = http.StatusInternalServerError
				}
			}
			var ev *zerolog.Event
			switch {
			case status >= 500:
				ev = log.Error()
			case status >= 400:
				ev = log.Warn()
			default:
				ev = log.Info()
			}
			ev = ev.
				Str("method", c.Request.Method).
				Str("path", reqPath).
				Int("status", status).
				Dur("latency", lat)
			if v, ok := c.Get("x-request-id"); ok {
				if s, ok := v.(string); ok && s != "" {
					ev = ev.Str("x_request_id", s)
				}
			}
			if raErr := c.Writer.Header().Get("X-RA-ERROR"); raErr != "" {
				ev = ev.Str("x_ra_error", raErr)
			}
			ev.Msg("http request")
			if r != nil {
				panic(r)
			}
		}()
		c.Next()
	}
}

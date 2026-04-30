package ra

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAccessLogExcludeSet(t *testing.T) {
	t.Parallel()
	m := buildAccessLogExcludeSet([]string{
		" /metrics ",
		"/health",
		"",
		"/swagger/ra.yaml",
	})
	require.NotNil(t, m)
	_, ok := m["/metrics"]
	assert.True(t, ok)
	_, ok = m["/health"]
	assert.True(t, ok)
	_, ok = m["/swagger/ra.yaml"]
	assert.True(t, ok)
}

func TestBuildAccessLogExcludeSet_EmptyYieldsNil(t *testing.T) {
	t.Parallel()
	assert.Nil(t, buildAccessLogExcludeSet(nil))
	assert.Nil(t, buildAccessLogExcludeSet([]string{""}))
}

func TestBuildAccessLogExcludeSet_CleansPaths(t *testing.T) {
	t.Parallel()
	m := buildAccessLogExcludeSet([]string{"/ready/", "metrics"})
	require.NotNil(t, m)
	_, ok := m["/ready"]
	assert.True(t, ok, "trailing slash normalized")
	_, ok = m["/metrics"]
	assert.True(t, ok, "leading slash added")
}

func TestAccessLogMiddleware_ExcludedPathSkipsLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	prev := log.Logger
	log.Logger = zerolog.New(&buf)
	t.Cleanup(func() { log.Logger = prev })

	r := gin.New()
	r.Use(AccessLogMiddleware([]string{"/metrics", "/swagger"}))
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/metrics", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/swagger", func(c *gin.Context) { c.Status(http.StatusOK) })

	// /ok — access line.
	buf.Reset()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ok", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
	assert.Contains(t, buf.String(), "http request")

	// /metrics?x=1 — тот же path для фильтра, access-лог не пишем.
	buf.Reset()
	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics?x=1", nil)
	r.ServeHTTP(httptest.NewRecorder(), req2)
	assert.NotContains(t, buf.String(), "http request")

	// /swagger/ — [path.Clean] совпадает с /swagger
	buf.Reset()
	req3 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/swagger/", nil)
	r.ServeHTTP(httptest.NewRecorder(), req3)
	assert.NotContains(t, buf.String(), "http request")
}

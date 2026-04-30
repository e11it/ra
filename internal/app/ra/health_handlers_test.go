package ra

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGinHealthAndReadyHandlers(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.ReleaseMode)

	t.Run("health returns 200", func(t *testing.T) {
		t.Parallel()

		r := &Ra{config: &Config{}}
		router := gin.New()
		router.GET("/health", r.HandleHealthGin)

		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "ok", w.Body.String())
	})

	t.Run("ready returns 503 with json contract", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{}
		cfg.Proxy.Enabled = true
		cfg.Proxy.ProxyHost = "http://127.0.0.1:1"
		r := &Ra{config: cfg}
		router := gin.New()
		router.GET("/ready", r.HandleReadyGin)

		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ready", nil)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusServiceUnavailable, w.Code)

		var got RAErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		require.Equal(t, ErrorCodeReadyUnavailable, got.ErrorCode)
	})
}

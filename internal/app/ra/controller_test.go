package ra

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReloadHandler_PublishesCompleteStateOrKeepsPreviousState(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	path := filepath.Join(t.TempDir(), "ra.yml")
	writeRuntimeConfig(t, path, ":8081", "old-user", ".*")
	ra, err := NewRA(path)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, authStatus(t, ra, "old-user"))

	writeRuntimeConfig(t, path, ":8082", "new-user", "[")
	reloaded, err := ra.ReloadHandler()

	assert.False(t, reloaded)
	require.Error(t, err)
	assert.Equal(t, ":8081", ra.GetServerAddr())
	assert.Equal(t, http.StatusOK, authStatus(t, ra, "old-user"))
	assert.Equal(t, http.StatusForbidden, authStatus(t, ra, "new-user"))

	writeRuntimeConfig(t, path, ":8082", "new-user", ".*")
	reloaded, err = ra.ReloadHandler()

	require.NoError(t, err)
	assert.True(t, reloaded)
	assert.Equal(t, ":8082", ra.GetServerAddr())
	assert.Equal(t, http.StatusForbidden, authStatus(t, ra, "old-user"))
	assert.Equal(t, http.StatusOK, authStatus(t, ra, "new-user"))
}

func TestReloadHandler_ConcurrentReaders(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	path := filepath.Join(t.TempDir(), "ra.yml")
	writeRuntimeConfig(t, path, ":8081", "any", ".*")
	ra, err := NewRA(path)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 50 {
				_ = ra.GetServerAddr()
				_ = ra.ProxyEnabled()
				_ = ra.GetShutdownTimeout()
				_ = ra.AccessLogExcludePaths()
				if status := authStatus(t, ra, "reader"); status != http.StatusOK {
					t.Errorf("unexpected auth status during reload: %d", status)
					return
				}
			}
		})
	}

	for i := range 20 {
		writeRuntimeConfig(t, path, fmt.Sprintf(":%d", 8082+i), "any", ".*")
		_, err := ra.ReloadHandler()
		require.NoError(t, err)
	}
	wg.Wait()
}

func TestReloadHandler_RetriesSameBytesAfterTransientBuildFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ra.yml")
	writeRuntimeConfig(t, path, ":8081", "any", ".*")
	ra, err := NewRA(path)
	require.NoError(t, err)

	writeRuntimeConfig(t, path, ":8082", "any", ".*")
	reloaded, err := ra.reloadConfig(func(_ *Config) (*runtimeState, error) {
		return nil, errors.New("transient build failure")
	})

	assert.False(t, reloaded)
	require.ErrorContains(t, err, "transient build failure")
	assert.Equal(t, ":8081", ra.GetServerAddr())

	reloaded, err = ra.ReloadHandler()

	require.NoError(t, err)
	assert.True(t, reloaded)
	assert.Equal(t, ":8082", ra.GetServerAddr())
}

func newRAFromConfig(cfg *Config) (*Ra, error) {
	ra := &Ra{}
	state, err := ra.buildRuntimeState(cfg)
	if err != nil {
		return nil, err
	}
	ra.state.Store(state)
	return ra, nil
}

func authStatus(t *testing.T, ra *Ra, user string) int {
	t.Helper()
	router := gin.New()
	router.Use(ra.GetAuthMiddlerware(false))
	router.GET("/auth", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth", http.NoBody)
	req.RemoteAddr = "192.0.2.10:4321"
	req.Header.Set("X-Original-Uri", "/topics/example")
	req.Header.Set("X-Original-Method", http.MethodGet)
	req.Header.Set("X-Authenticated-User", user)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Code
}

func writeRuntimeConfig(t *testing.T, path, addr, user, aclPath string) {
	t.Helper()
	contents := fmt.Sprintf(`
addr: %q
cache:
  enabled: false
identity:
  authenticated_user_header: X-Authenticated-User
  trusted_proxies: [192.0.2.0/24]
auth:
  acl:
    - path: %q
      users: [%s]
      methods: [any]
      contenttype: [any]
`, addr, aclPath, user)
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
}

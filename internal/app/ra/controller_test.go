package ra

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReloadHandler_PublishesCompleteStateOrKeepsPreviousState(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	path := filepath.Join(t.TempDir(), "ra.yml")
	writeRuntimeConfig(t, path, 5, "old-user", ".*")
	ra, err := NewRA(path)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, authStatus(t, ra, "old-user"))

	writeRuntimeConfig(t, path, 6, "new-user", "[")
	reloaded, err := ra.ReloadHandler()

	assert.False(t, reloaded)
	require.Error(t, err)
	assert.Equal(t, ":8081", ra.GetServerAddr())
	assert.Equal(t, uint(5), ra.GetShutdownTimeout())
	assert.Equal(t, http.StatusOK, authStatus(t, ra, "old-user"))
	assert.Equal(t, http.StatusForbidden, authStatus(t, ra, "new-user"))

	writeRuntimeConfig(t, path, 6, "new-user", ".*")
	reloaded, err = ra.ReloadHandler()

	require.NoError(t, err)
	assert.True(t, reloaded)
	assert.Equal(t, ":8081", ra.GetServerAddr())
	assert.Equal(t, uint(6), ra.GetShutdownTimeout())
	assert.Equal(t, http.StatusForbidden, authStatus(t, ra, "old-user"))
	assert.Equal(t, http.StatusOK, authStatus(t, ra, "new-user"))
}

func TestReloadHandler_ConcurrentReaders(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	path := filepath.Join(t.TempDir(), "ra.yml")
	writeRuntimeConfig(t, path, 5, "any", ".*")
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
		writeRuntimeConfig(t, path, uint(10+i), "any", ".*")
		_, err := ra.ReloadHandler()
		require.NoError(t, err)
	}
	wg.Wait()
}

func TestReloadHandler_RetriesSameBytesAfterTransientBuildFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ra.yml")
	writeRuntimeConfig(t, path, 5, "any", ".*")
	ra, err := NewRA(path)
	require.NoError(t, err)

	writeRuntimeConfig(t, path, 6, "any", ".*")
	reloaded, err := ra.reloadConfig(func(_ *Config) (*runtimeState, error) {
		return nil, errors.New("transient build failure")
	})

	assert.False(t, reloaded)
	require.ErrorContains(t, err, "transient build failure")
	assert.Equal(t, uint(5), ra.GetShutdownTimeout())

	reloaded, err = ra.ReloadHandler()

	require.NoError(t, err)
	assert.True(t, reloaded)
	assert.Equal(t, uint(6), ra.GetShutdownTimeout())
}

func TestReloadHandler_RejectsEachStartupOnlyField(t *testing.T) {
	tests := []struct {
		name      string
		fieldPath string
		mutate    func(*runtimeConfigFile)
	}{
		{name: "addr", fieldPath: "addr", mutate: func(cfg *runtimeConfigFile) { cfg.addr = ":9090" }},
		{name: "proxy enabled", fieldPath: "proxy.enabled", mutate: func(cfg *runtimeConfigFile) { cfg.proxyEnabled = true }},
		{name: "proxy host", fieldPath: "proxy.proxyhost", mutate: func(cfg *runtimeConfigFile) { cfg.proxyHost = "http://127.0.0.1:9998" }},
		{name: "access log exclude paths", fieldPath: "access_log.exclude_paths", mutate: func(cfg *runtimeConfigFile) {
			cfg.excludePaths = []string{"/health"}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ra.yml")
			active := defaultRuntimeConfigFile()
			writeRuntimeConfigFile(t, path, active)
			ra, err := NewRA(path)
			require.NoError(t, err)

			candidate := active
			tt.mutate(&candidate)
			writeRuntimeConfigFile(t, path, candidate)
			reloaded, err := ra.ReloadHandler()

			assert.False(t, reloaded)
			require.EqualError(t, err, "startup-only config fields changed: "+tt.fieldPath+"; restart required")
			assert.Equal(t, active.addr, ra.GetServerAddr())
			assert.Equal(t, active.proxyEnabled, ra.ProxyEnabled())
			assert.Equal(t, active.proxyHost, ra.currentState().config.Proxy.ProxyHost)
			assert.Equal(t, active.excludePaths, ra.AccessLogExcludePaths())
		})
	}
}

func TestReloadHandler_RejectsMultipleStartupOnlyFieldsDeterministicallyAndRetriesSameBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ra.yml")
	active := defaultRuntimeConfigFile()
	active.user = "old-user"
	writeRuntimeConfigFile(t, path, active)
	ra, err := NewRA(path)
	require.NoError(t, err)

	candidate := active
	candidate.addr = ":9090"
	candidate.proxyEnabled = true
	candidate.proxyHost = "http://127.0.0.1:9998"
	candidate.excludePaths = []string{"/health"}
	candidate.shutdownTimeout = 6
	candidate.user = "new-user"
	writeRuntimeConfigFile(t, path, candidate)
	wantError := "startup-only config fields changed: addr, proxy.enabled, proxy.proxyhost, access_log.exclude_paths; restart required"

	buildCalls := 0
	for range 2 {
		reloaded, reloadErr := ra.reloadConfig(func(config *Config) (*runtimeState, error) {
			buildCalls++
			return ra.buildRuntimeState(config)
		})
		assert.False(t, reloaded)
		require.EqualError(t, reloadErr, wantError)
	}
	assert.Zero(t, buildCalls)
	assert.Equal(t, active.addr, ra.GetServerAddr())
	assert.Equal(t, active.proxyHost, ra.currentState().config.Proxy.ProxyHost)
	assert.Equal(t, active.shutdownTimeout, ra.GetShutdownTimeout())
	assert.Equal(t, http.StatusOK, authStatus(t, ra, "old-user"))
	assert.Equal(t, http.StatusForbidden, authStatus(t, ra, "new-user"))
}

func TestReloadHandler_AppliesReloadableFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ra.yml")
	active := defaultRuntimeConfigFile()
	active.user = "old-user"
	writeRuntimeConfigFile(t, path, active)
	ra, err := NewRA(path)
	require.NoError(t, err)

	candidate := active
	candidate.shutdownTimeout = 9
	candidate.user = "new-user"
	writeRuntimeConfigFile(t, path, candidate)
	reloaded, err := ra.ReloadHandler()

	require.NoError(t, err)
	assert.True(t, reloaded)
	assert.Equal(t, uint(9), ra.GetShutdownTimeout())
	assert.Equal(t, http.StatusForbidden, authStatus(t, ra, "old-user"))
	assert.Equal(t, http.StatusOK, authStatus(t, ra, "new-user"))
}

func TestReloadHandler_AppliesReloadableFieldsWithEquivalentAccessLogExcludeSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ra.yml")
	active := defaultRuntimeConfigFile()
	active.user = "old-user"
	writeRuntimeConfigFile(t, path, active)
	ra, err := NewRA(path)
	require.NoError(t, err)

	candidate := active
	candidate.excludePaths = []string{" /health/ ", "metrics/", "/health", "   "}
	candidate.shutdownTimeout = 9
	candidate.user = "new-user"
	writeRuntimeConfigFile(t, path, candidate)
	reloaded, err := ra.ReloadHandler()

	require.NoError(t, err)
	assert.True(t, reloaded)
	assert.Equal(t, uint(9), ra.GetShutdownTimeout())
	assert.Equal(t, http.StatusForbidden, authStatus(t, ra, "old-user"))
	assert.Equal(t, http.StatusOK, authStatus(t, ra, "new-user"))
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

type runtimeConfigFile struct {
	addr            string
	shutdownTimeout uint
	proxyEnabled    bool
	proxyHost       string
	excludePaths    []string
	user            string
	aclPath         string
}

func defaultRuntimeConfigFile() runtimeConfigFile {
	return runtimeConfigFile{
		addr:            ":8081",
		shutdownTimeout: 5,
		proxyHost:       "http://127.0.0.1:9999",
		excludePaths:    []string{"/metrics", "/health"},
		user:            "any",
		aclPath:         ".*",
	}
}

func writeRuntimeConfig(t *testing.T, path string, shutdownTimeout uint, user, aclPath string) {
	t.Helper()
	cfg := defaultRuntimeConfigFile()
	cfg.shutdownTimeout = shutdownTimeout
	cfg.user = user
	cfg.aclPath = aclPath
	writeRuntimeConfigFile(t, path, cfg)
}

func writeRuntimeConfigFile(t *testing.T, path string, cfg runtimeConfigFile) {
	t.Helper()
	contents := fmt.Sprintf(`
addr: %q
shutdowntimeout: %d
access_log:
  exclude_paths: [%s]
proxy:
  enabled: %t
  proxyhost: %q
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
`, cfg.addr, cfg.shutdownTimeout, formatYAMLStringList(cfg.excludePaths), cfg.proxyEnabled, cfg.proxyHost, cfg.aclPath, cfg.user)
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
}

func formatYAMLStringList(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = fmt.Sprintf("%q", value)
	}
	return strings.Join(quoted, ", ")
}

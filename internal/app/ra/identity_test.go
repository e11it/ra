package ra

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/e11it/ra/pkg/auth"
)

func TestGetAuthRequest_UsesIdentityOnlyFromTrustedProxy(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	tests := []struct {
		name          string
		trusted       []string
		remoteAddr    string
		forwardedUser string
		wantUser      string
	}{
		{
			name:          "trusted exact IPv4",
			trusted:       []string{"192.0.2.10"},
			remoteAddr:    "192.0.2.10:4321",
			forwardedUser: "alice",
			wantUser:      "alice",
		},
		{
			name:          "trusted IPv4 CIDR",
			trusted:       []string{"192.0.2.0/24"},
			remoteAddr:    "192.0.2.99:4321",
			forwardedUser: "alice",
			wantUser:      "alice",
		},
		{
			name:          "trusted IPv6 CIDR",
			trusted:       []string{"2001:db8::/32"},
			remoteAddr:    "[2001:db8::10]:4321",
			forwardedUser: "alice",
			wantUser:      "alice",
		},
		{
			name:          "untrusted peer cannot assert identity",
			trusted:       []string{"192.0.2.0/24"},
			remoteAddr:    "198.51.100.7:4321",
			forwardedUser: "alice",
			wantUser:      "anon",
		},
		{
			name:          "missing forwarded identity is anonymous",
			trusted:       []string{"192.0.2.0/24"},
			remoteAddr:    "192.0.2.10:4321",
			forwardedUser: "",
			wantUser:      "anon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testRuntimeConfig()
			cfg.Identity.AuthenticatedUserHeader = "X-Authenticated-User"
			cfg.Identity.TrustedProxies = tt.trusted
			ra, err := newRAFromConfig(cfg)
			require.NoError(t, err)

			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				"/auth",
				http.NoBody,
			)
			ctx.Request.RemoteAddr = tt.remoteAddr
			ctx.Request.Header.Set("X-Original-Uri", "/topics/example")
			ctx.Request.Header.Set("X-Original-Method", "GET")
			ctx.Request.Header.Set("X-Authenticated-User", tt.forwardedUser)
			ctx.Request.SetBasicAuth("forged-basic-user", "irrelevant")

			got := ra.GetAuthRequest(ctx, false)

			assert.Equal(t, tt.wantUser, got.AuthUser)
		})
	}
}

func TestNewRAFromConfig_RejectsInvalidTrustedProxy(t *testing.T) {
	t.Parallel()

	cfg := testRuntimeConfig()
	cfg.Identity.TrustedProxies = []string{"not-an-ip"}

	ra, err := newRAFromConfig(cfg)

	assert.Nil(t, ra)
	require.Error(t, err)
	assert.ErrorContains(t, err, "parse trusted proxy")
}

func testRuntimeConfig() *Config {
	cfg := &Config{}
	cfg.Cache.Enabled = false
	cfg.Headers.AuthURL = "X-Original-Uri"
	cfg.Headers.IP = "X-Real-Ip"
	cfg.Headers.Method = "X-Original-Method"
	cfg.Identity.AuthenticatedUserHeader = "X-Authenticated-User"
	cfg.Auth.ACL = []auth.ACLRule{{
		Path:        ".*",
		Users:       []string{"any"},
		Methods:     []string{"any"},
		ContentType: []string{"any"},
	}}
	return cfg
}

package ra

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadinessError(t *testing.T) {
	t.Parallel()

	t.Run("proxy disabled is ready", func(t *testing.T) {
		t.Parallel()

		r, err := newRAFromConfig(testRuntimeConfig())
		require.NoError(t, err)
		err = r.readinessError()
		require.NoError(t, err)
	})

	t.Run("proxy enabled with reachable host is ready", func(t *testing.T) {
		t.Parallel()

		var lc net.ListenConfig
		ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, ln.Close()) })

		cfg := testRuntimeConfig()
		cfg.Proxy.Enabled = true
		cfg.Proxy.ProxyHost = "http://" + ln.Addr().String()
		r, err := newRAFromConfig(cfg)
		require.NoError(t, err)

		err = r.readinessError()
		require.NoError(t, err)
	})

	t.Run("proxy enabled with unreachable host is not ready", func(t *testing.T) {
		t.Parallel()

		cfg := testRuntimeConfig()
		cfg.Proxy.Enabled = true
		cfg.Proxy.ProxyHost = "http://127.0.0.1:1"
		r, err := newRAFromConfig(cfg)
		require.NoError(t, err)

		err = r.readinessError()
		require.Error(t, err)
	})

	t.Run("proxy enabled with invalid proxy host is not ready", func(t *testing.T) {
		t.Parallel()

		cfg := testRuntimeConfig()
		cfg.Proxy.Enabled = true
		cfg.Proxy.ProxyHost = "://bad"
		r, err := newRAFromConfig(cfg)
		require.NoError(t, err)

		err = r.readinessError()
		require.Error(t, err)
	})
}

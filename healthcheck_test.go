package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunHealthcheck(t *testing.T) {
	t.Parallel()

	t.Run("returns nil on 2xx", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		err := runHealthcheck(srv.URL, time.Second)
		require.NoError(t, err)
	})

	t.Run("returns error on non 2xx", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		err := runHealthcheck(srv.URL, time.Second)
		require.Error(t, err)
	})

	t.Run("returns error on invalid url", func(t *testing.T) {
		t.Parallel()
		err := runHealthcheck("://bad", time.Second)
		require.Error(t, err)
	})
}

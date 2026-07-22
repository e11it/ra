package ra

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/e11it/ra/pkg/validate"
)

type proxyTestCase struct {
	name              string
	restStatus        int
	restBody          string
	expectedBody      string
	expectedXError    string
	expectedHeaderSet bool
}

func TestProxyHandler_BestEffortErrorResponse(t *testing.T) {
	t.Parallel()

	tests := []proxyTestCase{
		{
			name:              "200 passthrough",
			restStatus:        http.StatusOK,
			restBody:          `{"offsets":[{"partition":0,"offset":1}]}`,
			expectedBody:      `{"offsets":[{"partition":0,"offset":1}]}`,
			expectedXError:    "",
			expectedHeaderSet: false,
		},
		{
			name:              "non-200 with valid json body",
			restStatus:        http.StatusBadRequest,
			restBody:          `{"error_code":123,"message":"bad payload"}`,
			expectedBody:      `{"error_code":123,"message":"bad payload"}`,
			expectedXError:    "123",
			expectedHeaderSet: true,
		},
		{
			name:              "non-200 with empty body",
			restStatus:        http.StatusBadRequest,
			restBody:          "",
			expectedBody:      "",
			expectedXError:    "",
			expectedHeaderSet: false,
		},
		{
			name:              "non-200 with non-json body",
			restStatus:        http.StatusBadRequest,
			restBody:          "bad request plain text",
			expectedBody:      "bad request plain text",
			expectedXError:    "",
			expectedHeaderSet: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runProxyCase(t, tt)
		})
	}
}

func runProxyCase(t *testing.T, tt proxyTestCase) {
	t.Helper()
	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/topics/dev.topic.v2" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(tt.restStatus)
		_, _ = w.Write([]byte(tt.restBody))
	}))
	defer rest.Close()

	cfg := testRuntimeConfig()
	cfg.Proxy.Enabled = true
	cfg.Proxy.ProxyHost = rest.URL

	ra, err := newRAFromConfig(cfg)
	require.NoError(t, err)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Any("/topics/*proxyPath",
		GetUUIDMiddlerware(),
		ra.GetAuthMiddlerware(true),
		ra.GetProxyHandler(),
	)

	srv := httptest.NewServer(router)
	defer srv.Close()

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		srv.URL+"/topics/dev.topic.v2",
		strings.NewReader(`{"records":[]}`),
	)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	assert.Equal(t, tt.restStatus, resp.StatusCode)
	assert.Equal(t, tt.expectedBody, string(respBody))
	assert.Equal(t, tt.expectedHeaderSet, resp.Header.Get("X-Error-Code") != "")
	assert.Equal(t, tt.expectedXError, resp.Header.Get("X-Error-Code"))
}

type noopValidator struct{}

func (noopValidator) Validate(_ []byte) *validate.Report {
	return validate.NewReport()
}

func TestAuthMiddleware_EmptyBodyWhenValidationEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		enabled         bool
		body            string
		withValidator   bool
		wantStatus      int
		wantProxyHits   int32
		wantErrorCode   int
		wantReasonMatch string
	}{
		{
			name:            "enabled rejects empty body",
			enabled:         true,
			body:            "   ",
			withValidator:   false,
			wantStatus:      http.StatusBadRequest,
			wantProxyHits:   0,
			wantErrorCode:   ErrorCodeMalformedBody,
			wantReasonMatch: "empty request body while body_validation.enabled=true",
		},
		{
			name:          "disabled keeps previous behavior",
			enabled:       false,
			body:          "",
			withValidator: false,
			wantStatus:    http.StatusOK,
			wantProxyHits: 1,
		},
		{
			name:          "enabled allows non-empty body with validator",
			enabled:       true,
			body:          `{"records":[{"value":{"x":1}}]}`,
			withValidator: true,
			wantStatus:    http.StatusOK,
			wantProxyHits: 1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var proxyHits int32
			rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				atomic.AddInt32(&proxyHits, 1)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer rest.Close()

			cfg := testRuntimeConfig()
			cfg.Proxy.Enabled = true
			cfg.Proxy.ProxyHost = rest.URL

			ra, err := newRAFromConfig(cfg)
			require.NoError(t, err)
			if tt.enabled || tt.withValidator {
				state := ra.currentState()
				replacement := *state
				config := *state.config
				config.BodyValidation.Enabled = tt.enabled
				replacement.config = &config
				if tt.withValidator {
					replacement.bodyValidator = noopValidator{}
				}
				ra.state.Store(&replacement)
			}

			gin.SetMode(gin.ReleaseMode)
			router := gin.New()
			router.Any("/topics/*proxyPath",
				GetUUIDMiddlerware(),
				ra.GetAuthMiddlerware(true),
				ra.GetProxyHandler(),
			)

			srv := httptest.NewServer(router)
			defer srv.Close()

			req, err := http.NewRequestWithContext(
				context.Background(),
				http.MethodPost,
				srv.URL+"/topics/dev.topic.v2",
				strings.NewReader(tt.body),
			)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}

			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read response body: %v", err)
			}

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			assert.Equal(t, tt.wantProxyHits, atomic.LoadInt32(&proxyHits))

			if tt.wantErrorCode != 0 {
				var payload RAErrorResponse
				if err := json.Unmarshal(respBody, &payload); err != nil {
					t.Fatalf("unmarshal error payload: %v", err)
				}
				assert.Equal(t, tt.wantErrorCode, payload.ErrorCode)
				assert.Contains(t, payload.Details.Reason, tt.wantReasonMatch)
			}
		})
	}
}

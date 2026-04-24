package ra

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/e11it/ra/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type allowAllAccessController struct{}

func (a *allowAllAccessController) Validate(_ *auth.AuthRequest) error {
	return nil
}

func TestProxyHandler_BestEffortErrorResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		restStatus        int
		restBody          string
		expectedBody      string
		expectedXError    string
		expectedHeaderSet bool
	}{
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

			rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/topics/dev.topic.v2" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				w.WriteHeader(tt.restStatus)
				_, _ = w.Write([]byte(tt.restBody))
			}))
			defer rest.Close()

			cfg := &Config{}
			cfg.Proxy.Enabled = true
			cfg.Proxy.ProxyHost = rest.URL

			ra := &Ra{
				config: cfg,
				auth:   &allowAllAccessController{},
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

			req, err := http.NewRequest(http.MethodPost, srv.URL+"/topics/dev.topic.v2", strings.NewReader(`{"records":[]}`))
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read response body: %v", err)
			}

			assert.Equal(t, tt.restStatus, resp.StatusCode)
			assert.Equal(t, tt.expectedBody, string(respBody))
			assert.Equal(t, tt.expectedHeaderSet, resp.Header.Get("X-Error-Code") != "")
			assert.Equal(t, tt.expectedXError, resp.Header.Get("X-Error-Code"))
		})
	}
}

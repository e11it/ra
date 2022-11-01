package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/e11it/ra/internal/app/ra"
	"github.com/gin-gonic/gin"
)

// Helper function to process a request and test its response
func testHTTPResponse(t *testing.T, r *gin.Engine, req *http.Request, f func(w *httptest.ResponseRecorder) bool) {
	// Create a response recorder
	w := httptest.NewRecorder()

	// Create the service and process the above request.
	r.ServeHTTP(w, req)

	if !f(w) {
		t.Fail()
	}
}

func testMiddlewareRequest(t *testing.T, r *gin.Engine, expectedHTTPCode int) {
	req, _ := http.NewRequest("GET", "/", nil)

	testHTTPResponse(t, r, req, func(w *httptest.ResponseRecorder) bool {
		return w.Code == expectedHTTPCode
	})
}

func createTestingAuthRouter(path string) *gin.Engine {
	newRa, err := ra.NewRA(path)
	if err != nil {
		return nil
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.Use(newRa.GetAuthMiddlerware())
	router.GET("/auth", func(c *gin.Context) {
		c.String(http.StatusOK, "Auth")
	})
	return router
}

func testGetAuthServer() *gin.Engine {
	// cfg := new(config)
	// cfg.Auth.Prefix = "/topics/"
	// cfg.Auth.URLValidReg = `^\d{3}-0\.[a-z0-9-]+\.(db|cdc|cmd|sys|log|tmp)\.[a-z0-9-.]+\.\d+$`
	// cfg.Auth.ContentTypeValidReg = `^(application/vnd.kafka.avro.v1+json)$`

	// cfg.Auth.ACL = []auth.AclRule{
	// 	{
	// 		Path:        `000-0.+?`,
	// 		Users:       []string{"sap"},
	// 		Methods:     []string{"any"},
	// 		ContentType: []string{`application/vnd.kafka.avro.v1+json`, `application/vnd.kafka.binary.v2+json`},
	// 	},
	// 	{
	// 		Path:        `000-0.sap-erp\.+?`,
	// 		Users:       []string{"sap"},
	// 		Methods:     []string{"any"},
	// 		ContentType: []string{`application/vnd.kafka.binary.v1+json`, `application/vnd.kafka.binary.v2+json`},
	// 	},
	// }

	return createTestingAuthRouter("example/_test/auth_server_config.yml")
}

func testGetSRServer() *gin.Engine {
	// Schema registry config
	// cfg := new(config)

	// cfg.Auth.ACL = []auth.AclRule{
	// 	{
	// 		Path:        `.*`,
	// 		Users:       []string{"any"},
	// 		Methods:     []string{"GET"},
	// 		ContentType: []string{"any"},
	// 	},
	// }

	return createTestingAuthRouter("example/_test/sr_server_config.yml")
}

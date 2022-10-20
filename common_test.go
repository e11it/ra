package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/e11it/ra/auth"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
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

func testGetAuthServer() *gin.Engine {
	cfg := new(config)
	cfg.Auth.SetDefauls()
	cfg.Auth.Prefix = "/topics/"
	cfg.Auth.URLValidReg = `^\d{3}-0\.[a-z0-9-]+\.(db|cdc|cmd|sys|log|tmp)\.[a-z0-9-.]+\.\d+$`
	cfg.Auth.ContentTypeValidReg = `^(application/vnd.kafka.avro.v1+json)$`

	cfg.Auth.ACL = []auth.ACLCfg{
		{
			Path:        `000-0.+?`,
			Users:       []string{"sap"},
			Methods:     []string{"any"},
			ContentType: []string{`application/vnd.kafka.avro.v1+json`, `application/vnd.kafka.binary.v2+json`},
		},
		{
			Path:        `000-0.sap-erp\.+?`,
			Users:       []string{"sap"},
			Methods:     []string{"any"},
			ContentType: []string{`application/vnd.kafka.binary.v1+json`, `application/vnd.kafka.binary.v2+json`},
		},
	}

	auth_m, err := auth.NewAuth(&cfg.Auth)
	if err != nil {
		log.WithError(err).Fatalln("Can't init auth module")
	}

	router, err := createAuthRouter(auth_m)
	if err != nil {
		return nil
	}

	return router
}

func testGetSRServer() *gin.Engine {
	// Schema registry config
	cfg := new(config)
	cfg.Auth.SetDefauls()

	cfg.Auth.ACL = []auth.ACLCfg{
		{
			Path:        `.*`,
			Users:       []string{"any"},
			Methods:     []string{"GET"},
			ContentType: []string{"any"},
		},
	}

	auth_m, err := auth.NewAuth(&cfg.Auth)
	if err != nil {
		log.WithError(err).Fatalln("Can't init auth module")
	}

	router, err := createAuthRouter(auth_m)
	if err != nil {
		return nil
	}

	return router
}

package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/e11it/ra/internal/app/ra"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestMain(m *testing.M) {
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestAuthRequest(t *testing.T) {
	router := testGetAuthServer()
	assert.NotNilf(t, router, "Error init router")

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/auth", nil)
	req.Header.Set("Content-Type", "application/vnd.kafka.binary.v2+json; charset=utf-8")
	req.Header.Set("X-Real-Ip", "10.48.5.59")
	req.Header.Set("X-Original-Uri", "/topics/000-0.sap-erp.db.operations.orders05.0")
	req.Header.Set("X-Original-Method", "POST")
	req.Header.Set("X-Service", "kafka-rest")
	req.Header.Set("Authorization", "Basic c2FwOnNlQzIzc0JGanV0azg5TnY=")
	router.ServeHTTP(w, req)

	// assert.False(t, called)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthAnyRequest(t *testing.T) {
	router := testGetAuthServer()
	assert.NotNilf(t, router, "Error init router")

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/auth", nil)
	req.Header.Set("Content-Type", "application/vnd.kafka.json.v2+json")
	req.Header.Set("X-Real-Ip", "10.48.5.59")
	req.Header.Set("X-Original-Uri", "/topics/000-0.iba.db.notify.datfiles-go.0")
	req.Header.Set("X-Original-Method", "POST")
	req.Header.Set("X-Service", "kafka-rest")
	req.Header.Set("Authorization", "Basic c2FwOnNlQzIzc0JGanV0azg5TnY=")
	router.ServeHTTP(w, req)

	// assert.False(t, called)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBadRequest(t *testing.T) {
	router := testGetAuthServer()
	assert.NotNilf(t, router, "Error init router")

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/auth", nil)
	req.Header.Set("Content-Type", "application/vnd.kafka.binary.v2+json")
	req.Header.Set("X-Real-Ip", "10.48.5.59")
	req.Header.Set("X-Original-Uri", "/topics/000-1.sap-erp.db.operations.orders05.0")
	req.Header.Set("X-Original-Method", "POST")
	req.Header.Set("X-Service", "kafka-rest")
	req.Header.Set("Authorization", "Basic c2FwOnNlQzIzc0JGanV0azg5TnY=")
	router.ServeHTTP(w, req)

	// assert.False(t, called)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSchemaRegistrySuccess(t *testing.T) {
	router := testGetSRServer()
	assert.NotNilf(t, router, "Error init router")

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/auth", nil)
	req.Header.Set("Content-Type", "application/vnd.kafka.binary.v2+json")
	req.Header.Set("X-Real-Ip", "10.48.5.59")
	req.Header.Set("X-Original-Uri", "/something")
	req.Header.Set("X-Original-Method", "GET")
	req.Header.Set("X-Service", "kafka-rest")
	req.Header.Set("Authorization", "Basic c2FwOnNlQzIzc0JGanV0azg5TnY=")
	router.ServeHTTP(w, req)

	// assert.False(t, called)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSchemaRegistryDeny(t *testing.T) {
	router := testGetSRServer()
	assert.NotNilf(t, router, "Error init router")

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/auth", nil)
	req.Header.Set("Content-Type", "application/vnd.kafka.binary.v2+json")
	req.Header.Set("X-Real-Ip", "10.48.5.59")
	req.Header.Set("X-Original-Uri", "/something")
	req.Header.Set("X-Original-Method", "DELETE")
	req.Header.Set("X-Service", "kafka-rest")
	req.Header.Set("Authorization", "Basic c2FwOnNlQzIzc0JGanV0azg5TnY=")
	router.ServeHTTP(w, req)

	// assert.False(t, called)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func BenchmarkFiber(b *testing.B) {
	ra, err := ra.NewRA("example/_test/auth_server_config.yml")
	if err != nil {
		b.Fatal(err)
	}

	app := fiber.New(fiber.Config{
		ReadTimeout: 2 * time.Second,
		IdleTimeout: 30 * time.Second,
	})
	app.Post("/auth", ra.GetFiberAuthMiddlerware(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	h := app.Handler()

	fctx := &fasthttp.RequestCtx{}
	fctx.Request.Header.SetMethod("POST")
	fctx.Request.SetRequestURI("/auth")
	fctx.Request.Header.Set("Content-Type", "application/vnd.kafka.binary.v2+json")
	fctx.Request.Header.Set("X-Real-Ip", "10.48.5.59")
	fctx.Request.Header.Set("X-Original-Uri", "000-0.sap-erp.db.operations.orders05.0")
	fctx.Request.Header.Set("X-Original-Method", "POST")
	fctx.Request.Header.Set("X-Service", "kafka-rest")
	fctx.Request.Header.Set("Authorization", "Basic c2FwOnNlQzIzc0JGanV0azg5TnY=")

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		h(fctx)
	}

	utils.AssertEqual(b, fiber.StatusOK, fctx.Response.Header.StatusCode())
}

func BenchmarkAuthRequest(b *testing.B) {
	router := testGetAuthServer()
	assert.NotNilf(b, router, "Error init router")

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("POST", "/auth", strings.NewReader("adfadsf"))
	req.Header.Set("Content-Type", "application/vnd.kafka.binary.v2+json")
	req.Header.Set("X-Real-Ip", "10.48.5.59")
	req.Header.Set("X-Original-Uri", "000-0.sap-erp.db.operations.orders05.0")
	req.Header.Set("X-Original-Method", "POST")
	req.Header.Set("X-Service", "kafka-rest")
	req.Header.Set("Authorization", "Basic c2FwOnNlQzIzc0JGanV0azg5TnY=")

	router.ServeHTTP(w, req)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, req)
	}
	// assert.False(t, called)
	// assert.Equal(t, http.StatusOK, w.Code)
}

//func TestPingRoute(t *testing.T) {
//	// The setupServer method, that we previously refactored
//	// is injected into a test server
//	ts := httptest.NewServer(testGetAuthServer())
//	// Shut down the server and block until all requests have gone through
//	defer ts.Close()
//
//	// Make a request to our server with the {base url}/ping
//	resp, err := http.Get(fmt.Sprintf("%s/", ts.URL))
//
//	if err != nil {
//		t.Fatalf("Expected no error, got %v", err)
//	}
//
//	if resp.StatusCode != 200 {
//		t.Fatalf("Expected status code 200, got %v", resp.StatusCode)
//	}
//
//	val, ok := resp.Header["Content-Type"]
//
//	// Assert that the "content-type" header is actually set
//	if !ok {
//		t.Fatalf("Expected Content-Type header to be set")
//	}
//
//	// Assert that it was set as expected
//	if val[0] != "application/json; charset=utf-8" {
//		t.Fatalf("Expected \"application/json; charset=utf-8\", got %s", val[0])
//	}
//}
//

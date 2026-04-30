package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type raErrorResp struct {
	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
	Details   struct {
		TraceID string `json:"trace_id"`
	} `json:"details"`
}

func assertDenyResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	var got raErrorResp
	err := json.Unmarshal(w.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.Equal(t, 40301, got.ErrorCode)
	assert.Equal(t, "Ra: auth denied", got.Message)
}

func TestMain(m *testing.M) {
	logrus.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func TestAuthRequest(t *testing.T) {
	router := testGetAuthServer()
	assert.NotNilf(t, router, "Error init router")

	w := httptest.NewRecorder()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/auth", http.NoBody)
	req.Header.Set("Content-Type", "application/vnd.kafka.binary.v2+json; charset=utf-8")
	req.Header.Set("X-Real-Ip", "10.48.5.59")
	req.Header.Set("X-Original-Uri", "/topics/000-0.sap-erp.db.operations.orders05.0")
	req.Header.Set("X-Original-Method", "POST")
	req.Header.Set("X-Service", "kafka-rest")
	req.Header.Set("Authorization", "Basic c2FwOnNlQzIzc0JGanV0azg5TnY=")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthRequest2(t *testing.T) {
	router := testGetAuthServer()
	assert.NotNilf(t, router, "Error init router")

	w := httptest.NewRecorder()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/auth", http.NoBody)
	req.Header.Set("Content-Type", "application/vnd.kafka.binary.v2+json; charset=utf-8")
	req.Header.Set("X-Real-Ip", "10.48.5.59")
	req.Header.Set("X-Original-Uri", "/topics/000-0.capital.db.operations.orders05.0")
	req.Header.Set("X-Original-Method", "POST")
	req.Header.Set("X-Service", "kafka-rest")
	req.SetBasicAuth("CapitalUserName", "passwordHere")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthAnyRequest(t *testing.T) {
	router := testGetAuthServer()
	assert.NotNilf(t, router, "Error init router")

	w := httptest.NewRecorder()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/auth", http.NoBody)
	req.Header.Set("Content-Type", "application/vnd.kafka.json.v2+json")
	req.Header.Set("X-Real-Ip", "10.48.5.59")
	req.Header.Set("X-Original-Uri", "/topics/000-0.iba.db.notify.datfiles-go.0")
	req.Header.Set("X-Original-Method", "POST")
	req.Header.Set("X-Service", "kafka-rest")
	req.Header.Set("Authorization", "Basic c2FwOnNlQzIzc0JGanV0azg5TnY=")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBadRequest(t *testing.T) {
	router := testGetAuthServer()
	assert.NotNilf(t, router, "Error init router")

	w := httptest.NewRecorder()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/auth", http.NoBody)
	req.Header.Set("Content-Type", "application/vnd.kafka.binary.v2+json")
	req.Header.Set("X-Real-Ip", "10.48.5.59")
	req.Header.Set("X-Original-Uri", "/topics/000-1.sap-erp.db.operations.orders05.0")
	req.Header.Set("X-Original-Method", "POST")
	req.Header.Set("X-Service", "kafka-rest")
	req.Header.Set("Authorization", "Basic c2FwOnNlQzIzc0JGanV0azg5TnY=")
	router.ServeHTTP(w, req)
	assertDenyResponse(t, w)
}

func TestSchemaRegistrySuccess(t *testing.T) {
	router := testGetSRServer()
	assert.NotNilf(t, router, "Error init router")

	w := httptest.NewRecorder()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/auth", http.NoBody)
	req.Header.Set("Content-Type", "application/vnd.kafka.binary.v2+json")
	req.Header.Set("X-Real-Ip", "10.48.5.59")
	req.Header.Set("X-Original-Uri", "/something")
	req.Header.Set("X-Original-Method", "GET")
	req.Header.Set("X-Service", "kafka-rest")
	req.Header.Set("Authorization", "Basic c2FwOnNlQzIzc0JGanV0azg5TnY=")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSchemaRegistrySuccess2(t *testing.T) {
	router := testGetSRServer()
	assert.NotNilf(t, router, "Error init router")

	w := httptest.NewRecorder()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/auth", http.NoBody)
	req.Header.Set("Content-Type", "")
	req.Header.Set("X-Real-Ip", "10.48.5.59")
	req.Header.Set("X-Original-Uri", "/subjects/000-0.namespace.db.cool-name.0-value/versions/latest")
	req.Header.Set("X-Original-Method", "GET")
	req.Header.Set("X-Service", "kafka-rest")
	req.Header.Set("Authorization", "Basic c2FwOnNlQzIzc0JGanV0azg5TnY=")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSchemaRegistryDeny(t *testing.T) {
	router := testGetSRServer()
	assert.NotNilf(t, router, "Error init router")

	w := httptest.NewRecorder()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/auth", http.NoBody)
	req.Header.Set("Content-Type", "application/vnd.kafka.binary.v2+json")
	req.Header.Set("X-Real-Ip", "10.48.5.59")
	req.Header.Set("X-Original-Uri", "/something")
	req.Header.Set("X-Original-Method", "DELETE")
	req.Header.Set("X-Service", "kafka-rest")
	req.Header.Set("Authorization", "Basic c2FwOnNlQzIzc0JGanV0azg5TnY=")
	router.ServeHTTP(w, req)
	assertDenyResponse(t, w)
}

func TestBodyValidation_GetRequestNotValidated(t *testing.T) {
	router := testGetBodyValidationServer()
	assert.NotNilf(t, router, "Error init router")

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/auth", http.NoBody)
	req.Header.Set("X-Original-Uri", "/topics/888-8.example.db.awesome.0")
	req.Header.Set("X-Original-Method", "GET")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code,
		"body validation применяется только к POST — GET должен пройти мимо")
}

func BenchmarkAuthRequest(b *testing.B) {
	router := testGetAuthServer()
	assert.NotNilf(b, router, "Error init router")

	w := httptest.NewRecorder()

	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth", strings.NewReader("adfadsf"))
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
}

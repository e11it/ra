package ra

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics stores Prometheus collectors used by RA.
type Metrics struct {
	registry *prometheus.Registry

	httpRequestsTotal *prometheus.CounterVec
	httpDuration      *prometheus.HistogramVec
	httpInFlight      prometheus.Gauge

	authDeniedTotal          prometheus.Counter
	bodyValidationFailed     prometheus.Counter
	proxyUpstreamErrorsTotal prometheus.Counter
	configReloadTotal        *prometheus.CounterVec
}

// NewMetrics initializes a dedicated registry and RA collectors.
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	factory := promauto.With(reg)

	m := &Metrics{
		registry: reg,
		httpRequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "ra_http_requests_total",
			Help: "Total HTTP requests handled by RA.",
		}, []string{"framework", "method", "route", "status_class", "outcome"}),
		httpDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "ra_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"framework", "method", "route"}),
		httpInFlight: factory.NewGauge(prometheus.GaugeOpts{
			Name: "ra_http_inflight_requests",
			Help: "Current number of in-flight HTTP requests.",
		}),
		authDeniedTotal: factory.NewCounter(prometheus.CounterOpts{
			Name: "ra_auth_denied_total",
			Help: "Number of requests denied by auth checks.",
		}),
		bodyValidationFailed: factory.NewCounter(prometheus.CounterOpts{
			Name: "ra_body_validation_failed_total",
			Help: "Number of requests rejected by body validation.",
		}),
		proxyUpstreamErrorsTotal: factory.NewCounter(prometheus.CounterOpts{
			Name: "ra_proxy_upstream_errors_total",
			Help: "Number of proxy requests that ended with upstream/server errors.",
		}),
		configReloadTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "ra_config_reload_total",
			Help: "Config reload attempts by result.",
		}, []string{"result"}),
	}

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return m
}

// Handler returns Prometheus exposition endpoint.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// ObserveReload increments reload counter with result label.
func (m *Metrics) ObserveReload(result string) {
	m.configReloadTotal.WithLabelValues(result).Inc()
}

// GinMiddleware records HTTP metrics for Gin handlers.
func (m *Metrics) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		m.httpInFlight.Inc()
		start := time.Now()
		c.Next()
		m.httpInFlight.Dec()

		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}
		method := c.Request.Method
		status := c.Writer.Status()

		m.observeRequest("gin", method, route, status, c.Writer.Header().Get("X-RA-ERROR"))
		m.httpDuration.WithLabelValues("gin", method, route).Observe(time.Since(start).Seconds())
	}
}

// FiberMiddleware records HTTP metrics for Fiber handlers.
func (m *Metrics) FiberMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		m.httpInFlight.Inc()
		start := time.Now()
		err := c.Next()
		m.httpInFlight.Dec()

		route := c.Route().Path
		if route == "" {
			route = "unknown"
		}
		status := c.Response().StatusCode()
		m.observeRequest("fiber", c.Method(), route, status, c.GetRespHeader("X-RA-ERROR"))
		m.httpDuration.WithLabelValues("fiber", c.Method(), route).Observe(time.Since(start).Seconds())
		return err
	}
}

func (m *Metrics) observeRequest(framework, method, route string, status int, raErr string) {
	statusClass := strconv.Itoa(status/100) + "xx"
	outcome := requestOutcome(status, route, raErr)
	m.httpRequestsTotal.WithLabelValues(framework, method, route, statusClass, outcome).Inc()

	switch outcome {
	case "auth_denied":
		m.authDeniedTotal.Inc()
	case "body_validation_failed":
		m.bodyValidationFailed.Inc()
	case "proxy_upstream_error":
		m.proxyUpstreamErrorsTotal.Inc()
	}
}

func requestOutcome(status int, route, raErr string) string {
	if status == http.StatusForbidden {
		return "auth_denied"
	}
	if status == http.StatusUnprocessableEntity && raErr != "" {
		return "body_validation_failed"
	}
	if strings.HasPrefix(route, "/topics/") && status >= http.StatusInternalServerError {
		return "proxy_upstream_error"
	}
	if status >= http.StatusBadRequest {
		return "client_or_server_error"
	}
	return "ok"
}

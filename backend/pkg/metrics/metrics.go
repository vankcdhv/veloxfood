// Package metrics exposes Prometheus instrumentation shared by every
// service: HTTP request counters/latency, gRPC client error counter and the
// circuit-breaker state gauge. Each service mounts /metrics via pkg/app.
package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests by service, method, route and status code.",
	}, []string{"service", "method", "route", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency by service, method and route.",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
	}, []string{"service", "method", "route"})

	grpcClientErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "grpc_client_errors_total",
		Help: "Failed outbound gRPC calls by target and status code.",
	}, []string{"target", "code"})

	circuitBreakerState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "circuit_breaker_state",
		Help: "Circuit breaker state per gRPC target (0=closed, 1=half-open, 2=open).",
	}, []string{"target"})
)

// Handler returns the /metrics endpoint handler.
func Handler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) { h.ServeHTTP(c.Writer, c.Request) }
}

// HTTPMiddleware records request count + latency. Uses the matched route
// template (c.FullPath) — not the raw URL — to keep label cardinality bounded.
func HTTPMiddleware(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched" // 404s share one label value
		}
		httpRequestsTotal.WithLabelValues(service, c.Request.Method, route, strconv.Itoa(c.Writer.Status())).Inc()
		httpRequestDuration.WithLabelValues(service, c.Request.Method, route).Observe(time.Since(start).Seconds())
	}
}

// RecordGRPCClientError counts a failed outbound call.
func RecordGRPCClientError(target, code string) {
	grpcClientErrorsTotal.WithLabelValues(target, code).Inc()
}

// SetBreakerState publishes the current breaker state for a target.
func SetBreakerState(target string, state float64) {
	circuitBreakerState.WithLabelValues(target).Set(state)
}

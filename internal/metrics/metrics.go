// Package metrics provides Prometheus HTTP metrics: a middleware that records
// request counts/latencies and the /metrics scrape handler.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests by service, method, route, and status.",
	}, []string{"service", "method", "route", "status"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency by service, method, and route.",
		Buckets: prometheus.DefBuckets,
	}, []string{"service", "method", "route"})
)

func Handler() http.Handler { return promhttp.Handler() }

func Middleware(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = "unknown"
			}
			httpRequests.WithLabelValues(service, r.Method, route, strconv.Itoa(ww.Status())).Inc()
			httpDuration.WithLabelValues(service, r.Method, route).Observe(time.Since(start).Seconds())
		})
	}
}

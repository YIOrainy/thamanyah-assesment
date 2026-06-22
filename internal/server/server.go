package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/swaggest/swgui/v5emb"

	"github.com/yazeedalorainy/thmanyah/internal/metrics"
)

const (
	defaultReadTimeout  = 5 * time.Second
	defaultWriteTimeout = 10 * time.Second

	rateLimitRequests = 100 // per IP
	rateLimitWindow   = time.Second
)

type Server struct {
	srv *http.Server
}

// Options configures cross-cutting server features.
type Options struct {
	OpenAPISpec    []byte // when set, serves /openapi.yaml + Swagger UI at /docs
	MetricsEnabled bool   // when true, serves Prometheus /metrics + request metrics
	ServiceName    string // metrics label (e.g. "cms", "discovery")
}

func New(addr string, app http.Handler, opts Options) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer)

	// /healthz, /metrics, and docs sit outside rate-limiting and compression.
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	if opts.MetricsEnabled {
		r.Handle("/metrics", metrics.Handler())
	}

	if len(opts.OpenAPISpec) > 0 {
		r.Get("/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/yaml")
			_, _ = w.Write(opts.OpenAPISpec)
		})
		docs := v5emb.New("Thmanyah API", "/openapi.yaml", "/docs/")
		r.Handle("/docs", docs)
		r.Handle("/docs/*", docs)
	}

	r.Group(func(r chi.Router) {
		if opts.MetricsEnabled {
			r.Use(metrics.Middleware(opts.ServiceName))
		}
		r.Use(httprate.LimitByIP(rateLimitRequests, rateLimitWindow))
		r.Use(middleware.Compress(5, "application/json", "application/problem+json", "text/plain"))
		r.Mount("/", app)
	})

	return &Server{srv: &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
	}}
}

func (s *Server) Start() {
	go func() {
		slog.Info("http server listening", "addr", s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server error", "error", err)
		}
	}()
}

func (s *Server) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		slog.Error("http server shutdown", "error", err)
	}
}

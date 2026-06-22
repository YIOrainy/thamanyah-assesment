package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/yazeedalorainy/thmanyah/api"
	"github.com/yazeedalorainy/thmanyah/internal/cache"
	"github.com/yazeedalorainy/thmanyah/internal/config"
	"github.com/yazeedalorainy/thmanyah/internal/discovery"
	"github.com/yazeedalorainy/thmanyah/internal/logging"
	"github.com/yazeedalorainy/thmanyah/internal/profiling"
	"github.com/yazeedalorainy/thmanyah/internal/server"
	"github.com/yazeedalorainy/thmanyah/internal/store"
)

func main() {
	configPath := flag.String("config", "config.example.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}
	logging.Init(cfg.LogLevel)
	slog.Info("config loaded",
		"service", "discovery",
		"path", *configPath,
		"log_level", cfg.LogLevel,
		"metrics_enabled", cfg.Observability.Metrics.Enabled,
		"profiling_enabled", cfg.Observability.Profiling.Enabled,
	)
	stopProfiling := profiling.Start(cfg.Observability.Profiling, "thmanyah.discovery")

	repos, err := store.New(cfg.Store)
	if err != nil {
		slog.Error("build store", "error", err)
		os.Exit(1)
	}
	c, closeCache := cache.New(cfg.Cache)
	router := discovery.NewRouter(repos.Shows, repos.Episodes, repos.Searcher, c, cfg.Cache.Redis.TTL.Duration())

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	addr := net.JoinHostPort(cfg.Server.Host, cfg.Server.DiscoveryPort)
	srv := server.New(addr, router, server.Options{
		OpenAPISpec:    api.Spec,
		MetricsEnabled: cfg.Observability.Metrics.Enabled,
		ServiceName:    "discovery",
	})
	srv.Start()
	slog.Info("discovery started", "addr", addr, "api_prefix", discovery.APIPrefix)

	<-ctx.Done()
	slog.Info("shutting down", "service", "discovery")
	srv.Close()
	closeCache()
	repos.Close()
	stopProfiling()
	slog.Info("shutdown complete", "service", "discovery")
}

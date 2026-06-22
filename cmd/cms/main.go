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
	"github.com/yazeedalorainy/thmanyah/internal/auth"
	"github.com/yazeedalorainy/thmanyah/internal/cms"
	"github.com/yazeedalorainy/thmanyah/internal/config"
	"github.com/yazeedalorainy/thmanyah/internal/ingestion"
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
		"service", "cms",
		"path", *configPath,
		"log_level", cfg.LogLevel,
		"metrics_enabled", cfg.Observability.Metrics.Enabled,
		"profiling_enabled", cfg.Observability.Profiling.Enabled,
	)
	stopProfiling := profiling.Start(cfg.Observability.Profiling, "thmanyah.cms")

	repos, err := store.New(cfg.Store)
	if err != nil {
		slog.Error("build store", "error", err)
		os.Exit(1)
	}
	if err := repos.BootstrapAdmin(context.Background(), cfg.Auth.BootstrapAdmin); err != nil {
		slog.Error("bootstrap admin", "error", err)
		os.Exit(1)
	}

	jwt := auth.NewJWT(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL.Duration())
	importSvc := ingestion.NewService(ingestion.Importers(cfg.Ingestion), repos.Shows, repos.Episodes, repos.Refs)
	router := cms.NewRouter(repos.Shows, repos.Episodes, repos.Users, jwt, importSvc)

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	addr := net.JoinHostPort(cfg.Server.Host, cfg.Server.CMSPort)
	srv := server.New(addr, router, server.Options{
		OpenAPISpec:    api.Spec,
		MetricsEnabled: cfg.Observability.Metrics.Enabled,
		ServiceName:    "cms",
	})
	srv.Start()
	slog.Info("cms started", "addr", addr, "api_prefix", cms.APIPrefix)

	<-ctx.Done()
	slog.Info("shutting down", "service", "cms")
	srv.Close()
	repos.Close()
	stopProfiling()
	slog.Info("shutdown complete", "service", "cms")
}

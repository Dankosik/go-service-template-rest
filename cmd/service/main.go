package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/app/ping"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/domain"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.Log.Level}))
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	metrics := telemetry.New()

	probes := make([]domain.ReadinessProbe, 0, 1)
	if cfg.Postgres.DSN != "" {
		pg, err := postgres.New(ctx, cfg.Postgres.DSN)
		if err != nil {
			log.Error("postgres init failed", "err", err)
			os.Exit(1)
		}
		defer pg.Close()
		probes = append(probes, pg)
	}

	healthSvc := health.New(probes...)
	pingSvc := ping.New()

	handler := httpx.NewRouter(log, httpx.Handlers{
		Health: healthSvc,
		Ping:   pingSvc,
	}, metrics)

	srv := httpx.New(httpx.Config{
		Addr:              cfg.HTTP.Addr,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
		MaxHeaderBytes:    cfg.HTTP.MaxHeaderBytes,
	}, handler)

	runErr := make(chan error, 1)
	go func() {
		log.Info("http server started", "addr", cfg.HTTP.Addr, "env", cfg.Env)
		runErr <- srv.Run()
	}()

	var failed bool
	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-runErr:
		if err != nil {
			failed = true
			log.Error("http server stopped with error", "err", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("graceful shutdown failed", "err", err)
		os.Exit(1)
	}

	if failed {
		os.Exit(1)
	}

	log.Info("shutdown complete")
}

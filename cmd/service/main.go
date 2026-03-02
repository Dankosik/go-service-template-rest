package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/app/ping"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/domain"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

const telemetryShutdownTimeout = 5 * time.Second

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() (runErr error) {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.Log.Level}))
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tracingShutdown, err := telemetry.SetupTracing(ctx, telemetry.TracingConfig{
		ServiceName:      cfg.OTel.ServiceName,
		DeploymentEnv:    cfg.Env,
		TracesSampler:    cfg.OTel.TracesSampler,
		TracesSamplerArg: cfg.OTel.TracesSamplerArg,
	})
	if err != nil {
		return fmt.Errorf("setup tracing: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), telemetryShutdownTimeout)
		defer cancel()
		if err := tracingShutdown(shutdownCtx); err != nil {
			log.Error("tracing shutdown failed", "err", err)
			if runErr == nil {
				runErr = fmt.Errorf("shutdown tracing: %w", err)
			}
		}
	}()

	metrics := telemetry.New()

	probes := make([]domain.ReadinessProbe, 0, 1)
	if cfg.Postgres.DSN != "" {
		pg, err := postgres.New(ctx, cfg.Postgres.DSN)
		if err != nil {
			return fmt.Errorf("postgres init failed: %w", err)
		}
		defer pg.Close()
		probes = append(probes, pg)
	}

	healthSvc := health.New(probes...)
	pingSvc := ping.New()

	handler := httpx.NewRouter(
		log,
		httpx.Handlers{
			Health: healthSvc,
			Ping:   pingSvc,
		},
		metrics,
		httpx.RouterConfig{MaxBodyBytes: cfg.HTTP.MaxBodyBytes},
	)

	srv := httpx.New(httpx.Config{
		Addr:              cfg.HTTP.Addr,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
		MaxHeaderBytes:    cfg.HTTP.MaxHeaderBytes,
	}, handler)

	runErrCh := make(chan error, 1)
	go func() {
		log.Info("http server started", "addr", cfg.HTTP.Addr, "env", cfg.Env)
		runErrCh <- srv.Run()
	}()

	var serverErr error
	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-runErrCh:
		serverErr = err
		if serverErr != nil {
			log.Error("http server stopped with error", "err", serverErr)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}
	if serverErr != nil {
		return fmt.Errorf("http server stopped with error: %w", serverErr)
	}

	log.Info("shutdown complete")
	return nil
}

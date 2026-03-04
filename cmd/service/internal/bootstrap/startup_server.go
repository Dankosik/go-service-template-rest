package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/config"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func serveHTTPRuntime(
	signalCtx context.Context,
	bootstrapCtx context.Context,
	bootstrapSpan trace.Span,
	cfg config.Config,
	log *slog.Logger,
	metrics *telemetry.Metrics,
	deployTelemetry *deployTelemetryRecorder,
	startupLifecycleStartedAt time.Time,
	healthSvc *health.Service,
	srv *httpx.Server,
) error {
	listener, err := net.Listen("tcp", cfg.HTTP.Addr)
	if err != nil {
		bootstrapSpan.RecordError(err)
		bootstrapSpan.SetAttributes(
			attribute.String("result", "error"),
			attribute.String("error.type", "startup_error"),
			attribute.String("failed.stage", "startup.http_listen"),
		)
		metrics.IncConfigValidationFailure("startup_error")
		metrics.IncConfigStartupOutcome("rejected")
		log.Error(
			"startup_blocked",
			startupLogArgs(
				bootstrapCtx,
				"startup_probes",
				"http_listen",
				"error",
				"error.type", "startup_error",
				"err", err,
			)...,
		)
		recordAdmissionFailureWithRollback(bootstrapCtx, deployTelemetry, "startup_error", "startup", startupLifecycleStartedAt)
		return fmt.Errorf("listen http server: %w", err)
	}

	metrics.IncConfigStartupOutcome("ready")
	bootstrapSpan.SetAttributes(attribute.String("result", "success"))
	deployTelemetry.RecordAdmission(bootstrapCtx, "success", "ready", "")

	runErrCh := make(chan error, 1)
	serverStartedAt := time.Now()
	go func() {
		log.Info("http server started", "addr", listener.Addr().String(), "env", cfg.App.Env)
		runErrCh <- srv.Serve(listener)
	}()

	var serverErr error
	select {
	case <-signalCtx.Done():
		log.Info("shutdown signal received")
	case err := <-runErrCh:
		serverErr = err
		if serverErr != nil {
			log.Error("http server stopped with error", "err", serverErr)
			recordRollbackFailure(signalCtx, deployTelemetry, "runtime_error", serverStartedAt)
		}
	}

	shutdownStartedAt := time.Now()
	if err := drainAndShutdown(signalCtx, cfg.HTTP.ShutdownTimeout, healthSvc, srv); err != nil {
		recordRollbackFailure(signalCtx, deployTelemetry, "shutdown_error", shutdownStartedAt)
		return err
	}
	if serverErr != nil {
		return fmt.Errorf("http server stopped with error: %w", serverErr)
	}

	log.Info("shutdown complete")
	return nil
}

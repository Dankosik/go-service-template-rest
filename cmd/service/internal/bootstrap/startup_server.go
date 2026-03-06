package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
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
	admissionReady <-chan struct{},
	closeReadiness func(),
	shutdownDelay time.Duration,
) error {
	if err := startupRuntimeContextErr(signalCtx, bootstrapCtx); err != nil {
		return rejectHTTPStartup(
			bootstrapCtx,
			bootstrapSpan,
			metrics,
			log,
			deployTelemetry,
			startupLifecycleStartedAt,
			"startup.http_listen",
			fmt.Errorf("startup canceled before http listen: %w", err),
		)
	}

	listener, err := net.Listen("tcp", cfg.HTTP.Addr)
	if err != nil {
		return rejectHTTPStartup(
			bootstrapCtx,
			bootstrapSpan,
			metrics,
			log,
			deployTelemetry,
			startupLifecycleStartedAt,
			"startup.http_listen",
			fmt.Errorf("listen http server: %w", err),
		)
	}
	if err := startupRuntimeContextErr(signalCtx, bootstrapCtx); err != nil {
		_ = listener.Close()
		return rejectHTTPStartup(
			bootstrapCtx,
			bootstrapSpan,
			metrics,
			log,
			deployTelemetry,
			startupLifecycleStartedAt,
			"startup.http_serve",
			fmt.Errorf("startup canceled before http serve: %w", err),
		)
	}

	runErrCh := make(chan error, 1)
	go func() {
		log.Info("http server started", "addr", listener.Addr().String(), "env", cfg.App.Env)
		runErrCh <- srv.Serve(listener)
	}()

	var serverErr error
	startupWatchCh := bootstrapCtx.Done()
	startupReadyCh := admissionReady

waitForStop:
	for {
		select {
		case <-startupReadyCh:
			startupReadyCh = nil
			startupWatchCh = nil
		case <-signalCtx.Done():
			log.Info("shutdown signal received")
			closeReadiness()
			recordAdmissionFailure(signalCtx, deployTelemetry, "startup_error", "readiness", startupLifecycleStartedAt)
			break waitForStop
		case <-startupWatchCh:
			closeReadiness()
			if signalCtx.Err() != nil {
				log.Info("shutdown signal received")
				recordAdmissionFailure(signalCtx, deployTelemetry, "startup_error", "readiness", startupLifecycleStartedAt)
				break waitForStop
			}
			log.Error("startup budget exhausted before readiness", "err", bootstrapCtx.Err())
			recordAdmissionFailure(bootstrapCtx, deployTelemetry, "startup_error", "readiness", startupLifecycleStartedAt)
			break waitForStop
		case err := <-runErrCh:
			serverErr = err
			if serverErr != nil {
				closeReadiness()
				log.Error("http server stopped with error", "err", serverErr)
				recordAdmissionFailure(signalCtx, deployTelemetry, "startup_error", "readiness", startupLifecycleStartedAt)
			}
			break waitForStop
		}
	}

	if err := drainAndShutdown(signalCtx, shutdownDelay, cfg.HTTP.ShutdownTimeout, healthSvc, srv); err != nil {
		closeReadiness()
		recordAdmissionFailure(signalCtx, deployTelemetry, "startup_error", "readiness", startupLifecycleStartedAt)
		return err
	}
	if serverErr != nil {
		return fmt.Errorf("http server stopped with error: %w", serverErr)
	}

	log.Info("shutdown complete")
	return nil
}

func startupRuntimeContextErr(signalCtx context.Context, bootstrapCtx context.Context) error {
	if err := signalCtx.Err(); err != nil {
		return err
	}
	if err := bootstrapCtx.Err(); err != nil {
		return err
	}
	return nil
}

func rejectHTTPStartup(
	bootstrapCtx context.Context,
	bootstrapSpan trace.Span,
	metrics *telemetry.Metrics,
	log *slog.Logger,
	deployTelemetry *deployTelemetryRecorder,
	startupLifecycleStartedAt time.Time,
	stage string,
	err error,
) error {
	bootstrapSpan.RecordError(err)
	bootstrapSpan.SetAttributes(
		attribute.String("result", "error"),
		attribute.String("error.type", "startup_error"),
		attribute.String("failed.stage", stage),
	)
	metrics.IncConfigValidationFailure("startup_error")
	metrics.IncConfigStartupOutcome("rejected")
	log.Error(
		"startup_blocked",
		startupLogArgs(
			bootstrapCtx,
			"startup_probes",
			strings.TrimPrefix(stage, "startup."),
			"error",
			"error.type", "startup_error",
			"err", err,
		)...,
	)
	recordAdmissionFailure(bootstrapCtx, deployTelemetry, "startup_error", "startup", startupLifecycleStartedAt)
	return err
}

package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/Dankosik/privacy-sanitization-service/internal/app/health"
	"github.com/Dankosik/privacy-sanitization-service/internal/config"
	"github.com/Dankosik/privacy-sanitization-service/internal/infra/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type runtimeServer interface {
	Serve(net.Listener) error
	Shutdown(context.Context) error
}

func serveHTTPRuntime(
	signalCtx context.Context,
	bootstrapCtx context.Context,
	bootstrapSpan trace.Span,
	cfg config.Config,
	log *slog.Logger,
	metrics *telemetry.Metrics,
	healthSvc *health.Service,
	srv runtimeServer,
	readinessCheck func(context.Context) error,
	admission *startupAdmissionController,
	shutdownDelay time.Duration,
) error {
	if err := startupRuntimeContextErr(signalCtx, bootstrapCtx); err != nil {
		return rejectHTTPStartup(
			bootstrapCtx,
			bootstrapSpan,
			metrics,
			log,
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
			"startup.http_serve",
			fmt.Errorf("startup canceled before http serve: %w", err),
		)
	}

	runErrCh := make(chan error, 1)
	go func() {
		log.Info("http server started", "addr", listener.Addr().String(), "env", cfg.App.Env)
		runErrCh <- srv.Serve(listener)
	}()

	admissionCtx, cancelAdmission := context.WithCancel(bootstrapCtx)
	defer cancelAdmission()

	admissionErrCh := startStartupAdmission(admissionCtx, readinessCheck, admission)
	var serverErr error
	var terminalErr error
	startupFailureRecorded := false
	startupWatchCh := bootstrapCtx.Done()
	recordPreReadyFailure := func(stage string, err error) {
		if startupFailureRecorded || admission.Ready() {
			return
		}
		if err != nil {
			bootstrapSpan.RecordError(err)
		}
		bootstrapSpan.SetAttributes(
			attribute.String("result", "error"),
			attribute.String("error.type", "startup_error"),
			attribute.String("failed.stage", stage),
		)
		metrics.IncConfigValidationFailure("startup_error")
		metrics.IncConfigStartupOutcome("rejected")
		startupFailureRecorded = true
	}

waitForStop:
	for {
		select {
		case err := <-admissionErrCh:
			admissionErrCh = nil
			if err != nil {
				terminalErr = rejectHTTPStartup(
					bootstrapCtx,
					bootstrapSpan,
					metrics,
					log,
					"startup.readiness",
					fmt.Errorf("startup readiness check failed: %w", err),
				)
				startupFailureRecorded = true
				break waitForStop
			}
			startupWatchCh = nil
		case <-signalCtx.Done():
			log.Info("shutdown signal received")
			recordPreReadyFailure("startup.readiness", signalCtx.Err())
			break waitForStop
		case <-startupWatchCh:
			if signalCtx.Err() != nil {
				log.Info("shutdown signal received")
				recordPreReadyFailure("startup.readiness", signalCtx.Err())
				break waitForStop
			}
			terminalErr = fmt.Errorf("startup budget exhausted before readiness: %w", bootstrapCtx.Err())
			log.Error("startup budget exhausted before readiness", "err", terminalErr)
			recordPreReadyFailure("startup.readiness", terminalErr)
			break waitForStop
		case err := <-runErrCh:
			serverErr = err
			if !admission.Ready() {
				if serverErr != nil {
					terminalErr = fmt.Errorf("http server stopped before readiness: %w", serverErr)
				} else {
					terminalErr = errors.New("http server stopped before readiness")
				}
				recordPreReadyFailure("startup.http_serve", terminalErr)
			}
			if serverErr != nil {
				log.Error("http server stopped with error", "err", serverErr)
			}
			break waitForStop
		}
	}

	cancelAdmission()

	effectiveShutdownDelay := shutdownDelay
	if !admission.Ready() {
		effectiveShutdownDelay = 0
	}
	if err := drainAndShutdown(signalCtx, effectiveShutdownDelay, cfg.HTTP.ShutdownTimeout, healthSvc, srv); err != nil {
		recordPreReadyFailure("startup.shutdown", err)
		if terminalErr != nil {
			return errors.Join(terminalErr, err)
		}
		return err
	}
	if terminalErr != nil {
		return terminalErr
	}
	if serverErr != nil {
		return fmt.Errorf("http server stopped with error: %w", serverErr)
	}

	log.Info("shutdown complete")
	return nil
}

func startStartupAdmission(
	bootstrapCtx context.Context,
	readinessCheck func(context.Context) error,
	admission *startupAdmissionController,
) <-chan error {
	resultCh := make(chan error, 1)

	go func() {
		readyCtx, cancel := withStageBudget(bootstrapCtx, startupAdmissionBudget)
		defer cancel()

		if err := readyCtx.Err(); err != nil {
			resultCh <- err
			return
		}
		if readinessCheck != nil {
			if err := readinessCheck(readyCtx); err != nil {
				resultCh <- err
				return
			}
		}

		admission.MarkReady(readyCtx)
		resultCh <- nil
	}()

	return resultCh
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
	return err
}

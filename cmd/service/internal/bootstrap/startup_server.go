package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel/trace"
)

type runtimeServer interface {
	Serve(net.Listener) error
	Shutdown(context.Context) error
}

type serveHTTPRuntimeArgs struct {
	signalCtx      context.Context
	bootstrapCtx   context.Context
	bootstrapSpan  trace.Span
	cfg            config.Config
	log            *slog.Logger
	metrics        *telemetry.Metrics
	healthSvc      *health.Service
	srv            runtimeServer
	readinessCheck func(context.Context) error
	admission      *startupAdmissionController
	shutdownDelay  time.Duration
}

func serveHTTPRuntime(args serveHTTPRuntimeArgs) error {
	if err := startupRuntimeContextErr(args.signalCtx, args.bootstrapCtx); err != nil {
		return rejectHTTPStartup(
			args.bootstrapCtx,
			args.bootstrapSpan,
			args.metrics,
			args.log,
			"startup.http_listen",
			fmt.Errorf("startup canceled before http listen: %w", err),
		)
	}

	listener, err := net.Listen("tcp", args.cfg.HTTP.Addr)
	if err != nil {
		return rejectHTTPStartup(
			args.bootstrapCtx,
			args.bootstrapSpan,
			args.metrics,
			args.log,
			"startup.http_listen",
			fmt.Errorf("listen http server: %w", err),
		)
	}
	if err := startupRuntimeContextErr(args.signalCtx, args.bootstrapCtx); err != nil {
		_ = listener.Close()
		return rejectHTTPStartup(
			args.bootstrapCtx,
			args.bootstrapSpan,
			args.metrics,
			args.log,
			"startup.http_serve",
			fmt.Errorf("startup canceled before http serve: %w", err),
		)
	}

	runErrCh := make(chan error, 1)
	go func() {
		args.log.Info("http server started", "addr", listener.Addr().String(), "env", args.cfg.App.Env)
		runErrCh <- args.srv.Serve(listener)
	}()

	admissionCtx, cancelAdmission := context.WithCancel(args.bootstrapCtx)
	defer cancelAdmission()

	admissionErrCh := startStartupAdmission(admissionCtx, args.readinessCheck, args.admission, args.cfg.HTTP.ReadinessTimeout)
	var serverErr error
	var terminalErr error
	startupFailureRecorded := false
	startupWatchCh := args.bootstrapCtx.Done()
	recordPreReadyFailure := func(stage string, err error) {
		if startupFailureRecorded || args.admission.Ready() {
			return
		}
		recordStartupRejection(args.bootstrapSpan, args.metrics, telemetry.StartupRejectionReasonStartupError, "startup_error", stage, err)
		startupFailureRecorded = true
	}

waitForStop:
	for {
		select {
		case err := <-admissionErrCh:
			admissionErrCh = nil
			if err != nil {
				terminalErr = rejectHTTPStartup(
					args.bootstrapCtx,
					args.bootstrapSpan,
					args.metrics,
					args.log,
					"startup.readiness",
					fmt.Errorf("startup readiness check failed: %w", err),
				)
				startupFailureRecorded = true
				break waitForStop
			}
			startupWatchCh = nil
		case <-args.signalCtx.Done():
			args.log.Info("shutdown signal received")
			recordPreReadyFailure("startup.readiness", args.signalCtx.Err())
			break waitForStop
		case <-startupWatchCh:
			if args.signalCtx.Err() != nil {
				args.log.Info("shutdown signal received")
				recordPreReadyFailure("startup.readiness", args.signalCtx.Err())
				break waitForStop
			}
			terminalErr = fmt.Errorf("startup budget exhausted before readiness: %w", args.bootstrapCtx.Err())
			args.log.Error("startup budget exhausted before readiness", "err", terminalErr)
			recordPreReadyFailure("startup.readiness", terminalErr)
			break waitForStop
		case err := <-runErrCh:
			serverErr = err
			if !args.admission.Ready() {
				if serverErr != nil {
					terminalErr = fmt.Errorf("http server stopped before readiness: %w", serverErr)
				} else {
					terminalErr = errors.New("http server stopped before readiness")
				}
				recordPreReadyFailure("startup.http_serve", terminalErr)
			}
			if serverErr != nil {
				args.log.Error("http server stopped with error", "err", serverErr)
			}
			break waitForStop
		}
	}

	cancelAdmission()

	effectiveShutdownDelay := args.shutdownDelay
	if !args.admission.Ready() {
		effectiveShutdownDelay = 0
	}
	if err := drainAndShutdown(args.signalCtx, effectiveShutdownDelay, args.cfg.HTTP.ShutdownTimeout, args.healthSvc, args.srv); err != nil {
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

	args.log.Info("shutdown complete")
	return nil
}

func startStartupAdmission(
	bootstrapCtx context.Context,
	readinessCheck func(context.Context) error,
	admission *startupAdmissionController,
	readinessTimeout time.Duration,
) <-chan error {
	resultCh := make(chan error, 1)

	go func() {
		readyCtx, cancel := withStageBudget(bootstrapCtx, readinessTimeout)
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
	recordStartupRejection(bootstrapSpan, metrics, telemetry.StartupRejectionReasonStartupError, "startup_error", stage, err)
	log.Error(
		"startup_blocked",
		startupLogArgs(
			bootstrapCtx,
			startupLogComponentStartupProbes,
			strings.TrimPrefix(stage, "startup."),
			"error",
			"error.type", "startup_error",
			"err", err,
		)...,
	)
	return err
}

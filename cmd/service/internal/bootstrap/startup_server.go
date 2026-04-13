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

type httpRuntimeWaitOutcome struct {
	ready                  bool
	terminalErr            error
	serverErr              error
	startupFailureRecorded bool
}

func serveHTTPRuntime(signalCtx context.Context, bootstrapCtx context.Context, args serveHTTPRuntimeArgs) error {
	if err := startupRuntimeContextErr(signalCtx, bootstrapCtx); err != nil {
		return rejectHTTPStartup(
			bootstrapCtx,
			args.bootstrapSpan,
			args.metrics,
			args.log,
			"startup.http_listen",
			fmt.Errorf("startup canceled before http listen: %w", err),
		)
	}

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(bootstrapCtx, "tcp", args.cfg.HTTP.Addr)
	if err != nil {
		return rejectHTTPStartup(
			bootstrapCtx,
			args.bootstrapSpan,
			args.metrics,
			args.log,
			"startup.http_listen",
			fmt.Errorf("listen http server: %w", err),
		)
	}
	if err := startupRuntimeContextErr(signalCtx, bootstrapCtx); err != nil {
		_ = listener.Close()
		return rejectHTTPStartup(
			bootstrapCtx,
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

	admissionCtx, cancelAdmission := context.WithCancel(bootstrapCtx)
	defer cancelAdmission()

	admissionErrCh := startStartupAdmission(admissionCtx, args.readinessCheck, args.cfg.HTTP.ReadinessTimeout)
	outcome := waitForHTTPRuntimePreReady(signalCtx, bootstrapCtx, args, admissionErrCh, runErrCh)
	if outcome.ready {
		outcome = waitForHTTPRuntimePostReady(signalCtx, args, runErrCh)
	}

	cancelAdmission()

	effectiveShutdownDelay := args.shutdownDelay
	if !args.admission.Ready() {
		effectiveShutdownDelay = 0
	}
	if err := drainAndShutdown(signalCtx, args.log, effectiveShutdownDelay, args.cfg.HTTP.ShutdownTimeout, args.healthSvc, args.srv); err != nil {
		outcome.recordPreReadyFailure(args, "startup.shutdown", err)
		if outcome.terminalErr != nil {
			return errors.Join(outcome.terminalErr, err)
		}
		return err
	}
	if outcome.terminalErr != nil {
		return outcome.terminalErr
	}
	if outcome.serverErr != nil {
		return fmt.Errorf("http server stopped with error: %w", outcome.serverErr)
	}

	args.log.Info("shutdown complete")
	return nil
}

func waitForHTTPRuntimePreReady(
	signalCtx context.Context,
	bootstrapCtx context.Context,
	args serveHTTPRuntimeArgs,
	admissionErrCh <-chan error,
	runErrCh <-chan error,
) httpRuntimeWaitOutcome {
	var outcome httpRuntimeWaitOutcome

	for {
		select {
		case err := <-admissionErrCh:
			if err != nil {
				outcome.terminalErr = rejectHTTPStartup(
					bootstrapCtx,
					args.bootstrapSpan,
					args.metrics,
					args.log,
					"startup.readiness",
					fmt.Errorf("startup readiness check failed: %w", err),
				)
				outcome.startupFailureRecorded = true
				return outcome
			}
			select {
			case err := <-runErrCh:
				outcome.handleServerStop(args, err)
				return outcome
			default:
			}
			args.admission.MarkReady()
			outcome.ready = true
			return outcome
		case <-signalCtx.Done():
			args.log.Info("shutdown signal received")
			outcome.recordPreReadyFailure(args, "startup.readiness", signalCtx.Err())
			return outcome
		case <-bootstrapCtx.Done():
			if signalCtx.Err() != nil {
				args.log.Info("shutdown signal received")
				outcome.recordPreReadyFailure(args, "startup.readiness", signalCtx.Err())
				return outcome
			}
			outcome.terminalErr = fmt.Errorf("startup budget exhausted before readiness: %w", bootstrapCtx.Err())
			args.log.Error("startup budget exhausted before readiness", "err", outcome.terminalErr)
			outcome.recordPreReadyFailure(args, "startup.readiness", outcome.terminalErr)
			return outcome
		case err := <-runErrCh:
			outcome.handleServerStop(args, err)
			return outcome
		}
	}
}

func waitForHTTPRuntimePostReady(signalCtx context.Context, args serveHTTPRuntimeArgs, runErrCh <-chan error) httpRuntimeWaitOutcome {
	var outcome httpRuntimeWaitOutcome

	select {
	case <-signalCtx.Done():
		args.log.Info("shutdown signal received")
	case err := <-runErrCh:
		outcome.handleServerStop(args, err)
	}

	return outcome
}

func (o *httpRuntimeWaitOutcome) handleServerStop(args serveHTTPRuntimeArgs, err error) {
	o.serverErr = err
	if !args.admission.Ready() {
		if o.serverErr != nil {
			o.terminalErr = fmt.Errorf("http server stopped before readiness: %w", o.serverErr)
		} else {
			o.terminalErr = errors.New("http server stopped before readiness")
		}
		o.recordPreReadyFailure(args, "startup.http_serve", o.terminalErr)
	}
	if o.serverErr != nil {
		args.log.Error("http server stopped with error", "err", o.serverErr)
	}
}

func (o *httpRuntimeWaitOutcome) recordPreReadyFailure(args serveHTTPRuntimeArgs, stage string, err error) {
	if o.startupFailureRecorded || args.admission.Ready() {
		return
	}
	recordStartupRejection(args.bootstrapSpan, args.metrics, telemetry.StartupRejectionReasonStartupError, "startup_error", stage, err)
	o.startupFailureRecorded = true
}

func startStartupAdmission(
	bootstrapCtx context.Context,
	readinessCheck func(context.Context) error,
	readinessTimeout time.Duration,
) <-chan error {
	resultCh := make(chan error, 1)

	go func() {
		readyCtx, cancel := withStageBudget(bootstrapCtx, readinessTimeout)
		defer cancel()

		if err := readyCtx.Err(); err != nil {
			resultCh <- fmt.Errorf("startup admission context: %w", err)
			return
		}
		if readinessCheck != nil {
			if err := readinessCheck(readyCtx); err != nil {
				resultCh <- err
				return
			}
		}
		if err := readyCtx.Err(); err != nil {
			resultCh <- fmt.Errorf("startup admission context: %w", err)
			return
		}
		resultCh <- nil
	}()

	return resultCh
}

func startupRuntimeContextErr(signalCtx context.Context, bootstrapCtx context.Context) error {
	if err := signalCtx.Err(); err != nil {
		return fmt.Errorf("startup signal context: %w", err)
	}
	if err := bootstrapCtx.Err(); err != nil {
		return fmt.Errorf("startup bootstrap context: %w", err)
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

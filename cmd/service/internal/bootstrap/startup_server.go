package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
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
	healthSvc      *health.Service
	srv            runtimeServer
	metricsSrv     runtimeServer
	readinessCheck func(context.Context) error
	admission      *startupAdmissionController
	shutdownDelay  time.Duration
}

type serverResult struct {
	name string
	err  error
}

func serveHTTPRuntime(signalCtx context.Context, bootstrapCtx context.Context, args serveHTTPRuntimeArgs) error {
	if err := startupRuntimeContextErr(signalCtx, bootstrapCtx); err != nil {
		return rejectHTTPStartup(
			bootstrapCtx,
			args.bootstrapSpan,
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
			args.log,
			"startup.http_listen",
			fmt.Errorf("listen http server: %w", err),
		)
	}

	var metricsListener net.Listener
	if args.metricsSrv != nil && args.cfg.Observability.Metrics.Addr != "" {
		metricsListener, err = listenConfig.Listen(bootstrapCtx, "tcp", args.cfg.Observability.Metrics.Addr)
		if err != nil {
			_ = listener.Close()
			return rejectHTTPStartup(
				bootstrapCtx,
				args.bootstrapSpan,
				args.log,
				"startup.metrics_listen",
				fmt.Errorf("listen metrics server: %w", err),
			)
		}
	}
	if err := startupRuntimeContextErr(signalCtx, bootstrapCtx); err != nil {
		_ = listener.Close()
		if metricsListener != nil {
			_ = metricsListener.Close()
		}
		return rejectHTTPStartup(
			bootstrapCtx,
			args.bootstrapSpan,
			args.log,
			"startup.http_serve",
			fmt.Errorf("startup canceled before http serve: %w", err),
		)
	}

	serverCount := 1
	if metricsListener != nil {
		serverCount++
	}
	runErrCh := make(chan serverResult, serverCount)
	go func() {
		args.log.Info("http server started", "addr", listener.Addr().String(), "env", args.cfg.App.Env)
		runErrCh <- serverResult{name: "http", err: normalizeServeError(args.srv.Serve(listener))}
	}()
	if metricsListener != nil {
		go func() {
			args.log.Info("metrics server started", "addr", metricsListener.Addr().String(), "env", args.cfg.App.Env)
			runErrCh <- serverResult{name: "metrics", err: normalizeServeError(args.metricsSrv.Serve(metricsListener))}
		}()
	}

	admissionCtx, cancelAdmission := context.WithCancel(bootstrapCtx)
	defer cancelAdmission()

	admissionErrCh := startStartupAdmission(admissionCtx, args.readinessCheck, args.cfg.HTTP.ReadinessTimeout)
	ready, stopRequested, terminalErr := waitForStartupAdmission(
		signalCtx,
		bootstrapCtx,
		args,
		admissionErrCh,
		runErrCh,
	)
	var serverErr error

	if ready && !stopRequested {
		select {
		case <-signalCtx.Done():
			args.log.Info("shutdown signal received")
		case result := <-runErrCh:
			serverErr = serverStoppedAfterReadiness(args.log, result)
		}
	}
	cancelAdmission()

	effectiveShutdownDelay := args.shutdownDelay
	if !ready {
		effectiveShutdownDelay = 0
	}
	servers := []shutdownServer{args.srv}
	if args.metricsSrv != nil {
		servers = append(servers, args.metricsSrv)
	}
	if err := drainAndShutdown(signalCtx, args.log, effectiveShutdownDelay, args.cfg.HTTP.ShutdownTimeout, args.healthSvc, servers...); err != nil {
		if terminalErr != nil {
			return errors.Join(terminalErr, err)
		}
		return errors.Join(serverErr, err)
	}
	if terminalErr != nil {
		return terminalErr
	}
	if serverErr != nil {
		return serverErr
	}

	args.log.Info("shutdown complete")
	return nil
}

func normalizeServeError(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func waitForStartupAdmission(
	signalCtx context.Context,
	bootstrapCtx context.Context,
	args serveHTTPRuntimeArgs,
	admissionErrCh <-chan error,
	runErrCh <-chan serverResult,
) (ready bool, stopRequested bool, terminalErr error) {
	select {
	case err := <-admissionErrCh:
		if err != nil {
			return false, false, rejectHTTPStartup(
				bootstrapCtx,
				args.bootstrapSpan,
				args.log,
				"startup.readiness",
				fmt.Errorf("startup readiness check failed: %w", err),
			)
		}
		select {
		case result := <-runErrCh:
			return false, false, serverStoppedBeforeReadiness(args, result)
		default:
			args.admission.MarkReady()
			return true, false, nil
		}
	case <-signalCtx.Done():
		args.log.Info("shutdown signal received")
		recordStartupRejection(args.bootstrapSpan, "startup_error", "startup.readiness", signalCtx.Err())
		return false, true, nil
	case <-bootstrapCtx.Done():
		select {
		case <-signalCtx.Done():
			args.log.Info("shutdown signal received")
			recordStartupRejection(args.bootstrapSpan, "startup_error", "startup.readiness", signalCtx.Err())
			return false, true, nil
		default:
		}
		err := fmt.Errorf("startup budget exhausted before readiness: %w", bootstrapCtx.Err())
		args.log.Error("startup budget exhausted before readiness", "err", err)
		recordStartupRejection(args.bootstrapSpan, "startup_error", "startup.readiness", err)
		return false, false, err
	case result := <-runErrCh:
		return false, false, serverStoppedBeforeReadiness(args, result)
	}
}

func serverStoppedBeforeReadiness(args serveHTTPRuntimeArgs, result serverResult) error {
	err := fmt.Errorf("%s server stopped before readiness", result.name)
	if result.err != nil {
		args.log.Error(result.name+" server stopped with error", "err", result.err)
		err = fmt.Errorf("%s server stopped before readiness: %w", result.name, result.err)
	}
	recordStartupRejection(args.bootstrapSpan, "startup_error", "startup."+result.name+"_serve", err)
	return err
}

func serverStoppedAfterReadiness(log *slog.Logger, result serverResult) error {
	if result.err == nil {
		return fmt.Errorf("%s server stopped unexpectedly", result.name)
	}
	log.Error(result.name+" server stopped with error", "err", result.err)
	return fmt.Errorf("%s server stopped with error: %w", result.name, result.err)
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
	log *slog.Logger,
	stage string,
	err error,
) error {
	recordStartupRejection(bootstrapSpan, "startup_error", stage, err)
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

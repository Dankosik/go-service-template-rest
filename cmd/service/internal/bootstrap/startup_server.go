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
)

// runtimeServer is the http.Server surface this package drives. Close is part of
// it because the drain needs a way to abandon connections a graceful shutdown
// gave up on; see forceCloseServers.
type runtimeServer interface {
	Serve(net.Listener) error
	Shutdown(context.Context) error
	Close() error
}

type serveHTTPRuntimeArgs struct {
	cfg            config.Config
	log            *slog.Logger
	healthSvc      *health.Service
	srv            runtimeServer
	metricsSrv     runtimeServer
	readinessCheck func(context.Context) error
	admission      *startupAdmissionController
	shutdownDelay  time.Duration
	// shutdown is the process-wide teardown deadline. It is armed here, at the
	// one point that knows serving has ended, and every stage after the drain
	// draws from it.
	shutdown *shutdownBudget
}

type serverResult struct {
	name string
	err  error
}

func serveHTTPRuntime(signalCtx context.Context, bootstrapCtx context.Context, args serveHTTPRuntimeArgs) error {
	if err := startupRuntimeContextErr(signalCtx, bootstrapCtx); err != nil {
		return rejectHTTPStartup(
			bootstrapCtx,
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
			args.log.InfoContext(signalCtx, "shutdown signal received")
		case result := <-runErrCh:
			serverErr = serverStoppedAfterReadiness(args.log, result)
		}
	}
	cancelAdmission()

	// The grace period starts now, not at process start: this is the moment the
	// platform began counting.
	args.shutdown.start()

	effectiveShutdownDelay := args.shutdownDelay
	if !ready {
		effectiveShutdownDelay = 0
	}
	// The diagnostics listener is deliberately not in this drain. Everything worth
	// measuring happens during the window it occupies: the readiness propagation
	// delay, up to the whole remaining shutdown budget of in-flight requests, and
	// the shed and timed-out responses they produce. The version this replaced closed
	// /metrics at the same instant as the API, so with the shipped scrape-only
	// configuration none of that window was ever collected — the Prometheus target
	// simply went down for the last fifteen seconds of every pod's life, which is
	// exactly the fifteen seconds a rolling deploy is judged on.
	drainErr := drainAndShutdown(
		signalCtx,
		args.log,
		effectiveShutdownDelay,
		// Clamped, so a drain cannot spend budget the stages after it need. The
		// configured value normally wins; validateShutdownGraceBudget is what
		// keeps that true rather than leaving it to chance here.
		args.shutdown.clamp(args.cfg.HTTP.ShutdownTimeout),
		args.healthSvc,
		args.srv,
	)
	// Stopped only now, so a scraper could still collect everything the drain
	// produced. It is stopped here rather than by the caller because this function
	// started its goroutine, and split ownership is what lets one escape.
	diagnosticsErr := shutdownDiagnostics(signalCtx, args.log, args.shutdown, args.metricsSrv)

	if drainErr != nil {
		if terminalErr != nil {
			return errors.Join(terminalErr, drainErr, diagnosticsErr)
		}
		return errors.Join(serverErr, drainErr, diagnosticsErr)
	}
	if terminalErr != nil {
		return errors.Join(terminalErr, diagnosticsErr)
	}
	if serverErr != nil {
		return errors.Join(serverErr, diagnosticsErr)
	}
	if diagnosticsErr != nil {
		return diagnosticsErr
	}

	args.log.InfoContext(signalCtx, "shutdown complete")
	return nil
}

// shutdownDiagnostics closes the private listener under its own budget, and is
// safe to call twice.
//
// It needs a bound of its own: an in-flight scrape holds the connection, and
// http.Server.Shutdown waits for active requests indefinitely — so without one a
// stalled scraper would park the process here and take the telemetry flush with it,
// which is the same failure the dependency close is bounded against.
func shutdownDiagnostics(base context.Context, log *slog.Logger, budget *shutdownBudget, server runtimeServer) error {
	if server == nil {
		return nil
	}

	err := server.Shutdown(budget.stage(base, diagnosticsShutdownTimeout))
	switch {
	case err == nil, errors.Is(err, http.ErrServerClosed):
		log.InfoContext(base, "diagnostics_stopped", startupLogArgs(startupLogComponentShutdown, "diagnostics", "success")...)
		return nil
	case errors.Is(err, context.DeadlineExceeded):
		// A scrape outlived the budget. Closing abandons it, which is the same
		// trade the API drain makes, and leaves the flush below able to run.
		closeErr := server.Close()
		log.WarnContext(
			base,
			"diagnostics_forced",
			startupLogArgs(
				startupLogComponentShutdown,
				"diagnostics",
				"degraded",
				"reason", "scrape_outlived_shutdown_budget",
			)...,
		)
		if closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
			return fmt.Errorf("close diagnostics server: %w", closeErr)
		}
		return nil
	default:
		return fmt.Errorf("shutdown diagnostics server: %w", err)
	}
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
				args.log,
				"startup.readiness",
				fmt.Errorf("startup readiness check failed: %w", err),
			)
		}
		select {
		case result := <-runErrCh:
			return false, false, serverStoppedBeforeReadiness(bootstrapCtx, args, result)
		default:
			args.admission.MarkReady()
			return true, false, nil
		}
	case <-signalCtx.Done():
		args.log.InfoContext(signalCtx, "shutdown signal received")
		return false, true, nil
	case <-bootstrapCtx.Done():
		select {
		case <-signalCtx.Done():
			args.log.InfoContext(signalCtx, "shutdown signal received")
			return false, true, nil
		default:
		}
		err := fmt.Errorf("startup budget exhausted before readiness: %w", bootstrapCtx.Err())
		args.log.ErrorContext(bootstrapCtx, "startup budget exhausted before readiness", "err", err)
		return false, false, err
	case result := <-runErrCh:
		return false, false, serverStoppedBeforeReadiness(bootstrapCtx, args, result)
	}
}

func serverStoppedBeforeReadiness(ctx context.Context, args serveHTTPRuntimeArgs, result serverResult) error {
	err := fmt.Errorf("%s server stopped before readiness", result.name)
	if result.err != nil {
		err = fmt.Errorf("%s server stopped before readiness: %w", result.name, result.err)
	}
	args.log.ErrorContext(
		ctx,
		"startup_blocked",
		startupLogArgs(
			startupLogComponentStartupProbes,
			result.name+"_serve",
			"error",
			"error.type", "startup_error",
			"err", err,
		)...,
	)
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
	log *slog.Logger,
	stage string,
	err error,
) error {
	log.ErrorContext(
		bootstrapCtx,
		"startup_blocked",
		startupLogArgs(
			startupLogComponentStartupProbes,
			strings.TrimPrefix(stage, "startup."),
			"error",
			"error.type", "startup_error",
			"err", err,
		)...,
	)
	return err
}

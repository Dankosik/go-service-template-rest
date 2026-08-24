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

	// profile:grpc:start
	grpcx "github.com/example/go-service-template-rest/internal/infra/grpc"
	// profile:grpc:end
	"golang.org/x/net/netutil"
)

// runtimeServer is the http.Server surface this package drives. Close is part of
// it because the drain needs a way to abandon connections a graceful shutdown
// gave up on; see forceCloseServers.
type runtimeServer interface {
	Serve(listener net.Listener) error
	Shutdown(ctx context.Context) error
	Close() error
}

// profile:grpc:start
type grpcRuntimeServer interface {
	runtimeServer
	SetServing(ready bool)
	StartDrain()
}

// profile:grpc:end

type serveRuntimeArgs struct {
	cfg       config.Config
	log       *slog.Logger
	healthSvc *health.Service
	httpSrv   runtimeServer
	// profile:grpc:start
	grpcSrv grpcRuntimeServer
	// profile:grpc:end
	metricsSrv         runtimeServer
	readinessCheck     func(context.Context) error
	backgroundFailures <-chan error
	admission          *startupAdmissionController
	onReady            func()
	shutdownDelay      time.Duration
	// profile:messaging-nats-jetstream:start
	preDrain func()
	// profile:messaging-nats-jetstream:end
	// shutdown is the process-wide teardown deadline. It is armed here, at the
	// one point that knows serving has ended, and every stage after the drain
	// draws from it.
	shutdown *shutdownBudget
}

type serverResult struct {
	name string
	err  error
}

type runtimeListeners struct {
	http net.Listener
	// profile:grpc:start
	grpc net.Listener
	// profile:grpc:end
	metrics net.Listener
}

func bindRuntimeListeners(ctx context.Context, args serveRuntimeArgs) (runtimeListeners, string, error) {
	var listeners runtimeListeners
	var listenConfig net.ListenConfig

	httpListener, err := listenConfig.Listen(ctx, "tcp", args.cfg.HTTP.Addr)
	if err != nil {
		return listeners, "startup.http_listen", fmt.Errorf("listen http server: %w", err)
	}
	listeners.http = boundedAPIListener(httpListener, args.cfg.HTTP.MaxConnections)

	// profile:grpc:start
	if args.grpcSrv != nil {
		grpcListener, grpcErr := listenConfig.Listen(ctx, "tcp", args.cfg.GRPC.Server.Addr)
		if grpcErr != nil {
			listeners.close()
			return runtimeListeners{}, "startup.grpc_listen", fmt.Errorf("listen gRPC server: %w", grpcErr)
		}
		listeners.grpc = boundedAPIListener(grpcListener, grpcx.MaxConnections)
	}
	// profile:grpc:end

	if args.metricsSrv != nil && args.cfg.Observability.Metrics.Addr != "" {
		metricsListener, metricsErr := listenConfig.Listen(ctx, "tcp", args.cfg.Observability.Metrics.Addr)
		if metricsErr != nil {
			listeners.close()
			return runtimeListeners{}, "startup.metrics_listen", fmt.Errorf("listen metrics server: %w", metricsErr)
		}
		listeners.metrics = metricsListener
	}
	return listeners, "", nil
}

func (l runtimeListeners) close() {
	if l.http != nil {
		_ = l.http.Close()
	}
	// profile:grpc:start
	if l.grpc != nil {
		_ = l.grpc.Close()
	}
	// profile:grpc:end
	if l.metrics != nil {
		_ = l.metrics.Close()
	}
}

func serveRuntime(signalCtx context.Context, bootstrapCtx context.Context, args serveRuntimeArgs) error {
	if err := startupRuntimeContextErr(signalCtx, bootstrapCtx); err != nil {
		return rejectHTTPStartup(
			bootstrapCtx,
			args.log,
			"startup.http_listen",
			fmt.Errorf("startup canceled before http listen: %w", err),
		)
	}

	listeners, stage, err := bindRuntimeListeners(bootstrapCtx, args)
	if err != nil {
		return rejectHTTPStartup(bootstrapCtx, args.log, stage, err)
	}
	listener := listeners.http
	// profile:grpc:start
	grpcListener := listeners.grpc
	// profile:grpc:end
	metricsListener := listeners.metrics

	if err := startupRuntimeContextErr(signalCtx, bootstrapCtx); err != nil {
		listeners.close()
		return rejectHTTPStartup(
			bootstrapCtx,
			args.log,
			"startup.http_serve",
			fmt.Errorf("startup canceled before http serve: %w", err),
		)
	}

	serverCount := 1
	// profile:grpc:start
	if grpcListener != nil {
		serverCount++
	}
	// profile:grpc:end
	if metricsListener != nil {
		serverCount++
	}
	runErrCh := make(chan serverResult, serverCount)
	go func() {
		args.log.InfoContext(bootstrapCtx, "http server started", "addr", listener.Addr().String(), "env", args.cfg.App.Env)
		runErrCh <- serverResult{name: "http", err: normalizeServeError(args.httpSrv.Serve(listener))}
	}()
	// profile:grpc:start
	if grpcListener != nil {
		go func() {
			args.log.InfoContext(bootstrapCtx, "gRPC server started", "addr", grpcListener.Addr().String(), "env", args.cfg.App.Env)
			runErrCh <- serverResult{name: "grpc", err: args.grpcSrv.Serve(grpcListener)}
		}()
	}
	// profile:grpc:end
	if metricsListener != nil {
		go func() {
			args.log.InfoContext(bootstrapCtx, "metrics server started", "addr", metricsListener.Addr().String(), "env", args.cfg.App.Env)
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
		serverErr, terminalErr = waitForRuntimeStop(signalCtx, args, runErrCh)
	}
	cancelAdmission()

	// The grace period starts now, not at process start: this is the moment the
	// platform began counting.
	args.shutdown.start()
	// profile:messaging-nats-jetstream:start
	if args.preDrain != nil {
		args.preDrain()
	}
	// profile:messaging-nats-jetstream:end

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
	drainer := startupDrainer(args.healthSvc)
	applicationServers := []shutdownServer{args.httpSrv}
	// profile:grpc:start
	if args.grpcSrv != nil {
		drainer = startupDrainSet{args.healthSvc, args.grpcSrv}
		applicationServers = append(applicationServers, args.grpcSrv)
	}
	// profile:grpc:end
	drainErr := drainAndShutdown(
		signalCtx,
		args.log,
		effectiveShutdownDelay,
		// Clamped, so a drain cannot spend budget the stages after it need. The
		// configured value normally wins; validateShutdownGraceBudget is what
		// keeps that true rather than leaving it to chance here.
		args.shutdown.clamp(args.cfg.HTTP.ShutdownTimeout),
		drainer,
		applicationServers...,
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

func waitForRuntimeStop(
	signalCtx context.Context,
	args serveRuntimeArgs,
	runErrCh <-chan serverResult,
) (serverErr error, terminalErr error) {
	select {
	case <-signalCtx.Done():
		args.log.InfoContext(signalCtx, "shutdown signal received")
	case result := <-runErrCh:
		serverErr = serverStoppedAfterReadiness(args.log, result)
	case err := <-args.backgroundFailures:
		terminalErr = fmt.Errorf("background task failed after readiness: %w", err)
	}
	return serverErr, terminalErr
}

// boundedAPIListener caps how many connections the API accepts at once.
//
// It covers the half of overload the middleware chain cannot see. MaxInFlight
// sheds inside a handler, so every connection beyond the limit has already cost a
// goroutine, two buffers, and a header parse up to http.max_header_bytes by the
// time it is rejected — a connection flood therefore grows the heap without bound
// behind a load shedder reporting that the service copes. Excess callers wait in
// the kernel accept queue instead, which costs this process nothing.
//
// The diagnostics listener is deliberately not capped: it serves a scraper, and
// a metrics endpoint that blocks during an incident is the wrong trade.
func boundedAPIListener(listener net.Listener, maxConnections int) net.Listener {
	if maxConnections <= 0 {
		return listener
	}
	return netutil.LimitListener(listener, maxConnections)
}

func normalizeServeError(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func serverStoppedAfterReadiness(log *slog.Logger, result serverResult) error {
	if result.err == nil {
		return fmt.Errorf("%s server stopped unexpectedly", result.name)
	}
	log.Error(result.name+" server stopped with error", "err", result.err)
	return fmt.Errorf("%s server stopped with error: %w", result.name, result.err)
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

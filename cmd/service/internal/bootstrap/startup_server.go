package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

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
	diagnosticsSrv     runtimeServer
	readinessCheck     func(context.Context) error
	backgroundFailures <-chan error
	admission          *startupAdmissionController
	onReady            func()
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

// boundServer is one server paired with the listener it serves. label names it
// in the start log; name identifies it in serverResult.
type boundServer struct {
	name     string
	label    string
	server   runtimeServer
	listener net.Listener
}

// bindRuntimeListeners binds every configured server's listener. On failure it
// closes the listeners already bound and reports the failed operation.
func bindRuntimeListeners(ctx context.Context, args serveRuntimeArgs) (bound []boundServer, failedOperation string, err error) {
	var listenConfig net.ListenConfig

	httpListener, err := listenConfig.Listen(ctx, "tcp", args.cfg.HTTP.Addr)
	if err != nil {
		return nil, "http_listen", fmt.Errorf("listen http server: %w", err)
	}
	bound = append(bound, boundServer{
		name:     "http",
		label:    "http",
		server:   args.httpSrv,
		listener: boundedAPIListener(httpListener, args.cfg.HTTP.MaxConnections),
	})

	// profile:grpc:start
	if args.grpcSrv != nil {
		grpcListener, grpcErr := listenConfig.Listen(ctx, "tcp", args.cfg.GRPC.Server.Addr)
		if grpcErr != nil {
			closeBoundListeners(bound)
			return nil, "grpc_listen", fmt.Errorf("listen gRPC server: %w", grpcErr)
		}
		bound = append(bound, boundServer{
			name:     "grpc",
			label:    "gRPC",
			server:   args.grpcSrv,
			listener: boundedAPIListener(grpcListener, grpcx.MaxConnections),
		})
	}
	// profile:grpc:end

	if args.diagnosticsSrv != nil && args.cfg.Observability.Metrics.Addr != "" {
		diagnosticsListener, diagnosticsListenErr := listenConfig.Listen(ctx, "tcp", args.cfg.Observability.Metrics.Addr)
		if diagnosticsListenErr != nil {
			closeBoundListeners(bound)
			return nil, "metrics_listen", fmt.Errorf("listen diagnostics server: %w", diagnosticsListenErr)
		}
		bound = append(bound, boundServer{
			name:     "diagnostics",
			label:    "diagnostics",
			server:   args.diagnosticsSrv,
			listener: diagnosticsListener,
		})
	}
	return bound, "", nil
}

func closeBoundListeners(bound []boundServer) {
	for _, b := range bound {
		_ = b.listener.Close()
	}
}

func serveRuntime(signalCtx context.Context, startupCtx context.Context, args serveRuntimeArgs) error {
	if err := startupRuntimeContextErr(signalCtx, startupCtx); err != nil {
		return rejectRuntimeStartup(
			startupCtx,
			args.log,
			"http_listen",
			fmt.Errorf("startup canceled before http listen: %w", err),
		)
	}

	bound, operation, err := bindRuntimeListeners(startupCtx, args)
	if err != nil {
		return rejectRuntimeStartup(startupCtx, args.log, operation, err)
	}

	if err := startupRuntimeContextErr(signalCtx, startupCtx); err != nil {
		closeBoundListeners(bound)
		return rejectRuntimeStartup(
			startupCtx,
			args.log,
			"http_serve",
			fmt.Errorf("startup canceled before http serve: %w", err),
		)
	}

	runErrCh := make(chan serverResult, len(bound))
	for _, b := range bound {
		go func() {
			args.log.InfoContext(startupCtx, b.label+" server started", "addr", b.listener.Addr().String(), "env", args.cfg.App.Env)
			runErrCh <- serverResult{name: b.name, err: normalizeServeError(b.server.Serve(b.listener))}
		}()
	}

	admissionCtx, cancelAdmission := context.WithCancel(startupCtx)
	defer cancelAdmission()

	admissionErrCh := startStartupAdmission(admissionCtx, args.readinessCheck, args.cfg.HTTP.ReadinessTimeout)
	ready, terminalErr := waitForStartupAdmission(
		signalCtx,
		startupCtx,
		args,
		admissionErrCh,
		runErrCh,
	)
	if ready {
		terminalErr = waitForRuntimeStop(signalCtx, args, runErrCh)
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

	effectiveReadinessPropagationDelay := args.cfg.HTTP.ReadinessPropagationDelay
	if !ready {
		effectiveReadinessPropagationDelay = 0
	}
	// The diagnostics listener is deliberately not in this drain. Everything worth
	// measuring happens during the window it occupies: the readiness propagation
	// delay, up to the whole remaining shutdown budget of in-flight requests, and
	// the shed and timed-out responses they produce. Closing /metrics with the API
	// would leave that window uncollected under the shipped scrape-only
	// configuration: the Prometheus target would go down for the end of every
	// pod's life, which is the part a rolling deploy is judged on.
	drainer := shutdownDrainer(args.healthSvc)
	applicationServers := []shutdownServer{args.httpSrv}
	// profile:grpc:start
	if args.grpcSrv != nil {
		drainer = shutdownDrainSet{args.healthSvc, args.grpcSrv}
		applicationServers = append(applicationServers, args.grpcSrv)
	}
	// profile:grpc:end
	drainErr := drainAndShutdown(
		signalCtx,
		args.log,
		effectiveReadinessPropagationDelay,
		// Clamped, so a drain cannot spend budget the stages after it need. The
		// configured value normally wins; validateShutdownGraceBudget is what
		// keeps that true rather than leaving it to chance here.
		args.shutdown.clamp(signalCtx, args.cfg.HTTP.ShutdownTimeout),
		drainer,
		applicationServers...,
	)
	// Stopped only now, so a scraper could still collect everything the drain
	// produced. It is stopped here rather than by the caller because this function
	// started its goroutine, and split ownership is what lets one escape.
	diagnosticsErr := shutdownDiagnostics(signalCtx, args.log, args.shutdown, args.diagnosticsSrv)

	if terminalErr != nil || drainErr != nil {
		return errors.Join(terminalErr, drainErr, diagnosticsErr)
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
) error {
	select {
	case <-signalCtx.Done():
		args.log.InfoContext(signalCtx, "shutdown signal received")
		return nil
	case result := <-runErrCh:
		return serverStoppedAfterReadiness(args.log, result)
	case err := <-args.backgroundFailures:
		return fmt.Errorf("background task failed after readiness: %w", err)
	}
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

func startupRuntimeContextErr(signalCtx context.Context, startupCtx context.Context) error {
	if err := signalCtx.Err(); err != nil {
		return fmt.Errorf("startup signal context: %w", err)
	}
	if err := startupCtx.Err(); err != nil {
		return fmt.Errorf("startup bootstrap context: %w", err)
	}
	return nil
}

func rejectRuntimeStartup(
	startupCtx context.Context,
	log *slog.Logger,
	operation string,
	err error,
) error {
	log.ErrorContext(
		startupCtx,
		"startup_blocked",
		startupLogArgs(
			startupLogComponentStartupProbes,
			operation,
			"error",
			"error.type", "startup_error",
			"err", err,
		)...,
	)
	return err
}

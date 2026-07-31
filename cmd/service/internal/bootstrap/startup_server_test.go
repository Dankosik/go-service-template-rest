package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
)

type fakeRuntimeServer struct {
	serveStarted  chan struct{}
	stopServe     chan struct{}
	stopServeOnce sync.Once
	onServe       func(net.Listener) error
	onShutdown    func(context.Context) error
	onClose       func() error
}

func newFakeRuntimeServer() *fakeRuntimeServer {
	return &fakeRuntimeServer{
		serveStarted: make(chan struct{}),
		stopServe:    make(chan struct{}),
	}
}

func (f *fakeRuntimeServer) Serve(listener net.Listener) error {
	if f.serveStarted != nil {
		close(f.serveStarted)
	}
	if f.onServe != nil {
		return f.onServe(listener)
	}

	<-f.stopServe
	_ = listener.Close()
	return nil
}

func (f *fakeRuntimeServer) Close() error {
	if f.stopServe != nil {
		f.stopServeOnce.Do(func() {
			close(f.stopServe)
		})
	}
	if f.onClose != nil {
		return f.onClose()
	}
	return nil
}

func (f *fakeRuntimeServer) Shutdown(ctx context.Context) error {
	if f.stopServe != nil {
		f.stopServeOnce.Do(func() {
			close(f.stopServe)
		})
	}
	if f.onShutdown != nil {
		return f.onShutdown(ctx)
	}
	return nil
}

// profile:grpc:start
type fakeGRPCRuntimeServer struct {
	*fakeRuntimeServer

	markServing     chan struct{}
	startDrain      chan struct{}
	markServingOnce sync.Once
	startDrainOnce  sync.Once
	// profile:authn-oidc-jwt:start
	authnReadiness chan bool
	// profile:authn-oidc-jwt:end
}

func newFakeGRPCRuntimeServer() *fakeGRPCRuntimeServer {
	return &fakeGRPCRuntimeServer{
		fakeRuntimeServer: newFakeRuntimeServer(),
		markServing:       make(chan struct{}),
		startDrain:        make(chan struct{}),
		// profile:authn-oidc-jwt:start
		authnReadiness: make(chan bool, 8),
		// profile:authn-oidc-jwt:end
	}
}

func (f *fakeGRPCRuntimeServer) MarkServing() {
	f.markServingOnce.Do(func() { close(f.markServing) })
}

// profile:authn-oidc-jwt:start
func (f *fakeGRPCRuntimeServer) SetAuthnReady(ready bool) {
	f.authnReadiness <- ready
}

// profile:authn-oidc-jwt:end

func (f *fakeGRPCRuntimeServer) StartDrain() {
	f.startDrainOnce.Do(func() { close(f.startDrain) })
}

// profile:grpc:end

func newTestStartupAdmissionController() *startupAdmissionController {
	return newStartupAdmissionController()
}

func TestStartupAdmissionControllerCheckReady(t *testing.T) {
	t.Parallel()

	admission := newTestStartupAdmissionController()

	err := admission.CheckReady(context.Background())
	if !errors.Is(err, errStartupAdmissionPending) {
		t.Fatalf("CheckReady() error = %v, want %v", err, errStartupAdmissionPending)
	}

	admission.MarkReady()
	if err := admission.CheckReady(context.Background()); err != nil {
		t.Fatalf("CheckReady() after MarkReady error = %v, want nil", err)
	}
}

func TestStartStartupAdmissionRejectsCanceledReadinessContextAfterSuccessfulCheck(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		bootstrapCtx, cancel := context.WithCancel(t.Context())
		defer cancel()

		resultCh := startStartupAdmission(bootstrapCtx, func(ctx context.Context) error {
			cancel()
			<-ctx.Done()
			return nil
		}, time.Second)

		err := <-resultCh
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("startStartupAdmission() error = %v, want wrapped %v", err, context.Canceled)
		}
	})
}

func TestServeHTTPRuntimeListenError(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	svc := health.New()

	err := serveRuntime(context.Background(), context.Background(), serveRuntimeArgs{
		cfg:            config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:-1", ShutdownTimeout: time.Second}},
		log:            logger,
		healthSvc:      svc,
		httpSrv:        newFakeRuntimeServer(),
		readinessCheck: func(context.Context) error { return nil },
		admission:      newTestStartupAdmissionController(),
		shutdown:       testShutdownBudget(),
	})

	if err == nil {
		t.Fatal("serveRuntime() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "listen http server") {
		t.Fatalf("serveRuntime() err = %v, want listen context", err)
	}
}

func TestServeHTTPRuntimeMetricsListenError(t *testing.T) {
	t.Parallel()

	occupied, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve metrics address: %v", err)
	}
	defer func() {
		_ = occupied.Close()
	}()

	err = serveRuntime(context.Background(), context.Background(), serveRuntimeArgs{
		cfg: config.Config{
			HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second},
			Observability: config.ObservabilityConfig{
				Metrics: config.MetricsConfig{Addr: occupied.Addr().String()},
			},
		},
		log:            slog.New(slog.DiscardHandler),
		healthSvc:      health.New(),
		httpSrv:        newFakeRuntimeServer(),
		metricsSrv:     newFakeRuntimeServer(),
		readinessCheck: func(context.Context) error { return nil },
		admission:      newTestStartupAdmissionController(),
		shutdown:       testShutdownBudget(),
	})

	if err == nil {
		t.Fatal("serveRuntime() error = nil, want metrics listen failure")
	}
	if !strings.Contains(err.Error(), "listen metrics server") {
		t.Fatalf("serveRuntime() err = %v, want metrics listen context", err)
	}
}

func TestServeHTTPRuntimeStartsAndStopsApplicationAndMetricsServers(t *testing.T) {
	t.Parallel()

	appSrv := newFakeRuntimeServer()
	metricsSrv := newFakeRuntimeServer()
	admission := newTestStartupAdmissionController()
	signalCtx, cancelSignal := context.WithCancel(context.Background())
	defer cancelSignal()
	bootstrapCtx := context.WithoutCancel(signalCtx)

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- serveRuntime(signalCtx, bootstrapCtx, serveRuntimeArgs{
			cfg: config.Config{
				App:  config.AppConfig{Env: "test"},
				HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second},
				Observability: config.ObservabilityConfig{
					Metrics: config.MetricsConfig{Addr: "127.0.0.1:0"},
				},
			},
			log:            slog.New(slog.DiscardHandler),
			healthSvc:      health.New(),
			httpSrv:        appSrv,
			metricsSrv:     metricsSrv,
			readinessCheck: func(context.Context) error { return nil },
			admission:      admission,
			shutdown:       testShutdownBudget(),
		})
	}()

	for name, started := range map[string]<-chan struct{}{
		"application": appSrv.serveStarted,
		"metrics":     metricsSrv.serveStarted,
	} {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatalf("%s server did not start", name)
		}
	}

	deadline := time.Now().Add(time.Second)
	for !admission.Ready() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !admission.Ready() {
		t.Fatal("startup admission was not marked ready")
	}

	cancelSignal()
	select {
	case err := <-runErrCh:
		if err != nil {
			t.Fatalf("serveRuntime() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveRuntime() did not stop both servers")
	}
}

func TestServeHTTPRuntimeStopsAfterBackgroundFailure(t *testing.T) {
	t.Parallel()

	srv := newFakeRuntimeServer()
	admission := newTestStartupAdmissionController()
	failures := make(chan error, 1)
	taskErr := errors.New("consumer lost its lease")

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- serveRuntime(context.Background(), context.Background(), serveRuntimeArgs{
			cfg: config.Config{
				App:  config.AppConfig{Env: "test"},
				HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second},
			},
			log:                slog.New(slog.DiscardHandler),
			healthSvc:          health.New(),
			httpSrv:            srv,
			readinessCheck:     func(context.Context) error { return nil },
			backgroundFailures: failures,
			admission:          admission,
			shutdown:           testShutdownBudget(),
		})
	}()

	select {
	case <-srv.serveStarted:
	case <-time.After(time.Second):
		t.Fatal("application server did not start")
	}
	deadline := time.Now().Add(time.Second)
	for !admission.Ready() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !admission.Ready() {
		t.Fatal("startup admission was not marked ready")
	}

	failures <- taskErr
	select {
	case err := <-runErrCh:
		if !errors.Is(err, taskErr) {
			t.Fatalf("serveRuntime() error = %v, want wrapped %v", err, taskErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveRuntime() did not stop after the background failure")
	}
}

func TestServeHTTPRuntimeRejectsCanceledStartupBeforeListen(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	svc := health.New()

	signalCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := serveRuntime(signalCtx, context.Background(), serveRuntimeArgs{
		cfg:            config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
		log:            logger,
		healthSvc:      svc,
		httpSrv:        newFakeRuntimeServer(),
		readinessCheck: func(context.Context) error { return nil },
		admission:      newTestStartupAdmissionController(),
		shutdown:       testShutdownBudget(),
	})

	if err == nil {
		t.Fatal("serveRuntime() error = nil, want non-nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("serveRuntime() err = %v, want wrapped %v", err, context.Canceled)
	}
}

func TestServeHTTPRuntimeMarksReadyWithoutExternalReadinessProbe(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	svc := health.New()
	srv := newFakeRuntimeServer()
	admission := newTestStartupAdmissionController()
	readinessChecked := make(chan struct{}, 1)

	signalCtx, cancelSignal := context.WithCancel(context.Background())
	defer cancelSignal()
	bootstrapCtx := context.WithoutCancel(signalCtx)

	runErrCh := make(chan error, 1)
	go func(signalCtx context.Context, bootstrapCtx context.Context) {
		runErrCh <- serveRuntime(signalCtx, bootstrapCtx, serveRuntimeArgs{
			cfg:       config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
			log:       logger,
			healthSvc: svc,
			httpSrv:   srv,
			readinessCheck: func(context.Context) error {
				select {
				case readinessChecked <- struct{}{}:
				default:
				}
				return nil
			},
			admission: admission,
			shutdown:  testShutdownBudget(),
		})
	}(signalCtx, bootstrapCtx)

	select {
	case <-readinessChecked:
	case <-time.After(time.Second):
		t.Fatal("internal readiness check was not executed")
	}

	deadline := time.Now().Add(time.Second)
	for !admission.Ready() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !admission.Ready() {
		t.Fatal("startup admission was not marked ready")
	}

	cancelSignal()

	select {
	case err := <-runErrCh:
		if err != nil {
			t.Fatalf("serveRuntime() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveRuntime() did not return after shutdown signal")
	}
}

func TestServeHTTPRuntimeRejectsStartupDeadlineBeforeReadiness(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	svc := health.New()

	bootstrapCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := serveRuntime(context.Background(), bootstrapCtx, serveRuntimeArgs{
		cfg:       config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
		log:       logger,
		healthSvc: svc,
		httpSrv:   newFakeRuntimeServer(),
		readinessCheck: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		admission: newTestStartupAdmissionController(),
		shutdown:  testShutdownBudget(),
	})

	if err == nil {
		t.Fatal("serveRuntime() error = nil, want non-nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("serveRuntime() error = %v, want wrapped %v", err, context.DeadlineExceeded)
	}
}

func TestServeHTTPRuntimeSkipsPropagationDelayBeforeAdmissionReady(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	svc := health.New()
	srv := newFakeRuntimeServer()
	shutdownCalled := false

	srv.onShutdown = func(context.Context) error {
		shutdownCalled = true
		return nil
	}

	err := serveRuntime(context.Background(), context.Background(), serveRuntimeArgs{
		cfg:       config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: 25 * time.Millisecond}},
		log:       logger,
		healthSvc: svc,
		httpSrv:   srv,
		readinessCheck: func(context.Context) error {
			return errors.New("readiness failed")
		},
		admission:     newTestStartupAdmissionController(),
		shutdown:      testShutdownBudget(),
		shutdownDelay: time.Hour,
	})

	if err == nil {
		t.Fatal("serveRuntime() error = nil, want non-nil")
	}
	if !shutdownCalled {
		t.Fatal("server shutdown was not called before admission-ready")
	}
	if !strings.Contains(err.Error(), "startup readiness check failed") {
		t.Fatalf("serveRuntime() err = %v, want startup readiness context", err)
	}
}

func TestServeHTTPRuntimeReturnsServeFailureBeforeAdmissionReady(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	svc := health.New()
	srv := newFakeRuntimeServer()
	srv.onServe = func(net.Listener) error {
		return errors.New("boom")
	}

	err := serveRuntime(context.Background(), context.Background(), serveRuntimeArgs{
		cfg:       config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
		log:       logger,
		healthSvc: svc,
		httpSrv:   srv,
		readinessCheck: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
				return nil
			}
		},
		admission: newTestStartupAdmissionController(),
		shutdown:  testShutdownBudget(),
	})

	if err == nil {
		t.Fatal("serveRuntime() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "http server stopped before readiness: boom") {
		t.Fatalf("serveRuntime() err = %v, want pre-readiness serve failure", err)
	}
}

func TestServeHTTPRuntimeReturnsPendingServeFailureBeforeMarkingAdmissionReady(t *testing.T) {
	t.Parallel()

	// Driven under synctest because "already pending" means two different things
	// on either side of one goroutine hop. The fake server signals when Serve
	// returns; the code under test only sees the failure once the serving
	// goroutine has delivered it to its channel. Waiting on the signal alone left
	// that hop unsynchronized, so admission occasionally won the race and the
	// assertion below failed for a reason the production code was not guilty of.
	synctest.Test(t, func(t *testing.T) {
		logger := slog.New(slog.DiscardHandler)
		svc := health.New()
		srv := newFakeRuntimeServer()
		admission := newTestStartupAdmissionController()
		serveReturned := make(chan struct{})
		serveErr := errors.New("serve failed while admission succeeded")

		srv.onServe = func(net.Listener) error {
			defer close(serveReturned)
			return serveErr
		}

		err := serveRuntime(context.Background(), context.Background(), serveRuntimeArgs{
			cfg:       config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
			log:       logger,
			healthSvc: svc,
			httpSrv:   srv,
			readinessCheck: func(ctx context.Context) error {
				select {
				case <-serveReturned:
				case <-ctx.Done():
					return ctx.Err()
				}
				// Every other goroutine is durably blocked or finished by the time
				// this returns, which is what makes the serve result actually
				// pending rather than merely imminent.
				synctest.Wait()
				return nil
			},
			admission: admission,
			shutdown:  testShutdownBudget(),
		})

		if err == nil {
			t.Fatal("serveRuntime() error = nil, want pending serve failure")
		}
		if !errors.Is(err, serveErr) {
			t.Fatalf("serveRuntime() error = %v, want wrapped %v", err, serveErr)
		}
		if admission.Ready() {
			t.Fatal("startup admission marked ready while serve failure was already pending")
		}
	})
}

// TestServeHTTPRuntimeStopsDiagnosticsAfterTheDrain pins the shutdown ordering that
// keeps the drain measurable.
//
// The version this replaced passed the diagnostics server into the same
// drainAndShutdown call as the API, so /metrics closed at the instant the drain
// began — and with the shipped scrape-only configuration the readiness propagation
// delay and the whole in-flight drain went uncollected. The assertion is on order
// because the failure mode is invisible in behavior: everything still shuts down,
// and the metrics for the last window of the pod's life simply do not exist.
// profile:grpc:start
func TestServeRuntimeCoordinatesGRPCReadinessAndDrain(t *testing.T) {
	t.Parallel()

	httpServer := newFakeRuntimeServer()
	grpcServer := newFakeGRPCRuntimeServer()
	signalCtx, cancelSignal := context.WithCancel(context.Background())
	result := make(chan error, 1)

	go func() {
		result <- serveRuntime(signalCtx, context.Background(), serveRuntimeArgs{
			cfg: config.Config{
				App:  config.AppConfig{Env: "test"},
				HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second},
				GRPC: config.GRPCConfig{Server: config.GRPCServerConfig{
					Addr:           "127.0.0.1:0",
					MaxConnections: 4,
				}},
			},
			log:            slog.New(slog.DiscardHandler),
			healthSvc:      health.New(),
			httpSrv:        httpServer,
			grpcSrv:        grpcServer,
			readinessCheck: func(context.Context) error { return nil },
			admission:      newTestStartupAdmissionController(),
			shutdown:       testShutdownBudget(),
		})
	}()

	for name, started := range map[string]<-chan struct{}{
		"http": httpServer.serveStarted,
		"grpc": grpcServer.serveStarted,
	} {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatalf("%s server did not start", name)
		}
	}
	select {
	case <-grpcServer.markServing:
	case <-time.After(time.Second):
		t.Fatal("gRPC health was not marked serving after startup admission")
	}

	cancelSignal()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("serveRuntime() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveRuntime() did not stop")
	}
	select {
	case <-grpcServer.startDrain:
	default:
		t.Fatal("gRPC health was not put into drain before shutdown")
	}
}

func TestServeRuntimeRejectsGRPCListenFailureBeforeServing(t *testing.T) {
	t.Parallel()

	var listenConfig net.ListenConfig
	occupied, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	t.Cleanup(func() { _ = occupied.Close() })

	grpcServer := newFakeGRPCRuntimeServer()
	err = serveRuntime(context.Background(), context.Background(), serveRuntimeArgs{
		cfg: config.Config{
			HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second},
			GRPC: config.GRPCConfig{Server: config.GRPCServerConfig{
				Addr:           occupied.Addr().String(),
				MaxConnections: 4,
			}},
		},
		log:       slog.New(slog.DiscardHandler),
		healthSvc: health.New(),
		httpSrv:   newFakeRuntimeServer(),
		grpcSrv:   grpcServer,
		admission: newTestStartupAdmissionController(),
		shutdown:  testShutdownBudget(),
	})
	if err == nil || !strings.Contains(err.Error(), "listen gRPC server") {
		t.Fatalf("serveRuntime() error = %v, want gRPC listen failure", err)
	}
	select {
	case <-grpcServer.serveStarted:
		t.Fatal("gRPC server started after listener acquisition failed")
	default:
	}
}

// profile:grpc:end

func TestServeHTTPRuntimeStopsDiagnosticsAfterTheDrain(t *testing.T) {
	t.Parallel()

	var (
		mu    sync.Mutex
		order []string
	)
	record := func(name string) {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, name)
	}

	apiServer := newFakeRuntimeServer()
	apiServer.onShutdown = func(context.Context) error {
		record("api_drained")
		_ = apiServer.Close()
		return nil
	}
	diagnosticsServer := newFakeRuntimeServer()
	diagnosticsServer.onShutdown = func(context.Context) error {
		record("diagnostics_stopped")
		_ = diagnosticsServer.Close()
		return nil
	}

	signalCtx, cancelSignal := context.WithCancel(context.Background())
	go func() {
		<-apiServer.serveStarted
		<-diagnosticsServer.serveStarted
		cancelSignal()
	}()

	err := serveRuntime(signalCtx, context.Background(), serveRuntimeArgs{
		cfg: config.Config{
			HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second},
			Observability: config.ObservabilityConfig{
				Metrics: config.MetricsConfig{Addr: "127.0.0.1:0"},
			},
		},
		log:            slog.New(slog.DiscardHandler),
		healthSvc:      health.New(),
		httpSrv:        apiServer,
		metricsSrv:     diagnosticsServer,
		readinessCheck: func(context.Context) error { return nil },
		admission:      newTestStartupAdmissionController(),
		shutdown:       testShutdownBudget(),
	})
	if err != nil {
		t.Fatalf("serveRuntime() error = %v, want nil", err)
	}

	mu.Lock()
	defer mu.Unlock()
	want := []string{"api_drained", "diagnostics_stopped"}
	if !slices.Equal(order, want) {
		t.Fatalf("shutdown order = %v, want %v", order, want)
	}
}

// TestShutdownDiagnosticsForcesCloseOnBudgetExhaustion pins the bound on the
// diagnostics stop. It runs after the API drain, so without a budget of its own a
// scrape that never completes would park the process here and take the telemetry
// flush with it — the same failure the dependency close is bounded against.
func TestShutdownDiagnosticsForcesCloseOnBudgetExhaustion(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		var closed atomic.Bool
		server := newFakeRuntimeServer()
		server.onShutdown = func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}
		server.onClose = func() error {
			closed.Store(true)
			return nil
		}

		var logged bytes.Buffer
		err := shutdownDiagnostics(context.Background(), slog.New(slog.NewJSONHandler(&logged, nil)), testShutdownBudget(), server)
		if err != nil {
			t.Fatalf("shutdownDiagnostics() error = %v, want the abandoned scrape reported as degraded, not failed", err)
		}
		if !closed.Load() {
			t.Fatal("shutdownDiagnostics() did not force the listener closed after its budget expired")
		}
		if !strings.Contains(logged.String(), "diagnostics_forced") {
			t.Fatalf("log = %q, want the forced close recorded", logged.String())
		}
	})
}

func TestShutdownDiagnosticsIgnoresAbsentServer(t *testing.T) {
	t.Parallel()

	if err := shutdownDiagnostics(context.Background(), slog.New(slog.DiscardHandler), testShutdownBudget(), nil); err != nil {
		t.Fatalf("shutdownDiagnostics(nil) error = %v, want nil", err)
	}
}

package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/config"
	"go.opentelemetry.io/otel/trace"
)

type fakeRuntimeServer struct {
	serveStarted  chan struct{}
	stopServe     chan struct{}
	stopServeOnce sync.Once
	onServe       func(net.Listener) error
	onShutdown    func(context.Context) error
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

func newTestStartupAdmissionController() *startupAdmissionController {
	return newStartupAdmissionController(
		newStartupSpanController(trace.SpanFromContext(context.Background()), func(context.Context) {}),
	)
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

	err := serveHTTPRuntime(context.Background(), context.Background(), serveHTTPRuntimeArgs{
		bootstrapSpan:  trace.SpanFromContext(context.Background()),
		cfg:            config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:-1", ShutdownTimeout: time.Second}},
		log:            logger,
		healthSvc:      svc,
		srv:            newFakeRuntimeServer(),
		readinessCheck: func(context.Context) error { return nil },
		admission:      newTestStartupAdmissionController(),
	})

	if err == nil {
		t.Fatal("serveHTTPRuntime() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "listen http server") {
		t.Fatalf("serveHTTPRuntime() err = %v, want listen context", err)
	}
}

func TestServeHTTPRuntimeRejectsCanceledStartupBeforeListen(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	svc := health.New()

	signalCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := serveHTTPRuntime(signalCtx, context.Background(), serveHTTPRuntimeArgs{
		bootstrapSpan:  trace.SpanFromContext(context.Background()),
		cfg:            config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
		log:            logger,
		healthSvc:      svc,
		srv:            newFakeRuntimeServer(),
		readinessCheck: func(context.Context) error { return nil },
		admission:      newTestStartupAdmissionController(),
	})

	if err == nil {
		t.Fatal("serveHTTPRuntime() error = nil, want non-nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("serveHTTPRuntime() err = %v, want wrapped %v", err, context.Canceled)
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
	bootstrapSpan := trace.SpanFromContext(bootstrapCtx)

	runErrCh := make(chan error, 1)
	go func(signalCtx context.Context, bootstrapCtx context.Context, bootstrapSpan trace.Span) {
		runErrCh <- serveHTTPRuntime(signalCtx, bootstrapCtx, serveHTTPRuntimeArgs{
			bootstrapSpan: bootstrapSpan,
			cfg:           config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
			log:           logger,
			healthSvc:     svc,
			srv:           srv,
			readinessCheck: func(context.Context) error {
				select {
				case readinessChecked <- struct{}{}:
				default:
				}
				return nil
			},
			admission: admission,
		})
	}(signalCtx, bootstrapCtx, bootstrapSpan)

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
			t.Fatalf("serveHTTPRuntime() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveHTTPRuntime() did not return after shutdown signal")
	}
}

func TestServeHTTPRuntimeRejectsStartupDeadlineBeforeReadiness(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	svc := health.New()

	bootstrapCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := serveHTTPRuntime(context.Background(), bootstrapCtx, serveHTTPRuntimeArgs{
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		cfg:           config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
		log:           logger,
		healthSvc:     svc,
		srv:           newFakeRuntimeServer(),
		readinessCheck: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		admission: newTestStartupAdmissionController(),
	})

	if err == nil {
		t.Fatal("serveHTTPRuntime() error = nil, want non-nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("serveHTTPRuntime() error = %v, want wrapped %v", err, context.DeadlineExceeded)
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

	err := serveHTTPRuntime(context.Background(), context.Background(), serveHTTPRuntimeArgs{
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		cfg:           config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: 25 * time.Millisecond}},
		log:           logger,
		healthSvc:     svc,
		srv:           srv,
		readinessCheck: func(context.Context) error {
			return errors.New("readiness failed")
		},
		admission:     newTestStartupAdmissionController(),
		shutdownDelay: time.Hour,
	})

	if err == nil {
		t.Fatal("serveHTTPRuntime() error = nil, want non-nil")
	}
	if !shutdownCalled {
		t.Fatal("server shutdown was not called before admission-ready")
	}
	if !strings.Contains(err.Error(), "startup readiness check failed") {
		t.Fatalf("serveHTTPRuntime() err = %v, want startup readiness context", err)
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

	err := serveHTTPRuntime(context.Background(), context.Background(), serveHTTPRuntimeArgs{
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		cfg:           config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
		log:           logger,
		healthSvc:     svc,
		srv:           srv,
		readinessCheck: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
				return nil
			}
		},
		admission: newTestStartupAdmissionController(),
	})

	if err == nil {
		t.Fatal("serveHTTPRuntime() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "http server stopped before readiness: boom") {
		t.Fatalf("serveHTTPRuntime() err = %v, want pre-readiness serve failure", err)
	}
}

func TestServeHTTPRuntimeReturnsPendingServeFailureBeforeMarkingAdmissionReady(t *testing.T) {
	t.Parallel()

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

	err := serveHTTPRuntime(context.Background(), context.Background(), serveHTTPRuntimeArgs{
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		cfg:           config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
		log:           logger,
		healthSvc:     svc,
		srv:           srv,
		readinessCheck: func(ctx context.Context) error {
			select {
			case <-serveReturned:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
		admission: admission,
	})

	if err == nil {
		t.Fatal("serveHTTPRuntime() error = nil, want pending serve failure")
	}
	if !errors.Is(err, serveErr) {
		t.Fatalf("serveHTTPRuntime() error = %v, want wrapped %v", err, serveErr)
	}
	if admission.Ready() {
		t.Fatal("startup admission marked ready while serve failure was already pending")
	}
}

package bootstrap

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Dankosik/privacy-sanitization-service/internal/app/health"
	"github.com/Dankosik/privacy-sanitization-service/internal/config"
	"github.com/Dankosik/privacy-sanitization-service/internal/infra/telemetry"
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

func newTestStartupAdmissionController(metrics *telemetry.Metrics) *startupAdmissionController {
	return newStartupAdmissionController(
		newStartupSpanController(trace.SpanFromContext(context.Background()), func(context.Context) {}),
		metrics,
	)
}

func TestServeHTTPRuntimeListenError(t *testing.T) {
	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := health.New()

	err := serveHTTPRuntime(
		context.Background(),
		context.Background(),
		trace.SpanFromContext(context.Background()),
		config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:-1", ShutdownTimeout: time.Second}},
		logger,
		metrics,
		svc,
		newFakeRuntimeServer(),
		func(context.Context) error { return nil },
		newTestStartupAdmissionController(metrics),
		0,
	)

	if err == nil {
		t.Fatal("serveHTTPRuntime() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "listen http server") {
		t.Fatalf("serveHTTPRuntime() err = %v, want listen context", err)
	}
}

func TestServeHTTPRuntimeRejectsCanceledStartupBeforeListen(t *testing.T) {
	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := health.New()

	signalCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := serveHTTPRuntime(
		signalCtx,
		context.Background(),
		trace.SpanFromContext(context.Background()),
		config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
		logger,
		metrics,
		svc,
		newFakeRuntimeServer(),
		func(context.Context) error { return nil },
		newTestStartupAdmissionController(metrics),
		0,
	)

	if err == nil {
		t.Fatal("serveHTTPRuntime() error = nil, want non-nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("serveHTTPRuntime() err = %v, want wrapped %v", err, context.Canceled)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `config_startup_outcome_total{outcome="rejected"} 1`) {
		t.Fatalf("metrics do not contain rejected startup outcome:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `config_validation_failures_total{reason="startup_error"} 1`) {
		t.Fatalf("metrics do not contain startup_error validation failure:\n%s", metricsText)
	}
}

func TestServeHTTPRuntimeMarksReadyWithoutExternalReadinessProbe(t *testing.T) {
	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := health.New()
	srv := newFakeRuntimeServer()
	admission := newTestStartupAdmissionController(metrics)
	readinessChecked := make(chan struct{}, 1)

	signalCtx, cancelSignal := context.WithCancel(context.Background())
	defer cancelSignal()
	bootstrapCtx := context.WithoutCancel(signalCtx)
	bootstrapSpan := trace.SpanFromContext(bootstrapCtx)

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- serveHTTPRuntime(
			signalCtx,
			bootstrapCtx,
			bootstrapSpan,
			config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
			logger,
			metrics,
			svc,
			srv,
			func(context.Context) error {
				select {
				case readinessChecked <- struct{}{}:
				default:
				}
				return nil
			},
			admission,
			0,
		)
	}()

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

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `config_startup_outcome_total{outcome="ready"} 1`) {
		t.Fatalf("metrics do not contain ready startup outcome:\n%s", metricsText)
	}
	if strings.Contains(metricsText, `config_startup_outcome_total{outcome="rejected"} 1`) {
		t.Fatalf("metrics unexpectedly contain rejected startup outcome:\n%s", metricsText)
	}
}

func TestServeHTTPRuntimeRejectsStartupDeadlineBeforeReadiness(t *testing.T) {
	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := health.New()

	bootstrapCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := serveHTTPRuntime(
		context.Background(),
		bootstrapCtx,
		trace.SpanFromContext(context.Background()),
		config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
		logger,
		metrics,
		svc,
		newFakeRuntimeServer(),
		func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		newTestStartupAdmissionController(metrics),
		0,
	)

	if err == nil {
		t.Fatal("serveHTTPRuntime() error = nil, want non-nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("serveHTTPRuntime() error = %v, want wrapped %v", err, context.DeadlineExceeded)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `config_startup_outcome_total{outcome="rejected"} 1`) {
		t.Fatalf("metrics do not contain rejected startup outcome:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `config_validation_failures_total{reason="startup_error"} 1`) {
		t.Fatalf("metrics do not contain startup_error validation failure:\n%s", metricsText)
	}
}

func TestServeHTTPRuntimeSkipsPropagationDelayBeforeAdmissionReady(t *testing.T) {
	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := health.New()
	srv := newFakeRuntimeServer()
	startedAt := time.Now()

	srv.onShutdown = func(context.Context) error {
		if elapsed := time.Since(startedAt); elapsed >= 100*time.Millisecond {
			t.Fatalf("shutdown started too late before admission-ready: %s", elapsed)
		}
		return nil
	}

	err := serveHTTPRuntime(
		context.Background(),
		context.Background(),
		trace.SpanFromContext(context.Background()),
		config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
		logger,
		metrics,
		svc,
		srv,
		func(context.Context) error {
			return errors.New("readiness failed")
		},
		newTestStartupAdmissionController(metrics),
		150*time.Millisecond,
	)

	if err == nil {
		t.Fatal("serveHTTPRuntime() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "startup readiness check failed") {
		t.Fatalf("serveHTTPRuntime() err = %v, want startup readiness context", err)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `config_startup_outcome_total{outcome="rejected"} 1`) {
		t.Fatalf("metrics do not contain rejected startup outcome:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `config_validation_failures_total{reason="startup_error"} 1`) {
		t.Fatalf("metrics do not contain startup_error validation failure:\n%s", metricsText)
	}
}

func TestServeHTTPRuntimeReturnsServeFailureBeforeAdmissionReady(t *testing.T) {
	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := health.New()
	srv := newFakeRuntimeServer()
	srv.onServe = func(net.Listener) error {
		return errors.New("boom")
	}

	err := serveHTTPRuntime(
		context.Background(),
		context.Background(),
		trace.SpanFromContext(context.Background()),
		config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
		logger,
		metrics,
		svc,
		srv,
		func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
				return nil
			}
		},
		newTestStartupAdmissionController(metrics),
		0,
	)

	if err == nil {
		t.Fatal("serveHTTPRuntime() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "http server stopped before readiness: boom") {
		t.Fatalf("serveHTTPRuntime() err = %v, want pre-readiness serve failure", err)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `config_startup_outcome_total{outcome="rejected"} 1`) {
		t.Fatalf("metrics do not contain rejected startup outcome:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `config_validation_failures_total{reason="startup_error"} 1`) {
		t.Fatalf("metrics do not contain startup_error validation failure:\n%s", metricsText)
	}
}

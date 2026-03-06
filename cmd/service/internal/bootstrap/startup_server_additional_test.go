package bootstrap

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/config"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel"
)

func TestServeHTTPRuntimeListenError(t *testing.T) {
	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	recorder := newDeployTelemetryRecorder(logger, metrics, "test")
	svc := health.New()
	srv := httpx.New(httpx.Config{Addr: "127.0.0.1:0"}, http.NewServeMux())

	_, span := otel.Tracer("test").Start(context.Background(), "bootstrap-server")
	err := serveHTTPRuntime(
		context.Background(),
		context.Background(),
		span,
		config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:-1", ShutdownTimeout: time.Second}},
		logger,
		metrics,
		recorder,
		time.Now(),
		svc,
		srv,
		nil,
		func() {},
		0,
	)
	span.End()

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
	recorder := newDeployTelemetryRecorder(logger, metrics, "test")
	svc := health.New()
	srv := httpx.New(httpx.Config{Addr: "127.0.0.1:0"}, http.NewServeMux())

	signalCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, span := otel.Tracer("test").Start(context.Background(), "bootstrap-server")
	err := serveHTTPRuntime(
		signalCtx,
		context.Background(),
		span,
		config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
		logger,
		metrics,
		recorder,
		time.Now(),
		svc,
		srv,
		nil,
		func() {},
		0,
	)
	span.End()

	if err == nil {
		t.Fatal("serveHTTPRuntime() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "startup canceled before http listen") {
		t.Fatalf("serveHTTPRuntime() err = %v, want canceled-before-listen context", err)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if strings.Contains(metricsText, `deploy_health_admission_total{environment="test",reason_class="ready",result="success"} 1`) {
		t.Fatalf("metrics unexpectedly contain success admission:\n%s", metricsText)
	}
}

func TestServeHTTPRuntimeRejectsStartupDeadlineBeforeReadiness(t *testing.T) {
	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	recorder := newDeployTelemetryRecorder(logger, metrics, "test")
	svc := health.New()
	srv := httpx.New(httpx.Config{Addr: "127.0.0.1:0"}, http.NewServeMux())

	bootstrapCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, span := otel.Tracer("test").Start(context.Background(), "bootstrap-server")
	err := serveHTTPRuntime(
		context.Background(),
		bootstrapCtx,
		span,
		config.Config{HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second}},
		logger,
		metrics,
		recorder,
		time.Now(),
		svc,
		srv,
		make(chan struct{}),
		func() {},
		0,
	)
	span.End()

	if err != nil {
		t.Fatalf("serveHTTPRuntime() error = %v, want nil", err)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `deploy_health_admission_total{environment="test",reason_class="startup_error",result="failure"} 1`) {
		t.Fatalf("metrics do not contain startup deadline admission failure:\n%s", metricsText)
	}
	if strings.Contains(metricsText, `deploy_health_admission_total{environment="test",reason_class="ready",result="success"} 1`) {
		t.Fatalf("metrics unexpectedly contain readiness success after startup deadline:\n%s", metricsText)
	}
}

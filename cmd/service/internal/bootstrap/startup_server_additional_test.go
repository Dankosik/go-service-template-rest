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
	)
	span.End()

	if err == nil {
		t.Fatal("serveHTTPRuntime() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "listen http server") {
		t.Fatalf("serveHTTPRuntime() err = %v, want listen context", err)
	}
}

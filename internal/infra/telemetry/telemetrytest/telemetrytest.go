// Package telemetrytest owns shared test helpers for process-wide
// OpenTelemetry state: installing recording providers, restoring previous
// globals, and clearing ambient exporter environment. Tests that use these
// helpers mutate process-wide state and must not run in parallel.
package telemetrytest

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

const providerShutdownTimeout = 5 * time.Second

// RestoreGlobals snapshots the process-wide tracer provider, meter provider,
// and text-map propagator, and restores them when the test finishes. Use it
// when production setup code installs global telemetry during the test.
func RestoreGlobals(tb testing.TB) {
	tb.Helper()

	previousTracerProvider := otel.GetTracerProvider()
	previousMeterProvider := otel.GetMeterProvider()
	previousPropagator := otel.GetTextMapPropagator()
	tb.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		otel.SetMeterProvider(previousMeterProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})
}

// InstallSpanRecorder installs a process-wide tracer provider that samples
// and records every span, plus the W3C trace-context propagator. The previous
// tracer provider and propagator are restored and the temporary provider is
// shut down when the test finishes.
func InstallSpanRecorder(tb testing.TB) *tracetest.SpanRecorder {
	tb.Helper()

	previousTracerProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(recorder),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	tb.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		otel.SetTextMapPropagator(previousPropagator)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), providerShutdownTimeout)
		defer cancel()
		if err := provider.Shutdown(shutdownCtx); err != nil {
			tb.Errorf("shutdown test tracer provider: %v", err)
		}
	})
	return recorder
}

// ClearAmbientExporterEnv blanks every ambient OTLP exporter and proxy
// environment variable for the duration of the test so exporter policy is
// driven only by explicit configuration.
func ClearAmbientExporterEnv(tb testing.TB) {
	tb.Helper()

	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "OTEL_EXPORTER_OTLP_") {
			tb.Setenv(name, "")
		}
	}
	for _, name := range []string{
		"HTTP_PROXY",
		"HTTPS_PROXY",
		"NO_PROXY",
		"http_proxy",
		"https_proxy",
		"no_proxy",
	} {
		tb.Setenv(name, "")
	}
}

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

// shutdownAfterTest releases a provider when the test finishes, under a bound so
// an exporter that cannot drain fails the test instead of hanging the package.
//
// Every constructor here routes through it. The copies it replaced were the same
// six lines with a different noun, and what they disagreed on was the part that
// matters least visibly: whether a provider that failed to shut down was
// reported at all.
func shutdownAfterTest(tb testing.TB, kind string, shutdown func(context.Context) error) {
	tb.Helper()

	tb.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), providerShutdownTimeout)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			tb.Errorf("shutdown test %s provider: %v", kind, err)
		}
	})
}

// NewRecordingTracerProvider returns a tracer provider that samples and records
// every span, and the recorder holding them.
//
// It installs nothing process-wide, so a test that hands the provider to the
// code under test may run in parallel. Use [InstallSpanRecorder] instead when
// that code reads the global provider.
func NewRecordingTracerProvider(tb testing.TB) (*tracetest.SpanRecorder, *sdktrace.TracerProvider) {
	tb.Helper()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(recorder),
	)
	shutdownAfterTest(tb, "tracer", provider.Shutdown)
	return recorder, provider
}

// RestoreGlobals snapshots the process-wide tracer provider, meter provider,
// text-map propagator, and SDK error handler, and restores them when the test
// finishes. Use it when production setup code installs global telemetry.
func RestoreGlobals(tb testing.TB) {
	tb.Helper()

	previousTracerProvider := otel.GetTracerProvider()
	previousMeterProvider := otel.GetMeterProvider()
	previousPropagator := otel.GetTextMapPropagator()
	previousErrorHandler := otel.GetErrorHandler()
	tb.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		otel.SetMeterProvider(previousMeterProvider)
		otel.SetTextMapPropagator(previousPropagator)
		otel.SetErrorHandler(previousErrorHandler)
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
	recorder, provider := NewRecordingTracerProvider(tb)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	// Registered after the shutdown NewRecordingTracerProvider owns, so cleanup
	// runs in the order the globals require: the process stops pointing at this
	// provider before it is released.
	tb.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		otel.SetTextMapPropagator(previousPropagator)
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

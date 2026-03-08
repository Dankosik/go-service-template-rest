package bootstrap

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func installTestTracerProvider(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	previousProvider := otel.GetTracerProvider()
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider()
	tracerProvider.RegisterSpanProcessor(spanRecorder)
	otel.SetTracerProvider(tracerProvider)

	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer shutdownCancel()
		if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
			t.Errorf("tracerProvider.Shutdown() error = %v", err)
		}
	})

	return spanRecorder
}

func assertSpanStringAttribute(t *testing.T, span sdktrace.ReadOnlySpan, key, want string) {
	t.Helper()

	for _, attr := range span.Attributes() {
		if string(attr.Key) != key {
			continue
		}
		if got := attr.Value.AsString(); got != want {
			t.Fatalf("span %q attribute %q = %q, want %q", span.Name(), key, got, want)
		}
		return
	}

	t.Fatalf("span %q missing attribute %q", span.Name(), key)
}

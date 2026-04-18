package bootstrap

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
)

//nolint:paralleltest // Installs a process-wide tracer provider for span capture.
func TestStartupSpanControllerMarkReadyEndsSpanBeforeCleanup(t *testing.T) {
	spanRecorder := installTestTracerProvider(t)
	_, span := otel.Tracer("test").Start(context.Background(), "config.bootstrap")

	cleanupCalls := 0
	controller := newStartupSpanController(span, func(context.Context) {
		cleanupCalls++
		ended := spanRecorder.Ended()
		if len(ended) != 1 {
			t.Fatalf("ended spans during cleanup = %d, want 1", len(ended))
		}
		assertSpanStringAttribute(t, ended[0], "result", "success")
	})

	controller.MarkReady()
	if ended := spanRecorder.Ended(); len(ended) != 1 {
		t.Fatalf("ended spans after MarkReady() = %d, want 1", len(ended))
	}

	controller.Close(context.Background())
	if cleanupCalls != 1 {
		t.Fatalf("cleanup calls = %d, want 1", cleanupCalls)
	}
	if ended := spanRecorder.Ended(); len(ended) != 1 {
		t.Fatalf("ended spans after Close() = %d, want 1", len(ended))
	}

	controller.Close(context.Background())
	if cleanupCalls != 1 {
		t.Fatalf("cleanup calls after repeated Close() = %d, want 1", cleanupCalls)
	}
	if ended := spanRecorder.Ended(); len(ended) != 1 {
		t.Fatalf("ended spans after repeated Close() = %d, want 1", len(ended))
	}
}

//nolint:paralleltest // Installs a process-wide tracer provider for span capture.
func TestStartupSpanControllerCloseEndsSpanBeforeCleanup(t *testing.T) {
	spanRecorder := installTestTracerProvider(t)
	_, span := otel.Tracer("test").Start(context.Background(), "config.bootstrap")

	cleanupCalls := 0
	controller := newStartupSpanController(span, func(context.Context) {
		cleanupCalls++
		if ended := spanRecorder.Ended(); len(ended) != 1 {
			t.Fatalf("ended spans during cleanup = %d, want 1", len(ended))
		}
	})

	controller.Close(context.Background())
	if cleanupCalls != 1 {
		t.Fatalf("cleanup calls = %d, want 1", cleanupCalls)
	}
	if ended := spanRecorder.Ended(); len(ended) != 1 {
		t.Fatalf("ended spans after Close() = %d, want 1", len(ended))
	}

	controller.Close(context.Background())
	if cleanupCalls != 1 {
		t.Fatalf("cleanup calls after repeated Close() = %d, want 1", cleanupCalls)
	}
	if ended := spanRecorder.Ended(); len(ended) != 1 {
		t.Fatalf("ended spans after repeated Close() = %d, want 1", len(ended))
	}
}

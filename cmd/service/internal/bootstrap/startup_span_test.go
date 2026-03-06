package bootstrap

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
)

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
}

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
}

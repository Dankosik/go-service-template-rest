package postgresoutbox

import (
	"context"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// The creation context survives the round trip through the column, which is the
// whole of R1: a publication minutes later can still name the operation that
// produced the event.
//
// These tests install process-wide OpenTelemetry state and therefore do not run
// in parallel; telemetrytest owns that contract.
func TestCreationContextRoundTrip(t *testing.T) {
	t.Parallel()
	telemetrytest.InstallSpanRecorder(t)

	ctx, span := otel.GetTracerProvider().Tracer("test").Start(context.Background(), "producing operation")
	defer span.End()
	wanted := trace.SpanContextFromContext(ctx)

	stored, degraded := captureCreationContext(ctx)
	if degraded {
		t.Fatal("capture inside a trace reported degraded")
	}
	if !strings.Contains(string(stored), "traceparent") {
		t.Fatalf("stored context = %q, want a traceparent entry", stored)
	}

	// The relay restores it from bytes alone, with none of the producing
	// process's state.
	restored := trace.SpanContextFromContext(
		otel.GetTextMapPropagator().Extract(context.Background(), creationContextFromStored(stored)),
	)
	if restored.TraceID() != wanted.TraceID() {
		t.Fatalf("restored trace = %s, want %s", restored.TraceID(), wanted.TraceID())
	}
	if restored.SpanID() != wanted.SpanID() {
		t.Fatalf("restored span = %s, want %s", restored.SpanID(), wanted.SpanID())
	}
}

// Appending outside a trace is ordinary — a migration, a backfill, a test — so
// it stores the empty object and reports no degradation. Only a context that
// existed and could not be stored is degraded.
func TestCreationContextAbsentIsNotDegraded(t *testing.T) {
	t.Parallel()
	telemetrytest.InstallSpanRecorder(t)

	stored, degraded := captureCreationContext(context.Background())
	if degraded {
		t.Error("capture outside a trace reported degraded")
	}
	if string(stored) != "{}" {
		t.Errorf("stored context = %q, want the empty object", stored)
	}
	if carrier := creationContextFromStored(stored); carrier != nil {
		t.Errorf("empty object restored to %v, want no carrier", carrier)
	}
}

// A propagator emitting more than the column allows degrades to an absent
// context. It must never fail the append: the context comes from ambient
// telemetry configuration rather than from the caller, so rejecting the event
// would let a telemetry setting decide whether a business event is stored.
func TestCreationContextTooLargeDegradesInsteadOfFailing(t *testing.T) {
	t.Parallel()
	telemetrytest.RestoreGlobals(t)
	otel.SetTextMapPropagator(oversizePropagator{})

	stored, degraded := captureCreationContext(context.Background())
	if !degraded {
		t.Error("an oversize context did not report degraded")
	}
	if string(stored) != "{}" {
		t.Errorf("stored context = %q, want the empty object", stored)
	}
}

// Bytes the column's IS JSON OBJECT check would refuse can only arrive from a
// writer that bypassed it. Reading them yields no carrier rather than a
// failure, because a stored trace context must not be able to fail a
// publication.
func TestCreationContextFromUnreadableBytes(t *testing.T) {
	t.Parallel()

	for name, stored := range map[string][]byte{
		"not json":         []byte(`{not json`),
		"not an object":    []byte(`["traceparent"]`),
		"non-string value": []byte(`{"traceparent":1}`),
		"empty":            nil,
	} {
		if carrier := creationContextFromStored(stored); carrier != nil {
			t.Errorf("%s restored to %v, want no carrier", name, carrier)
		}
	}
}

// oversizePropagator injects more than maxTraceContextBytes allows.
type oversizePropagator struct{}

func (oversizePropagator) Inject(_ context.Context, carrier propagation.TextMapCarrier) {
	carrier.Set("traceparent", strings.Repeat("x", maxTraceContextBytes+1))
}

func (oversizePropagator) Extract(ctx context.Context, _ propagation.TextMapCarrier) context.Context {
	return ctx
}

func (oversizePropagator) Fields() []string { return []string{"traceparent"} }

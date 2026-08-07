package postgresoutbox

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// R2: a publication is joinable to the operation that produced it. These tests
// install process-wide OpenTelemetry state and do not run in parallel.

// The publish span links to the creation context rather than descending from
// it, so the producing request's trace does not stay open for the whole backlog
// and redrive horizon. The link is what carries the join.
func TestPublishSpanLinksToTheProducingOperation(t *testing.T) {
	recorder := telemetrytest.InstallSpanRecorder(t)
	telemetry := newTracingTelemetry(t)

	producing, span := otel.GetTracerProvider().Tracer("test").Start(context.Background(), "producing operation")
	stored, _ := captureCreationContext(producing)
	span.End()
	origin := trace.SpanContextFromContext(producing)

	event := outboxEventForUnit()
	event.Destination, event.ID = "orders.events", "event-1"
	event.traceContext = creationContextFromStored(stored)

	_, publish := telemetry.StartPublish(context.Background(), event)
	telemetry.EndPublish(publish, nil, classNone)

	recorded := spanNamed(t, recorder, "publish orders.events")
	if len(recorded.Links()) != 1 {
		t.Fatalf("publish span carries %d links, want exactly one to the producing operation", len(recorded.Links()))
	}
	if got := recorded.Links()[0].SpanContext.TraceID(); got != origin.TraceID() {
		t.Errorf("link trace = %s, want the producing trace %s", got, origin.TraceID())
	}
	// A link, not a parent: sharing the trace id would mean the producing
	// request's trace stays open until this publication happens.
	if recorded.SpanContext().TraceID() == origin.TraceID() {
		t.Error("publish span shares the producing trace id; it must be a linked root")
	}
	if recorded.SpanKind() != trace.SpanKindProducer {
		t.Errorf("publish span kind = %v, want producer", recorded.SpanKind())
	}
}

// An event appended outside a trace still publishes and is still observable.
// The span must carry no link at all — in particular it must not link to
// whatever span the relay itself is running inside, which is what extracting
// into the live context instead of a blank one would produce.
func TestPublishSpanWithoutCreationContextLinksToNothing(t *testing.T) {
	recorder := telemetrytest.InstallSpanRecorder(t)
	telemetry := newTracingTelemetry(t)

	// A relay-side span is deliberately active while the publication starts.
	relayCtx, relaySpan := otel.GetTracerProvider().Tracer("test").Start(context.Background(), "relay cycle")
	defer relaySpan.End()

	event := outboxEventForUnit()
	event.Destination, event.ID = "orders.events", "event-2"

	_, publish := telemetry.StartPublish(relayCtx, event)
	telemetry.EndPublish(publish, nil, classNone)

	recorded := spanNamed(t, recorder, "publish orders.events")
	if len(recorded.Links()) != 0 {
		t.Fatalf("publish span carries %d links, want none for an event with no creation context",
			len(recorded.Links()))
	}
}

// A failed publication reports the same bounded class its metric sample
// carries, so a trace and a dashboard name one condition. The broker's own
// error text never reaches the span.
func TestPublishSpanRecordsTheBoundedErrorClass(t *testing.T) {
	recorder := telemetrytest.InstallSpanRecorder(t)
	telemetry := newTracingTelemetry(t)

	event := outboxEventForUnit()
	event.Destination, event.ID = "orders.events", "event-3"
	_, publish := telemetry.StartPublish(context.Background(), event)
	telemetry.EndPublish(publish, errors.New("broker said no at 10.0.0.5:4222"), classPublisherRejected)

	recorded := spanNamed(t, recorder, "publish orders.events")
	if recorded.Status().Code != codes.Error {
		t.Errorf("publish span status = %v, want error", recorded.Status().Code)
	}
	if recorded.Status().Description != classPublisherRejected {
		t.Errorf("publish span status description = %q, want the bounded class %q",
			recorded.Status().Description, classPublisherRejected)
	}
	for _, attribute := range recorded.Attributes() {
		if value := attribute.Value.AsString(); value != "" && strings.Contains(value, "10.0.0.5") {
			t.Errorf("publish span attribute %s leaked broker error text: %q", attribute.Key, value)
		}
	}
}

// A nil Telemetry is a working no-op everywhere else in this package, and the
// publication path is no exception: it must return a usable context and a span
// that can be ended.
func TestPublishSpanOnNilTelemetryIsSafe(t *testing.T) {
	t.Parallel()

	var telemetry *Telemetry
	ctx, span := telemetry.StartPublish(context.Background(), outboxEventForUnit())
	if ctx == nil {
		t.Fatal("StartPublish(nil telemetry) returned no context")
	}
	telemetry.EndPublish(span, errors.New("failed"), classPublisherTemporary)
}

func newTracingTelemetry(tb testing.TB) *Telemetry {
	tb.Helper()

	telemetry, err := NewTelemetry(nil, nil)
	if err != nil {
		tb.Fatalf("NewTelemetry: %v", err)
	}
	tb.Cleanup(telemetry.Close)
	return telemetry
}

//nolint:ireturn // The SDK's recorder reports spans as its own interface.
func spanNamed(tb testing.TB, recorder *tracetest.SpanRecorder, name string) sdktrace.ReadOnlySpan {
	tb.Helper()

	for _, span := range recorder.Ended() {
		if span.Name() == name {
			return span
		}
	}
	tb.Fatalf("no span named %q was recorded", name)
	return nil
}

package correlationpolicy

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestPolicyAndReservedFieldsAreClosed(t *testing.T) {
	t.Parallel()

	for _, policy := range []Policy{None, TraceContext, TrustedService} {
		if !policy.Valid() {
			t.Fatalf("Policy(%d).Valid() = false, want true", policy)
		}
	}
	if Policy(99).Valid() {
		t.Fatal("unknown policy was accepted")
	}
	if got, want := ReservedFields("x-request-id"), []string{"traceparent", "tracestate", "baggage", "x-request-id"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ReservedFields() = %#v, want %#v", got, want)
	}
	reserved := Reserved("x-request-id")
	for _, key := range []string{"TraceParent", "TRACESTATE", "baggage", "X-Request-ID"} {
		if !reserved(key) {
			t.Fatalf("Reserved() did not match %q", key)
		}
	}
	if reserved("authorization") {
		t.Fatal("Reserved() matched an unrelated header")
	}
}

func TestPropagatorEmitsOnlySelectedCorrelation(t *testing.T) {
	t.Parallel()

	span := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1}, SpanID: trace.SpanID{1}, TraceFlags: trace.FlagsSampled,
	})
	ctx := reqctx.ContextWithRequestID(trace.ContextWithSpanContext(context.Background(), span), "request-1")

	none := http.Header{}
	NewPropagator(None, reqctx.RequestIDHeader).Inject(ctx, propagation.HeaderCarrier(none))
	if len(none) != 0 {
		t.Fatalf("None propagator wrote %#v", none)
	}

	traceOnly := http.Header{}
	NewPropagator(TraceContext, reqctx.RequestIDHeader).Inject(ctx, propagation.HeaderCarrier(traceOnly))
	if traceOnly.Get("Traceparent") == "" || traceOnly.Get(reqctx.RequestIDHeader) != "" {
		t.Fatalf("TraceContext propagator wrote %#v, want only trace context", traceOnly)
	}

	trusted := http.Header{}
	propagator := NewPropagator(TrustedService, reqctx.RequestIDHeader)
	propagator.Inject(ctx, propagation.HeaderCarrier(trusted))
	if trusted.Get("Traceparent") == "" || trusted.Get(reqctx.RequestIDHeader) != "request-1" {
		t.Fatalf("TrustedService propagator wrote %#v, want trace and request id", trusted)
	}
	if got := propagator.Extract(ctx, propagation.HeaderCarrier(trusted)); got != ctx {
		t.Fatal("outbound propagator adopted response correlation")
	}
	if got := propagator.Fields(); !reflect.DeepEqual(got, append(propagation.TraceContext{}.Fields(), reqctx.RequestIDHeader)) {
		t.Fatalf("Fields() = %#v, want trusted outbound fields", got)
	}
}

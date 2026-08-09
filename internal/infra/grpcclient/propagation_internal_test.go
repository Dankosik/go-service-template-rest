// Correlation trust boundary, white-box half: does each seam in propagation.go
// hold on its own — the correlationpolicy propagator emitting and declaring exactly the selected
// fields, and the two sanitizers stripping reserved keys without mutating the
// caller's own metadata or call options?
//
// It drives the unexported seams directly, so it needs no server and can reach
// what the wire half cannot distinguish, such as the pointer form of a per-RPC
// credential call option.
//
// propagation_test.go asserts what actually reaches a server, and the resolver
// half of the same boundary starts at resolver_internal_test.go.

package grpcclient

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/observability/correlationpolicy"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

func TestPolicyPropagatorOwnsRemoteCorrelationFields(t *testing.T) {
	t.Parallel()

	var traceID trace.TraceID
	traceID[len(traceID)-1] = 1
	var spanID trace.SpanID
	spanID[len(spanID)-1] = 2
	ctx := trace.ContextWithSpanContext(t.Context(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	}))
	ctx = reqctx.ContextWithRequestID(ctx, "trusted_request_123")

	for _, testCase := range []struct {
		name          string
		policy        PropagationPolicy
		wantTrace     bool
		wantRequestID bool
	}{
		{name: "none", policy: PropagationNone},
		{name: "trace context", policy: PropagationTraceContext, wantTrace: true},
		{
			name:          "trusted service",
			policy:        PropagationTrustedService,
			wantTrace:     true,
			wantRequestID: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			carrier := propagation.MapCarrier{}
			correlationpolicy.NewPropagator(testCase.policy, requestIDMetadataKey).Inject(ctx, carrier)

			if got := carrier.Get("traceparent"); (got != "") != testCase.wantTrace {
				t.Fatalf("traceparent = %q, want present %v", got, testCase.wantTrace)
			} else if got != "" && !strings.Contains(got, traceID.String()) {
				t.Fatalf("traceparent = %q, want trace ID %s", got, traceID)
			}
			if got := carrier.Get(requestIDMetadataKey); (got != "") != testCase.wantRequestID {
				t.Fatalf("x-request-id = %q, want present %v", got, testCase.wantRequestID)
			}
			if got := carrier.Get("baggage"); got != "" {
				t.Fatalf("baggage = %q, want absent", got)
			}
		})
	}
}

// TestPolicyPropagatorFieldsMatchWhatInjectEmits holds the three-place invariant
// the package doc states and nothing else checked: a correlation value is
// claimed in reservedCorrelationMetadataKeys, emitted by Inject under its
// policy, and declared by Fields, and any two without the third is a defect.
//
// Claiming without emitting silently drops a caller's value; emitting without
// claiming lets a caller forge one; emitting without declaring leaves an
// injecting carrier free to size itself without room for the key. The test above
// names three keys it expects, so it cannot see a fourth added to two places of
// the three — this one compares the sets themselves and needs no edit to cover
// the next key.
func TestPolicyPropagatorFieldsMatchWhatInjectEmits(t *testing.T) {
	t.Parallel()

	// A non-empty trace state is what makes the comparison below an equality
	// rather than a containment: W3C trace context declares both traceparent and
	// tracestate, but only emits the second when the span carries one.
	traceState, err := trace.ParseTraceState("vendor=value")
	if err != nil {
		t.Fatalf("trace.ParseTraceState() error = %v", err)
	}
	ctx := trace.ContextWithSpanContext(t.Context(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{2},
		TraceFlags: trace.FlagsSampled,
		TraceState: traceState,
	}))
	ctx = reqctx.ContextWithRequestID(ctx, "fields_request_123")

	for _, testCase := range []struct {
		name   string
		policy PropagationPolicy
	}{
		{name: "none", policy: PropagationNone},
		{name: "trace context", policy: PropagationTraceContext},
		{name: "trusted service", policy: PropagationTrustedService},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			propagator := correlationpolicy.NewPropagator(testCase.policy, requestIDMetadataKey)
			carrier := propagation.MapCarrier{}
			propagator.Inject(ctx, carrier)

			emitted := slices.Sorted(maps.Keys(carrier))
			declared := slices.Sorted(slices.Values(propagator.Fields()))
			if !slices.Equal(emitted, declared) {
				t.Fatalf(
					"Inject emitted %v but Fields declares %v; a correlation value belongs in both",
					emitted,
					declared,
				)
			}
			for _, key := range emitted {
				if !reservedCorrelationMetadataKey(key) {
					t.Fatalf(
						"Inject emits %q, which reservedCorrelationMetadataKeys does not claim; "+
							"an unclaimed key is one a caller can forge",
						key,
					)
				}
			}
		})
	}
}

func TestPolicyPropagatorOmitsInvalidContextRequestID(t *testing.T) {
	t.Parallel()

	ctx := reqctx.ContextWithRequestID(t.Context(), "contains spaces")
	carrier := propagation.MapCarrier{}
	correlationpolicy.NewPropagator(PropagationTrustedService, requestIDMetadataKey).Inject(ctx, carrier)
	if got := carrier.Get(requestIDMetadataKey); got != "" {
		t.Fatalf("x-request-id = %q, want invalid value omitted", got)
	}
}

func TestClientInterceptorsSanitizeCallerMetadataWithoutMutation(t *testing.T) {
	t.Parallel()

	original := metadata.MD{
		"TraceParent":   {"stale-parent"},
		"tracestate":    {"stale-state"},
		"BAGGAGE":       {"private=value"},
		"x-request-id":  {"stale-request"},
		"authorization": {"Bearer retained"},
	}
	ctx := metadata.NewOutgoingContext(t.Context(), original)

	var unaryCalled bool
	err := propagationUnaryInterceptor(
		ctx,
		"/test.Service/Unary",
		nil,
		nil,
		nil,
		func(callCtx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			unaryCalled = true
			assertSanitizedOutgoingMetadata(callCtx, t)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("unary interceptor error = %v", err)
	}
	if !unaryCalled {
		t.Fatal("unary invoker was not called")
	}

	var streamCalled bool
	_, err = propagationStreamInterceptor(
		ctx,
		&grpc.StreamDesc{},
		nil,
		"/test.Service/Stream",
		func(callCtx context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
			streamCalled = true
			assertSanitizedOutgoingMetadata(callCtx, t)
			// The interceptor test observes context and options before any
			// concrete stream is needed.
			return nil, nil //nolint:nilnil // No concrete stream is consumed by this interceptor unit test.
		},
	)
	if err != nil {
		t.Fatalf("stream interceptor error = %v", err)
	}
	if !streamCalled {
		t.Fatal("streamer was not called")
	}
	if got := original["TraceParent"]; len(got) != 1 || got[0] != "stale-parent" {
		t.Fatalf("caller metadata traceparent = %v, want unchanged", got)
	}
	if got := original["authorization"]; len(got) != 1 || got[0] != "Bearer retained" {
		t.Fatalf("caller metadata authorization = %v, want unchanged", got)
	}
}

func TestSanitizingPerRPCCredentialsPreserveAuthSecurityAndErrors(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("credential lookup failed")
	source := map[string]string{
		"authorization": "Bearer retained",
		"TraceParent":   "stale-parent",
		"tracestate":    "stale-state",
		"BAGGAGE":       "private=value",
		"x-request-id":  "stale-request",
	}
	base := staticPerRPCCredentials{
		values:          source,
		err:             sentinel,
		requireSecurity: true,
	}
	wrapped := wrapPerRPCCredentials(base)
	if wrapped == nil {
		t.Fatal("wrapPerRPCCredentials() = nil for non-nil base")
	}

	got, err := wrapped.GetRequestMetadata(t.Context(), "test:///service")
	if !errors.Is(err, sentinel) {
		t.Fatalf("GetRequestMetadata() error = %v, want sentinel", err)
	}
	if got["authorization"] != "Bearer retained" {
		t.Fatalf("authorization = %q, want retained", got["authorization"])
	}
	for _, key := range reservedCorrelationMetadataKeys {
		if value := credentialValueEqualFold(got, key); value != "" {
			t.Fatalf("%s = %q, want removed", key, value)
		}
	}
	if source["TraceParent"] != "stale-parent" {
		t.Fatalf("caller credential map was mutated: %v", source)
	}
	if !wrapped.RequireTransportSecurity() {
		t.Fatal("RequireTransportSecurity() = false, want delegated true")
	}
}

func TestCallOptionSanitizationClonesValueAndPointerCredentials(t *testing.T) {
	t.Parallel()

	base := &staticPerRPCCredentials{
		values: map[string]string{
			"authorization": "Bearer retained",
			"traceparent":   "stale-parent",
		},
	}
	pointerOption := &grpc.PerRPCCredsCallOption{Creds: base}
	options := []grpc.CallOption{
		grpc.PerRPCCredentials(base),
		pointerOption,
	}

	sanitized := sanitizeCallOptions(options)
	if len(sanitized) != len(options) {
		t.Fatalf("sanitized option count = %d, want %d", len(sanitized), len(options))
	}
	for index, option := range sanitized {
		perRPC, ok := option.(grpc.PerRPCCredsCallOption)
		if !ok {
			t.Fatalf("option %d type = %T, want grpc.PerRPCCredsCallOption", index, option)
		}
		values, err := perRPC.Creds.GetRequestMetadata(t.Context())
		if err != nil {
			t.Fatalf("option %d credential error = %v", index, err)
		}
		if values["authorization"] != "Bearer retained" || values["traceparent"] != "" {
			t.Fatalf("option %d metadata = %v, want auth only", index, values)
		}
	}
	if pointerOption.Creds != base {
		t.Fatal("caller-owned pointer call option was mutated")
	}
}

func assertSanitizedOutgoingMetadata(ctx context.Context, t *testing.T) {
	t.Helper()

	values, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("sanitized context has no outgoing metadata")
	}
	if got := values.Get("authorization"); len(got) != 1 || got[0] != "Bearer retained" {
		t.Fatalf("authorization = %v, want retained", got)
	}
	for _, key := range reservedCorrelationMetadataKeys {
		if value := metadataFirstEqualFold(values, key); value != "" {
			t.Fatalf("%s = %q, want removed", key, value)
		}
	}
}

func metadataFirstEqualFold(values metadata.MD, wanted string) string {
	for key, candidates := range values {
		if strings.EqualFold(key, wanted) && len(candidates) > 0 {
			return candidates[0]
		}
	}
	return ""
}

func credentialValueEqualFold(values map[string]string, wanted string) string {
	for key, value := range values {
		if strings.EqualFold(key, wanted) {
			return value
		}
	}
	return ""
}

// staticPerRPCCredentials is field for field the livePerRPCCredentials that
// propagation_test.go declares; that file names the boundary the two sit on and
// why one cannot be shared.
type staticPerRPCCredentials struct {
	values          map[string]string
	err             error
	requireSecurity bool
}

func (c staticPerRPCCredentials) GetRequestMetadata(
	context.Context,
	...string,
) (map[string]string, error) {
	return c.values, c.err
}

func (c staticPerRPCCredentials) RequireTransportSecurity() bool {
	return c.requireSecurity
}

var _ credentials.PerRPCCredentials = staticPerRPCCredentials{}

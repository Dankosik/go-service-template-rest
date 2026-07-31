//nolint:staticcheck // This file proves grpc-go's deprecated but live resolver metadata path is sanitized.
package grpcclient

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
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

			carrier := propagation.MapCarrier{
				"traceparent":  "stale",
				"tracestate":   "stale",
				"baggage":      "secret=value",
				"x-request-id": "stale",
			}
			for key := range carrier {
				delete(carrier, key)
			}
			policyPropagator{policy: testCase.policy}.Inject(ctx, carrier)

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

func TestPolicyPropagatorOmitsInvalidContextRequestID(t *testing.T) {
	t.Parallel()

	ctx := reqctx.ContextWithRequestID(t.Context(), "contains spaces")
	carrier := propagation.MapCarrier{}
	policyPropagator{policy: PropagationTrustedService}.Inject(ctx, carrier)
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

func TestResolverWrapperSanitizesEveryAddressAndPreservesLifecycle(t *testing.T) {
	t.Parallel()

	baseResolver := &recordingResolver{}
	baseBuilder := &recordingResolverBuilder{
		scheme:   "recording",
		resolver: baseResolver,
	}
	downstream := &recordingResolverClientConn{}
	wrapped := wrapResolverBuilder(baseBuilder)

	gotResolver, err := wrapped.Build(resolver.Target{}, downstream, resolver.BuildOptions{})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if gotResolver != baseResolver {
		t.Fatalf("Build() resolver = %T, want original resolver", gotResolver)
	}

	addressAttributes := attributes.New("address", "top")
	balancerAttributes := attributes.New("balancer", "top")
	endpointAttributes := attributes.New("endpoint", "one")
	endpointAddressAttributes := attributes.New("address", "endpoint")
	endpointBalancerAttributes := attributes.New("balancer", "endpoint")
	stateAttributes := attributes.New("state", "retained")
	serviceConfig := &serviceconfig.ParseResult{Err: errors.New("opaque parsed service config")}
	state := resolver.State{
		Addresses: []resolver.Address{{
			Addr:               "127.0.0.1:9000",
			ServerName:         "service.internal",
			Attributes:         addressAttributes,
			BalancerAttributes: balancerAttributes,
			Metadata:           map[string]string{"traceparent": "stale"},
		}},
		Endpoints: []resolver.Endpoint{{
			Attributes: endpointAttributes,
			Addresses: []resolver.Address{{
				Addr:               "127.0.0.1:9001",
				ServerName:         "endpoint.service.internal",
				Attributes:         endpointAddressAttributes,
				BalancerAttributes: endpointBalancerAttributes,
				Metadata:           map[string]string{"x-request-id": "stale"},
			}},
		}},
		ServiceConfig: serviceConfig,
		Attributes:    stateAttributes,
	}
	if err := baseBuilder.connection.UpdateState(state); err != nil {
		t.Fatalf("UpdateState() error = %v", err)
	}
	baseBuilder.connection.NewAddress(state.Addresses)

	assertResolverMetadataRemoved(t, downstream.state.Addresses)
	assertResolverMetadataRemoved(t, downstream.state.Endpoints[0].Addresses)
	assertResolverMetadataRemoved(t, downstream.addresses)
	if state.Addresses[0].Metadata == nil || state.Endpoints[0].Addresses[0].Metadata == nil {
		t.Fatal("caller-owned resolver state was mutated")
	}
	if downstream.updateCalls != 1 || downstream.newAddressCalls != 1 {
		t.Fatalf(
			"resolver callbacks = UpdateState %d NewAddress %d, want 1 each",
			downstream.updateCalls,
			downstream.newAddressCalls,
		)
	}
	if downstream.state.Addresses[0].Addr != state.Addresses[0].Addr ||
		downstream.state.Addresses[0].ServerName != state.Addresses[0].ServerName ||
		downstream.state.Addresses[0].Attributes != addressAttributes ||
		downstream.state.Addresses[0].BalancerAttributes != balancerAttributes {
		t.Fatalf("top-level address fields were not preserved: %+v", downstream.state.Addresses[0])
	}
	if downstream.state.Endpoints[0].Attributes != endpointAttributes ||
		downstream.state.Endpoints[0].Addresses[0].Addr !=
			state.Endpoints[0].Addresses[0].Addr ||
		downstream.state.Endpoints[0].Addresses[0].ServerName !=
			state.Endpoints[0].Addresses[0].ServerName ||
		downstream.state.Endpoints[0].Addresses[0].Attributes != endpointAddressAttributes ||
		downstream.state.Endpoints[0].Addresses[0].BalancerAttributes !=
			endpointBalancerAttributes {
		t.Fatalf("endpoint fields were not preserved: %+v", downstream.state.Endpoints[0])
	}
	if downstream.state.ServiceConfig != serviceConfig || downstream.state.Attributes != stateAttributes {
		t.Fatalf("state-owned fields were not preserved: %+v", downstream.state)
	}
	if downstream.addresses[0].Addr != state.Addresses[0].Addr ||
		downstream.addresses[0].ServerName != state.Addresses[0].ServerName ||
		downstream.addresses[0].Attributes != addressAttributes ||
		downstream.addresses[0].BalancerAttributes != balancerAttributes {
		t.Fatalf("NewAddress fields were not preserved: %+v", downstream.addresses[0])
	}
	downstream.state.Addresses[0].Addr = "changed"
	if state.Addresses[0].Addr != "127.0.0.1:9000" {
		t.Fatal("resolver address slices still alias caller state")
	}
	downstream.state.Endpoints[0].Addresses[0].Addr = "changed"
	if state.Endpoints[0].Addresses[0].Addr != "127.0.0.1:9001" {
		t.Fatal("resolver endpoint address slices still alias caller state")
	}
	downstream.addresses[0].Addr = "changed"
	if state.Addresses[0].Addr != "127.0.0.1:9000" {
		t.Fatal("deprecated NewAddress slice still aliases caller state")
	}

	updateErr := errors.New("resolver state rejected")
	downstream.updateErr = updateErr
	if err := baseBuilder.connection.UpdateState(state); !errors.Is(err, updateErr) {
		t.Fatalf("UpdateState() error = %v, want sentinel", err)
	}
	if downstream.updateCalls != 2 {
		t.Fatalf("UpdateState() calls = %d, want 2", downstream.updateCalls)
	}

	reportErr := errors.New("resolver lookup failed")
	baseBuilder.connection.ReportError(reportErr)
	if downstream.reportCalls != 1 || !errors.Is(downstream.reportErr, reportErr) {
		t.Fatalf(
			"ReportError() = calls %d error %v, want one sentinel callback",
			downstream.reportCalls,
			downstream.reportErr,
		)
	}

	parseResult := &serviceconfig.ParseResult{Err: errors.New("parsed config sentinel")}
	downstream.parseResult = parseResult
	if got := baseBuilder.connection.ParseServiceConfig(`{"loadBalancingConfig":[]}`); got != parseResult {
		t.Fatalf("ParseServiceConfig() = %p, want %p", got, parseResult)
	}
	if downstream.parseCalls != 1 ||
		downstream.parseInput != `{"loadBalancingConfig":[]}` {
		t.Fatalf(
			"ParseServiceConfig() callback = calls %d input %q",
			downstream.parseCalls,
			downstream.parseInput,
		)
	}

	gotResolver.ResolveNow(resolver.ResolveNowOptions{})
	gotResolver.Close()
	if baseResolver.resolveNowCalls != 1 || baseResolver.closeCalls != 1 {
		t.Fatalf(
			"resolver lifecycle calls = resolve %d close %d, want 1 each",
			baseResolver.resolveNowCalls,
			baseResolver.closeCalls,
		)
	}
}

func TestResolverWrapperPreservesOptionalAuthorityOverride(t *testing.T) {
	t.Parallel()

	base := &authorityResolverBuilder{
		recordingResolverBuilder: recordingResolverBuilder{
			scheme:   "authority",
			resolver: &recordingResolver{},
		},
		authority: "service.internal",
	}
	wrapped := wrapResolverBuilder(base)
	authority, ok := wrapped.(resolver.AuthorityOverrider)
	if !ok {
		t.Fatal("wrapped resolver lost AuthorityOverrider")
	}
	if got := authority.OverrideAuthority(resolver.Target{}); got != base.authority {
		t.Fatalf("OverrideAuthority() = %q, want %q", got, base.authority)
	}

	ordinary := &recordingResolverBuilder{
		scheme:   "ordinary",
		resolver: &recordingResolver{},
	}
	if _, ok := wrapResolverBuilder(ordinary).(resolver.AuthorityOverrider); ok {
		t.Fatal("ordinary resolver unexpectedly gained AuthorityOverrider")
	}
}

func TestResolverWrapperPreservesBuildError(t *testing.T) {
	t.Parallel()

	buildErr := errors.New("resolver build failed")
	base := &recordingResolverBuilder{
		scheme:   "failing",
		buildErr: buildErr,
	}
	wrapped := wrapResolverBuilder(base)
	if _, err := wrapped.Build(
		resolver.Target{},
		&recordingResolverClientConn{},
		resolver.BuildOptions{},
	); !errors.Is(err, buildErr) {
		t.Fatalf("Build() error = %v, want sentinel", err)
	}
	if base.buildCalls != 1 {
		t.Fatalf("Build() calls = %d, want 1", base.buildCalls)
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

type recordingResolver struct {
	resolveNowCalls int
	closeCalls      int
}

func (r *recordingResolver) ResolveNow(resolver.ResolveNowOptions) {
	r.resolveNowCalls++
}

func (r *recordingResolver) Close() {
	r.closeCalls++
}

type recordingResolverBuilder struct {
	scheme     string
	resolver   resolver.Resolver
	connection resolver.ClientConn
	buildErr   error
	buildCalls int
}

func (b *recordingResolverBuilder) Build( //nolint:ireturn // Implements resolver.Builder for delegation proof.
	_ resolver.Target,
	connection resolver.ClientConn,
	_ resolver.BuildOptions,
) (resolver.Resolver, error) {
	b.buildCalls++
	b.connection = connection
	return b.resolver, b.buildErr
}

func (b *recordingResolverBuilder) Scheme() string {
	return b.scheme
}

type authorityResolverBuilder struct {
	recordingResolverBuilder

	authority string
}

func (b authorityResolverBuilder) OverrideAuthority(resolver.Target) string {
	return b.authority
}

type recordingResolverClientConn struct {
	state           resolver.State
	addresses       []resolver.Address
	updateErr       error
	reportErr       error
	parseResult     *serviceconfig.ParseResult
	parseInput      string
	updateCalls     int
	reportCalls     int
	newAddressCalls int
	parseCalls      int
}

func (c *recordingResolverClientConn) UpdateState(state resolver.State) error {
	c.updateCalls++
	c.state = state
	return c.updateErr
}

func (c *recordingResolverClientConn) ReportError(err error) {
	c.reportCalls++
	c.reportErr = err
}

func (c *recordingResolverClientConn) NewAddress(addresses []resolver.Address) {
	c.newAddressCalls++
	c.addresses = addresses
}

func (c *recordingResolverClientConn) ParseServiceConfig(
	serviceConfig string,
) *serviceconfig.ParseResult {
	c.parseCalls++
	c.parseInput = serviceConfig
	return c.parseResult
}

func assertResolverMetadataRemoved(t *testing.T, addresses []resolver.Address) {
	t.Helper()

	for _, address := range addresses {
		if address.Metadata != nil {
			t.Fatalf("resolver address %q metadata = %v, want nil", address.Addr, address.Metadata)
		}
	}
}

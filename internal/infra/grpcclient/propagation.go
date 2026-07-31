package grpcclient

import (
	"context"
	"net/url"
	"slices"
	"strings"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
)

const requestIDMetadataKey = "x-request-id"

var reservedCorrelationMetadataKeys = [...]string{
	"traceparent",
	"tracestate",
	"baggage",
	requestIDMetadataKey,
}

// PropagationPolicy controls which locally owned correlation values may cross
// this client's trust boundary.
type PropagationPolicy uint8

const (
	// PropagationNone keeps local client telemetry but emits no remote
	// correlation metadata.
	PropagationNone PropagationPolicy = iota
	// PropagationTraceContext emits the W3C trace context only.
	PropagationTraceContext
	// PropagationTrustedService emits the W3C trace context and an accepted
	// request ID to a trusted service.
	PropagationTrustedService
)

func (p PropagationPolicy) valid() bool {
	return p <= PropagationTrustedService
}

type policyPropagator struct {
	policy PropagationPolicy
}

func (p policyPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	if p.policy == PropagationNone {
		return
	}

	propagation.TraceContext{}.Inject(ctx, carrier)
	if p.policy != PropagationTrustedService {
		return
	}
	requestID := reqctx.RequestID(ctx)
	if reqctx.ValidRequestID(requestID) {
		carrier.Set(requestIDMetadataKey, requestID)
	}
}

func (policyPropagator) Extract(ctx context.Context, _ propagation.TextMapCarrier) context.Context {
	return ctx
}

func (p policyPropagator) Fields() []string {
	if p.policy == PropagationNone {
		return nil
	}
	fields := propagation.TraceContext{}.Fields()
	if p.policy == PropagationTrustedService {
		fields = append(fields, requestIDMetadataKey)
	}
	return fields
}

func propagationUnaryInterceptor(
	ctx context.Context,
	method string,
	request any,
	reply any,
	connection *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	options ...grpc.CallOption,
) error {
	return invoker(
		sanitizeOutgoingContext(ctx),
		method,
		request,
		reply,
		connection,
		sanitizeCallOptions(options)...,
	)
}

func propagationStreamInterceptor( //nolint:ireturn // grpc.StreamClientInterceptor requires grpc.ClientStream.
	ctx context.Context,
	description *grpc.StreamDesc,
	connection *grpc.ClientConn,
	method string,
	streamer grpc.Streamer,
	options ...grpc.CallOption,
) (grpc.ClientStream, error) {
	return streamer(
		sanitizeOutgoingContext(ctx),
		description,
		connection,
		method,
		sanitizeCallOptions(options)...,
	)
}

func sanitizeOutgoingContext(ctx context.Context) context.Context {
	outgoing, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return ctx
	}
	sanitized := outgoing.Copy()
	deleteReservedMetadata(sanitized)
	return metadata.NewOutgoingContext(ctx, sanitized)
}

func sanitizeCallOptions(options []grpc.CallOption) []grpc.CallOption {
	sanitized := slices.Clone(options)
	for index, option := range sanitized {
		switch perRPC := option.(type) {
		case grpc.PerRPCCredsCallOption:
			perRPC.Creds = wrapPerRPCCredentials(perRPC.Creds)
			sanitized[index] = perRPC
		case *grpc.PerRPCCredsCallOption:
			if perRPC == nil {
				continue
			}
			cloned := *perRPC
			cloned.Creds = wrapPerRPCCredentials(cloned.Creds)
			sanitized[index] = cloned
		}
	}
	return sanitized
}

func wrapPerRPCCredentials( //nolint:ireturn // The gRPC call option owns this interface contract.
	base credentials.PerRPCCredentials,
) credentials.PerRPCCredentials {
	if base == nil {
		return nil
	}
	return sanitizingPerRPCCredentials{base: base}
}

type sanitizingPerRPCCredentials struct {
	base credentials.PerRPCCredentials
}

func (c sanitizingPerRPCCredentials) GetRequestMetadata(
	ctx context.Context,
	uri ...string,
) (map[string]string, error) {
	values, err := c.base.GetRequestMetadata(ctx, uri...)
	if values == nil {
		// Preserve the credential owner's exact error; gRPC maps it into the
		// authentication failure seen by the caller.
		//nolint:wrapcheck // Preserve the credential provider's authentication failure unchanged.
		return nil, err
	}
	sanitized := make(map[string]string, len(values))
	for key, value := range values {
		if !reservedCorrelationMetadataKey(key) {
			sanitized[key] = value
		}
	}
	// Preserve the credential owner's exact error while replacing only its
	// returned metadata map.
	//nolint:wrapcheck // Preserve the credential provider's authentication failure unchanged.
	return sanitized, err
}

func (c sanitizingPerRPCCredentials) RequireTransportSecurity() bool {
	return c.base.RequireTransportSecurity()
}

func deleteReservedMetadata(values metadata.MD) {
	for key := range values {
		if reservedCorrelationMetadataKey(key) {
			delete(values, key)
		}
	}
}

func reservedCorrelationMetadataKey(key string) bool {
	for _, reserved := range reservedCorrelationMetadataKeys {
		if strings.EqualFold(key, reserved) {
			return true
		}
	}
	return false
}

func sanitizingResolverBuilders(target string) []resolver.Builder {
	schemes := []string{"dns", resolver.GetDefaultScheme()}
	if parsed, err := url.Parse(target); err == nil {
		schemes = append(schemes, parsed.Scheme)
	}

	seen := make(map[string]struct{}, len(schemes))
	builders := make([]resolver.Builder, 0, len(schemes))
	for _, scheme := range schemes {
		base := resolver.Get(scheme)
		if base == nil {
			continue
		}
		if _, ok := seen[base.Scheme()]; ok {
			continue
		}
		seen[base.Scheme()] = struct{}{}
		builders = append(builders, wrapResolverBuilder(base))
	}
	return builders
}

func wrapResolverBuilder( //nolint:ireturn // Optional AuthorityOverrider support requires two concrete wrappers.
	base resolver.Builder,
) resolver.Builder {
	wrapped := sanitizingResolverBuilder{base: base}
	authority, ok := base.(resolver.AuthorityOverrider)
	if !ok {
		return wrapped
	}
	return sanitizingAuthorityResolverBuilder{
		sanitizingResolverBuilder: wrapped,
		authority:                 authority,
	}
}

type sanitizingResolverBuilder struct {
	base resolver.Builder
}

func (b sanitizingResolverBuilder) Build( //nolint:ireturn // Implements resolver.Builder.
	target resolver.Target,
	connection resolver.ClientConn,
	options resolver.BuildOptions,
) (resolver.Resolver, error) {
	//nolint:wrapcheck // Preserve the selected resolver's exact build failure.
	return b.base.Build(target, sanitizingResolverClientConn{ClientConn: connection}, options)
}

func (b sanitizingResolverBuilder) Scheme() string {
	return b.base.Scheme()
}

type sanitizingAuthorityResolverBuilder struct {
	sanitizingResolverBuilder

	authority resolver.AuthorityOverrider
}

func (b sanitizingAuthorityResolverBuilder) OverrideAuthority(target resolver.Target) string {
	return b.authority.OverrideAuthority(target)
}

type sanitizingResolverClientConn struct {
	resolver.ClientConn
}

func (c sanitizingResolverClientConn) UpdateState(state resolver.State) error {
	// The resolver contract requires returning the downstream ClientConn error
	// unchanged; callers may use its identity for recovery.
	//nolint:wrapcheck // Preserve resolver.ClientConn recovery semantics and error identity.
	return c.ClientConn.UpdateState(sanitizeResolverState(state))
}

func (c sanitizingResolverClientConn) NewAddress(addresses []resolver.Address) {
	c.ClientConn.NewAddress( //nolint:staticcheck // Compatibility path required by resolver.ClientConn.
		sanitizeResolverAddresses(addresses),
	)
}

func sanitizeResolverState(state resolver.State) resolver.State {
	state.Addresses = sanitizeResolverAddresses(state.Addresses)
	state.Endpoints = slices.Clone(state.Endpoints)
	for index := range state.Endpoints {
		state.Endpoints[index].Addresses = sanitizeResolverAddresses(state.Endpoints[index].Addresses)
	}
	return state
}

func sanitizeResolverAddresses(addresses []resolver.Address) []resolver.Address {
	sanitized := slices.Clone(addresses)
	for index := range sanitized {
		sanitized[index].Metadata = nil //nolint:staticcheck // Deprecated Metadata remains a live grpc-go disclosure path.
	}
	return sanitized
}

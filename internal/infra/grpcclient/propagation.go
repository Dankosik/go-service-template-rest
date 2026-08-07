package grpcclient

import (
	"context"
	"slices"
	"strings"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const requestIDMetadataKey = reqctx.RequestIDMetadataKey

// reservedCorrelationMetadataKeys are the keys this client alone may set on an
// outgoing RPC.
//
// The invariant the rest of this file exists to hold: policyPropagator is the
// only permitted source of these keys, so what crosses the trust boundary is
// exactly what PropagationPolicy selected. Three separate paths can otherwise
// put a value under one of these keys on the wire, and each has its own strip
// seam in this package, applied before OpenTelemetry injects the allowlist:
//
//	caller-supplied outgoing metadata   -> sanitizeOutgoingContext
//	credential-supplied metadata        -> sanitizingPerRPCCredentials
//	resolver-supplied address metadata  -> sanitizeResolverAddresses, resolver.go
//
// Removing any one of them widens the trust boundary without changing a single
// call site; the propagation and resolver-selection tests are what fail.
// client.go's WithNoProxy and WithDisableServiceConfig close the two remaining
// routes by which a peer could introduce a fourth.
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

// sanitizeCallOptions rewraps any per-RPC credential so the metadata it returns
// passes the same strip seam as everything else bound for the wire.
//
// Both forms of the option are reachable, which is why the switch has two arms
// that look redundant. grpc.PerRPCCredentials returns the value form, but
// grpc.CallOption is satisfied by a pointer to it as well, so a caller that
// builds the option as a composite literal and takes its address produces the
// pointer form. Each arm writes a copy into the cloned slice rather than
// mutating in place: rewriting through the pointer would change the caller's
// own option value.
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

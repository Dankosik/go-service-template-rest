package grpcclient

import (
	"context"
	"maps"
	"slices"

	"github.com/example/go-service-template-rest/internal/observability/correlationpolicy"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const requestIDMetadataKey = reqctx.RequestIDMetadataKey

// reservedCorrelationMetadataKey reports a key this client alone may set on an
// outgoing RPC. correlationpolicy owns which keys those are and how one is
// matched, so this client and the HTTP one cannot read one list two ways.
//
// The invariant this file holds: the correlationpolicy propagator is the only
// permitted source of these keys, so what crosses the trust boundary is exactly
// what PropagationPolicy selected. Three other paths can put a value under one of
// these keys, each with its own strip seam, applied before OpenTelemetry injects
// the allowlist:
//
//	caller-supplied outgoing metadata   -> sanitizeOutgoingContext
//	credential-supplied metadata        -> sanitizingPerRPCCredentials
//	resolver-supplied address metadata  -> sanitizeResolverAddresses, resolver.go
//
// Removing any one of them widens the trust boundary without changing a call
// site; the propagation and resolver-selection tests are what fail. client.go's
// WithNoProxy and WithDisableServiceConfig close the two remaining routes a peer
// could use. The latter refuses only resolver-supplied service config; the
// client's own default is not a route a peer controls.
var reservedCorrelationMetadataKey = correlationpolicy.Reserved(requestIDMetadataKey)

// PropagationPolicy controls which locally owned correlation values may cross
// this client's trust boundary.
//
// It is an alias rather than a second enum so the httpclient half cannot answer
// the same question differently; correlationpolicy owns the rule.
type PropagationPolicy = correlationpolicy.Policy

const (
	// PropagationNone keeps local client telemetry but emits no remote
	// correlation metadata.
	PropagationNone = correlationpolicy.None
	// PropagationTraceContext emits the W3C trace context only.
	PropagationTraceContext = correlationpolicy.TraceContext
	// PropagationTrustedService emits the W3C trace context and an accepted
	// request ID to a trusted service.
	PropagationTrustedService = correlationpolicy.TrustedService
)

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
// The switch's two arms only look redundant: grpc.CallOption is satisfied by both
// the value form grpc.PerRPCCredentials returns and a pointer to it, which a
// caller taking the address of a composite literal produces. Each arm writes a
// copy into the cloned slice, because rewriting through the pointer would change
// the caller's own option value.
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

func wrapPerRPCCredentials( //nolint:ireturn // gRPC credential options own this interface contract.
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
	sanitized := maps.Clone(values)
	maps.DeleteFunc(sanitized, func(key, _ string) bool {
		return reservedCorrelationMetadataKey(key)
	})
	// Preserve the credential owner's exact error while replacing only its
	// returned metadata map.
	//nolint:wrapcheck // Preserve the credential provider's authentication failure unchanged.
	return sanitized, err
}

func (c sanitizingPerRPCCredentials) RequireTransportSecurity() bool {
	return c.base.RequireTransportSecurity()
}

func deleteReservedMetadata(values metadata.MD) {
	maps.DeleteFunc(values, func(key string, _ []string) bool {
		return reservedCorrelationMetadataKey(key)
	})
}

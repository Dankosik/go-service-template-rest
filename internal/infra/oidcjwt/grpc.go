package oidcjwt

import (
	"context"
	"errors"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryInterceptor authenticates every non-health unary RPC once.
func (v *Verifier) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		clean, incoming := splitAuthorizationMetadata(ctx)
		if info != nil && publicHealthMethod(info.FullMethod) {
			return handler(clean, request)
		}
		authenticated, err := v.authenticateRPC(clean, incoming)
		if err != nil {
			return nil, grpcAuthenticationError(err)
		}
		return handler(authenticated, request)
	}
}

// StreamInterceptor authenticates every non-health stream once and replaces
// the handler-visible stream context with the immutable principal context.
func (v *Verifier) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		clean, incoming := splitAuthorizationMetadata(stream.Context())
		if info != nil && publicHealthMethod(info.FullMethod) {
			return handler(server, serverStreamWithContext{ServerStream: stream, ctx: clean})
		}
		authenticated, err := v.authenticateRPC(clean, incoming)
		if err != nil {
			return grpcAuthenticationError(err)
		}
		return handler(server, serverStreamWithContext{ServerStream: stream, ctx: authenticated})
	}
}

// publicHealthMethod is an exact allowlist rather than a prefix match on the
// health service. This is the boundary that decides which RPCs need no
// credential, so a method grpc-go adds to grpc.health.v1.Health later must be
// authenticated until someone deliberately publishes it. The transport adapter
// in internal/infra/grpc matches the same service by prefix on purpose: it only
// governs admission, logging, and telemetry, where over-matching is cheap.
func publicHealthMethod(fullMethod string) bool {
	return fullMethod == healthpb.Health_Check_FullMethodName ||
		fullMethod == healthpb.Health_Watch_FullMethodName
}

// authenticateRPC returns ctx carrying the principal proven by incoming.
//
// The credential arrives as metadata rather than as a second context so that it
// cannot be swapped with ctx: ctx is the handler-visible context, whose
// authorization metadata has already been removed, so reading the credential
// from it would find nothing. The compiler now rejects the mistake that this
// signature previously had to describe.
func (v *Verifier) authenticateRPC(
	ctx context.Context,
	incoming metadata.MD,
) (context.Context, error) {
	principal, err := v.verifyCredential(ctx, incoming.Get("authorization"), TransportGRPC)
	if err != nil {
		return nil, err
	}
	return reqctx.ContextWithPrincipal(ctx, principal), nil
}

// splitAuthorizationMetadata returns the context a handler may see and the
// credential exactly as it arrived. Producing both from one call is what keeps
// them consistent: past this point the credential exists only as the returned
// metadata, and every context in play has already had it removed.
func splitAuthorizationMetadata(ctx context.Context) (context.Context, metadata.MD) {
	incoming, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, nil
	}
	clean := incoming.Copy()
	clean.Delete("authorization")
	return metadata.NewIncomingContext(ctx, clean), incoming
}

// grpcAuthenticationError renders a verification failure as the status the
// caller receives.
//
// A plain status.Error is the whole of what this owes the transport: the policy
// boundary in internal/infra/grpc trusts any status a policy interceptor
// returns directly, so a policy needs no counterpart of that package's own
// provenance marker.
//
// The arms below are grouped by the answer rather than in [Kind] declaration
// order, which makes the two categories that get their own status visible as the
// exceptions they are; the exhaustive linter holds the set complete either way.
//
// KindUntrustedTransport is named here and cannot arrive here. Only the HTTP
// adapter produces it, from trustedHTTPRequest; this transport's equivalent is
// validateGRPCAuthnTransport in internal/config, which refuses to start a service
// whose gRPC server is not TLS while this profile is on. The arm is not dead
// code, because a Kind may not be omitted from this switch — answering it as a
// rejected credential keeps the switch exhaustive without inventing a status for
// a case that cannot reach here.
func grpcAuthenticationError(err error) error {
	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "authentication canceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, "authentication deadline exceeded")
	}
	kind, _ := KindOf(err)
	switch kind {
	case KindOversize:
		return status.Error(codes.ResourceExhausted, "authentication credential is too large")
	case KindUnavailable:
		return status.Error(codes.Unavailable, "authentication trust is unavailable")
	case KindMissing, KindMalformed, KindInvalid, KindUntrustedTransport:
		return unauthenticatedCredential()
	default:
		// Unreachable for a declared Kind, because the mandatory exhaustive linter
		// holds the arms above to the full set. It answers the same way as the
		// credential arm so that an error arriving with no Kind at all — one the
		// context tests above did not claim — still fails closed.
		return unauthenticatedCredential()
	}
}

// unauthenticatedCredential is the one answer every rejected credential
// receives, whichever category refused it. A single message for the whole set is
// deliberate: the caller learns that its credential was not accepted and nothing
// about which check said so.
func unauthenticatedCredential() error {
	return status.Error(codes.Unauthenticated, "authentication credential is missing or invalid")
}

// serverStreamWithContext replaces the context a streaming handler observes.
//
// internal/infra/grpc declares an identical type for the same reason. Sharing
// one would make this package import that transport adapter, which is the
// dependency direction its Options seam exists to avoid, so the duplication is
// deliberate.
type serverStreamWithContext struct {
	grpc.ServerStream

	ctx context.Context //nolint:containedctx // grpc.ServerStream requires Context to return the authenticated RPC context.
}

func (s serverStreamWithContext) Context() context.Context {
	return s.ctx
}

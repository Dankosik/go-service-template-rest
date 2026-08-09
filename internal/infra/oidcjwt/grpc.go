package oidcjwt

import (
	"context"
	"errors"
	"time"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryInterceptor authenticates every unary RPC except the public health
// Check probe once.
func (v *Verifier) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		clean, incoming := splitAuthorizationMetadata(ctx)
		if publicHealthMethod(info.FullMethod) {
			return handler(clean, request)
		}
		authenticated, _, err := v.authenticateRPC(clean, incoming)
		if err != nil {
			return nil, grpcAuthenticationError(err)
		}
		return handler(authenticated, request)
	}
}

// StreamInterceptor authenticates every stream once and bounds its
// handler-visible context by the verified token lifetime.
func (v *Verifier) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		_ *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		clean, incoming := splitAuthorizationMetadata(stream.Context())
		authenticated, expiresAt, err := v.authenticateRPC(clean, incoming)
		if err != nil {
			return grpcAuthenticationError(err)
		}
		bounded, cancel := context.WithDeadline(authenticated, expiresAt.Add(ClockSkew))
		defer cancel()
		return handler(server, serverStreamWithContext{ServerStream: stream, ctx: bounded})
	}
}

// publicHealthMethod is an exact allowlist rather than a prefix match on the
// health service. This is the boundary that decides which RPC needs no
// credential, so a method grpc-go adds to grpc.health.v1.Health later must be
// authenticated until someone deliberately publishes it. The transport adapter
// in internal/infra/grpc also exempts only Check from admission; its wider
// health-service prefix governs logging and telemetry alone.
func publicHealthMethod(fullMethod string) bool {
	return fullMethod == healthpb.Health_Check_FullMethodName
}

// authenticateRPC returns ctx carrying the principal proven by incoming and the
// expiry that bounds a streaming RPC.
//
// The credential arrives as metadata rather than as a second context so that the
// two cannot be swapped: ctx is the handler-visible context, whose authorization
// metadata has already been removed, so reading the credential from it would
// find nothing.
func (v *Verifier) authenticateRPC(
	ctx context.Context,
	incoming metadata.MD,
) (context.Context, time.Time, error) {
	verified, err := v.verifyCredential(ctx, incoming.Get("authorization"), transportGRPC)
	if err != nil {
		return nil, time.Time{}, err
	}
	return reqctx.ContextWithPrincipal(ctx, verified.principal), verified.expiresAt, nil
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
// boundary in internal/infra/grpc trusts any status a policy interceptor returns
// directly, so a policy needs no counterpart of that package's own provenance
// marker.
//
// The arms are grouped by the answer rather than in [Kind] declaration order,
// which makes the two categories that get their own status visible as the
// exceptions they are; the exhaustive linter holds the set complete either way.
//
// KindUntrustedTransport is named here and cannot arrive here: only the HTTP
// adapter produces it, and this transport's equivalent is
// validateGRPCAuthnTransport in internal/config, which refuses to start a
// service whose gRPC server is not TLS while this profile is on. The arm is not
// dead code — a Kind may not be omitted from this switch — and answering it as a
// rejected credential keeps the set complete without inventing a status.
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
	case KindMissing, KindMalformed, KindInvalid, KindLifetime, KindUntrustedTransport:
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

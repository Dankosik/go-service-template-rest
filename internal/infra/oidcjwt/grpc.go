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
		clean := withoutAuthorizationMetadata(ctx)
		if info != nil && publicHealthMethod(info.FullMethod) {
			return handler(clean, request)
		}
		authenticated, err := v.authenticateRPC(clean, ctx)
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
		clean := withoutAuthorizationMetadata(stream.Context())
		if info != nil && publicHealthMethod(info.FullMethod) {
			return handler(server, serverStreamWithContext{ServerStream: stream, ctx: clean})
		}
		authenticated, err := v.authenticateRPC(clean, stream.Context())
		if err != nil {
			return grpcAuthenticationError(err)
		}
		return handler(server, serverStreamWithContext{ServerStream: stream, ctx: authenticated})
	}
}

func publicHealthMethod(fullMethod string) bool {
	return fullMethod == healthpb.Health_Check_FullMethodName ||
		fullMethod == healthpb.Health_Watch_FullMethodName
}

func (v *Verifier) authenticateRPC(clean, source context.Context) (context.Context, error) {
	incoming, _ := metadata.FromIncomingContext(source)
	token, err := bearerToken(incoming.Get("authorization"))
	if err != nil {
		v.metrics.recordVerification(clean, TransportGRPC, err)
		return nil, err
	}
	principal, err := v.Verify(clean, token, TransportGRPC)
	if err != nil {
		return nil, err
	}
	return reqctx.ContextWithPrincipal(clean, principal), nil
}

func withoutAuthorizationMetadata(ctx context.Context) context.Context {
	incoming, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	clean := incoming.Copy()
	clean.Delete("authorization")
	return metadata.NewIncomingContext(ctx, clean)
}

func grpcAuthenticationError(err error) error {
	if errors.Is(err, context.Canceled) {
		return ownedAuthenticationStatus(codes.Canceled, "authentication canceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ownedAuthenticationStatus(codes.DeadlineExceeded, "authentication deadline exceeded")
	}
	kind, _ := KindOf(err)
	switch kind {
	case KindOversize:
		return ownedAuthenticationStatus(codes.ResourceExhausted, "authentication credential is too large")
	case KindUnavailable:
		return ownedAuthenticationStatus(codes.Unavailable, "authentication trust is unavailable")
	case KindMissing, KindMalformed, KindInvalid, KindUntrustedTransport:
		return ownedAuthenticationStatus(codes.Unauthenticated, "authentication credential is missing or invalid")
	default:
		return ownedAuthenticationStatus(codes.Unauthenticated, "authentication credential is missing or invalid")
	}
}

type serverStreamWithContext struct {
	grpc.ServerStream

	ctx context.Context //nolint:containedctx // grpc.ServerStream requires Context to return the authenticated RPC context.
}

func (s serverStreamWithContext) Context() context.Context {
	return s.ctx
}

type authenticationStatusError struct {
	status *status.Status
}

func ownedAuthenticationStatus(code codes.Code, message string) error {
	return &authenticationStatusError{status: status.New(code, message)}
}

func (e *authenticationStatusError) Error() string {
	return e.status.Err().Error()
}

func (e *authenticationStatusError) GRPCStatus() *status.Status {
	return e.status
}

package grpcx

import (
	"context"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

const requestIDMetadataKey = reqctx.RequestIDMetadataKey

func correlationUnaryInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		enriched, requestID := contextWithRequestID(ctx)
		if err := grpc.SetHeader(enriched, metadata.Pairs(requestIDMetadataKey, requestID)); err != nil {
			return nil, correlationFailure(enriched, log, info.FullMethod, err)
		}
		return handler(enriched, request)
	}
}

func correlationStreamInterceptor(log *slog.Logger) grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		enriched, requestID := contextWithRequestID(stream.Context())
		if err := stream.SetHeader(metadata.Pairs(requestIDMetadataKey, requestID)); err != nil {
			return correlationFailure(enriched, log, info.FullMethod, err)
		}
		return handler(server, serverStreamWithContext{ServerStream: stream, ctx: enriched})
	}
}

// correlationFailure answers an RPC whose response metadata could not be
// published, and is the reason correlation carries a logger at all.
//
// Correlation is the outermost interceptor, so it is the one position whose own
// failure never reaches the access log below it: that record would otherwise be
// the only trace of an RPC this transport answered. The status stays generic for
// the same reason every other owned status does — grpc-go's transport error is
// not the caller's business.
func correlationFailure(ctx context.Context, log *slog.Logger, method string, err error) error {
	log.LogAttrs(
		ctx,
		slog.LevelError,
		"grpc_correlation_failed",
		slog.String("rpc.method", method),
		slog.String("error", err.Error()),
	)
	return ownedStatus(codes.Internal, "failed to prepare response metadata")
}

func contextWithRequestID(ctx context.Context) (context.Context, string) {
	incoming, _ := metadata.FromIncomingContext(ctx)
	values := incoming.Get(requestIDMetadataKey)
	// Exactly one value is a candidate. With two or more, which identifier the
	// logs, the response metadata, and the next hop agree on would depend on an
	// ordering the caller controls, so none of them is accepted and a fresh ID
	// is minted instead. This is deliberately stricter than the HTTP transport,
	// where Header.Get takes the first of several.
	candidate := ""
	if len(values) == 1 {
		candidate = values[0]
	}
	return reqctx.ContextWithAcceptedRequestID(ctx, candidate)
}

// serverStreamWithContext replaces the context a streaming handler observes.
// The package doc owns why a policy package declares its own copy rather than
// sharing this one.
// profile:authn-oidc-jwt:start
// internal/infra/oidcjwt/grpc.go is that copy.
// profile:authn-oidc-jwt:end
type serverStreamWithContext struct {
	grpc.ServerStream

	ctx context.Context //nolint:containedctx // grpc.ServerStream requires Context to return the enriched RPC context.
}

func (s serverStreamWithContext) Context() context.Context {
	return s.ctx
}

package grpcx

import (
	"context"
	"log/slog"
	"time"

	"buf.build/go/protovalidate"
	"google.golang.org/grpc"
)

type aroundRPC func(context.Context, string, func(context.Context) error) error

func asUnaryInterceptor(around aroundRPC) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		var response any
		err := around(ctx, info.FullMethod, func(callCtx context.Context) error {
			var callErr error
			response, callErr = handler(callCtx, request)
			return callErr
		})
		return response, err
	}
}

func asStreamInterceptor(around aroundRPC) grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := stream.Context()
		return around(ctx, info.FullMethod, func(callCtx context.Context) error {
			if callCtx == ctx {
				return handler(server, stream)
			}
			return handler(server, serverStreamWithContext{ServerStream: stream, ctx: callCtx})
		})
	}
}

type serverStreamWithContext struct {
	grpc.ServerStream

	ctx context.Context //nolint:containedctx // grpc.ServerStream requires Context.
}

func (s serverStreamWithContext) Context() context.Context { return s.ctx }

func unaryChain(
	log *slog.Logger,
	admission admissionPolicy,
	timeout time.Duration,
	supplied []grpc.UnaryServerInterceptor,
	validator protovalidate.Validator,
	handlerErrors aroundRPC,
) []grpc.UnaryServerInterceptor {
	chain := []grpc.UnaryServerInterceptor{
		asUnaryInterceptor(recoveryAround(log)),
		asUnaryInterceptor(deadlineAround(timeout)),
		asUnaryInterceptor(admission.around),
		asUnaryInterceptor(policyErrorBoundary(log)),
	}
	chain = append(chain, supplied...)
	chain = append(chain, validationUnaryInterceptor(log, validator))
	return append(chain, asUnaryInterceptor(handlerErrors))
}

func streamChain(
	log *slog.Logger,
	admission admissionPolicy,
	supplied []grpc.StreamServerInterceptor,
	validator protovalidate.Validator,
	handlerErrors aroundRPC,
) []grpc.StreamServerInterceptor {
	chain := []grpc.StreamServerInterceptor{
		asStreamInterceptor(recoveryAround(log)),
		asStreamInterceptor(admission.around),
		asStreamInterceptor(policyErrorBoundary(log)),
	}
	chain = append(chain, supplied...)
	chain = append(chain, validationStreamInterceptor(log, validator))
	return append(chain, asStreamInterceptor(handlerErrors))
}

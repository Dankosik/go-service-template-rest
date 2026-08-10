package grpcx

import (
	"context"
	"log/slog"
	"runtime/debug"

	"github.com/example/go-service-template-rest/internal/failure"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
	"google.golang.org/grpc/codes"
)

func recoveryAround(log *slog.Logger) aroundRPC {
	return func(ctx context.Context, fullMethod string, call func(context.Context) error) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logRecoveredPanic(ctx, log, fullMethod, recovered)
				err = ownedStatus(codes.Internal, failure.SanitizedDetail)
			}
		}()
		return call(ctx)
	}
}

func logRecoveredPanic(ctx context.Context, log *slog.Logger, method string, recovered any) {
	// debug.Stack is taken here, inside the deferred recovery, because that is
	// the only point the panicking frames still exist.
	log.ErrorContext(
		ctx,
		"grpc_panic_recovered",
		append([]any{"rpc.method", method}, logctx.PanicAttrs(recovered, debug.Stack())...)...,
	)
}

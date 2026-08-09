package grpcx

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/example/go-service-template-rest/internal/failure"
	"google.golang.org/grpc/codes"
)

func recoveryAround(log *slog.Logger) aroundRPC {
	return func(ctx context.Context, fullMethod string, call func(context.Context) error) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logRecoveredPanic(ctx, log, fullMethod, recovered)
				err = ownedStatus(codes.Internal, sanitizedFailureDetail)
			}
		}()
		return call(ctx)
	}
}

func logRecoveredPanic(ctx context.Context, log *slog.Logger, method string, recovered any) {
	log.ErrorContext(
		ctx,
		"grpc_panic_recovered",
		"rpc.method", method,
		"panic.class", failure.PanicClass(recovered),
		"panic.type", fmt.Sprintf("%T", recovered),
		"stack", string(debug.Stack()),
	)
}

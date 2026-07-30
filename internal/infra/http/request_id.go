package httpx

import (
	"context"

	"github.com/example/go-service-template-rest/internal/reqctx"
)

const maxRequestIDLength = reqctx.MaxRequestIDLength

// contextWithRequestID validates the candidate, generates a replacement when needed,
// and returns both the enriched context and the accepted identifier.
//
// The carrier lives in internal/reqctx rather than here so a feature package can
// read the identifier it needs for its own records. A key owned by this package
// would be unreachable from one: depguard forbids a feature package from
// importing an infra adapter.
func contextWithRequestID(ctx context.Context, candidate string) (context.Context, string) {
	return reqctx.ContextWithAcceptedRequestID(ctx, candidate)
}

// requestIDFromContext returns the validated request identifier stored in ctx.
func requestIDFromContext(ctx context.Context) string {
	return reqctx.RequestID(ctx)
}

func validRequestID(value string) bool {
	return reqctx.ValidRequestID(value)
}

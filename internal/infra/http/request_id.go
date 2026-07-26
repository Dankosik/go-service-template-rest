package httpx

import (
	"context"
	"crypto/rand"
	"strings"

	"github.com/example/go-service-template-rest/internal/reqctx"
)

const maxRequestIDLength = 128

// contextWithRequestID validates the candidate, generates a replacement when needed,
// and returns both the enriched context and the accepted identifier.
//
// The carrier lives in internal/reqctx rather than here so a feature package can
// read the identifier it needs for its own records. A key owned by this package
// would be unreachable from one: depguard forbids a feature package from
// importing an infra adapter.
func contextWithRequestID(ctx context.Context, candidate string) (context.Context, string) {
	requestID := strings.TrimSpace(candidate)
	if !validRequestID(requestID) {
		requestID = newRequestID()
	}
	return reqctx.ContextWithRequestID(ctx, requestID), requestID
}

// requestIDFromContext returns the validated request identifier stored in ctx.
func requestIDFromContext(ctx context.Context) string {
	return reqctx.RequestID(ctx)
}

func validRequestID(value string) bool {
	if len(value) == 0 || len(value) > maxRequestIDLength {
		return false
	}
	for i := 0; i < len(value); i++ {
		b := value[i]
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') {
			continue
		}
		switch b {
		case '.', '_', '~', '-':
			continue
		default:
			return false
		}
	}
	return true
}

func newRequestID() string {
	return rand.Text()
}

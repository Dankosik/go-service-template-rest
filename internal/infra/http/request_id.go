package httpx

import (
	"context"
	"crypto/rand"
	"strings"
)

const maxRequestIDLength = 128

type requestIDContextKey struct{}

// contextWithRequestID validates the candidate, generates a replacement when needed,
// and returns both the enriched context and the accepted identifier.
func contextWithRequestID(ctx context.Context, candidate string) (context.Context, string) {
	requestID := strings.TrimSpace(candidate)
	if !validRequestID(requestID) {
		requestID = newRequestID()
	}
	return context.WithValue(ctx, requestIDContextKey{}, requestID), requestID
}

// requestIDFromContext returns the validated request identifier stored in ctx.
func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
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

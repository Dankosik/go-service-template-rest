package reqctx

import (
	"context"
	"crypto/rand"
	"strings"
)

// MaxRequestIDLength is the shared wire limit for a correlation identifier.
const MaxRequestIDLength = 128

// RequestIDHeader and RequestIDMetadataKey are one wire name in the two
// spellings the transports require: net/http canonicalizes header keys, gRPC
// requires lowercase metadata keys. This package owns both because every
// transport adapter must agree on them and none may import another;
// TestRequestIDWireNamesAreOneName proves the two equal.
const (
	RequestIDHeader      = "X-Request-ID"
	RequestIDMetadataKey = "x-request-id"
)

type requestIDContextKey struct{}

// ContextWithRequestID returns ctx carrying the correlation identifier.
// Generating and validating the value belongs to the transport adapter.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// RequestID returns the correlation identifier, or the empty string when the
// request did not travel through the correlation middleware. Trace and span
// identifiers are reachable through go.opentelemetry.io/otel/trace; this is the
// one correlation value that is not.
func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func ContextWithAcceptedRequestID(ctx context.Context, candidate string) (context.Context, string) {
	requestID := strings.TrimSpace(candidate)
	if !ValidRequestID(requestID) {
		requestID = rand.Text()
	}
	return ContextWithRequestID(ctx, requestID), requestID
}

// ValidRequestID reports whether value is safe to carry in logs and response
// metadata. The alphabet is transport-neutral and unchanged from the original
// HTTP contract.
func ValidRequestID(value string) bool {
	if value == "" || len(value) > MaxRequestIDLength {
		return false
	}
	for index := range len(value) {
		character := value[index]
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') {
			continue
		}
		switch character {
		case '.', '_', '~', '-':
			continue
		default:
			return false
		}
	}
	return true
}

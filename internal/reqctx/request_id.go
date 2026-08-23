package reqctx

import (
	"context"
	"crypto/rand"
	"regexp"
	"strings"
)

const maxRequestIDLength = 128

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)

// RequestIDHeader is shared by the inbound HTTP adapter and the fixed outbound
// HTTP sanitizer, which must agree on the field they accept or remove.
const RequestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

func contextWithRequestID(ctx context.Context, requestID string) context.Context {
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
	if !validRequestID(requestID) {
		requestID = rand.Text()
	}
	return contextWithRequestID(ctx, requestID), requestID
}

func validRequestID(value string) bool {
	return len(value) <= maxRequestIDLength && requestIDPattern.MatchString(value)
}

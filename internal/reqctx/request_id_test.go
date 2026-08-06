package reqctx_test

import (
	"context"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/reqctx"
)

// TestRequestIDWireNamesAreOneName holds the invariant that lets four transport
// adapters derive their spelling from this package instead of restating it: the
// HTTP and gRPC forms must stay the same name, each in the casing its protocol
// requires.
func TestRequestIDWireNamesAreOneName(t *testing.T) {
	t.Parallel()

	if !strings.EqualFold(reqctx.RequestIDHeader, reqctx.RequestIDMetadataKey) {
		t.Fatalf(
			"RequestIDHeader = %q and RequestIDMetadataKey = %q are different names, want the same name in two casings",
			reqctx.RequestIDHeader,
			reqctx.RequestIDMetadataKey,
		)
	}
	// gRPC lowercases metadata keys on the wire and matches them lowercased, so
	// a key in any other casing silently fails to match. HTTP needs no
	// equivalent check: net/http canonicalizes on both Get and Set, which is why
	// RequestIDHeader may keep the "X-Request-ID" spelling the OpenAPI contract
	// documents rather than Go's "X-Request-Id" canonical form.
	if got := strings.ToLower(reqctx.RequestIDMetadataKey); got != reqctx.RequestIDMetadataKey {
		t.Fatalf("RequestIDMetadataKey = %q, want lowercase %q for gRPC metadata", reqctx.RequestIDMetadataKey, got)
	}
}

func TestContextWithAcceptedRequestID(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name      string
		candidate string
		want      string
	}{
		{name: "accepts safe value", candidate: "request_123.~", want: "request_123.~"},
		{name: "trims safe value", candidate: " request-123 ", want: "request-123"},
		{name: "replaces empty"},
		{name: "replaces unsafe", candidate: "user@example.com"},
		{name: "replaces overlong", candidate: strings.Repeat("a", reqctx.MaxRequestIDLength+1)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx, got := reqctx.ContextWithAcceptedRequestID(context.Background(), testCase.candidate)
			if testCase.want != "" && got != testCase.want {
				t.Fatalf("accepted request ID = %q, want %q", got, testCase.want)
			}
			if !reqctx.ValidRequestID(got) {
				t.Fatalf("accepted request ID %q is invalid", got)
			}
			if fromContext := reqctx.RequestID(ctx); fromContext != got {
				t.Fatalf("RequestID() = %q, want %q", fromContext, got)
			}
		})
	}
}

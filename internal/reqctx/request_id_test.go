package reqctx_test

import (
	"context"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/reqctx"
)

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

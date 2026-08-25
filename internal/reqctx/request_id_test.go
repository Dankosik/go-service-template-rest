package reqctx_test

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/reqctx"
)

func TestContextWithAcceptedRequestID(t *testing.T) {
	t.Parallel()
	safeRequestID := regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)

	for _, testCase := range []struct {
		name      string
		candidate string
		want      string
	}{
		{name: "accepts safe value", candidate: "request_123.~", want: "request_123.~"},
		{name: "trims safe value", candidate: " request-123 ", want: "request-123"},
		{name: "replaces empty"},
		{name: "replaces unsafe", candidate: "user@example.com"},
		{name: "replaces overlong", candidate: strings.Repeat("a", 129)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx, got := reqctx.ContextWithAcceptedRequestID(context.Background(), testCase.candidate)
			if testCase.want != "" && got != testCase.want {
				t.Fatalf("accepted request ID = %q, want %q", got, testCase.want)
			}
			if len(got) > 128 || !safeRequestID.MatchString(got) {
				t.Fatalf("accepted request ID %q is invalid", got)
			}
			if fromContext := reqctx.RequestID(ctx); fromContext != got {
				t.Fatalf("RequestID() = %q, want %q", fromContext, got)
			}
		})
	}
}

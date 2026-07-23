package httpx

import (
	"context"
	"strings"
	"testing"
)

func TestContextWithRequestID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		candidate string
		want      string
	}{
		{name: "preserves valid", candidate: "caller.ID_1~ok", want: "caller.ID_1~ok"},
		{name: "trims valid", candidate: "  caller-id  ", want: "caller-id"},
		{name: "generates when empty"},
		{name: "generates when overlong", candidate: strings.Repeat("a", maxRequestIDLength+1)},
		{name: "generates for non ASCII", candidate: "запрос"},
		{name: "generates for invalid punctuation", candidate: "caller/id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, got := contextWithRequestID(context.Background(), tt.candidate)
			if tt.want != "" && got != tt.want {
				t.Fatalf("ContextWithRequestID() = %q, want %q", got, tt.want)
			}
			if tt.want == "" && (got == "" || got == strings.TrimSpace(tt.candidate)) {
				t.Fatalf("ContextWithRequestID() generated %q for invalid candidate %q", got, tt.candidate)
			}
			if !validRequestID(got) {
				t.Fatalf("ContextWithRequestID() returned invalid request ID %q", got)
			}
			if fromContext := requestIDFromContext(ctx); fromContext != got {
				t.Fatalf("RequestIDFromContext() = %q, want %q", fromContext, got)
			}
		})
	}
}

func TestRequestIDFromContextWithoutValue(t *testing.T) {
	t.Parallel()

	if got := requestIDFromContext(context.Background()); got != "" {
		t.Fatalf("RequestIDFromContext() = %q, want empty", got)
	}
}

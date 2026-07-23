package bootstrap

import (
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func assertSpanStringAttribute(t *testing.T, span sdktrace.ReadOnlySpan, key, want string) {
	t.Helper()

	for _, attr := range span.Attributes() {
		if string(attr.Key) != key {
			continue
		}
		if got := attr.Value.AsString(); got != want {
			t.Fatalf("span %q attribute %q = %q, want %q", span.Name(), key, got, want)
		}
		return
	}

	t.Fatalf("span %q missing attribute %q", span.Name(), key)
}

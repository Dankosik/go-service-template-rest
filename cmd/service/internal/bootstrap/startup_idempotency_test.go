package bootstrap

import (
	"log/slog"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
)

func TestHTTPIdempotencyDisabledIsInert(t *testing.T) {
	t.Parallel()

	runtime, err := initHTTPIdempotencyRuntime(t.Context(), config.Config{}, nil, slog.Default(), false)
	if err != nil {
		t.Fatalf("initHTTPIdempotencyRuntime(disabled): %v", err)
	}
	if runtime.store != nil || runtime.cleanup != nil {
		t.Fatalf("disabled runtime = %#v, want inert", runtime)
	}
	if err := runtime.Run(t.Context()); err != nil {
		t.Fatalf("disabled Run(): %v", err)
	}
}

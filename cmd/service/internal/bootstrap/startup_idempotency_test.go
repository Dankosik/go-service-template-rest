package bootstrap

import (
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
)

func TestHTTPIdempotencyDisabledIsInert(t *testing.T) {
	t.Parallel()

	store, err := initHTTPIdempotencyRuntime(t.Context(), config.Config{}, nil, false)
	if err != nil {
		t.Fatalf("initHTTPIdempotencyRuntime(disabled): %v", err)
	}
	if store != nil {
		t.Fatalf("disabled store = %#v, want nil", store)
	}
}

package bootstrap

import (
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
)

func TestMessagingCompositionDisabledHasNoRuntimeOrReadiness(t *testing.T) {
	runtime, err := initMessagingRuntime(t.Context(), config.MessagingConfig{}, nil)
	if err != nil {
		t.Fatalf("initMessagingRuntime(disabled) error = %v", err)
	}
	if len(runtime.ReadinessProbes()) != 0 {
		t.Fatalf("disabled messaging probes = %d", len(runtime.ReadinessProbes()))
	}
	runtime.StartDrain()
	if err := runtime.Shutdown(t.Context()); err != nil {
		t.Fatalf("disabled messaging Shutdown() error = %v", err)
	}
}

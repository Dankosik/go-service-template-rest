package bootstrap

import (
	"testing"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
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

func TestMessagingCompositionSplitsCanonicalURLs(t *testing.T) {
	got := runtimeopts.Messaging(config.MessagingConfig{URLs: "tls://one.example:4222,tls://two.example:4222"}).URLs
	if len(got) != 2 || got[0] != "tls://one.example:4222" || got[1] != "tls://two.example:4222" {
		t.Fatalf("runtimeopts.Messaging().URLs = %v", got)
	}
}

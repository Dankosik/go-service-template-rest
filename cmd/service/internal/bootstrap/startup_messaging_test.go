package bootstrap

import (
	"testing"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
)

func TestMessagingCompositionDisabledHasNoRuntimeOrReadiness(t *testing.T) {
	runtime, err := initMessagingRuntime(t.Context(), config.MessagingConfig{}, nil)
	if err != nil {
		t.Fatalf("initMessagingRuntime(disabled) error = %v", err)
	}
	if publisher, publisherErr := runtime.Publisher(natsjs.Route{Type: "test", Version: 1, Subject: "events.test"}); publisher != nil || publisherErr == nil {
		t.Fatalf("disabled messaging runtime = publisher %v, error %v", publisher, publisherErr)
	}
	if len(runtime.ReadinessProbes()) != 0 {
		t.Fatalf("disabled messaging probes = %d", len(runtime.ReadinessProbes()))
	}
	if !runtime.Ready() {
		t.Fatal("disabled messaging runtime is not ready")
	}
	runtime.StartDrain()
	if err := runtime.Shutdown(t.Context()); err != nil {
		t.Fatalf("disabled messaging Shutdown() error = %v", err)
	}
}

func TestMessagingCompositionReadinessUsesImmediateClientState(t *testing.T) {
	runtime := messagingRuntime{client: new(natsjs.Client)}
	if runtime.Ready() {
		t.Fatal("disconnected messaging runtime reported ready")
	}
}

func TestMessagingCompositionSplitsCanonicalURLs(t *testing.T) {
	got := runtimeopts.Messaging(config.MessagingConfig{URLs: "tls://one.example:4222,tls://two.example:4222"}).URLs
	if len(got) != 2 || got[0] != "tls://one.example:4222" || got[1] != "tls://two.example:4222" {
		t.Fatalf("runtimeopts.Messaging().URLs = %v", got)
	}
}

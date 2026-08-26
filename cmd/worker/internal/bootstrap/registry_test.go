package bootstrap

import (
	"context"
	"testing"

	"github.com/example/go-service-template-rest/internal/domainevent"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
)

func testRegistry(t *testing.T, eventType string, handler func(context.Context, string) error) *natsjs.Registry {
	t.Helper()
	kind := domainevent.Define[string](eventType, 1)
	registry, err := natsjs.NewRegistry(natsjs.Route{Type: eventType, Version: 1, Subject: "events.test"})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	if err := registry.Handle(kind, func(ctx context.Context, event domainevent.Typed[string]) error {
		return handler(ctx, event.Payload)
	}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	return registry
}

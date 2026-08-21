package natsjs

import (
	"context"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/domainevent"
	"github.com/nats-io/nats.go/jetstream"
)

type typedPayload struct {
	Value string `json:"value"`
}

func TestTypedPublisherAndHandlerHideBrokerFields(t *testing.T) {
	kind := domainevent.Define[typedPayload]("example.updated", 1)
	registry, err := NewRegistry(Route{Type: kind.Type, Version: kind.Version, Subject: "events.example"})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	seen := make(chan domainevent.Typed[typedPayload], 1)
	if err := domainevent.Handle(registry, kind, func(_ context.Context, event domainevent.Typed[typedPayload]) error {
		seen <- event
		return nil
	}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	handler, err := registry.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}
	createdAt := time.Unix(10, 0).UTC()
	if err := handler(t.Context(), Message{
		messageID: "event-1", eventType: kind.Type, schema: "v1", createdAt: createdAt,
		payload: []byte(`{"value":"handled"}`),
	}); err != nil {
		t.Fatalf("typed handler error = %v", err)
	}
	if got := <-seen; got.ID != "event-1" || got.Payload.Value != "handled" || !got.OccurredAt.Equal(createdAt) {
		t.Fatalf("typed event = %#v", got)
	}

	broker := &recordingJetStream{ack: &jetstream.PubAck{Stream: "EVENTS", Sequence: 1}}
	publisher, err := registry.Publisher(unitClient(t, broker, RoleProducer).Producer())
	if err != nil {
		t.Fatalf("Publisher() error = %v", err)
	}
	event, err := kind.New("event-2", createdAt, typedPayload{Value: "published"})
	if err != nil {
		t.Fatalf("Kind.New() error = %v", err)
	}
	if err := publisher.Publish(t.Context(), event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if broker.published.Subject != "events.example" || broker.published.Header.Get(headerMessageID) != "event-2" {
		t.Fatalf("wire publication = %#v", broker.published)
	}
}

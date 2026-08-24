package natsjs

import (
	"context"
	"errors"
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
	if err := Handle(registry, kind, func(_ context.Context, event domainevent.Typed[typedPayload]) error {
		seen <- event
		return nil
	}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	handler, err := registry.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}
	createdAt := time.Unix(10, 0).In(time.FixedZone("UTC-like", 0))
	if err := handler(t.Context(), Message{
		subject: "events.example", messageID: "event-1", eventType: kind.Type, schema: "v1", createdAt: createdAt,
		payload: []byte(`{"value":"handled"}`),
	}); err != nil {
		t.Fatalf("typed handler error = %v", err)
	}
	if got := <-seen; got.ID != "event-1" || got.Payload.Value != "handled" || !got.OccurredAt.Equal(createdAt) {
		t.Fatalf("typed event = %#v", got)
	}
	err = handler(t.Context(), Message{
		subject: "events.other", messageID: "event-2", eventType: kind.Type, schema: "v1", createdAt: createdAt,
		payload: []byte(`{"value":"wrong route"}`),
	})
	if !IsPermanent(err) {
		t.Fatalf("wrong-subject error = %v, want permanent", err)
	}
	select {
	case event := <-seen:
		t.Fatalf("wrong-subject event reached handler: %#v", event)
	default:
	}

	broker := &recordingJetStream{ack: &jetstream.PubAck{Stream: "EVENTS", Sequence: 1}}
	publisher, err := registry.Publisher(unitClient(t, broker).Producer())
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

func TestTypedHandlerContract(t *testing.T) {
	t.Parallel()

	kind := domainevent.Define[typedPayload]("example.updated", 1)
	noop := func(context.Context, domainevent.Typed[typedPayload]) error { return nil }
	if err := Handle(nil, kind, noop); !errors.Is(err, ErrRejected) {
		t.Fatalf("Handle(nil registry) error = %v, want ErrRejected", err)
	}
	registry, err := NewRegistry(Route{Type: kind.Type, Version: kind.Version, Subject: "events.example"})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	if err := Handle[typedPayload](registry, kind, nil); !errors.Is(err, ErrRejected) {
		t.Fatalf("Handle(nil handler) error = %v, want ErrRejected", err)
	}
	called := false
	if err := Handle(registry, kind, func(context.Context, domainevent.Typed[typedPayload]) error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if err := Handle(registry, kind, noop); !errors.Is(err, ErrRejected) {
		t.Fatalf("Handle(duplicate) error = %v, want ErrRejected", err)
	}
	handler, err := registry.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}
	err = handler(t.Context(), Message{
		messageID: "event-1", eventType: kind.Type, schema: "v1",
		createdAt: time.Unix(1, 0).UTC(), payload: []byte("{"),
	})
	if !IsPermanent(err) || called {
		t.Fatalf("invalid payload error = %v, handler called = %t", err, called)
	}
}

func TestSchemaVersionRequiresCanonicalSpelling(t *testing.T) {
	t.Parallel()

	if got, err := schemaVersion("v1"); err != nil || got != 1 {
		t.Fatalf("schemaVersion(v1) = %d, %v", got, err)
	}
	for _, schema := range []string{"1", "v0", "v01", "v+1", "v65536"} {
		if _, err := schemaVersion(schema); err == nil {
			t.Errorf("schemaVersion(%q) error = nil", schema)
		}
	}
}

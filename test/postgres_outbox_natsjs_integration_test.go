//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
)

type natsOutboxPublisher struct {
	producer *natsjs.Producer
}

func (publisher natsOutboxPublisher) Publish(ctx context.Context, event postgresoutbox.Event) error {
	_, err := publisher.producer.Publish(ctx, natsjs.Event{
		Subject:       event.Destination,
		MessageID:     event.ID,
		PublicationID: event.ID,
		Type:          event.Type,
		Schema:        event.Schema,
		OrderingKey:   event.OrderingKey,
		CreatedAt:     event.OccurredAt,
		Payload:       event.Payload,
	})
	if errors.Is(err, natsjs.ErrRejected) {
		return fmt.Errorf("%w: %v", postgresoutbox.ErrPermanentPublication, err)
	}
	return err
}

func TestPostgresOutboxNATSConformance(t *testing.T) {
	fixture := newNATSFixture(t)
	publisher := natsOutboxPublisher{producer: fixture.client(t, natsjs.RoleProducer).Producer()}
	ctx, pool, store := newOutboxFixture(t)

	received := make(chan natsjs.Message, 1)
	_, _, _ = fixture.worker(t, func(_ context.Context, message natsjs.Message) error {
		received <- message
		return nil
	}, func(config *natsjs.WorkerConfig) {
		config.Consumer = "outbox-conformance"
	})

	event := outboxEvent("outbox-nats-conformance")
	event.Destination = sourceSubject
	event.Payload = []byte(`{"broker":"durable"}`)
	event.OrderingKey = "account-42"
	event.OrderingSequence = 7
	mustAppendOutbox(t, ctx, pool, store, event)
	relay := mustNewOutboxRelay(t, store, publisher, nil, testRelayConfig())
	result := runOutboxRelay(ctx, relay)
	waitForOutboxCount(t, ctx, pool, "published_at IS NOT NULL", 1)
	message := receive(t, received, 10*time.Second, "durable outbox JetStream message")
	relay.StartDrain()
	assertRelayResult(t, result, true, nil)
	if message.MessageID() != event.ID || message.PublicationID() != event.ID ||
		message.Subject() != event.Destination || message.Type() != event.Type ||
		message.Schema() != event.Schema || message.OrderingKey() != event.OrderingKey ||
		string(message.Payload()) != string(event.Payload) {
		t.Fatalf("JetStream message = id %q publication %q subject %q type %q schema %q key %q payload %q, want outbox event %+v",
			message.MessageID(), message.PublicationID(), message.Subject(), message.Type(), message.Schema(),
			message.OrderingKey(), message.Payload(), event)
	}

	rejected := outboxEvent("outbox-nats-rejected")
	rejected.Destination = "invalid subject"
	mustAppendOutbox(t, ctx, pool, store, rejected)
	relay = mustNewOutboxRelay(t, store, publisher, nil, testRelayConfig())
	result = runOutboxRelay(ctx, relay)
	waitForOutboxCount(t, ctx, pool, "poisoned_at IS NOT NULL", 1)
	relay.StartDrain()
	assertRelayResult(t, result, true, nil)
	record, err := store.Get(ctx, rejected.ID)
	if err != nil {
		t.Fatalf("Get(rejected): %v", err)
	}
	if !record.PublishedAt.IsZero() || record.PoisonedAt.IsZero() || record.LastErrorClass != "publisher_permanent" {
		t.Fatalf("rejected outbox record published=%v poisoned=%v class=%q, want unfinished poison",
			record.PublishedAt, record.PoisonedAt, record.LastErrorClass)
	}
}

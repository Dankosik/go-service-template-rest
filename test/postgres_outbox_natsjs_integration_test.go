//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func TestPostgresOutboxNATSConformance(t *testing.T) {
	t.Parallel()
	telemetrytest.InstallSpanRecorder(t)
	fixture := newNATSFixture(t)
	publisher := natsjs.NewOutboxPublisher(fixture.client(t, natsjs.RoleProducer).Producer())
	ctx, pool, store := newOutboxFixture(t)

	type delivery struct {
		message natsjs.Message
		traceID trace.TraceID
	}
	received := make(chan delivery, 1)
	_, _, _ = fixture.worker(t, func(ctx context.Context, message natsjs.Message) error {
		received <- delivery{message: message, traceID: trace.SpanContextFromContext(ctx).TraceID()}
		return nil
	}, func(config *natsjs.WorkerConfig) {
		config.Consumer = "outbox-conformance"
	})

	event := outboxEvent("outbox-nats-conformance")
	event.Destination = sourceSubject
	event.Payload = []byte(`{"broker":"durable"}`)
	event.OrderingKey = "account-42"
	event.OrderingSequence = 7
	producing, span := otel.GetTracerProvider().Tracer("integration").Start(ctx, "create outbox event")
	origin := trace.SpanContextFromContext(producing)
	if err := pool.InTx(producing, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return store.Append(producing, tx, event)
	}); err != nil {
		t.Fatalf("append traced outbox event: %v", err)
	}
	span.End()
	relay := mustNewOutboxRelay(t, store, publisher, nil, testRelayConfig())
	result := runOutboxRelay(ctx, relay)
	waitForOutboxCount(t, ctx, pool, "published_at IS NOT NULL", 1)
	delivered := waittest.Receive(t, received, 10*time.Second, "durable outbox JetStream message")
	message := delivered.message
	relay.StartDrain()
	assertRelayResult(t, result, nil)
	if message.MessageID() != event.ID || message.PublicationID() != event.ID ||
		message.Subject() != event.Destination || message.Type() != event.Type ||
		message.Schema() != event.Schema || message.OrderingKey() != event.OrderingKey ||
		string(message.Payload()) != string(event.Payload) {
		t.Fatalf("JetStream message = id %q publication %q subject %q type %q schema %q key %q payload %q, want outbox event %+v",
			message.MessageID(), message.PublicationID(), message.Subject(), message.Type(), message.Schema(),
			message.OrderingKey(), message.Payload(), event)
	}
	if delivered.traceID != origin.TraceID() {
		t.Fatalf("worker trace = %s, want producing trace %s", delivered.traceID, origin.TraceID())
	}

	rejected := outboxEvent("outbox-nats-rejected")
	rejected.Destination = "invalid subject"
	mustAppendOutbox(t, ctx, pool, store, rejected)
	relay = mustNewOutboxRelay(t, store, publisher, nil, testRelayConfig())
	result = runOutboxRelay(ctx, relay)
	waitForOutboxCount(t, ctx, pool, "poisoned_at IS NOT NULL", 1)
	relay.StartDrain()
	assertRelayResult(t, result, nil)
	record, err := store.Get(ctx, rejected.ID)
	if err != nil {
		t.Fatalf("Get(rejected): %v", err)
	}
	if !record.PublishedAt.IsZero() || record.PoisonedAt.IsZero() || record.LastErrorClass != "publisher_permanent" {
		t.Fatalf("rejected outbox record published=%v poisoned=%v class=%q, want unfinished poison",
			record.PublishedAt, record.PoisonedAt, record.LastErrorClass)
	}
}

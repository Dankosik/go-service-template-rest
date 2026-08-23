package natsjs

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

func TestOutboxWorkerPublishesStableWireIdentityAndTrace(t *testing.T) {
	t.Parallel()

	broker := &recordingJetStream{ack: &jetstream.PubAck{Stream: "EVENTS", Sequence: 1}}
	worker, err := NewOutboxWorker(unitClient(t, broker).Producer())
	if err != nil {
		t.Fatalf("NewOutboxWorker() error = %v", err)
	}
	const traceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	metadata, err := json.Marshal(map[string]string{"traceparent": traceparent})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	args := postgresoutbox.PublishJob{
		ID: "event-1", Type: "order.updated", Version: 1,
		OccurredAt: time.Unix(1, 0).UTC(), Payload: json.RawMessage(`{"order_id":"order-1"}`),
		Subject: "events.orders",
	}
	if err := worker.Work(t.Context(), &river.Job[postgresoutbox.PublishJob]{
		JobRow: &rivertype.JobRow{Metadata: metadata},
		Args:   args,
	}); err != nil {
		t.Fatalf("Work() error = %v", err)
	}

	if broker.published.Subject != args.Subject ||
		broker.published.Header.Get(headerMessageID) != args.ID ||
		broker.published.Header.Get(jetstream.MsgIDHeader) != args.ID ||
		broker.published.Header.Get(headerEventType) != args.Type ||
		broker.published.Header.Get(headerEventSchema) != "v1" ||
		broker.published.Header.Get("traceparent") != traceparent ||
		!bytes.Equal(broker.published.Data, args.Payload) {
		t.Fatalf("published message = %#v", broker.published)
	}
}

func TestNewOutboxAppenderRejectsInvalidRoutes(t *testing.T) {
	t.Parallel()

	for name, routes := range map[string][]Route{
		"empty":    nil,
		"wildcard": {{Type: "order.updated", Version: 1, Subject: "events.*"}},
		"version":  {{Type: "order.updated", Subject: "events.orders"}},
		"duplicate": {
			{Type: "order.updated", Version: 1, Subject: "events.orders"},
			{Type: "order.updated", Version: 1, Subject: "events.orders"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := NewOutboxAppender(testMaxPayloadBytes, routes...); err == nil {
				t.Fatal("NewOutboxAppender() error = nil")
			}
		})
	}
}

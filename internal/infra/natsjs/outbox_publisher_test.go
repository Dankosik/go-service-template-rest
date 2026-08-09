package natsjs

import (
	"context"
	"errors"
	"maps"
	"sync"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestOutboxPublisher(t *testing.T) {
	if NewOutboxPublisher(nil) != nil {
		t.Fatal("NewOutboxPublisher(nil) must return a nil interface")
	}

	t.Run("acknowledgement and envelope", func(t *testing.T) {
		broker := &recordingJetStream{ack: &jetstream.PubAck{Stream: "EVENTS", Sequence: 7}}
		publisher := NewOutboxPublisher(unitClient(t, broker, RoleProducer).Producer())
		ctx := trace.ContextWithSpanContext(t.Context(), trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2}, TraceFlags: trace.FlagsSampled,
		}))
		event := validOutboxEvent()
		if err := publisher.Publish(ctx, event); err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
		if broker.published.Subject != event.Destination || string(broker.published.Data) != string(event.Payload) ||
			broker.published.Header.Get(headerMessageID) != event.ID ||
			broker.published.Header.Get(headerPublicationID) != event.ID ||
			broker.published.Header.Get(headerEventType) != event.Type ||
			broker.published.Header.Get(headerEventSchema) != event.Schema ||
			broker.published.Header.Get(headerOrderingKey) != event.OrderingKey {
			t.Fatalf("published message = %#v, want mapped outbox event", broker.published)
		}
		if got := broker.published.Header.Get("traceparent"); got != "" {
			t.Fatalf("traceparent = %q, want relay-attempt context removed", got)
		}
	})

	t.Run("classification", func(t *testing.T) {
		tests := []struct {
			name      string
			mutate    func(*postgresoutbox.Event)
			brokerErr error
			cancel    bool
			want      error
		}{
			{name: "immutable envelope", mutate: func(event *postgresoutbox.Event) { event.Destination = "invalid subject" }, want: postgresoutbox.ErrPermanentPublication},
			{name: "definite refusal", brokerErr: nats.ErrNoResponders, want: postgresoutbox.ErrPublicationNotAccepted},
			{name: "ambiguous acknowledgement", brokerErr: errors.New("connection vanished"), want: ErrAmbiguous},
			{name: "pre-dispatch cancellation", cancel: true, want: postgresoutbox.ErrPublicationNotAccepted},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				broker := &recordingJetStream{ack: &jetstream.PubAck{Stream: "EVENTS"}, err: test.brokerErr}
				publisher := NewOutboxPublisher(unitClient(t, broker, RoleProducer).Producer())
				event := validOutboxEvent()
				if test.mutate != nil {
					test.mutate(&event)
				}
				ctx := t.Context()
				if test.cancel {
					var cancel context.CancelFunc
					ctx, cancel = context.WithCancel(ctx)
					cancel()
				}
				err := publisher.Publish(ctx, event)
				if !errors.Is(err, test.want) {
					t.Fatalf("Publish() error = %v, want %v", err, test.want)
				}
				if errors.Is(test.want, ErrAmbiguous) && (errors.Is(err, postgresoutbox.ErrPermanentPublication) || errors.Is(err, postgresoutbox.ErrPublicationNotAccepted)) {
					t.Fatalf("ambiguous error became definitive: %v", err)
				}
			})
		}
	})

	t.Run("concurrent calls retain identity and W3C carrier", func(t *testing.T) {
		broker := newBarrierJetStream("event-one", "event-two")
		client := unitClient(t, broker, RoleProducer)
		client.producer = newProducer(client, 2, client.cfg.MaxPayloadBytes)
		publisher := outboxPublisher{producer: client.Producer()}
		type concurrentEvent struct {
			event   Event
			carrier propagation.MapCarrier
		}
		events := []concurrentEvent{
			{
				event:   validMappedOutboxEvent("event-one"),
				carrier: propagation.MapCarrier{"traceparent": "00-11111111111111111111111111111111-1111111111111111-01"},
			},
			{
				event:   validMappedOutboxEvent("event-two"),
				carrier: propagation.MapCarrier{"traceparent": "00-22222222222222222222222222222222-2222222222222222-01"},
			},
		}
		errs := make(chan error, 2)
		for _, event := range events {
			go func() { errs <- publisher.publish(t.Context(), event.event, event.carrier) }()
		}
		arrived := broker.wait(t)
		close(broker.release[arrived[1]])
		close(broker.release[arrived[0]])
		for range events {
			if err := <-errs; err != nil {
				t.Fatalf("concurrent Publish() error = %v", err)
			}
		}
		if got := broker.publications(); !maps.Equal(got, map[string]barrierPublication{
			"event-one": {publicationID: "event-one", traceID: trace.TraceID{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11}},
			"event-two": {publicationID: "event-two", traceID: trace.TraceID{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22}},
		}) {
			t.Fatalf("published identity/carriers = %#v", got)
		}
	})
}

func validMappedOutboxEvent(id string) Event {
	return Event{
		Subject: "events.created", MessageID: id, PublicationID: id, Type: "created", Schema: "v1",
		OrderingKey: "account-1", CreatedAt: time.Unix(123, 456).UTC(), Payload: []byte(`{"id":1}`),
	}
}

func validOutboxEvent() postgresoutbox.Event {
	return postgresoutbox.Event{
		ID: "event-1", Type: "created", Source: "orders", Destination: "events.created", Schema: "v1",
		OccurredAt: time.Unix(123, 456).UTC(), Payload: []byte(`{"id":1}`), Metadata: []byte(`{"request":"caller-owned"}`),
		OrderingKey: "account-1", OrderingSequence: 1,
	}
}

type barrierJetStream struct {
	jetstream.JetStream

	arrived chan string
	release map[string]chan struct{}

	mu        sync.Mutex
	published map[string]barrierPublication
}

type barrierPublication struct {
	publicationID string
	traceID       trace.TraceID
}

func newBarrierJetStream(ids ...string) *barrierJetStream {
	broker := &barrierJetStream{
		arrived:   make(chan string, len(ids)),
		release:   make(map[string]chan struct{}, len(ids)),
		published: make(map[string]barrierPublication, len(ids)),
	}
	for _, id := range ids {
		broker.release[id] = make(chan struct{})
	}
	return broker
}

func (broker *barrierJetStream) PublishMsg(ctx context.Context, msg *nats.Msg, _ ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	messageID := msg.Header.Get(headerMessageID)
	broker.arrived <- messageID
	<-broker.release[messageID]
	spanContext := trace.SpanContextFromContext(propagation.TraceContext{}.Extract(
		ctx, propagation.MapCarrier{"traceparent": msg.Header.Get("traceparent")},
	))
	broker.mu.Lock()
	broker.published[messageID] = barrierPublication{
		publicationID: msg.Header.Get(headerPublicationID),
		traceID:       spanContext.TraceID(),
	}
	broker.mu.Unlock()
	return &jetstream.PubAck{Stream: "EVENTS"}, nil
}

func (broker *barrierJetStream) wait(t *testing.T) []string {
	t.Helper()
	arrived := make([]string, 0, cap(broker.arrived))
	for range cap(broker.arrived) {
		select {
		case id := <-broker.arrived:
			arrived = append(arrived, id)
		case <-time.After(time.Second):
			t.Fatal("concurrent publish did not reach broker barrier")
		}
	}
	return arrived
}

func (broker *barrierJetStream) publications() map[string]barrierPublication {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	return maps.Clone(broker.published)
}

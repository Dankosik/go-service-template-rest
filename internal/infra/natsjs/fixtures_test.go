package natsjs

import (
	"context"
	"log/slog"
	"maps"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// The in-process doubles every unit test in this package builds on. They stand
// in for the broker (recordingJetStream), one delivery (fakeMsg), and the two
// runtime objects a test needs without a live connection.

type recordingJetStream struct {
	jetstream.JetStream

	ack       *jetstream.PubAck
	err       error
	published *nats.Msg
}

func (s *recordingJetStream) PublishMsg(_ context.Context, msg *nats.Msg, _ ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	s.published = &nats.Msg{Subject: msg.Subject, Header: maps.Clone(msg.Header), Data: append([]byte(nil), msg.Data...)}
	return s.ack, s.err
}

func unitClient(t *testing.T, broker jetstream.JetStream, role Role) *Client {
	t.Helper()
	sig, err := newTelemetry(Observability{Logger: slog.New(slog.DiscardHandler)}, role, func() bool { return false })
	if err != nil {
		t.Fatalf("newTelemetry() error = %v", err)
	}
	t.Cleanup(sig.close)
	cfg := testConfig()
	cfg.Stream = "EVENTS"
	client := &Client{
		cfg:         cfg,
		js:          broker,
		telemetry:   sig,
		events:      make(chan connectionEvent, 1),
		reconnected: make(chan struct{}, 1),
		terminal:    make(chan error, 1),
		closed:      make(chan struct{}),
	}
	client.producer = newProducer(client, 1, cfg.MaxPayloadBytes)
	return client
}

func unitWorker(t *testing.T, broker jetstream.JetStream, handler Handler) *Worker {
	t.Helper()
	client := unitClient(t, broker, RoleWorker)
	cfg := testWorkerConfig()
	cfg.Consumer = "events-worker"
	cfg.FilterSubject = "events.>"
	cfg.DeadLetterSubject = "dead.events"
	return &Worker{
		client:    client,
		cfg:       cfg,
		dlqStream: "EVENTS_DLQ",
		handler:   handler,
		fatal:     make(chan error, 1),
		runDone:   make(chan struct{}),
	}
}

func unitSource(t *testing.T, delivered uint64) *fakeMsg {
	t.Helper()
	msg, err := buildNATSMessage(t.Context(), validTestEvent(), testMaxPayloadBytes)
	if err != nil {
		t.Fatalf("buildNATSMessage() error = %v", err)
	}
	return &fakeMsg{
		subject: msg.Subject,
		header:  msg.Header,
		data:    msg.Data,
		metadata: &jetstream.MsgMetadata{
			Sequence:     jetstream.SequencePair{Stream: 3, Consumer: 2},
			NumDelivered: delivered,
			Timestamp:    time.Unix(100, 0).UTC(),
			Stream:       "EVENTS",
			Consumer:     "events-worker",
		},
	}
}

func validTestEvent() Event {
	return Event{
		Subject:       "events.created",
		MessageID:     "message-1",
		PublicationID: "publication-1",
		Type:          "created",
		Schema:        "v1",
		OrderingKey:   "account-1",
		CreatedAt:     time.Unix(123, 456).UTC(),
		Payload:       []byte("payload"),
	}
}

func eventHeaders(event Event) nats.Header {
	header := make(nats.Header)
	header.Set(headerMessageID, event.MessageID)
	header.Set(headerPublicationID, event.PublicationID)
	header.Set(headerEventType, event.Type)
	header.Set(headerEventSchema, event.Schema)
	header.Set(headerOrderingKey, event.OrderingKey)
	header.Set(headerCreatedAt, event.CreatedAt.Format(time.RFC3339Nano))
	return header
}

type fakeMsg struct {
	subject     string
	header      nats.Header
	data        []byte
	metadata    *jetstream.MsgMetadata
	metadataErr error
	ackErr      error
	nakErr      error
	ackCount    int
	nakDelay    time.Duration
}

func (m *fakeMsg) Metadata() (*jetstream.MsgMetadata, error) { return m.metadata, m.metadataErr }
func (m *fakeMsg) Data() []byte                              { return m.data }
func (m *fakeMsg) Headers() nats.Header                      { return m.header }
func (m *fakeMsg) Subject() string                           { return m.subject }
func (m *fakeMsg) Reply() string                             { return "" }
func (m *fakeMsg) Ack() error                                { return m.ackErr }
func (m *fakeMsg) DoubleAck(context.Context) error {
	m.ackCount++
	return m.ackErr
}
func (m *fakeMsg) Nak() error { return m.nakErr }
func (m *fakeMsg) NakWithDelay(delay time.Duration) error {
	m.nakDelay = delay
	return m.nakErr
}
func (m *fakeMsg) InProgress() error           { return nil }
func (m *fakeMsg) Term() error                 { return nil }
func (m *fakeMsg) TermWithReason(string) error { return nil }

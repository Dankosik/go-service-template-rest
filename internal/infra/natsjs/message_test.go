package natsjs

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestEventValidation(t *testing.T) {
	event := validTestEvent()
	if err := validateEvent(event, len(event.Payload)); err != nil {
		t.Fatalf("validateEvent(valid) error = %v", err)
	}
	cases := map[string]func(*Event){
		"subject":        func(event *Event) { event.Subject = "events.*" },
		"message ID":     func(event *Event) { event.MessageID = "" },
		"publication ID": func(event *Event) { event.PublicationID = "bad\nvalue" },
		"type":           func(event *Event) { event.Type = "" },
		"schema":         func(event *Event) { event.Schema = "" },
		"creation time":  func(event *Event) { event.CreatedAt = time.Now() },
		"payload":        func(event *Event) { event.Payload = append(event.Payload, 0) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			candidate := event
			candidate.Payload = slices.Clone(event.Payload)
			mutate(&candidate)
			if err := validateEvent(candidate, len(event.Payload)); !errors.Is(err, ErrRejected) {
				t.Fatalf("validateEvent() error = %v, want ErrRejected", err)
			}
		})
	}
}

func TestProducerAdmissionAndCopy(t *testing.T) {
	p := newProducer(nil, 1, testMaxPayloadBytes)
	if err := p.begin(); err != nil {
		t.Fatalf("begin(first) error = %v", err)
	}
	if err := p.begin(); !errors.Is(err, ErrCapacity) {
		t.Fatalf("begin(over capacity) error = %v, want ErrCapacity", err)
	}
	p.end()
	p.stop()
	if err := p.begin(); !errors.Is(err, ErrDraining) {
		t.Fatalf("begin(after stop) error = %v, want ErrDraining", err)
	}

	event := validTestEvent()
	ctx := reqctx.ContextWithRequestID(context.Background(), "request-1")
	msg, err := buildNATSMessage(ctx, event, testMaxPayloadBytes)
	if err != nil {
		t.Fatalf("buildNATSMessage() error = %v", err)
	}
	event.Payload[0] = 'X'
	if string(msg.Data) != "payload" {
		t.Fatalf("message payload aliased caller memory: %q", msg.Data)
	}
	if got := msg.Header.Get(headerCorrelationID); got != "request-1" {
		t.Fatalf("Correlation-Id = %q, want request-1", got)
	}
}

func TestMessageIsImmutable(t *testing.T) {
	msg := &fakeMsg{
		subject: "events.created",
		header:  eventHeaders(validTestEvent()),
		data:    []byte("payload"),
		metadata: &jetstream.MsgMetadata{
			Sequence:     jetstream.SequencePair{Stream: 3, Consumer: 2},
			NumDelivered: 2,
			NumPending:   4,
			Timestamp:    time.Unix(100, 0).UTC(),
			Stream:       "EVENTS",
			Consumer:     "events-worker",
		},
	}
	_, decoded, err := decodeMessage(msg, msg.metadata)
	if err != nil {
		t.Fatalf("decodeMessage() error = %v", err)
	}
	msg.data[0] = 'X'
	first := decoded.Payload()
	first[0] = 'Y'
	if got := string(decoded.Payload()); got != "payload" {
		t.Fatalf("decoded payload mutated through alias: %q", got)
	}
	if decoded.PublicationID() != "publication-1" || decoded.Type() != "created" || decoded.Schema() != "v1" ||
		decoded.OrderingKey() != "account-1" || decoded.CorrelationID() != "" || !decoded.CreatedAt().Equal(validTestEvent().CreatedAt) {
		t.Fatalf("decoded accessors returned inconsistent envelope: publication=%q type=%q schema=%q key=%q correlation=%q created=%v",
			decoded.PublicationID(), decoded.Type(), decoded.Schema(), decoded.OrderingKey(), decoded.CorrelationID(), decoded.CreatedAt())
	}
	if id := NewID(); id == "" {
		t.Fatal("NewID() returned empty identity")
	}
	carrier := headerCarrier(nats.Header{})
	carrier.Set("test", "value")
	if carrier.Get("test") != "value" || !slices.Contains(carrier.Keys(), "test") {
		t.Fatalf("header carrier = %#v", carrier)
	}
}

func TestPermanent(t *testing.T) {
	want := errors.New("poison")
	err := Permanent(want)
	if !isPermanent(err) || !errors.Is(err, want) {
		t.Fatalf("Permanent() = %v, want permanent wrapped error", err)
	}
	if !strings.Contains(err.Error(), "poison") {
		t.Fatalf("Permanent().Error() = %q", err.Error())
	}
	if Permanent(nil) != nil {
		t.Fatal("Permanent(nil) must be nil")
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

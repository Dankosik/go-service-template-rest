package natsjs

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

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
	decoded, _, err := decodeMessage(msg, msg.metadata)
	if err != nil {
		t.Fatalf("decodeMessage() error = %v", err)
	}
	msg.data[0] = 'X'
	first := decoded.Payload()
	first[0] = 'Y'
	if got := string(decoded.Payload()); got != "payload" {
		t.Fatalf("decoded payload mutated through alias: %q", got)
	}
	if decoded.MessageID() != "message-1" || decoded.PublicationID() != "publication-1" || decoded.Type() != "created" || decoded.Schema() != "v1" ||
		decoded.OrderingKey() != "account-1" || decoded.CorrelationID() != "" || !decoded.CreatedAt().Equal(validTestEvent().CreatedAt) {
		t.Fatalf("decoded accessors returned inconsistent envelope: message=%q publication=%q type=%q schema=%q key=%q correlation=%q created=%v",
			decoded.MessageID(), decoded.PublicationID(), decoded.Type(), decoded.Schema(), decoded.OrderingKey(), decoded.CorrelationID(), decoded.CreatedAt())
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

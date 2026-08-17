package natsjs

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// unitDeadLetter builds the record the worker would have written to the
// dead-letter stream for one source delivery, stored under its own stream and
// sequence the way the broker would have stored it.
func unitDeadLetter(t *testing.T, reason string) *fakeMsg {
	t.Helper()
	source := unitSource(t, 3)
	decoded, _, err := decodeMessage(source, source.metadata)
	if err != nil {
		t.Fatalf("decodeMessage() error = %v", err)
	}
	transfer, _ := deadLetterMessage(source, source.metadata, decoded, reason)
	return &fakeMsg{
		subject: "dead.events",
		header:  transfer.Header,
		data:    transfer.Data,
		metadata: &jetstream.MsgMetadata{
			Sequence:     jetstream.SequencePair{Stream: 11, Consumer: 7},
			NumDelivered: 1,
			Timestamp:    time.Unix(500, 0).UTC(),
			Stream:       "EVENTS_DLQ",
			Consumer:     "dlq-reader",
		},
	}
}

func TestRestoreDeadLetterReturnsTheOriginalPublication(t *testing.T) {
	t.Parallel()
	restored, err := RestoreDeadLetter(unitDeadLetter(t, deadLetterExhausted))
	if err != nil {
		t.Fatalf("RestoreDeadLetter() error = %v", err)
	}

	original := validTestEvent()
	if restored.Subject != original.Subject {
		t.Errorf("Subject = %q, want the subject it failed on %q", restored.Subject, original.Subject)
	}
	if restored.MessageID != original.MessageID {
		t.Errorf("MessageID = %q, want the logical id preserved as %q", restored.MessageID, original.MessageID)
	}
	if restored.Type != original.Type || restored.Schema != original.Schema {
		t.Errorf("Type/Schema = %q/%q, want %q/%q", restored.Type, restored.Schema, original.Type, original.Schema)
	}
	if restored.OrderingKey != original.OrderingKey {
		t.Errorf("OrderingKey = %q, want %q", restored.OrderingKey, original.OrderingKey)
	}
	if !restored.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", restored.CreatedAt, original.CreatedAt)
	}
	if !bytes.Equal(restored.Payload, original.Payload) {
		t.Errorf("Payload = %q, want the exact source bytes %q", restored.Payload, original.Payload)
	}
}

// The publication id is the one identity a redrive must not carry over:
// reusing it would have the broker recognize a duplicate and store nothing.
func TestRestoreDeadLetterReplacesThePublicationID(t *testing.T) {
	t.Parallel()
	record := unitDeadLetter(t, deadLetterExhausted)
	restored, err := RestoreDeadLetter(record)
	if err != nil {
		t.Fatalf("RestoreDeadLetter() error = %v", err)
	}
	if restored.PublicationID == validTestEvent().PublicationID {
		t.Fatal("PublicationID was carried over; the broker would discard the redrive as a duplicate")
	}
	if restored.PublicationID == record.header.Get(headerPublicationID) {
		t.Fatal("PublicationID reused the dead-letter transfer id")
	}

	// Same record restored twice is one publication, so a redrive retried after
	// an ambiguous publish deduplicates instead of delivering the work twice.
	again, err := RestoreDeadLetter(record)
	if err != nil {
		t.Fatalf("RestoreDeadLetter() second error = %v", err)
	}
	if again.PublicationID != restored.PublicationID {
		t.Errorf("PublicationID = %q on restore, %q on retry; a retried redrive would duplicate", restored.PublicationID, again.PublicationID)
	}
}

// Two dead-letter records are two publications even when they carry the same
// logical message, because each is its own place in the stream.
func TestRestoreDeadLetterSeparatesDistinctRecords(t *testing.T) {
	t.Parallel()
	first := unitDeadLetter(t, deadLetterExhausted)
	second := unitDeadLetter(t, deadLetterExhausted)
	second.metadata.Sequence.Stream = 12

	restoredFirst, err := RestoreDeadLetter(first)
	if err != nil {
		t.Fatalf("RestoreDeadLetter(first) error = %v", err)
	}
	restoredSecond, err := RestoreDeadLetter(second)
	if err != nil {
		t.Fatalf("RestoreDeadLetter(second) error = %v", err)
	}
	if restoredFirst.PublicationID == restoredSecond.PublicationID {
		t.Error("two stored records derived one publication id; the second redrive would be discarded")
	}
}

func TestRestoreDeadLetterRejectsWhatItCannotRebuild(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*fakeMsg){
		"malformed record kept no creation time": func(msg *fakeMsg) {
			msg.header.Del(headerCreatedAt)
		},
		"creation time is unparsable": func(msg *fakeMsg) {
			msg.header.Set(headerCreatedAt, "not-a-timestamp")
		},
		"origin subject is absent": func(msg *fakeMsg) {
			msg.header.Del(headerOriginalSubject)
		},
		"origin subject is a wildcard": func(msg *fakeMsg) {
			msg.header.Set(headerOriginalSubject, "events.*")
		},
		"logical message id is absent": func(msg *fakeMsg) {
			msg.header.Del(headerMessageID)
		},
		"event type is absent": func(msg *fakeMsg) {
			msg.header.Del(headerEventType)
		},
		"delivery metadata is unavailable": func(msg *fakeMsg) {
			msg.metadataErr = errors.New("no metadata")
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			record := unitDeadLetter(t, deadLetterMalformed)
			mutate(record)
			if _, err := RestoreDeadLetter(record); !errors.Is(err, ErrRejected) {
				t.Fatalf("RestoreDeadLetter() error = %v, want ErrRejected", err)
			}
		})
	}

	if _, err := RestoreDeadLetter(nil); !errors.Is(err, ErrRejected) {
		t.Fatalf("RestoreDeadLetter(nil) error = %v, want ErrRejected", err)
	}
}

// The restore is only useful if what it returns survives the publish path it
// exists to feed, so this drives the whole operator route: a dead-letter record
// goes back onto the broker addressed to the subject it originally failed on.
func TestRestoredDeadLetterRepublishesOnTheOriginalSubject(t *testing.T) {
	t.Parallel()
	broker := &recordingJetStream{ack: &jetstream.PubAck{Stream: "EVENTS", Sequence: 21}}
	client := unitClient(t, broker, RoleProducer)

	event, err := RestoreDeadLetter(unitDeadLetter(t, deadLetterExhausted))
	if err != nil {
		t.Fatalf("RestoreDeadLetter() error = %v", err)
	}
	if _, err := client.Producer().Publish(t.Context(), event); err != nil {
		t.Fatalf("Publish(restored) error = %v", err)
	}

	original := validTestEvent()
	if broker.published == nil {
		t.Fatal("Publish(restored) sent nothing to the broker")
	}
	if broker.published.Subject != original.Subject {
		t.Errorf("republished on %q, want the subject it originally failed on %q", broker.published.Subject, original.Subject)
	}
	if got := broker.published.Header.Get(headerMessageID); got != original.MessageID {
		t.Errorf("republished Message-Id = %q, want the preserved logical id %q", got, original.MessageID)
	}
	if got := broker.published.Header.Get(headerPublicationID); got == original.PublicationID {
		t.Error("republished with the original publication id; the broker would discard it as a duplicate")
	}
	if !bytes.Equal(broker.published.Data, original.Payload) {
		t.Errorf("republished payload = %q, want the exact source bytes %q", broker.published.Data, original.Payload)
	}
}

func TestDeadLetterReasonReportsWhyTheRecordWasTransferred(t *testing.T) {
	t.Parallel()
	for _, reason := range []string{deadLetterMalformed, deadLetterExhausted, deadLetterPermanent} {
		t.Run(reason, func(t *testing.T) {
			t.Parallel()
			if got := DeadLetterReason(unitDeadLetter(t, reason)); got != reason {
				t.Errorf("DeadLetterReason() = %q, want %q", got, reason)
			}
		})
	}
	if got := DeadLetterReason(nil); got != "" {
		t.Errorf("DeadLetterReason(nil) = %q, want empty", got)
	}
	if got := DeadLetterReason(unitSource(t, 1)); got != "" {
		t.Errorf("DeadLetterReason(source) = %q, want empty for a record this package did not transfer", got)
	}
}

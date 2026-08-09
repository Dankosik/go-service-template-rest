package natsjs

import (
	"fmt"
	"slices"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// RestoreDeadLetter rebuilds the event one dead-letter record came from, so an
// operator can republish it through [Producer.Publish] on the subject it
// originally failed on. It is the inverse of the transfer deadLetterMessage
// builds, and it lives here because the header names it reads are this
// package's own: nothing outside can reconstruct the envelope without copying
// string literals no test would pin.
//
// The restored event keeps the logical [Event.MessageID], so a consumer that
// deduplicates on it — through the PostgreSQL inbox or its own key — treats the
// redrive as the delivery it already refused rather than as new work. It does
// not keep the publication id: that one identifies a publication, and reusing
// it would have the broker recognize the redrive as a duplicate and store
// nothing, which is the opposite of the intent.
//
// The replacement is derived from the dead-letter record's own place in its
// stream rather than minted fresh, for the reason deadLetterTransferID is
// derived too: restoring one record twice yields one publication id, so a
// redrive retried after an ambiguous publish deduplicates at the broker instead
// of delivering the same work twice.
//
// A record dead-lettered as malformed is the one that does not come back. Its
// envelope is what failed to decode, so the identity this needs was never
// there, and it returns [ErrRejected] rather than a partially invented event.
// Those records are an operator's to inspect and republish by hand.
func RestoreDeadLetter(msg jetstream.Msg) (Event, error) {
	if msg == nil {
		return Event{}, fmt.Errorf("%w: dead-letter message is required", ErrRejected)
	}
	metadata, err := msg.Metadata()
	if err != nil {
		return Event{}, fmt.Errorf("%w: dead-letter metadata unavailable", ErrRejected)
	}
	header := msg.Headers()
	createdAt, err := time.Parse(time.RFC3339Nano, header.Get(headerCreatedAt))
	if err != nil || createdAt.IsZero() {
		return Event{}, fmt.Errorf("%w: dead-letter record carries no restorable creation time", ErrRejected)
	}
	event := Event{
		Subject:   header.Get(headerOriginalSubject),
		MessageID: header.Get(headerMessageID),
		PublicationID: streamRecordID(
			redrivePublicationPrefix,
			metadata.Stream,
			metadata.Sequence.Stream,
			metadata.Timestamp,
			header.Get(headerPublicationID),
		),
		Type:        header.Get(headerEventType),
		Schema:      header.Get(headerEventSchema),
		OrderingKey: header.Get(headerOrderingKey),
		CreatedAt:   createdAt.UTC(),
		Payload:     slices.Clone(msg.Data()),
	}
	// The payload bound belongs to the producer this event is about to go
	// through, which owns the configured maximum; everything checked here is the
	// identity that must have survived the transfer.
	if err := validateEvent(event, len(event.Payload)); err != nil {
		return Event{}, fmt.Errorf("%w: dead-letter record is not restorable", err)
	}
	return event, nil
}

// DeadLetterReason is why the worker moved one record to the dead-letter
// stream: "malformed", "exhausted", or "permanent". It is empty for a record
// this package did not transfer.
//
// An operator reads it to decide whether a redrive can succeed at all. Only
// "exhausted" describes a failure a later attempt may survive unchanged;
// "permanent" was the handler's own verdict and "malformed" never decoded, so
// both need the cause addressed before the record is worth republishing.
func DeadLetterReason(msg jetstream.Msg) string {
	if msg == nil {
		return ""
	}
	return msg.Headers().Get(headerDeadLetterReason)
}

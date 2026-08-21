package natsjs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// The dead-letter sub-protocol: a delivery the worker gave up on becomes a
// transfer onto the dead-letter stream, and an operator turns that transfer back
// into a republishable event. deadLetterMessage builds the transfer and
// RestoreDeadLetter is its exact inverse, which is why the two stay together.
//
// This is separate from message_wire.go because it changes for different
// reasons: a redrive rule, or what an operator needs to see about why a record
// stopped, rather than the envelope every message carries. The Original-* header
// names it writes are declared with the rest of the wire contract in
// message_wire.go, because they are one published contract.

// The Dead-Letter-Reason values. They belong here rather than with the metric
// and log labels in vocabulary.go because they travel on the wire to whatever
// consumes the dead-letter stream: they are a published contract, and renaming
// one is a consumer-visible change. worker_delivery.go names them; only
// Worker.deadLetter consumes them.
const (
	deadLetterMalformed = "malformed"
	deadLetterExhausted = "exhausted"
	deadLetterPermanent = "permanent"
)

// deadLetterMessage builds the transfer envelope: the original identity headers
// and trace context, the Original-* record of where the message came from, and a
// transfer id derived from that origin.
func deadLetterMessage(source jetstream.Msg, metadata *jetstream.MsgMetadata, decoded Message, reason string) (*nats.Msg, string) {
	header := make(nats.Header)
	carryIdentityHeaders(header, source.Headers())
	header.Set(headerOriginalSubject, source.Subject())
	header.Set(headerDeadLetterReason, reason)

	transferID := deadLetterTransferID(source, metadata)
	header.Set(jetstream.MsgIDHeader, transferID)
	// Message-Id in descending order of what it is worth to whoever reads the
	// dead-letter stream: the decoded id when the envelope parsed, otherwise
	// whatever carryIdentityHeaders forwarded, otherwise the transfer id so the
	// header is never absent. A malformed message reaches here with no decoded
	// id at all, which is exactly when the last fallback matters.
	switch {
	case decoded.messageID != "":
		header.Set(headerMessageID, decoded.messageID)
	case header.Get(headerMessageID) == "":
		header.Set(headerMessageID, transferID)
	}
	return &nats.Msg{Header: header, Data: slices.Clone(source.Data())}, transferID
}

// RestoreDeadLetter rebuilds the event one dead-letter record came from, so an
// operator can republish it through [Producer.Publish] on the subject it
// originally failed on. It is the inverse of the transfer deadLetterMessage
// builds, and it lives here because the header names it reads are this
// package's own: nothing outside can reconstruct the envelope without copying
// string literals no test would pin.
//
// The restored event keeps the logical [Event.MessageID], so a consumer that
// deduplicates on its own durable key treats the redrive as the delivery it
// already refused rather than as new work. It does
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
			header.Get(jetstream.MsgIDHeader),
		),
		Type:      header.Get(headerEventType),
		Schema:    header.Get(headerEventSchema),
		CreatedAt: createdAt.UTC(),
		Payload:   slices.Clone(msg.Data()),
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

// carryIdentityHeaders forwards the publisher's own envelope and trace context
// onto the transfer. An absent header is left absent rather than set empty, so
// a consumer can tell "the publisher did not send this" from "it sent a blank".
func carryIdentityHeaders(header, source nats.Header) {
	for _, name := range []string{
		headerMessageID, headerEventType, headerEventSchema, headerCreatedAt,
		"traceparent", "tracestate",
	} {
		if value := source.Get(name); value != "" {
			header.Set(name, value)
		}
	}
}

// deadLetterTransferID derives the transfer's deduplication id from the origin
// rather than minting a fresh one, so a transfer retried after an ambiguous
// publish deduplicates at the dead-letter stream instead of accumulating
// copies. The inputs are what identify one source delivery: the stream and
// sequence that stored it, its store timestamp, and the publisher's own id.
func deadLetterTransferID(source jetstream.Msg, metadata *jetstream.MsgMetadata) string {
	return streamRecordID(
		deadLetterTransferPrefix,
		metadata.Stream,
		metadata.Sequence.Stream,
		metadata.Timestamp,
		source.Headers().Get(jetstream.MsgIDHeader),
	)
}

// The two prefixes streamRecordID is called with. They only make the derived
// id legible to whoever reads it off a message; nothing parses one back.
const (
	deadLetterTransferPrefix = "dlq-"
	redrivePublicationPrefix = "redrive-"
)

// streamRecordID derives a publication id from one stored record's own place in
// a stream, so re-deriving it from that same record yields the same id and the
// broker deduplicates a retried publication instead of storing a second copy.
// Both directions of the dead-letter path need that property: the transfer into
// the stream and the redrive back out of it.
func streamRecordID(prefix, stream string, sequence uint64, storedAt time.Time, publicationID string) string {
	identity := strings.Join([]string{
		stream,
		strconv.FormatUint(sequence, 10),
		storedAt.UTC().Format(time.RFC3339Nano),
		publicationID,
	}, "\x00")
	digest := sha256.Sum256([]byte(identity))
	return prefix + hex.EncodeToString(digest[:])
}

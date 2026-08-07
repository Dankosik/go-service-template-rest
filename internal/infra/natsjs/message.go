package natsjs

import (
	"context"
	"crypto/rand"
	"errors"
	"slices"
	"time"
)

// This file is the surface a feature author writes against: the [Event] handed
// to [Producer.Publish], the [Message] a [Handler] receives, and [Permanent] for
// rejecting one. How any of it is encoded onto NATS is message_wire.go's
// business and nothing here depends on it.

// Event is one occurrence to publish. Every identity field is required and
// bounded by maxHeaderValueBytes, because each travels as a message header.
type Event struct {
	Subject       string
	MessageID     string
	PublicationID string
	Type          string
	Schema        string
	// OrderingKey travels to the consumer as data and nothing here acts on it.
	// This package does not serialize a key: the broker assigns its own stream
	// sequence and a worker above MaxConcurrency=1 runs one key's handlers
	// concurrently, which TestNATSOrderingKeyDoesNotSerialize pins. A consumer
	// that needs per-key order owns it — see the two shapes in
	// docs/durable-messaging.md.
	OrderingKey string
	CreatedAt   time.Time
	Payload     []byte
}

// PublishResult is where the broker stored an accepted event. Duplicate means
// the broker recognized PublicationID and stored nothing new, which is the
// success case for a retried publish rather than an error.
type PublishResult struct {
	Stream    string
	Sequence  uint64
	Duplicate bool
}

// DeliveryMetadata is the broker's own account of one delivery, as opposed to
// anything the publisher put in the envelope.
type DeliveryMetadata struct {
	Stream, Consumer                 string
	StreamSequence, ConsumerSequence uint64
	NumDelivered, NumPending         uint64
	StoredAt                         time.Time
}

// Message is one delivered event. Its fields are unexported and read through
// the accessors below so a handler cannot mutate what a redelivery would
// re-decode.
type Message struct {
	subject       string
	messageID     string
	publicationID string
	eventType     string
	schema        string
	orderingKey   string
	correlationID string
	createdAt     time.Time
	payload       []byte
	metadata      DeliveryMetadata
}

// Payload returns a copy, so the handler may keep or modify it after returning
// and a redelivery still sees the original bytes. Each call copies, so a
// handler that needs the payload more than once should hold the first result.
func (m Message) Payload() []byte { return slices.Clone(m.payload) }

func (m Message) Subject() string            { return m.subject }
func (m Message) MessageID() string          { return m.messageID }
func (m Message) PublicationID() string      { return m.publicationID }
func (m Message) Type() string               { return m.eventType }
func (m Message) Schema() string             { return m.schema }
func (m Message) OrderingKey() string        { return m.orderingKey }
func (m Message) CorrelationID() string      { return m.correlationID }
func (m Message) CreatedAt() time.Time       { return m.createdAt }
func (m Message) Metadata() DeliveryMetadata { return m.metadata }

// Handler is the feature-owned behavior one delivery invokes. Returning nil
// acknowledges the message; returning an error retries it after the configured
// delay, until the attempt budget — the first delivery plus one per configured
// retry delay — sends it to the dead-letter stream. Wrap with [Permanent] to
// skip that budget. A handler runs under its own timeout and must tolerate
// duplicates, because delivery is at-least-once.
type Handler func(context.Context, Message) error

type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

// Permanent marks a handler failure that retrying the same bytes cannot fix, so
// the message goes to the dead-letter stream on this attempt rather than after
// its budget runs out.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanentError{err: err}
}

func isPermanent(err error) bool {
	var target permanentError
	return errors.As(err, &target)
}

func NewID() string { return rand.Text() }

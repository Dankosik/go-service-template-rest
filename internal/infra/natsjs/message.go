package natsjs

import (
	"context"
	"crypto/rand"
	"slices"
	"time"
)

// These are adapter-internal wire shapes. Business composition uses [Registry]
// and [Publisher]; typed feature handlers are registered through domainevent.

// Event is one occurrence to publish. Every identity field is required and
// bounded by maxHeaderValueBytes, because each travels as a message header.
type Event struct {
	Subject       string
	MessageID     string
	PublicationID string
	Type          string
	// Schema versions Payload's shape for consumers. Nothing here parses it, and
	// no repository gate checks it the way OpenAPI and Buf check this service's
	// other published contracts — the emitting feature owns event compatibility
	// on its own. A stream retains what it was given and a dead-letter record can
	// be redriven long after, so a consumer must keep reading every version still
	// present, not only the one being published today.
	Schema    string
	CreatedAt time.Time
	Payload   []byte
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
func (m Message) CreatedAt() time.Time       { return m.createdAt }
func (m Message) Metadata() DeliveryMetadata { return m.metadata }

// Handler is the raw adapter behavior produced by [Registry.Handler]. Returning nil
// acknowledges the message; returning an error retries it after the configured
// delay, until the attempt budget — the first delivery plus one per configured
// retry delay — sends it to the dead-letter stream. Wrap with [Permanent] to
// skip that budget. A handler runs under its own timeout and must tolerate
// duplicates, because delivery is at-least-once.
type Handler func(context.Context, Message) error

func NewID() string { return rand.Text() }

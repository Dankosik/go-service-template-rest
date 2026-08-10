package postgresoutbox

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"
	"unicode"
	"unicode/utf8"

	"go.opentelemetry.io/otel/propagation"
)

// The envelope's size limits. migrations/000001_postgres_outbox.sql restates
// every one of them as a CHECK literal, because PostgreSQL is what stops a
// writer that bypasses Append. Change these first, then the schema:
// TestEnvelopeLimitsMatchMigrationChecks compares the two and fails on a limit
// only one side learned about.
const (
	maxTextBytes     = 256
	maxPayloadBytes  = 256 << 10
	maxMetadataBytes = 32 << 10
	maxEnvelopeBytes = 288 << 10
	// jsonWhitespace is the insignificant whitespace RFC 8259 allows before a
	// JSON value, which the stored `IS JSON OBJECT` check also skips.
	jsonWhitespace = " \t\r\n"
)

// Event is an immutable broker-neutral publication occurrence. Payload and
// Metadata are stored and retried as these exact bytes.
//
// Every text field is required, must be valid UTF-8 without control
// characters, and is limited to maxTextBytes; Payload is limited to
// maxPayloadBytes, Metadata to maxMetadataBytes, and the whole stored envelope
// to maxEnvelopeBytes. [Store.Append] rejects a violation with [ErrInvalidEvent]
// before sending anything, and the stored CHECK constraints mirror the same
// rules.
type Event struct {
	// ID is the publication identity the broker and its consumers deduplicate
	// on. It survives retry and redrive unchanged. Use [NewID] unless the
	// feature already owns an identity with the same guarantee.
	ID string
	// Type names the occurrence for consumers, such as "order.updated".
	Type string
	// Source names the emitting service or bounded context.
	Source string
	// Destination is the broker address the adapter routes on — a NATS subject,
	// a Kafka topic, an exchange. This package never interprets it; the
	// [Publisher] implementation decides what it means.
	Destination string
	// Schema versions Payload's shape for consumers, such as "v1". The outbox
	// neither parses nor validates Payload against it, and no repository gate
	// checks it the way OpenAPI and Buf check this service's other published
	// contracts — the emitting feature owns event compatibility on its own.
	// A stored event is retried and redriven as the exact bytes it was appended
	// with, so a consumer must keep reading every version still present in the
	// table, not only the one being written today.
	Schema string
	// OccurredAt is when the domain event happened, not when it was published.
	// It must be non-zero and UTC.
	OccurredAt time.Time
	// Payload is the exact JSON bytes to publish. They are stored and retried
	// byte for byte and are never normalized through jsonb.
	Payload json.RawMessage
	// Metadata is exact JSON-object bytes travelling beside Payload, for
	// whatever correlation the feature owns. Empty defaults to {}. Trace context
	// is not carried here — the outbox captures and stores its own; see
	// [Event.CreationContext].
	Metadata json.RawMessage
	// OrderingKey groups events that must publish in sequence order, typically
	// the aggregate id. Only the earliest unpublished sequence for a key is
	// claimable, so a key's events never publish out of order. It must be set
	// together with OrderingSequence, and an empty key opts out of ordering.
	//
	// The guarantee ends at [Publisher]. Whether the broker and its consumers
	// keep a key in order is theirs — see docs/postgres-transactional-outbox.md.
	// profile:messaging-nats-jetstream:start
	// The adapter worked in test/postgres_outbox_natsjs_integration_test.go does
	// not.
	// profile:messaging-nats-jetstream:end
	OrderingKey string
	// OrderingSequence is this event's position within OrderingKey and must be
	// positive and strictly above the key's retained high-water mark — take it
	// from the aggregate's own revision, under the lock that serializes the
	// domain mutation. A key therefore supports one writer at a time; a lower
	// or equal sequence is rejected with [ErrOrderingSequence].
	OrderingSequence int64
	// traceContext is the trace context that was active when this event was
	// appended. It is unexported because the outbox owns it rather than the
	// caller: [Store.Append] captures it from its own context, so no caller can
	// set or forge one, and [Store.Claim] restores it. Read it through
	// [Event.CreationContext]. tracecontext.go owns both directions.
	traceContext propagation.MapCarrier
}

// CreationContext is the trace context that was active when this event was
// appended, in the carrier an OpenTelemetry propagator consumes directly:
//
//	ctx = otel.GetTextMapPropagator().Extract(ctx, event.CreationContext())
//
// A [Publisher] forwards these entries onto its broker's headers so a consumer
// joins the same origin. Nothing here checks that it did — the same posture
// [Event.Metadata] already has.
//
// It is empty for an event appended outside a trace, and for one whose context
// could not be stored. Both extract to an unset context rather than failing,
// because a trace context never decides whether an event is delivered.
//
// The returned carrier is the event's own map rather than a copy, for the
// reason [net/http.Header] is one: every consumer only reads it, and copying
// would allocate once per claimed event. Do not write to it.
func (e Event) CreationContext() propagation.MapCarrier {
	return e.traceContext
}

// NewID mints one opaque token. A feature uses it for [Event.ID], the identity
// consumers deduplicate on; [Store.Claim] uses it for the lease token that
// fences one batch against another relay replica. Neither is derived from the
// other, and nothing reads structure out of either — both only need to be
// unique and unguessable.
func NewID() string {
	return rand.Text()
}

func (e Event) Validate() error {
	e = e.withDefaults()

	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "id", value: e.ID},
		{name: "type", value: e.Type},
		{name: "source", value: e.Source},
		{name: "destination", value: e.Destination},
		{name: "schema", value: e.Schema},
	} {
		if err := validateText(ErrInvalidEvent, field.name, field.value, maxTextBytes); err != nil {
			return err
		}
	}
	if e.OccurredAt.IsZero() {
		return fmt.Errorf("%w: occurred_at is required", ErrInvalidEvent)
	}
	if _, offset := e.OccurredAt.Zone(); offset != 0 {
		return fmt.Errorf("%w: occurred_at must use UTC", ErrInvalidEvent)
	}
	if err := validateJSON("payload", e.Payload, 1, maxPayloadBytes); err != nil {
		return err
	}
	if err := validateMetadata(e.Metadata); err != nil {
		return err
	}

	hasOrderingKey := e.OrderingKey != ""
	hasOrderingSequence := e.OrderingSequence != 0
	if hasOrderingKey != hasOrderingSequence {
		return fmt.Errorf("%w: ordering key and sequence must be present together", ErrInvalidEvent)
	}
	if hasOrderingKey {
		if err := validateText(ErrInvalidEvent, "ordering_key", e.OrderingKey, maxTextBytes); err != nil {
			return err
		}
		if e.OrderingSequence < 1 {
			return fmt.Errorf("%w: ordering_sequence must be positive", ErrInvalidEvent)
		}
	}

	total := len(e.ID) + len(e.Type) + len(e.Source) + len(e.Destination) +
		len(e.Schema) + len(e.OrderingKey) + len(e.Payload) + len(e.Metadata)
	if total > maxEnvelopeBytes {
		return fmt.Errorf("%w: envelope is %d bytes, limit is %d", ErrInvalidEvent, total, maxEnvelopeBytes)
	}
	return nil
}

func (e Event) withDefaults() Event {
	if len(e.Metadata) == 0 {
		e.Metadata = json.RawMessage("{}")
	}
	return e
}

// validateText applies one text rule set under the sentinel its caller owns.
// [Event.Validate] passes [ErrInvalidEvent], because the fault is in an
// envelope being stored; the [Store] identity checks pass [ErrConfig], because
// the fault is in the call rather than in any event. Keeping those apart is
// what lets a caller tell a rejected event from a bad id, lease token, or error
// class — ErrInvalidEvent means one Event failed Validate, and nothing else.
func validateText(sentinel error, name, value string, limit int) error {
	if value == "" {
		return fmt.Errorf("%w: %s is required", sentinel, name)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%w: %s must be valid UTF-8", sentinel, name)
	}
	if len(value) > limit {
		return fmt.Errorf("%w: %s is %d bytes, limit is %d", sentinel, name, len(value), limit)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("%w: %s contains a control character", sentinel, name)
		}
	}
	return nil
}

func validateJSON(name string, value []byte, minimum, maximum int) error {
	if len(value) < minimum || len(value) > maximum {
		return fmt.Errorf("%w: %s is %d bytes, accepted range is [%d,%d]", ErrInvalidEvent, name, len(value), minimum, maximum)
	}
	if !utf8.Valid(value) || !json.Valid(value) {
		return fmt.Errorf("%w: %s must be valid UTF-8 JSON", ErrInvalidEvent, name)
	}
	return nil
}

// validateMetadata accepts the same bytes the stored `IS JSON OBJECT` check
// accepts: valid JSON whose first non-whitespace byte opens an object. Deciding
// it on the leading byte keeps the append path free of a second full parse, and
// JSON has no other value that can start with '{'.
func validateMetadata(value []byte) error {
	if err := validateJSON("metadata", value, 2, maxMetadataBytes); err != nil {
		return err
	}
	// validateJSON already proved these bytes are one well-formed JSON value, so
	// a non-empty remainder is guaranteed and its first byte decides the kind.
	if bytes.TrimLeft(value, jsonWhitespace)[0] != '{' {
		return fmt.Errorf("%w: metadata must be a JSON object", ErrInvalidEvent)
	}
	return nil
}

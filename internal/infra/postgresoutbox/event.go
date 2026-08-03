package postgresoutbox

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	maxTextBytes     = 256
	maxPayloadBytes  = 256 << 10
	maxMetadataBytes = 32 << 10
	maxEnvelopeBytes = 288 << 10
	// jsonWhitespace is the insignificant whitespace RFC 8259 allows before a
	// JSON value, which the stored `IS JSON OBJECT` check also skips.
	jsonWhitespace = " \t\r\n"
)

var ErrInvalidEvent = errors.New("invalid outbox event")

// Event is an immutable broker-neutral publication occurrence. Payload and
// Metadata are stored and retried as these exact bytes.
type Event struct {
	ID               string
	Type             string
	Source           string
	Destination      string
	Schema           string
	OccurredAt       time.Time
	Payload          json.RawMessage
	Metadata         json.RawMessage
	OrderingKey      string
	OrderingSequence int64
}

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
		if err := validateText(field.name, field.value, maxTextBytes); err != nil {
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
		if err := validateText("ordering_key", e.OrderingKey, maxTextBytes); err != nil {
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

func validateText(name, value string, limit int) error {
	if value == "" {
		return fmt.Errorf("%w: %s is required", ErrInvalidEvent, name)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%w: %s must be valid UTF-8", ErrInvalidEvent, name)
	}
	if len(value) > limit {
		return fmt.Errorf("%w: %s is %d bytes, limit is %d", ErrInvalidEvent, name, len(value), limit)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("%w: %s contains a control character", ErrInvalidEvent, name)
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

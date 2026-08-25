package domainevent

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Event is one domain occurrence encoded for publication. Treat it as
// immutable after construction; value copies share Payload's backing bytes.
// Broker routing and delivery policy deliberately live outside this type.
type Event struct {
	ID         string
	Type       string
	Version    uint16
	OccurredAt time.Time
	Payload    json.RawMessage
}

// Kind binds one Go payload type to its stable event name and version. It is
// transport-neutral: broker routing is supplied separately by composition.
type Kind[T any] struct {
	Type    string
	Version uint16
}

// Define declares a typed event kind. Invalid constants are rejected when
// [Kind.New] validates the event, keeping declaration convenient without an init panic.
func Define[T any](eventType string, version uint16) Kind[T] {
	return Kind[T]{Type: eventType, Version: version}
}

// Typed is the shape delivered to a typed handler.
type Typed[T any] struct {
	ID         string
	OccurredAt time.Time
	Payload    T
}

// New encodes a typed payload once, before the caller enters its transaction.
func (k Kind[T]) New(id string, occurredAt time.Time, payload T) (Event, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return Event{}, fmt.Errorf("marshal domain event payload: %w", err)
	}
	event := Event{
		ID:         id,
		Type:       k.Type,
		Version:    k.Version,
		OccurredAt: occurredAt,
		Payload:    encoded,
	}
	if err := event.Validate(); err != nil {
		return Event{}, err
	}
	return event, nil
}

func (e Event) Validate() error {
	if err := validateText("id", e.ID); err != nil {
		return err
	}
	if err := validateText("type", e.Type); err != nil {
		return err
	}
	if e.Version == 0 {
		return errors.New("version must be positive")
	}
	if e.OccurredAt.IsZero() {
		return errors.New("occurred_at is required")
	}
	if _, offset := e.OccurredAt.Zone(); offset != 0 {
		return errors.New("occurred_at must use UTC")
	}
	if len(e.Payload) == 0 || !json.Valid(e.Payload) {
		return errors.New("payload must be valid JSON")
	}
	return nil
}

func validateText(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", name)
	}
	if !utf8.ValidString(value) || strings.ContainsFunc(value, unicode.IsControl) {
		return fmt.Errorf("%s must be valid text without control characters", name)
	}
	return nil
}

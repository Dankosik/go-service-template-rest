package domainevent

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const maxTextBytes = 256

// Event is one immutable domain occurrence ready for durable publication.
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

// Define declares a typed event kind. Invalid constants are rejected when New
// validates the event, keeping declaration convenient without an init panic.
func Define[T any](eventType string, version uint16) Kind[T] {
	return Kind[T]{Type: eventType, Version: version}
}

// Typed is the shape delivered to a typed handler.
type Typed[T any] struct {
	ID         string
	OccurredAt time.Time
	Payload    T
}

func (k Kind[T]) New(id string, occurredAt time.Time, payload T) (Event, error) {
	return New(id, k.Type, k.Version, occurredAt, payload)
}

// Registrar is the transport-neutral seam implemented by a messaging adapter.
type Registrar interface {
	Register(eventType string, version uint16, handler func(context.Context, Event) error) error
}

// Handle registers a typed handler without exposing subjects, headers,
// delivery attempts, or acknowledgements to business code.
func Handle[T any](registrar Registrar, kind Kind[T], handler func(context.Context, Typed[T]) error) error {
	if registrar == nil {
		return errors.New("event registrar is required")
	}
	if handler == nil {
		return errors.New("event handler is required")
	}
	err := registrar.Register(kind.Type, kind.Version, func(ctx context.Context, event Event) error {
		var payload T
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return Permanent(fmt.Errorf("decode %s v%d: %w", event.Type, event.Version, err))
		}
		return handler(ctx, Typed[T]{ID: event.ID, OccurredAt: event.OccurredAt, Payload: payload})
	})
	if err != nil {
		return fmt.Errorf("register typed event handler: %w", err)
	}
	return nil
}

type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

// Permanent marks bytes that retrying unchanged cannot make processable.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanentError{err: err}
}

func IsPermanent(err error) bool {
	var target permanentError
	return errors.As(err, &target)
}

// New encodes a typed payload once, before the caller enters its transaction.
func New(id, eventType string, version uint16, occurredAt time.Time, payload any) (Event, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return Event{}, fmt.Errorf("marshal domain event payload: %w", err)
	}
	event := Event{
		ID:         id,
		Type:       eventType,
		Version:    version,
		OccurredAt: occurredAt,
		Payload:    encoded,
	}
	if err := event.Validate(); err != nil {
		return Event{}, err
	}
	return event, nil
}

func NewID() string {
	return rand.Text()
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
	if len(value) > maxTextBytes {
		return fmt.Errorf("%s exceeds %d bytes", name, maxTextBytes)
	}
	if !utf8.ValidString(value) || strings.ContainsFunc(value, unicode.IsControl) {
		return fmt.Errorf("%s must be valid text without control characters", name)
	}
	return nil
}

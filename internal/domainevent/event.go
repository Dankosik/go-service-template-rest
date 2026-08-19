package domainevent

import (
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

package natsjs

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func TestEventValidation(t *testing.T) {
	event := validTestEvent()
	if err := validateEvent(event, len(event.Payload)); err != nil {
		t.Fatalf("validateEvent(valid) error = %v", err)
	}
	cases := map[string]func(*Event){
		"subject":        func(event *Event) { event.Subject = "events.*" },
		"message ID":     func(event *Event) { event.MessageID = "" },
		"publication ID": func(event *Event) { event.PublicationID = "bad\nvalue" },
		"type":           func(event *Event) { event.Type = "" },
		"schema":         func(event *Event) { event.Schema = "" },
		"creation time":  func(event *Event) { event.CreatedAt = time.Now() },
		"payload":        func(event *Event) { event.Payload = append(event.Payload, 0) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			candidate := event
			candidate.Payload = slices.Clone(event.Payload)
			mutate(&candidate)
			if err := validateEvent(candidate, len(event.Payload)); !errors.Is(err, ErrRejected) {
				t.Fatalf("validateEvent() error = %v, want ErrRejected", err)
			}
		})
	}
}

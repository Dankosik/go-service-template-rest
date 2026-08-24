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
	zeroOffset := event
	zeroOffset.CreatedAt = event.CreatedAt.In(time.FixedZone("UTC-like", 0))
	if err := validateEvent(zeroOffset, len(zeroOffset.Payload)); err != nil {
		t.Fatalf("validateEvent(zero-offset time) error = %v", err)
	}
	cases := map[string]func(*Event){
		"subject":        func(event *Event) { event.Subject = "events.*" },
		"message ID":     func(event *Event) { event.MessageID = "" },
		"publication ID": func(event *Event) { event.PublicationID = "bad\nvalue" },
		"type":           func(event *Event) { event.Type = "" },
		"schema":         func(event *Event) { event.Schema = "" },
		"creation time":  func(event *Event) { event.CreatedAt = time.Unix(1, 0).In(time.FixedZone("offset", 60)) },
		"payload":        func(event *Event) { event.Payload = append(event.Payload, 0) },
		// Ranging a string yields U+FFFD per invalid byte, and U+FFFD is not a
		// control character, so the control scan alone accepted a header value
		// that is not text — leaving the consumer that decodes it to fail
		// instead. The sibling durable-identity validators reject both of these
		// at their own boundary.
		"invalid UTF-8 message ID": func(event *Event) { event.MessageID = "id-\xff\xfe" },
		// C0 was refused and C1 was not. Both are equally unreadable wherever a
		// header is printed rather than parsed.
		"C1 control in publication ID": func(event *Event) { event.PublicationID = "pub-\u0085-id" },
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

package domainevent

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Parallel()

	type payload struct {
		OrderID string `json:"order_id"`
	}
	event, err := New("event-1", "order.updated", 1, time.Unix(1, 0).UTC(), payload{OrderID: "order-1"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if string(event.Payload) != `{"order_id":"order-1"}` {
		t.Fatalf("Payload = %s", event.Payload)
	}

	for name, mutate := range map[string]func(*Event){
		"missing id":      func(event *Event) { event.ID = "" },
		"invalid id text": func(event *Event) { event.ID = string([]byte{0xff}) },
		"controlled id":   func(event *Event) { event.ID = "bad\n" },
		"oversized id":    func(event *Event) { event.ID = strings.Repeat("x", maxTextBytes+1) },
		"invalid type":    func(event *Event) { event.Type = "bad\n" },
		"missing version": func(event *Event) { event.Version = 0 },
		"missing time":    func(event *Event) { event.OccurredAt = time.Time{} },
		"non UTC time":    func(event *Event) { event.OccurredAt = time.Unix(1, 0).In(time.FixedZone("offset", 60)) },
		"missing payload": func(event *Event) { event.Payload = nil },
		"invalid payload": func(event *Event) { event.Payload = []byte("{") },
		"oversized type":  func(event *Event) { event.Type = strings.Repeat("x", maxTextBytes+1) },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			invalid := event
			mutate(&invalid)
			if err := invalid.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
	if _, err := New("event-2", "order.updated", 1, time.Unix(1, 0).UTC(), func() {}); err == nil {
		t.Fatal("New() accepted an unencodable payload")
	}
	if id := NewID(); id == "" {
		t.Fatal("NewID() returned an empty identity")
	}
}

func TestPermanent(t *testing.T) {
	want := errors.New("poison")
	err := Permanent(want)
	if !IsPermanent(err) || !errors.Is(err, want) {
		t.Fatalf("Permanent() = %v", err)
	}
	if Permanent(nil) != nil {
		t.Fatal("Permanent(nil) must be nil")
	}
}

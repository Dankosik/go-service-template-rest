package domainevent

import (
	"strings"
	"testing"
	"time"
)

func TestKindNew(t *testing.T) {
	t.Parallel()

	type payload struct {
		OrderID string `json:"order_id"`
	}
	kind := Define[payload]("order.updated", 1)
	event, err := kind.New("event-1", time.Unix(1, 0).UTC(), payload{OrderID: "order-1"})
	if err != nil {
		t.Fatalf("Kind.New() error = %v", err)
	}
	if string(event.Payload) != `{"order_id":"order-1"}` {
		t.Fatalf("Payload = %s", event.Payload)
	}
	if event.ID != "event-1" || event.Type != "order.updated" || event.Version != 1 ||
		!event.OccurredAt.Equal(time.Unix(1, 0)) {
		t.Fatalf("Event = %#v", event)
	}

	for name, mutate := range map[string]func(*Event){
		"missing id":      func(event *Event) { event.ID = "" },
		"invalid id text": func(event *Event) { event.ID = string([]byte{0xff}) },
		"controlled id":   func(event *Event) { event.ID = "bad\n" },
		"invalid type":    func(event *Event) { event.Type = "bad\n" },
		"missing version": func(event *Event) { event.Version = 0 },
		"missing time":    func(event *Event) { event.OccurredAt = time.Time{} },
		"non UTC time":    func(event *Event) { event.OccurredAt = time.Unix(1, 0).In(time.FixedZone("offset", 60)) },
		"missing payload": func(event *Event) { event.Payload = nil },
		"invalid payload": func(event *Event) { event.Payload = []byte("{") },
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
	zeroOffset := event
	zeroOffset.OccurredAt = time.Unix(1, 0).In(time.FixedZone("UTC-like", 0))
	if err := zeroOffset.Validate(); err != nil {
		t.Fatalf("Validate(zero-offset time) error = %v", err)
	}
	for name, mutate := range map[string]func(*Event){
		"long id":   func(event *Event) { event.ID = strings.Repeat("x", 257) },
		"long type": func(event *Event) { event.Type = strings.Repeat("x", 257) },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			longText := event
			mutate(&longText)
			if err := longText.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
	unencodable := Define[func()]("order.updated", 1)
	if _, err := unencodable.New("event-2", time.Unix(1, 0).UTC(), func() {}); err == nil {
		t.Fatal("Kind.New() accepted an unencodable payload")
	}
}

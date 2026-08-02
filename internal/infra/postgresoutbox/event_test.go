package postgresoutbox

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestEventValidation(t *testing.T) {
	t.Parallel()

	valid := Event{
		ID:          "event-1",
		Type:        "order.created",
		Source:      "orders",
		Destination: "events",
		Schema:      "v1",
		OccurredAt:  time.Unix(1, 0).UTC(),
		Payload:     []byte(`{"order_id":"1"}`),
		Metadata:    []byte(`{"trace_id":"a"}`),
	}

	tests := map[string]func(Event) Event{
		"missing id": func(event Event) Event {
			event.ID = ""
			return event
		},
		"control text": func(event Event) Event {
			event.Type = "order\ncreated"
			return event
		},
		"non UTC occurred at": func(event Event) Event {
			event.OccurredAt = event.OccurredAt.In(time.FixedZone("east", 60))
			return event
		},
		"invalid payload": func(event Event) Event {
			event.Payload = []byte(`{"broken"`)
			return event
		},
		"metadata array": func(event Event) Event {
			event.Metadata = []byte(`[]`)
			return event
		},
		"ordering key only": func(event Event) Event {
			event.OrderingKey = "order-1"
			return event
		},
		"ordering sequence only": func(event Event) Event {
			event.OrderingSequence = 1
			return event
		},
		"oversized text": func(event Event) Event {
			event.Source = strings.Repeat("s", maxTextBytes+1)
			return event
		},
		"oversized payload": func(event Event) Event {
			event.Payload = append(append([]byte{'"'}, bytes.Repeat([]byte{'p'}, maxPayloadBytes-1)...), '"')
			return event
		},
		"oversized metadata": func(event Event) Event {
			event.Metadata = []byte(`{"m":"` + strings.Repeat("m", maxMetadataBytes) + `"}`)
			return event
		},
		"oversized envelope": func(event Event) Event {
			event.Payload = []byte(`"` + strings.Repeat("p", maxPayloadBytes-2) + `"`)
			event.Metadata = []byte(`{"m":"` + strings.Repeat("m", maxMetadataBytes-8) + `"}`)
			return event
		},
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("valid event: %v", err)
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := mutate(valid).Validate(); !errors.Is(err, ErrInvalidEvent) {
				t.Fatalf("Validate() error = %v, want ErrInvalidEvent", err)
			}
		})
	}
}

func TestEventBoundaryBytesAndDefaults(t *testing.T) {
	t.Parallel()
	if first, second := NewID(), NewID(); first == "" || first == second {
		t.Fatalf("NewID() values = %q, %q", first, second)
	}

	event := Event{
		ID:               "e",
		Type:             "t",
		Source:           "s",
		Destination:      "d",
		Schema:           "v",
		OccurredAt:       time.Unix(1, 0).UTC(),
		Payload:          []byte(`"` + strings.Repeat("p", maxPayloadBytes-2) + `"`),
		OrderingKey:      "k",
		OrderingSequence: 1,
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("maximum payload with default metadata: %v", err)
	}
	if got := string(event.withDefaults().Metadata); got != "{}" {
		t.Fatalf("default metadata = %q, want {}", got)
	}

	event.Payload = []byte(`{}`)
	event.Metadata = []byte(`{"m":"` + strings.Repeat("m", maxMetadataBytes-8) + `"}`)
	if got, want := len(event.Metadata), maxMetadataBytes; got != want {
		t.Fatalf("metadata bytes = %d, want %d", got, want)
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("maximum metadata: %v", err)
	}
}

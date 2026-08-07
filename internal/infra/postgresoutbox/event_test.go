package postgresoutbox

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// Every size this package rejects an envelope on is restated in the migration as
// a CHECK literal, because PostgreSQL is what stops a writer that bypasses
// Append. Nothing else compares the two, and raising a Go constant alone would
// turn a rejection this package owns — returned before any statement is sent —
// into a check violation inside the feature's own transaction, rolling back the
// mutation the event was appended with.
//
// The table is every column each constant governs, including maxErrorClassBytes
// from store_rows.go: the claim is that the schema and this package describe one
// set of limits, and splitting it across two test files would prove neither half.
// Matching on the rendered CHECK text means reformatting the migration fails this
// test; the wanted string is in the failure, so the repair is mechanical.
func TestEnvelopeLimitsMatchMigrationChecks(t *testing.T) {
	t.Parallel()

	const migrationPath = "../../../migrations/000001_postgres_outbox.sql"
	schema, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read %s: %v", migrationPath, err)
	}
	for _, check := range []struct {
		column string
		want   string
	}{
		{column: "id", want: fmt.Sprintf("octet_length(id) BETWEEN 1 AND %d", maxTextBytes)},
		{column: "event_type", want: fmt.Sprintf("octet_length(event_type) BETWEEN 1 AND %d", maxTextBytes)},
		{column: "source", want: fmt.Sprintf("octet_length(source) BETWEEN 1 AND %d", maxTextBytes)},
		{column: "destination", want: fmt.Sprintf("octet_length(destination) BETWEEN 1 AND %d", maxTextBytes)},
		{column: "schema_name", want: fmt.Sprintf("octet_length(schema_name) BETWEEN 1 AND %d", maxTextBytes)},
		{column: "ordering_key", want: fmt.Sprintf("octet_length(ordering_key) BETWEEN 1 AND %d", maxTextBytes)},
		{column: "lease_token", want: fmt.Sprintf("octet_length(lease_token) BETWEEN 1 AND %d", maxTextBytes)},
		{column: "payload", want: fmt.Sprintf("octet_length(payload) BETWEEN 1 AND %d", maxPayloadBytes)},
		{column: "metadata", want: fmt.Sprintf("octet_length(metadata) BETWEEN 2 AND %d", maxMetadataBytes)},
		{column: "envelope total", want: fmt.Sprintf("octet_length(metadata) <= %d", maxEnvelopeBytes)},
		{
			column: "last_error_class",
			want:   fmt.Sprintf("octet_length(last_error_class) BETWEEN 1 AND %d", maxErrorClassBytes),
		},
	} {
		if !bytes.Contains(schema, []byte(check.want)) {
			t.Errorf("migration does not bound %s as this package does: want %q", check.column, check.want)
		}
	}
}

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
		"non UTF-8 text": func(event Event) Event {
			event.Type = "\xff"
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
		"negative ordering sequence": func(event Event) Event {
			event.OrderingKey, event.OrderingSequence = "order-1", -1
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

// Every rejection Validate owns lives in TestEventValidation's table above.
// These rules are also the ones the stored CHECK constraints mirror, so a
// feature that violates one sees a single ErrInvalidEvent instead of a database
// failure — which is why they are proven here rather than only in integration.

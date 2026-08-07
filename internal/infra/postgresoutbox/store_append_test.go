package postgresoutbox

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestStoreAppendValidationAndInsert(t *testing.T) {
	t.Parallel()

	store := stubbedStore(databaseStub{})
	tx := &transactionStub{}
	if err := store.Append(t.Context(), tx); err != nil {
		t.Fatalf("Append(no events) error = %v", err)
	}
	event := outboxEventForUnit()
	event.Type, event.Source, event.Destination, event.Schema = "type", "source", "destination", "v1"
	event.OccurredAt = time.Unix(1, 0).UTC()
	// Derived from the valid event, so clearing the id is what makes it invalid.
	// Built the other way round it would be rejected for the fields
	// outboxEventForUnit never sets, and the assertion would hold either way.
	invalid := event
	invalid.ID = ""
	if err := store.Append(t.Context(), tx, invalid); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("Append(invalid) error = %v", err)
	}
	// One invalid event keeps the whole call off the wire, so a caller never
	// commits part of what it asked to append.
	if err := store.Append(t.Context(), tx, event, invalid); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("Append(partly invalid) error = %v", err)
	}
	if tx.statements != 0 {
		t.Fatalf("rejected append sent %d statements, want 0", tx.statements)
	}

	// A call with no ordering key takes the statement that never touches a
	// head, which is the one that carries no ordering columns.
	second := event
	second.ID = "second"
	if err := store.Append(t.Context(), tx, event, second); err != nil {
		t.Fatalf("Append(unordered) error = %v", err)
	}
	if tx.statements != 1 || len(tx.arguments) != 9 {
		t.Fatalf("Append(unordered) sent %d statements with %d column arrays, want 1 and 9",
			tx.statements, len(tx.arguments))
	}
	// One creation context per event, and the same one for every event of the
	// call: they share the caller's context, so a second encoding would be a
	// second copy of one value.
	contexts, ok := tx.arguments[8].([][]byte)
	if !ok || len(contexts) != 2 {
		t.Fatalf("Append(unordered) trace contexts = %v", tx.arguments[8])
	}
	// Appending outside a trace stores the empty object rather than failing.
	for index, stored := range contexts {
		if string(stored) != "{}" {
			t.Fatalf("Append(unordered) trace context %d = %q, want the empty object", index, stored)
		}
	}

	// Ordered and unordered events travel together, so a mixed call is still
	// one statement however many events it carries.
	tx.statements = 0
	ordered := event
	ordered.ID, ordered.OrderingKey, ordered.OrderingSequence = "ordered", "key", 1
	if err := store.Append(t.Context(), tx, event, second, ordered); err != nil {
		t.Fatalf("Append(mixed) error = %v", err)
	}
	if tx.statements != 1 || len(tx.arguments) != 11 {
		t.Fatalf("Append(mixed) sent %d statements with %d column arrays, want 1 and 11",
			tx.statements, len(tx.arguments))
	}
	if keys, ok := tx.arguments[9].([]string); !ok || len(keys) != 3 ||
		keys[0] != "" || keys[1] != "" || keys[2] != "key" {
		t.Fatalf("Append(mixed) ordering keys = %v", tx.arguments[9])
	}

	// A returned row is a key whose first sequence did not clear its retained
	// high-water mark, and the message names that key rather than the call.
	tx.rejected = []pgx.Row{orderingRejectionRow{key: "key", sequence: 1}}
	err := store.Append(t.Context(), tx, ordered)
	rejection := fmt.Sprintf("%v", err)
	if !errors.Is(err, ErrOrderingSequence) || !strings.Contains(rejection, `key "key" sequence 1`) {
		t.Fatalf("Append(rejected sequence) error = %v", err)
	}
	tx.rejected = nil
	tx.err = errors.New("insert")
	if err := store.Append(t.Context(), tx, event); !errors.Is(err, tx.err) {
		t.Fatalf("Append(database) error = %v", err)
	}
}

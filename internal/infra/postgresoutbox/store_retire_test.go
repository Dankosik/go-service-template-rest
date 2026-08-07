package postgresoutbox

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// R4 at the Go boundary: what the call refuses before reaching PostgreSQL, and
// what it makes of the statement's rejection report. The precondition itself —
// that a key with unpublished events keeps its mark, and that a racing append
// serializes against the head lock — is PostgreSQL's, and is proven in
// test/postgres_outbox_integration_test.go rather than against a stub.
func TestStoreRetireOrderingKeysValidation(t *testing.T) {
	t.Parallel()

	store := stubbedStore(databaseStub{})
	tx := &transactionStub{}

	// An empty call is a no-op rather than an error, so a caller retiring
	// whatever its aggregate happens to own needs no length check of its own.
	if err := store.RetireOrderingKeys(t.Context(), tx); err != nil {
		t.Fatalf("RetireOrderingKeys(no keys) error = %v", err)
	}
	if err := store.RetireOrderingKeys(t.Context(), nil, "key"); !errors.Is(err, ErrConfig) {
		t.Fatalf("RetireOrderingKeys(no transaction) error = %v", err)
	}
	// A bad key is a fault in the call rather than in an envelope, so it reports
	// ErrConfig like every other identity check — never ErrInvalidEvent, which
	// means one Event failed Validate and nothing else.
	if err := store.RetireOrderingKeys(t.Context(), tx, ""); !errors.Is(err, ErrConfig) {
		t.Fatalf("RetireOrderingKeys(empty key) error = %v", err)
	}
	if err := store.RetireOrderingKeys(t.Context(), tx, strings.Repeat("k", maxTextBytes+1)); !errors.Is(err, ErrConfig) {
		t.Fatalf("RetireOrderingKeys(oversize key) error = %v", err)
	}
	// One invalid key keeps the whole call off the wire, the same all-or-nothing
	// the append gives a caller.
	if err := store.RetireOrderingKeys(t.Context(), tx, "good", ""); !errors.Is(err, ErrConfig) {
		t.Fatalf("RetireOrderingKeys(partly invalid) error = %v", err)
	}
	if tx.statements != 0 {
		t.Fatalf("rejected retirement sent %d statements, want 0", tx.statements)
	}
}

func TestStoreRetireOrderingKeysOutcomes(t *testing.T) {
	t.Parallel()

	store := stubbedStore(databaseStub{})

	// No rows returned is the normal path: every named key was either retired or
	// was never there, and both are success.
	drained := &transactionStub{}
	if err := store.RetireOrderingKeys(t.Context(), drained, "drained", "never-seen"); err != nil {
		t.Fatalf("RetireOrderingKeys(drained) error = %v", err)
	}
	if drained.statements != 1 {
		t.Fatalf("RetireOrderingKeys(drained) sent %d statements, want 1", drained.statements)
	}
	if keys, ok := drained.arguments[0].([]string); !ok || len(keys) != 2 {
		t.Fatalf("RetireOrderingKeys bound %v, want the key array", drained.arguments[0])
	}

	// A returned row is a key that still owns unpublished events. It is a
	// distinct sentinel so a caller can absorb it — "this aggregate is not
	// finished draining" is an ordinary outcome, unlike a database failure.
	active := &transactionStub{rejected: []pgx.Row{singleTextRow("busy"), singleTextRow("also-busy")}}
	err := store.RetireOrderingKeys(t.Context(), active, "busy", "also-busy")
	if err == nil {
		t.Fatal("RetireOrderingKeys(active) error = nil, want ErrOrderingKeyActive")
	}
	if !errors.Is(err, ErrOrderingKeyActive) {
		t.Fatalf("RetireOrderingKeys(active) error = %v, want ErrOrderingKeyActive", err)
	}
	message := err.Error()
	if !strings.Contains(message, `"busy"`) || !strings.Contains(message, "of 2") {
		t.Errorf("RetireOrderingKeys(active) message = %q, want the first key and the count", message)
	}
	// Every caller-owned key would be an unbounded number of identifiers in a
	// string a caller may log; the count carries the rest.
	if strings.Contains(message, "also-busy") {
		t.Errorf("RetireOrderingKeys(active) message lists every key: %q", message)
	}

	failing := &transactionStub{err: errors.New("statement failed")}
	if err := store.RetireOrderingKeys(t.Context(), failing, "key"); !errors.Is(err, failing.err) {
		t.Fatalf("RetireOrderingKeys(database) error = %v", err)
	}
}

// A refused retirement is a rejection rather than a fault on the operation
// metric, which is what keeps an ordinary "not finished draining" outcome out of
// a database-error dashboard.
func TestRetireOrderingKeysReportsRejectionNotFailure(t *testing.T) {
	t.Parallel()

	outcome, errorClass := storeOutcome(ErrOrderingKeyActive)
	if outcome != outcomeRejected || errorClass != classValidation {
		t.Errorf("storeOutcome(ErrOrderingKeyActive) = %s/%s, want %s/%s",
			outcome, errorClass, outcomeRejected, classValidation)
	}
}

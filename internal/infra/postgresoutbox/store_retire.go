package postgresoutbox

import (
	"context"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

// This file owns the end of an ordering key's life, which is the one thing the
// rest of the package deliberately never decides.
//
// outbox_ordering_heads holds one row per ordering key ever appended, and
// cleanup never removes one: the retained high-water mark is what rejects a
// replayed sequence, and it has to outlive the events it was derived from to do
// that. A service keyed on a per-aggregate identity therefore accumulates a row
// per aggregate for the life of the database, and nothing inside the outbox can
// prove that a given key will never be used again. Only the feature that owns
// the aggregate knows that, so retirement is something it asserts rather than
// something this package infers.
//
// Nothing here is time-based, and that is the whole safety argument. An idle key
// is not a terminal key: a quiet aggregate that wakes up must still have its
// replayed sequences rejected. A TTL would silently trade that protection for
// disk, for every key that merely went quiet.

// RetireOrderingKeys removes the retained high-water mark of every terminal
// ordering key it is given, inside the transaction the caller owns. It neither
// begins nor commits that transaction, so the assertion "this aggregate is
// finished" commits atomically with whatever domain write makes it true.
//
// A key with unpublished events is refused with [ErrOrderingKeyActive] and
// nothing is retired — not that key and not the rest of the call, so a caller
// never has to reason about a partly applied retirement. A key that is unknown
// or already retired succeeds and changes nothing, which makes a repeated call
// idempotent.
//
// After retirement the key's sequence space restarts: a later append is
// accepted at any positive sequence, including one already used before. That is
// exactly the protection the caller traded away by asserting terminality, and it
// is why the assertion is the caller's. Retire a key whose identity can recur
// and replayed sequences for it stop being rejected.
//
// Concurrency is settled by PostgreSQL rather than here. The statement locks the
// same head rows in the same order the ordered append takes, so an append for
// one of these keys serializes either before this call — and is then visible as
// an unpublished event, refusing the retirement — or after it, and then
// establishes a fresh mark from its own sequence.
func (s *Store) RetireOrderingKeys(ctx context.Context, tx pgx.Tx, keys ...string) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "retire_ordering_key", started, err) }()
	if !s.valid() {
		return errStoreRequired()
	}
	if tx == nil {
		return fmt.Errorf("%w: transaction is required", ErrConfig)
	}
	if len(keys) == 0 {
		return nil
	}
	for _, key := range keys {
		if err := validateText(ErrConfig, "ordering_key", key, maxTextBytes); err != nil {
			return err
		}
	}

	active, err := sqlcgen.New(tx).RetireOutboxOrderingKeys(ctx, keys)
	if err != nil {
		return fmt.Errorf("retire outbox ordering keys: %w", err)
	}
	if len(active) > 0 {
		// The first key is enough for the caller to act on, and the count says
		// whether it was the only one. Listing every key would put an unbounded
		// number of caller-owned identifiers into an error string that a caller
		// may well log.
		return fmt.Errorf(
			"%w: key %q is one of %d still holding unpublished events",
			ErrOrderingKeyActive, active[0], len(active),
		)
	}
	return nil
}

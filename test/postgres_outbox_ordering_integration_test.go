//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/jackc/pgx/v5"
)

func TestPostgresOutboxOrderingAuthority(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newOutboxFixture(t)
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("ordered-2", "account-1", 2))

	err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return store.Append(ctx, tx, orderedEvent("ordered-1", "account-1", 1))
	})
	if !errors.Is(err, postgresoutbox.ErrOrderingSequence) {
		t.Fatalf("lower sequence error = %v, want ErrOrderingSequence", err)
	}
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("ordered-4", "account-1", 4))

	var highWater int64
	if err := pool.PGX().QueryRow(ctx, `SELECT last_sequence FROM outbox_ordering_heads WHERE ordering_key = 'account-1'`).Scan(&highWater); err != nil {
		t.Fatalf("read ordering high-water: %v", err)
	}
	if highWater != 4 {
		t.Fatalf("ordering high-water = %d, want 4", highWater)
	}

	for _, wantID := range []string{"ordered-2", "ordered-4"} {
		claim, err := claimOutboxEvent(ctx, store, time.Minute)
		if err != nil {
			t.Fatalf("Claim() for %s: %v", wantID, err)
		}
		if claim.Event.ID != wantID {
			t.Fatalf("claimed %q, want %q", claim.Event.ID, wantID)
		}
		if err := markOutboxPublished(ctx, store, claim); err != nil {
			t.Fatalf("MarkPublished(%s): %v", wantID, err)
		}
	}
	if _, err := pool.PGX().Exec(ctx, "UPDATE outbox_events SET published_at = clock_timestamp() - interval '2 hours'"); err != nil {
		t.Fatalf("backdate published rows: %v", err)
	}
	if _, err := store.CleanupPublished(ctx, time.Hour, 10); err != nil {
		t.Fatalf("CleanupPublished(): %v", err)
	}
	err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return store.Append(ctx, tx, orderedEvent("ordered-3", "account-1", 3))
	})
	if !errors.Is(err, postgresoutbox.ErrOrderingSequence) {
		t.Fatalf("post-cleanup lower sequence error = %v, want ErrOrderingSequence", err)
	}

	type appendResult struct {
		sequence int64
		err      error
	}
	results := make(chan appendResult, 2)
	for _, sequence := range []int64{5, 6} {
		go func() {
			err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
				return store.Append(ctx, tx, orderedEvent(fmt.Sprintf("ordered-%d", sequence), "account-1", sequence))
			})
			results <- appendResult{sequence: sequence, err: err}
		}()
	}
	for range 2 {
		result := <-results
		if result.sequence == 6 && result.err != nil {
			t.Fatalf("highest concurrent sequence rejected: %v", result.err)
		}
		if result.sequence == 5 && result.err != nil && !errors.Is(result.err, postgresoutbox.ErrOrderingSequence) {
			t.Fatalf("lower concurrent sequence error = %v", result.err)
		}
	}
	if err := pool.PGX().QueryRow(ctx, `SELECT last_sequence FROM outbox_ordering_heads WHERE ordering_key = 'account-1'`).Scan(&highWater); err != nil {
		t.Fatalf("read concurrent ordering high-water: %v", err)
	}
	if highWater != 6 {
		t.Fatalf("concurrent ordering high-water = %d, want 6", highWater)
	}
}

func TestPostgresOutboxOrderingHandoffRace(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newOutboxFixture(t)
	const iterations = 24
	for iteration := range iterations {
		key := fmt.Sprintf("handoff-%02d", iteration)
		mustAppendOutbox(t, ctx, pool, store, orderedEvent(key+"-1", key, 1))
		first := mustClaimOutbox(t, ctx, store)

		start := make(chan struct{})
		errs := make(chan error, 2)
		go func() {
			<-start
			errs <- pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
				return store.Append(ctx, tx, orderedEvent(key+"-2", key, 2))
			})
		}()
		go func() {
			<-start
			errs <- markOutboxPublished(ctx, store, first)
		}()
		close(start)
		for range 2 {
			if err := <-errs; err != nil {
				t.Fatalf("iteration %d concurrent append/mark: %v", iteration, err)
			}
		}

		second := mustClaimOutbox(t, ctx, store)
		if second.Event.ID != key+"-2" {
			t.Fatalf("iteration %d next claim = %q, want %q", iteration, second.Event.ID, key+"-2")
		}
		if err := markOutboxPublished(ctx, store, second); err != nil {
			t.Fatalf("iteration %d mark successor: %v", iteration, err)
		}
	}
}

func TestPostgresOutboxOrderingHandoffAfterBlockedSnapshot(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newOutboxFixture(t)
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("snapshot-1", "snapshot", 1))
	first := mustClaimOutbox(t, ctx, store)

	tx, err := pool.PGX().Begin(ctx)
	if err != nil {
		t.Fatalf("begin append: %v", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	var blockerPID int
	if err := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&blockerPID); err != nil {
		t.Fatalf("read append backend PID: %v", err)
	}
	if err := store.Append(ctx, tx, orderedEvent("snapshot-2", "snapshot", 2)); err != nil {
		t.Fatalf("append successor: %v", err)
	}

	marked := make(chan error, 1)
	go func() { marked <- markOutboxPublished(ctx, store, first) }()
	waitForOutbox(t,
		func() string { return "the mark to wait for the ordering head lock" },
		func() bool {
			// Finishing before the append commits would mean the mark never took
			// the head lock, which is the opposite of what this proves.
			select {
			case err := <-marked:
				t.Fatalf("mark completed before append commit: %v", err)
			default:
			}
			return outboxBlockedBy(t, ctx, pool, blockerPID)
		})

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit successor: %v", err)
	}
	if err := <-marked; err != nil {
		if !errors.Is(err, postgresoutbox.ErrLeaseLost) {
			t.Fatalf("mark after blocked snapshot: %v", err)
		}
		if err := markOutboxPublished(ctx, store, first); err != nil {
			t.Fatalf("retry mark with fresh snapshot: %v", err)
		}
	}
	second := mustClaimOutbox(t, ctx, store)
	if second.Event.ID != "snapshot-2" {
		t.Fatalf("next claim = %q, want snapshot-2", second.Event.ID)
	}
}

func TestPostgresOutboxOrderedMarkFencesClaimIdentity(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newOutboxFixture(t)
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("identity-a", "identity-a", 1))
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("identity-b", "identity-b", 1))
	claim := mustClaimOutbox(t, ctx, store)
	if claim.Event.ID != "identity-a" {
		t.Fatalf("first claim = %q, want identity-a", claim.Event.ID)
	}

	mutated := claim
	mutated.Event.OrderingKey = "identity-b"
	if err := markOutboxPublished(ctx, store, mutated); !errors.Is(err, postgresoutbox.ErrLeaseLost) {
		t.Fatalf("mutated claim mark = %v, want ErrLeaseLost", err)
	}
	record, err := store.Get(ctx, claim.Event.ID)
	if err != nil {
		t.Fatalf("Get(%s): %v", claim.Event.ID, err)
	}
	if !record.PublishedAt.IsZero() {
		t.Fatal("mutated ordered claim marked the original event published")
	}
	if err := markOutboxPublished(ctx, store, claim); err != nil {
		t.Fatalf("mark original claim: %v", err)
	}
}

// One ordering key walks the whole set of states that can block its successor:
// a live lease, retry-wait, lease recovery behind a foreign row lock, and
// poison until redrive. Each phase below builds on the durable state the
// previous one left, so the comments mark where one ends and the next begins.
func TestPostgresOutboxOrderingClaims(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newOutboxFixture(t)
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("key-1", "key", 1))
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("key-2", "key", 2))
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("key-3", "key", 3))
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("unordered"))

	// Only the earliest unpublished sequence of a key is claimable, and an
	// unordered event is never blocked by one.
	first := mustClaimOutbox(t, ctx, store)
	if first.Event.ID != "key-1" {
		t.Fatalf("first claim = %q, want key-1", first.Event.ID)
	}
	second := mustClaimOutbox(t, ctx, store)
	if second.Event.ID != "unordered" {
		t.Fatalf("claim while key predecessor leased = %q, want unordered", second.Event.ID)
	}
	if err := markOutboxPublished(ctx, store, second); err != nil {
		t.Fatalf("publish unordered: %v", err)
	}
	if err := markOutboxPublished(ctx, store, first); err != nil {
		t.Fatalf("publish key-1: %v", err)
	}
	third := mustClaimOutbox(t, ctx, store)
	if third.Event.ID != "key-2" {
		t.Fatalf("post-predecessor claim = %q, want key-2", third.Event.ID)
	}
	// Retry-wait blocks the key just as a lease does: key-3 stays unclaimable
	// while key-2 waits out its backoff, and becomes claimable only after
	// key-2 itself is claimed again.
	if err := scheduleOutboxRetry(ctx, store, third.Event.ID, third.Token, "publisher_temporary", time.Hour); err != nil {
		t.Fatalf("schedule key-2 retry: %v", err)
	}
	if _, err := claimOutboxEvent(ctx, store, time.Minute); !errors.Is(err, errNoOutboxWork) {
		t.Fatalf("Claim() behind retry-wait predecessor = %v, want ErrNoWork", err)
	}
	if _, err := pool.PGX().Exec(ctx, "UPDATE outbox_events SET available_at = clock_timestamp() WHERE id = 'key-2'"); err != nil {
		t.Fatalf("make key-2 retry eligible: %v", err)
	}
	retryClaim, err := claimOutboxEvent(ctx, store, shortOutboxLease)
	if err != nil || retryClaim.Event.ID != "key-2" {
		t.Fatalf("retry claim = %+v, %v; want key-2", retryClaim, err)
	}
	// Lease recovery must reach a predecessor whose row another transaction
	// holds. The claim skips it while the lock is held rather than blocking or
	// jumping ahead to key-3, and picks it up once the lock is released. The
	// explicit FOR UPDATE is what makes that a proven wait rather than a
	// timing assumption.
	expireOutboxLease(t, ctx, pool)
	lockTx, err := pool.PGX().Begin(ctx)
	if err != nil {
		t.Fatalf("begin recovery predecessor lock: %v", err)
	}
	if _, err := lockTx.Exec(ctx, "SELECT id FROM outbox_events WHERE id = 'key-2' FOR UPDATE"); err != nil {
		_ = lockTx.Rollback(context.WithoutCancel(ctx))
		t.Fatalf("lock recovery predecessor: %v", err)
	}
	if _, err := claimOutboxEvent(ctx, store, time.Minute); !errors.Is(err, errNoOutboxWork) {
		_ = lockTx.Rollback(context.WithoutCancel(ctx))
		t.Fatalf("Claim() around locked recovery predecessor = %v, want ErrNoWork", err)
	}
	if err := lockTx.Rollback(ctx); err != nil {
		t.Fatalf("release recovery predecessor lock: %v", err)
	}
	recoveryClaim, err := claimOutboxEvent(ctx, store, time.Minute)
	if err != nil || recoveryClaim.Event.ID != "key-2" {
		t.Fatalf("recovery claim = %+v, %v; want predecessor key-2", recoveryClaim, err)
	}
	// The recovered claim fenced the earlier one: the stale token can no longer
	// poison the row, and only the current lease can. Poison then blocks the key
	// until an operator redrive re-admits it.
	if err := poisonOutboxEvent(ctx, store, retryClaim.Event.ID, retryClaim.Token, "publisher_permanent"); !errors.Is(err, postgresoutbox.ErrLeaseLost) {
		t.Fatalf("stale poison fence = %v, want ErrLeaseLost", err)
	}
	if err := poisonOutboxEvent(ctx, store, recoveryClaim.Event.ID, recoveryClaim.Token, "publisher_permanent"); err != nil {
		t.Fatalf("poison recovered key-2: %v", err)
	}
	if _, err := claimOutboxEvent(ctx, store, time.Minute); !errors.Is(err, errNoOutboxWork) {
		t.Fatalf("Claim() behind poison = %v, want ErrNoWork", err)
	}
	observation, err := store.Observe(ctx)
	if err != nil {
		t.Fatalf("Observe(): %v", err)
	}
	if observation.EligibleCount != 0 || observation.PoisonCount != 1 {
		t.Fatalf("observation eligible=%d poison=%d, want 0 and 1", observation.EligibleCount, observation.PoisonCount)
	}
	if err := store.Redrive(ctx, third.Event.ID, "redrive-ordering"); err != nil {
		t.Fatalf("redrive key-2: %v", err)
	}
	if claim := mustClaimOutbox(t, ctx, store); claim.Event.ID != "key-2" {
		t.Fatalf("claim after redrive = %q, want key-2", claim.Event.ID)
	}
}

// The claim query hands out at most one ready event per ordering key, which is
// what makes concurrent publication of a batch safe.
func TestPostgresOutboxBatchClaimHoldsOneEventPerOrderingKey(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newOutboxFixture(t)
	for sequence := int64(1); sequence <= 4; sequence++ {
		mustAppendOutbox(t, ctx, pool, store, orderedEvent(fmt.Sprintf("ordered-%d", sequence), "one-key", sequence))
	}
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("unordered"))

	batch, err := store.Claim(ctx, time.Minute, 100, 5)
	if err != nil {
		t.Fatalf("Claim(): %v", err)
	}
	keys := make(map[string]int, len(batch.Events))
	for _, claimed := range batch.Events {
		keys[claimed.Event.OrderingKey]++
	}
	if len(batch.Events) != 2 || keys["one-key"] != 1 || keys[""] != 1 {
		t.Fatalf("claimed %d events with keys %v, want the ordering head plus the unordered event", len(batch.Events), keys)
	}
}

// One statement finalizes every ordered acknowledgement a lease holds: each
// key's head advances to its own successor and that successor becomes
// claimable, so a batch of ordered events costs one round trip rather than one
// per event.
func TestPostgresOutboxOrderedBatchFinalizationAdvancesEveryKey(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newOutboxFixture(t)
	const keys = 3
	for key := 1; key <= keys; key++ {
		for sequence := int64(1); sequence <= 2; sequence++ {
			mustAppendOutbox(t, ctx, pool, store, orderedEvent(
				fmt.Sprintf("key-%d-seq-%d", key, sequence), fmt.Sprintf("key-%d", key), sequence))
		}
	}

	batch, err := store.Claim(ctx, time.Minute, 100, 5)
	if err != nil {
		t.Fatalf("Claim(): %v", err)
	}
	if len(batch.Events) != keys {
		t.Fatalf("claimed %d events, want one ordering head per key", len(batch.Events))
	}
	directives := make([]postgresoutbox.OrderedDirective, len(batch.Events))
	for index, claimed := range batch.Events {
		directives[index] = postgresoutbox.OrderedDirective{
			ID:               claimed.Event.ID,
			OrderingKey:      claimed.Event.OrderingKey,
			OrderingSequence: claimed.Event.OrderingSequence,
		}
	}
	marked, err := store.MarkOrderedPublishedBatch(ctx, batch.Token, directives)
	if err != nil || len(marked) != keys {
		t.Fatalf("MarkOrderedPublishedBatch() = %v, %v, want %d finalized", marked, err, keys)
	}

	successors, err := store.Claim(ctx, time.Minute, 100, 5)
	if err != nil {
		t.Fatalf("Claim(successors): %v", err)
	}
	sequences := make(map[string]int64, len(successors.Events))
	for _, claimed := range successors.Events {
		sequences[claimed.Event.OrderingKey] = claimed.Event.OrderingSequence
	}
	if len(successors.Events) != keys {
		t.Fatalf("claimed %d successors %v, want one per key", len(successors.Events), sequences)
	}
	for key := 1; key <= keys; key++ {
		if sequences[fmt.Sprintf("key-%d", key)] != 2 {
			t.Fatalf("successor sequences = %v, want sequence 2 for every key", sequences)
		}
	}
}

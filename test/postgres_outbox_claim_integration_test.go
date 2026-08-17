//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
)

func TestPostgresOutboxConcurrentClaims(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newOutboxFixture(t)
	const eventCount = 24
	for i := range eventCount {
		mustAppendOutbox(t, ctx, pool, store, outboxEvent(fmt.Sprintf("concurrent-%02d", i)))
	}

	start := make(chan struct{})
	results := make(chan outboxClaim, eventCount)
	errs := make(chan error, eventCount)
	var workers sync.WaitGroup
	for range eventCount {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			claim, err := claimOutboxEvent(ctx, store, time.Minute)
			if err != nil {
				errs <- err
				return
			}
			results <- claim
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Errorf("concurrent Claim(): %v", err)
	}
	seen := make(map[string]struct{}, eventCount)
	for claim := range results {
		if _, duplicate := seen[claim.Event.ID]; duplicate {
			t.Errorf("event %q claimed twice", claim.Event.ID)
		}
		seen[claim.Event.ID] = struct{}{}
	}
	if len(seen) != eventCount {
		t.Fatalf("unique claims = %d, want %d", len(seen), eventCount)
	}

	mustAppendOutbox(t, ctx, pool, store, outboxEvent("locked-first"))
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("locked-second"))
	tx, err := pool.PGX().Begin(ctx)
	if err != nil {
		t.Fatalf("begin lock holder: %v", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err := tx.Exec(ctx, "SELECT id FROM outbox_events WHERE id = 'locked-first' FOR UPDATE"); err != nil {
		t.Fatalf("lock first event: %v", err)
	}
	claim, err := claimOutboxEvent(ctx, store, time.Minute)
	if err != nil {
		t.Fatalf("Claim() around held lock: %v", err)
	}
	if claim.Event.ID != "locked-second" {
		t.Fatalf("held-lock claim = %q, want locked-second", claim.Event.ID)
	}
}

func TestPostgresOutboxLeaseExpiryAndFence(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newOutboxFixture(t)
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("lease"))
	first, err := claimOutboxEvent(ctx, store, shortOutboxLease)
	if err != nil {
		t.Fatalf("first Claim(): %v", err)
	}
	expireOutboxLease(t, ctx, pool)
	second, err := claimOutboxEvent(ctx, store, time.Minute)
	if err != nil {
		t.Fatalf("recovery Claim(): %v", err)
	}
	if second.Event.ID != first.Event.ID || second.Token == first.Token {
		t.Fatalf("recovery claim id/token = %q/%q, first %q/%q", second.Event.ID, second.Token, first.Event.ID, first.Token)
	}
	if second.CycleAttemptCount != 2 || second.TotalAttemptCount != 2 {
		t.Fatalf("recovery attempts = %d/%d, want 2/2", second.CycleAttemptCount, second.TotalAttemptCount)
	}
	if err := markOutboxPublished(ctx, store, first); !errors.Is(err, postgresoutbox.ErrLeaseLost) {
		t.Fatalf("stale MarkPublished() = %v, want ErrLeaseLost", err)
	}
	if err := scheduleOutboxRetry(ctx, store, first.Event.ID, first.Token, "publisher_temporary", 0); !errors.Is(err, postgresoutbox.ErrLeaseLost) {
		t.Fatalf("stale ScheduleRetry() = %v, want ErrLeaseLost", err)
	}
	if err := poisonOutboxEvent(ctx, store, first.Event.ID, first.Token, "publisher_permanent"); !errors.Is(err, postgresoutbox.ErrLeaseLost) {
		t.Fatalf("stale MarkPoisoned() = %v, want ErrLeaseLost", err)
	}
	if err := markOutboxPublished(ctx, store, second); err != nil {
		t.Fatalf("current MarkPublished(): %v", err)
	}
}

func TestPostgresOutboxCrashAfterClaim(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newOutboxFixture(t)
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("abandoned"))
	first, err := claimOutboxEvent(ctx, store, shortOutboxLease)
	if err != nil {
		t.Fatalf("Claim(): %v", err)
	}
	store = nil
	expireOutboxLease(t, ctx, pool)
	restarted, err := postgresoutbox.NewStore(pool, nil)
	if err != nil {
		t.Fatalf("NewStore() after abandoned claim: %v", err)
	}
	second, err := claimOutboxEvent(ctx, restarted, time.Minute)
	if err != nil {
		t.Fatalf("Claim() after abandoned claim: %v", err)
	}
	if second.Event.ID != first.Event.ID || second.Token == first.Token {
		t.Fatalf("restarted claim id/token = %q/%q, first %q/%q", second.Event.ID, second.Token, first.Event.ID, first.Token)
	}
}

// Redrive's SELECT ... FOR UPDATE and its read-then-write of the audit ledger
// exist for this case, not for the sequential one: two operators submitting the
// same audit id at once. The loser blocks on the row lock, then re-reads the
// ledger on a snapshot that can see the winner's insert, and returns nil rather
// than starting a second delivery cycle.
//
// Sequential idempotence cannot catch a regression here — a Redrive that dropped
// the lock would still pass it, because the second call reads a committed row.
func TestPostgresOutboxConcurrentRedriveIsIdempotent(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newOutboxFixture(t)
	const racers = 4
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("race-poison"))
	claim := mustClaimOutbox(t, ctx, store)
	if err := poisonOutboxEvent(ctx, store, claim.Event.ID, claim.Token, "publisher_permanent"); err != nil {
		t.Fatalf("MarkPoisoned(): %v", err)
	}

	start := make(chan struct{})
	errs := make(chan error, racers)
	for range racers {
		go func() {
			<-start
			errs <- store.Redrive(ctx, claim.Event.ID, "race-audit")
		}()
	}
	close(start)
	for range racers {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent Redrive(): %v", err)
		}
	}

	record, err := store.Get(ctx, claim.Event.ID)
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if record.RedriveCount != 1 || record.LastRedriveID != "race-audit" || !record.PoisonedAt.IsZero() {
		t.Fatalf("after %d concurrent redrives count=%d audit=%q poisoned=%v, want one cycle",
			racers, record.RedriveCount, record.LastRedriveID, record.PoisonedAt)
	}
	var ledgerRows int
	if err := pool.PGX().QueryRow(ctx,
		"SELECT count(*) FROM outbox_redrives WHERE event_id = $1", claim.Event.ID).Scan(&ledgerRows); err != nil {
		t.Fatalf("count redrive ledger: %v", err)
	}
	if ledgerRows != 1 {
		t.Fatalf("redrive ledger rows = %d, want 1", ledgerRows)
	}
}

func TestPostgresOutboxRedrive(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newOutboxFixture(t)
	event := outboxEvent("poison")
	event.Payload = []byte(" {\n \"redrive\" : true\n} ")
	event.Metadata = []byte(`{"b":2,"a":1}`)
	mustAppendOutbox(t, ctx, pool, store, event)
	if err := store.Redrive(ctx, event.ID, "pending-audit"); !errors.Is(err, postgresoutbox.ErrOperatorStateConflict) {
		t.Fatalf("pending Redrive() = %v, want ErrOperatorStateConflict", err)
	}
	claim := mustClaimOutbox(t, ctx, store)
	if err := poisonOutboxEvent(ctx, store, claim.Event.ID, claim.Token, "publisher_permanent"); err != nil {
		t.Fatalf("MarkPoisoned(): %v", err)
	}
	if err := store.Redrive(ctx, claim.Event.ID, "audit-1"); err != nil {
		t.Fatalf("Redrive(): %v", err)
	}
	if err := store.Redrive(ctx, claim.Event.ID, "audit-1"); err != nil {
		t.Fatalf("idempotent Redrive(): %v", err)
	}
	record, err := store.Get(ctx, claim.Event.ID)
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if record.RedriveCount != 1 || record.CycleAttemptCount != 0 || !record.PoisonedAt.IsZero() {
		t.Fatalf("redriven record = count %d cycle %d poisoned %v", record.RedriveCount, record.CycleAttemptCount, record.PoisonedAt)
	}
	if !bytes.Equal(record.Event.Payload, event.Payload) || !bytes.Equal(record.Event.Metadata, event.Metadata) {
		t.Fatalf("redrive changed exact envelope bytes: payload %q metadata %q", record.Event.Payload, record.Event.Metadata)
	}

	claim = mustClaimOutbox(t, ctx, store)
	if claim.CycleAttemptCount != 1 || claim.TotalAttemptCount != 2 {
		t.Fatalf("post-redrive attempts = %d/%d, want 1/2", claim.CycleAttemptCount, claim.TotalAttemptCount)
	}
	if err := poisonOutboxEvent(ctx, store, claim.Event.ID, claim.Token, "publisher_permanent"); err != nil {
		t.Fatalf("second MarkPoisoned(): %v", err)
	}
	if err := store.Redrive(ctx, claim.Event.ID, "audit-2"); err != nil {
		t.Fatalf("second Redrive(): %v", err)
	}

	mustAppendOutbox(t, ctx, pool, store, outboxEvent("other-poison"))
	other := mustClaimOutbox(t, ctx, store)
	if other.Event.ID != "poison" {
		t.Fatalf("next claim = %q, want redriven poison", other.Event.ID)
	}
	if err := markOutboxPublished(ctx, store, other); err != nil {
		t.Fatalf("publish redriven event: %v", err)
	}
	if err := store.Redrive(ctx, other.Event.ID, "audit-3"); !errors.Is(err, postgresoutbox.ErrOperatorStateConflict) {
		t.Fatalf("redrive published event = %v, want ErrOperatorStateConflict", err)
	}
	other = mustClaimOutbox(t, ctx, store)
	if err := poisonOutboxEvent(ctx, store, other.Event.ID, other.Token, "publisher_permanent"); err != nil {
		t.Fatalf("poison other event: %v", err)
	}
	if err := store.Redrive(ctx, other.Event.ID, "audit-2"); !errors.Is(err, postgresoutbox.ErrOperatorAuditConflict) {
		t.Fatalf("cross-event audit reuse = %v, want ErrOperatorAuditConflict", err)
	}
	if _, err := pool.PGX().Exec(ctx, "UPDATE outbox_events SET published_at = clock_timestamp() - interval '2 hours' WHERE id = 'poison'"); err != nil {
		t.Fatalf("backdate redriven published event: %v", err)
	}
	if deleted, err := store.CleanupPublished(ctx, time.Hour, 1); err != nil || deleted != 1 {
		t.Fatalf("cleanup redriven event = %d, %v; want 1, nil", deleted, err)
	}
	var auditRows int
	if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM outbox_redrives WHERE event_id = 'poison'").Scan(&auditRows); err != nil {
		t.Fatalf("count cascaded redrive rows: %v", err)
	}
	if auditRows != 2 {
		t.Fatalf("redrive audit rows after event cleanup = %d, want 2 retained audit records", auditRows)
	}
}

func TestPostgresOutboxConcurrentUnknownActions(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newOutboxFixture(t)
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("unknown-race"))
	claim := mustClaimOutbox(t, ctx, store)
	poisonOutcomeUnknown(t, ctx, store, claim)
	lockTx, err := pool.PGX().Begin(ctx)
	if err != nil {
		t.Fatalf("begin operator action gate: %v", err)
	}
	t.Cleanup(func() { _ = lockTx.Rollback(context.Background()) })
	var blockerPID int
	if err := lockTx.QueryRow(ctx, `
		SELECT pg_backend_pid() FROM outbox_events WHERE id = $1 FOR UPDATE
	`, claim.Event.ID).Scan(&blockerPID); err != nil {
		t.Fatalf("lock unknown event: %v", err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	go func() {
		<-start
		errs <- store.RedriveUnknown(ctx, claim.Event.ID, "race-redrive")
	}()
	go func() {
		<-start
		errs <- store.ConfirmAccepted(ctx, claim.Event.ID, "race-confirm")
	}()
	close(start)
	waitForOutbox(t,
		func() string { return "both concurrent unknown actions to wait for the event lock" },
		func() bool {
			return outboxBackendCount(t, ctx, pool,
				"wait_event_type = 'Lock' AND query LIKE '%name: LockOutboxEventForAction :one%'") == 2
		},
	)
	if err := lockTx.Rollback(ctx); err != nil {
		t.Fatalf("release operator action gate: %v", err)
	}

	succeeded := 0
	conflicted := 0
	for range 2 {
		err := <-errs
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, postgresoutbox.ErrOperatorStateConflict),
			errors.Is(err, postgresoutbox.ErrOperatorAuditConflict):
			conflicted++
		default:
			t.Fatalf("operator race error = %v", err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("operator race success/conflict = %d/%d, want 1/1", succeeded, conflicted)
	}
	var audits int
	if err := pool.PGX().QueryRow(ctx,
		"SELECT count(*) FROM outbox_redrives WHERE event_id = $1", claim.Event.ID,
	).Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("operator race audits = %d, %v; want 1", audits, err)
	}

	mustAppendOutbox(t, ctx, pool, store, outboxEvent("unknown-race-replay"))
	replayClaim := mustClaimOutbox(t, ctx, store)
	if replayClaim.Event.ID != "unknown-race-replay" {
		if err := markOutboxPublished(ctx, store, replayClaim); err != nil {
			t.Fatalf("finish winning redrive before replay race: %v", err)
		}
		replayClaim = mustClaimOutbox(t, ctx, store)
	}
	if replayClaim.Event.ID != "unknown-race-replay" {
		t.Fatalf("replay race claim = %q, want unknown-race-replay", replayClaim.Event.ID)
	}
	poisonOutcomeUnknown(t, ctx, store, replayClaim)
	const replayRacers = 4
	replayStart := make(chan struct{})
	replayErrors := make(chan error, replayRacers)
	for range replayRacers {
		go func() {
			<-replayStart
			replayErrors <- store.RedriveUnknown(ctx, replayClaim.Event.ID, "same-unknown-action")
		}()
	}
	close(replayStart)
	for range replayRacers {
		if err := <-replayErrors; err != nil {
			t.Fatalf("same audited action race: %v", err)
		}
	}
	replayed, err := store.Get(ctx, replayClaim.Event.ID)
	if err != nil || replayed.RedriveCount != 1 {
		t.Fatalf("same audited action replay = %+v, %v", replayed, err)
	}
}

//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresOutboxAuditedUnknownRecovery(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("operator-ready"))
	if err := store.RedriveUnknown(ctx, "operator-ready", "invalid-ready-redrive"); !errors.Is(err, postgresoutbox.ErrOperatorStateConflict) {
		t.Fatalf("RedriveUnknown(ready) = %v, want state conflict", err)
	}
	if err := store.ConfirmAccepted(ctx, "operator-ready", "invalid-ready-confirm"); !errors.Is(err, postgresoutbox.ErrOperatorStateConflict) {
		t.Fatalf("ConfirmAccepted(ready) = %v, want state conflict", err)
	}
	if err := store.ConfirmAccepted(ctx, "missing-operator-event", "missing-confirm"); !errors.Is(err, postgresoutbox.ErrNotFound) {
		t.Fatalf("ConfirmAccepted(missing) = %v, want not found", err)
	}
	ready := mustClaimOutbox(t, ctx, store)
	if err := poisonOutboxEvent(ctx, store, ready.Event.ID, ready.Token, "publisher_permanent"); err != nil {
		t.Fatalf("mark deterministic operator poison: %v", err)
	}
	if err := store.RedriveUnknown(ctx, ready.Event.ID, "invalid-poison-redrive"); !errors.Is(err, postgresoutbox.ErrOperatorStateConflict) {
		t.Fatalf("RedriveUnknown(deterministic poison) = %v, want state conflict", err)
	}
	if err := store.Redrive(ctx, ready.Event.ID, "deterministic-redrive"); err != nil {
		t.Fatalf("Redrive(deterministic poison): %v", err)
	}
	ready = mustClaimOutbox(t, ctx, store)
	if err := markOutboxPublished(ctx, store, ready); err != nil {
		t.Fatalf("publish redriven deterministic event: %v", err)
	}

	mustAppendOutbox(t, ctx, pool, store, outboxEvent("unknown-recovery"))
	claim := mustClaimOutbox(t, ctx, store)
	poisonOutcomeUnknown(t, ctx, store, claim)

	if err := store.RedriveUnknown(ctx, claim.Event.ID, "unknown-redrive"); err != nil {
		t.Fatalf("RedriveUnknown(): %v", err)
	}
	if err := store.RedriveUnknown(ctx, claim.Event.ID, "unknown-redrive"); err != nil {
		t.Fatalf("RedriveUnknown(replay): %v", err)
	}
	if err := store.ConfirmAccepted(ctx, claim.Event.ID, "unknown-redrive"); !errors.Is(err, postgresoutbox.ErrOperatorAuditConflict) {
		t.Fatalf("cross-action audit reuse = %v, want audit conflict", err)
	}
	record, err := store.Get(ctx, claim.Event.ID)
	if err != nil || record.RedriveCount != 1 || record.TotalAttemptCount != 1 ||
		!record.PublicationUncertain || !record.PoisonedAt.IsZero() {
		t.Fatalf("redriven unknown = %+v, %v", record, err)
	}
	claim = mustClaimOutbox(t, ctx, store)
	poisonOutcomeUnknown(t, ctx, store, claim)
	if err := store.ConfirmAccepted(ctx, claim.Event.ID, "unknown-confirm"); err != nil {
		t.Fatalf("ConfirmAccepted(): %v", err)
	}
	if err := store.RedriveUnknown(ctx, claim.Event.ID, "unknown-confirm"); !errors.Is(err, postgresoutbox.ErrOperatorAuditConflict) {
		t.Fatalf("confirmation audit reused for redrive = %v, want audit conflict", err)
	}
	if err := store.ConfirmAccepted(ctx, claim.Event.ID, "unknown-confirm"); err != nil {
		t.Fatalf("ConfirmAccepted(replay): %v", err)
	}
	record, err = store.Get(ctx, claim.Event.ID)
	if err != nil || record.PublishedAt.IsZero() || !record.PublicationUncertain || !record.PoisonedAt.IsZero() {
		t.Fatalf("confirmed unknown = %+v, %v", record, err)
	}

	if _, err := pool.Exec(ctx,
		"UPDATE outbox_events SET published_at = clock_timestamp() - interval '2 hours' WHERE id = $1",
		claim.Event.ID,
	); err != nil {
		t.Fatalf("age confirmed event: %v", err)
	}
	if deleted, err := store.CleanupPublished(ctx, time.Hour, 1); err != nil || deleted != 1 {
		t.Fatalf("CleanupPublished() = %d, %v", deleted, err)
	}
	if err := store.ConfirmAccepted(ctx, claim.Event.ID, "unknown-confirm"); err != nil {
		t.Fatalf("ConfirmAccepted(replay after cleanup): %v", err)
	}

	mustAppendOutbox(t, ctx, pool, store, orderedEvent("unknown-ordered-1", "unknown-order", 1))
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("unknown-ordered-2", "unknown-order", 2))
	ordered := mustClaimOutbox(t, ctx, store)
	poisonOutcomeUnknown(t, ctx, store, ordered)
	if err := store.ConfirmAccepted(ctx, ordered.Event.ID, "ordered-confirm"); err != nil {
		t.Fatalf("ConfirmAccepted(ordered): %v", err)
	}
	if successor := mustClaimOutbox(t, ctx, store); successor.Event.ID != "unknown-ordered-2" {
		t.Fatalf("ordered successor = %q, want unknown-ordered-2", successor.Event.ID)
	}

	var action, eventID string
	var cycle *int32
	if err := pool.QueryRow(ctx, `
		SELECT action_kind, event_id, cycle_number
		FROM outbox_redrives WHERE audit_id = 'unknown-confirm'
	`).Scan(&action, &eventID, &cycle); err != nil {
		t.Fatalf("read retained confirmation audit: %v", err)
	}
	if action != "confirm_accepted" || eventID != "unknown-recovery" || cycle != nil {
		t.Fatalf("confirmation audit = %q/%q/%v", action, eventID, cycle)
	}
}

func TestPostgresOutboxRetryAndPoison(t *testing.T) {
	t.Run("exhaustion", func(t *testing.T) {
		ctx, pool, store := newOutboxFixture(t)
		mustAppendOutbox(t, ctx, pool, store, outboxEvent("exhaustion"))
		var attempts atomic.Int64
		publisher := testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
			attempts.Add(1)
			return fmt.Errorf("broker rejected: %w", postgresoutbox.ErrPublicationNotAccepted)
		})
		config := testRelayConfig()
		config.MaxAttempts = 3
		config.RetryBase = time.Nanosecond
		config.RetryMax = time.Nanosecond
		relay := mustNewOutboxRelay(t, store, publisher, nil, config)
		result := runOutboxRelay(ctx, relay)
		waitForOutboxCount(t, ctx, pool, "poisoned_at IS NOT NULL", 1)
		relay.StartDrain()
		assertRelayResult(t, result, nil)
		record, err := store.Get(ctx, "exhaustion")
		if err != nil {
			t.Fatalf("Get(): %v", err)
		}
		if attempts.Load() != 3 || record.CycleAttemptCount != 3 || record.LastErrorClass != "attempt_exhausted" {
			t.Fatalf("attempts publisher/db/class = %d/%d/%q, want 3/3/attempt_exhausted", attempts.Load(), record.CycleAttemptCount, record.LastErrorClass)
		}
	})
}

func TestPostgresOutboxStickyAtLimitQuarantinesWithoutPublish(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("ambiguous-limit"))
	config := testRelayConfig()
	config.MaxAttempts = 2
	config.RetryBase = time.Nanosecond
	config.RetryMax = time.Nanosecond

	var attempts atomic.Int64
	relay := mustNewOutboxRelay(t, store, testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
		attempts.Add(1)
		return errors.New("ambiguous publication")
	}), nil, config)
	result := runOutboxRelay(ctx, relay)
	waitForOutboxCount(t, ctx, pool, "last_error_class = 'outcome_unknown'", 1)
	relay.StartDrain()
	assertRelayResult(t, result, nil)

	record, err := store.Get(ctx, "ambiguous-limit")
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if attempts.Load() != 2 || record.CycleAttemptCount != 2 ||
		!record.PublicationUncertain || record.PoisonedAt.IsZero() || record.LastErrorClass != "outcome_unknown" {
		t.Fatalf("unknown state attempts=%d cycle=%d sticky=%t poisoned=%v class=%q",
			attempts.Load(), record.CycleAttemptCount, record.PublicationUncertain,
			record.PoisonedAt, record.LastErrorClass)
	}
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("preclaim-limit"))
	if _, err := pool.Exec(ctx, `
		UPDATE outbox_events
		SET cycle_attempt_count = 2, total_attempt_count = 2,
			publication_uncertain = true, last_error_class = 'publisher_temporary'
		WHERE id = 'preclaim-limit'
	`); err != nil {
		t.Fatalf("seed sticky at-limit row: %v", err)
	}
	batch, err := store.Claim(ctx, time.Minute, 1, config.MaxAttempts)
	if err != nil || len(batch.Events) != 0 {
		t.Fatalf("Claim(at limit) = %+v, %v; want no republish", batch, err)
	}
	observation, err := store.Observe(ctx)
	if err != nil || observation.OutcomeUnknownCount != 2 || observation.PoisonCount != 0 {
		t.Fatalf("Observe() = %+v, %v; want two outcome_unknown and no deterministic poison", observation, err)
	}
	if deleted, err := store.CleanupPublished(ctx, time.Nanosecond, 10); err != nil || deleted != 0 {
		t.Fatalf("CleanupPublished(unknown) = %d, %v; want retained", deleted, err)
	}
	restarted, err := postgresoutbox.NewStore(pool, nil)
	if err != nil {
		t.Fatalf("NewStore(restart): %v", err)
	}
	mustAppendOutbox(t, ctx, pool, restarted, outboxEvent("crash-at-limit"))
	crashed, err := restarted.Claim(ctx, shortOutboxLease, 1, 1)
	if err != nil || len(crashed.Events) != 1 || crashed.Events[0].PublicationUncertain {
		t.Fatalf("initial crash claim = %+v, %v", crashed, err)
	}
	expireOutboxLease(t, ctx, pool)
	restarted, err = postgresoutbox.NewStore(pool, nil)
	if err != nil {
		t.Fatalf("NewStore(after crash): %v", err)
	}
	recovered, err := restarted.Claim(ctx, time.Minute, 1, 1)
	if err != nil || len(recovered.Events) != 0 {
		t.Fatalf("recovery claim at limit = %+v, %v; want quarantine without publish", recovered, err)
	}
	crashRecord, err := restarted.Get(ctx, "crash-at-limit")
	if err != nil || !crashRecord.PublicationUncertain || crashRecord.PoisonedAt.IsZero() ||
		crashRecord.LastErrorClass != "outcome_unknown" {
		t.Fatalf("crash recovery state = %+v, %v", crashRecord, err)
	}

	mustAppendOutbox(t, ctx, pool, restarted, orderedEvent("sticky-ordered-1", "sticky-order", 1))
	mustAppendOutbox(t, ctx, pool, restarted, orderedEvent("sticky-ordered-2", "sticky-order", 2))
	mustAppendOutbox(t, ctx, pool, restarted, outboxEvent("sticky-unrelated"))
	var callsMu sync.Mutex
	calls := map[string]int{}
	publisher := testPublisherFunc(func(_ context.Context, event postgresoutbox.Event) error {
		callsMu.Lock()
		calls[event.ID]++
		callsMu.Unlock()
		if event.ID == "sticky-ordered-1" {
			return errors.New("ambiguous ordered publication")
		}
		return nil
	})
	relay = mustNewOutboxRelay(t, restarted, publisher, nil, config)
	result = runOutboxRelay(ctx, relay)
	waitForOutboxCount(t, ctx, pool,
		"id = 'sticky-ordered-1' AND last_error_class = 'outcome_unknown'", 1)
	waitForOutboxCount(t, ctx, pool,
		"id = 'sticky-unrelated' AND published_at IS NOT NULL", 1)
	relay.StartDrain()
	assertRelayResult(t, result, nil)
	callsMu.Lock()
	orderedCalls, successorCalls, unrelatedCalls := calls["sticky-ordered-1"], calls["sticky-ordered-2"], calls["sticky-unrelated"]
	callsMu.Unlock()
	if orderedCalls != 2 || successorCalls != 0 || unrelatedCalls != 1 {
		t.Fatalf("ordered/successor/unrelated calls = %d/%d/%d, want 2/0/1",
			orderedCalls, successorCalls, unrelatedCalls)
	}

	mustAppendOutbox(t, ctx, pool, restarted, orderedEvent("sticky-ack-1", "sticky-ack", 1))
	mustAppendOutbox(t, ctx, pool, restarted, orderedEvent("sticky-ack-2", "sticky-ack", 2))
	ackAttempts := atomic.Int64{}
	relay = mustNewOutboxRelay(t, restarted, testPublisherFunc(func(_ context.Context, event postgresoutbox.Event) error {
		if event.ID == "sticky-ack-1" && ackAttempts.Add(1) == 1 {
			return errors.New("ambiguous before acknowledgement")
		}
		return nil
	}), nil, config)
	result = runOutboxRelay(ctx, relay)
	waitForOutboxCount(t, ctx, pool,
		"id IN ('sticky-ack-1', 'sticky-ack-2') AND published_at IS NOT NULL", 2)
	relay.StartDrain()
	assertRelayResult(t, result, nil)
	ack, err := restarted.Get(ctx, "sticky-ack-1")
	if err != nil || !ack.PublicationUncertain || ack.PublishedAt.IsZero() {
		t.Fatalf("ack after uncertainty = %+v, %v", ack, err)
	}
}

func TestPostgresOutboxLegacyUncertaintyClassification(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	for _, id := range []string{
		"legacy-unattempted", "legacy-published", "legacy-retry",
		"legacy-leased", "legacy-poison", "legacy-at-limit",
	} {
		mustAppendOutbox(t, ctx, pool, store, outboxEvent(id))
	}
	if _, err := pool.Exec(ctx, `
		UPDATE outbox_events SET publication_uncertain = NULL;
		UPDATE outbox_events SET published_at = clock_timestamp(), cycle_attempt_count = 1,
			total_attempt_count = 1 WHERE id = 'legacy-published';
		UPDATE outbox_events SET cycle_attempt_count = 1, total_attempt_count = 1,
			available_at = clock_timestamp() + interval '1 hour', last_error_class = 'publisher_temporary'
			WHERE id = 'legacy-retry';
		UPDATE outbox_events SET cycle_attempt_count = 1, total_attempt_count = 1,
			lease_token = 'legacy-lease', lease_expires_at = clock_timestamp() + interval '1 hour'
			WHERE id = 'legacy-leased';
		UPDATE outbox_events SET cycle_attempt_count = 1, total_attempt_count = 1,
			poisoned_at = clock_timestamp(), last_error_class = 'publisher_permanent'
			WHERE id = 'legacy-poison';
		UPDATE outbox_events SET cycle_attempt_count = 3, total_attempt_count = 3,
			last_error_class = 'publisher_temporary' WHERE id = 'legacy-at-limit'
	`); err != nil {
		t.Fatalf("prepare legacy rows: %v", err)
	}
	var receiptsBefore, headsBefore int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM outbox_commit_receipts),
		(SELECT count(*) FROM outbox_ordering_heads)
	`).Scan(&receiptsBefore, &headsBefore); err != nil {
		t.Fatalf("read classification authority counts: %v", err)
	}

	var batches []int
	for {
		classified, err := store.ClassifyLegacyUncertainty(ctx, 3, 2)
		if err != nil {
			t.Fatalf("ClassifyLegacyUncertainty(): %v", err)
		}
		batches = append(batches, classified)
		if classified == 0 {
			break
		}
	}
	if !slices.Equal(batches, []int{2, 2, 2, 0}) {
		t.Fatalf("classification batches = %v, want [2 2 2 0]", batches)
	}
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("legacy-locked"))
	if _, err := pool.Exec(ctx, `
		UPDATE outbox_events SET publication_uncertain = NULL,
			cycle_attempt_count = 1, total_attempt_count = 1
		WHERE id = 'legacy-locked'
	`); err != nil {
		t.Fatalf("prepare locked legacy row: %v", err)
	}
	lockTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin legacy classification lock: %v", err)
	}
	t.Cleanup(func() { _ = lockTx.Rollback(context.Background()) })
	var blockerPID int
	if err := lockTx.QueryRow(ctx, `
		SELECT pg_backend_pid() FROM outbox_events
		WHERE id = 'legacy-locked' FOR UPDATE
	`).Scan(&blockerPID); err != nil {
		t.Fatalf("lock legacy classification row: %v", err)
	}
	classifyCtx, cancelClassification := context.WithCancel(ctx)
	classificationResult := make(chan error, 1)
	go func() {
		_, err := store.ClassifyLegacyUncertainty(classifyCtx, 3, 2)
		classificationResult <- err
	}()
	waitForOutbox(t,
		func() string { return "legacy classification to wait for the locked candidate" },
		func() bool { return outboxBlockedBy(t, ctx, pool, blockerPID) },
	)
	cancelClassification()
	if err := <-classificationResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("locked classification error = %v, want cancellation instead of false zero", err)
	} else if pgErr, ok := errors.AsType[*pgconn.PgError](err); !ok || pgErr.Code != "57014" {
		t.Fatalf("locked classification error = %v, want caller cancellation with SQLSTATE 57014", err)
	}
	if err := lockTx.Rollback(ctx); err != nil {
		t.Fatalf("release legacy classification lock: %v", err)
	}
	serverLock, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin server-cancelled classification lock: %v", err)
	}
	t.Cleanup(func() { _ = serverLock.Rollback(context.Background()) })
	var serverBlockerPID int
	if err := serverLock.QueryRow(ctx, `SELECT pg_backend_pid() FROM outbox_events WHERE id = 'legacy-locked' FOR UPDATE`).Scan(&serverBlockerPID); err != nil {
		t.Fatalf("lock server-cancelled legacy row: %v", err)
	}
	serverResult := make(chan error, 1)
	go func() {
		_, err := store.ClassifyLegacyUncertainty(ctx, 3, 2)
		serverResult <- err
	}()
	serverPID := outboxBlockedPID(t, ctx, pool, serverBlockerPID)
	if err := pool.QueryRow(ctx, `SELECT pg_cancel_backend($1)`, serverPID).Scan(new(bool)); err != nil {
		t.Fatalf("cancel classification backend: %v", err)
	}
	if err := <-serverResult; errors.Is(err, context.Canceled) {
		t.Fatalf("server-cancelled classification error = %v, did not want caller cancellation", err)
	} else if pgErr, ok := errors.AsType[*pgconn.PgError](err); !ok || pgErr.Code != "57014" {
		t.Fatalf("server-cancelled classification error = %v, want SQLSTATE 57014", err)
	}
	if err := serverLock.Rollback(ctx); err != nil {
		t.Fatalf("release server-cancelled legacy lock: %v", err)
	}
	if classified, err := store.ClassifyLegacyUncertainty(ctx, 3, 2); err != nil || classified < 0 || classified > 1 {
		t.Fatalf("resume classification = %d, %v; want zero or one monotonic row", classified, err)
	}
	if classified, err := store.ClassifyLegacyUncertainty(ctx, 3, 2); err != nil || classified != 0 {
		t.Fatalf("authoritative final classification = %d, %v; want zero", classified, err)
	}
	var receiptsAfter, headsAfter int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM outbox_commit_receipts),
		(SELECT count(*) FROM outbox_ordering_heads)
	`).Scan(&receiptsAfter, &headsAfter); err != nil {
		t.Fatalf("read post-classification authority counts: %v", err)
	}
	if receiptsAfter != receiptsBefore+1 || headsAfter != headsBefore {
		t.Fatalf("classification changed receipts/heads = %d/%d, want append-only %d/%d",
			receiptsAfter, headsAfter, receiptsBefore+1, headsBefore)
	}

	for _, test := range []struct {
		id        string
		uncertain bool
		unknown   bool
		leased    bool
	}{
		{id: "legacy-unattempted"},
		{id: "legacy-published"},
		{id: "legacy-retry", uncertain: true},
		{id: "legacy-leased", uncertain: true, leased: true},
		{id: "legacy-poison", uncertain: true, unknown: true},
		{id: "legacy-at-limit", uncertain: true, unknown: true},
	} {
		record, err := store.Get(ctx, test.id)
		if err != nil {
			t.Fatalf("Get(%s): %v", test.id, err)
		}
		if record.PublicationUncertain != test.uncertain || (!record.LeaseExpiresAt.IsZero()) != test.leased ||
			(record.LastErrorClass == "outcome_unknown") != test.unknown || (!record.PoisonedAt.IsZero()) != test.unknown {
			t.Errorf("classified %s = sticky=%t leased=%t poisoned=%t class=%q",
				test.id, record.PublicationUncertain, !record.LeaseExpiresAt.IsZero(),
				!record.PoisonedAt.IsZero(), record.LastErrorClass)
		}
	}
}

func outboxBlockedPID(t *testing.T, ctx context.Context, pool *pgxpool.Pool, blockerPID int) int {
	t.Helper()
	var pid int
	waitForOutbox(t, func() string { return "legacy classification to block" }, func() bool {
		if err := pool.QueryRow(ctx, `
			SELECT COALESCE((SELECT pid FROM pg_stat_activity
			WHERE $1 = ANY(pg_blocking_pids(pid)) LIMIT 1), 0)`, blockerPID).Scan(&pid); err != nil {
			t.Fatalf("read blocked classification backend: %v", err)
		}
		return pid != 0
	})
	return pid
}

func TestPostgresOutboxUnknownObservationAndDiscovery(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	for index := 1; index <= 3; index++ {
		mustAppendOutbox(t, ctx, pool, store, outboxEvent(fmt.Sprintf("unknown-observe-%d", index)))
		unknown := mustClaimOutbox(t, ctx, store)
		poisonOutcomeUnknown(t, ctx, store, unknown)
	}
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("poison-observe"))
	poison := mustClaimOutbox(t, ctx, store)
	if err := poisonOutboxEvent(ctx, store, poison.Event.ID, poison.Token, "publisher_permanent"); err != nil {
		t.Fatalf("mark deterministic poison: %v", err)
	}

	observation, err := store.Observe(ctx)
	if err != nil {
		t.Fatalf("Observe(): %v", err)
	}
	if observation.OutcomeUnknownCount != 3 || observation.PoisonCount != 1 ||
		observation.ReceiptsBytes == 0 || observation.ReceiptsIndexBytes == 0 {
		t.Fatalf("unknown observation = %+v", observation)
	}
	const discovery = `
		SELECT id, destination, event_type, schema_name, cycle_attempt_count,
			total_attempt_count, last_attempt_at, poisoned_at, last_error_class,
			publication_uncertain
		FROM outbox_events
		WHERE published_at IS NULL
		  AND poisoned_at IS NOT NULL
		  AND publication_uncertain IS TRUE
		  AND last_error_class = 'outcome_unknown'
		  AND (poisoned_at, id) > ($1, $2)
		ORDER BY poisoned_at, id
		LIMIT 2
	`
	rows, err := pool.Query(ctx, discovery, time.Unix(0, 0).UTC(), "")
	if err != nil {
		t.Fatalf("outcome-unknown discovery: %v", err)
	}
	var columns []string
	for _, field := range rows.FieldDescriptions() {
		columns = append(columns, field.Name)
	}
	wantColumns := []string{
		"id", "destination", "event_type", "schema_name", "cycle_attempt_count",
		"total_attempt_count", "last_attempt_at", "poisoned_at", "last_error_class",
		"publication_uncertain",
	}
	var discovered []string
	var cursorAt time.Time
	var cursorID string
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			t.Fatalf("read discovery page: %v", err)
		}
		cursorID = values[0].(string)
		cursorAt = values[7].(time.Time)
		discovered = append(discovered, cursorID)
	}
	rows.Close()
	if !slices.Equal(columns, wantColumns) || len(discovered) != 2 {
		t.Fatalf("discovery columns/first page = %v/%v, want %v/two rows", columns, discovered, wantColumns)
	}
	rows, err = pool.Query(ctx, discovery, cursorAt, cursorID)
	if err != nil {
		t.Fatalf("outcome-unknown second discovery page: %v", err)
	}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			t.Fatalf("read second discovery page: %v", err)
		}
		discovered = append(discovered, values[0].(string))
	}
	rows.Close()
	seen := map[string]bool{}
	for _, id := range discovered {
		seen[id] = true
	}
	if len(discovered) != 3 || len(seen) != 3 {
		t.Fatalf("discovery pages = %v, want three unique outcome-unknown events", discovered)
	}
}

func TestPostgresOutboxFailedObservationStaysStaleAndUnready(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	reader, telemetry := newOutboxTelemetry(t)
	config := testRelayConfig()
	config.PollInterval = time.Hour
	config.ObservationInterval = 50 * time.Millisecond
	relay := mustNewOutboxRelay(t, store, testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
		return nil
	}), telemetry, config)
	result := runOutboxRelay(ctx, relay)
	waitForOutboxOperationCount(t, reader, "claim", "empty", 1)
	before := collectOutboxProcessMetrics(t, reader)
	if before.ready != 1 || before.observedAt == 0 {
		t.Fatalf("initial ready/observation = %d/%f, want 1/>0", before.ready, before.observedAt)
	}
	if _, err := pool.Exec(ctx, "DROP TABLE outbox_events CASCADE"); err != nil {
		t.Fatalf("remove schema for fatal observation: %v", err)
	}
	failed := readRelayResult(t, result)
	if failed.Err == nil || failed.CleanupUnsafe || relay.Ready() {
		t.Fatalf("failed observation result=%+v ready=%t", failed, relay.Ready())
	}
	after := collectOutboxProcessMetrics(t, reader)
	if after.ready != 0 || after.observedAt != before.observedAt {
		t.Fatalf("failed observation ready/timestamp = %d/%f, want 0/%f", after.ready, after.observedAt, before.observedAt)
	}
}

//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/jackc/pgx/v5"
)

func TestPostgresOutboxEnvelope(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	payload := []byte(" {\n  \"id\" : \"42\"\n} ")
	metadata := []byte(`{"z":1,"a":"two"}`)
	event := outboxEvent("exact-bytes")
	event.Payload = payload
	event.Metadata = metadata
	mustAppendOutbox(t, ctx, pool, store, event)

	record, err := store.Get(ctx, event.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !bytes.Equal(record.Event.Payload, payload) || !bytes.Equal(record.Event.Metadata, metadata) {
		t.Fatalf("stored bytes = payload %q metadata %q, want exact input bytes", record.Event.Payload, record.Event.Metadata)
	}
	record.Event.Payload[0] = 'x'
	again, err := store.Get(ctx, event.ID)
	if err != nil {
		t.Fatalf("second Get() error = %v", err)
	}
	if !bytes.Equal(again.Event.Payload, payload) {
		t.Fatal("mutating returned payload changed durable bytes")
	}
	for _, test := range []struct {
		id       string
		payload  []byte
		metadata []byte
	}{
		{id: "json-large-number", payload: []byte(`1e1000000`), metadata: []byte(`{}`)},
		{id: "json-null-codepoint-payload", payload: []byte(`"\u0000"`), metadata: []byte(`{}`)},
		{id: "json-null-codepoint-metadata", payload: []byte(`{}`), metadata: []byte(`{"value":"\u0000"}`)},
	} {
		event := outboxEvent(test.id)
		event.Payload = test.payload
		event.Metadata = test.metadata
		mustAppendOutbox(t, ctx, pool, store, event)
		stored, err := store.Get(ctx, event.ID)
		if err != nil {
			t.Fatalf("Get(%s): %v", event.ID, err)
		}
		if !bytes.Equal(stored.Event.Payload, test.payload) || !bytes.Equal(stored.Event.Metadata, test.metadata) {
			t.Fatalf("stored JSON bytes for %s = %q/%q, want %q/%q", event.ID, stored.Event.Payload, stored.Event.Metadata, test.payload, test.metadata)
		}
	}

	_, err = pool.PGX().Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata
		) VALUES ('invalid-bypass', 't', 's', 'd', 'v', clock_timestamp(),
			convert_to('not-json', 'UTF8'), convert_to('{}', 'UTF8'))`)
	if err == nil {
		t.Fatal("direct invalid JSON insert succeeded, want database constraint rejection")
	}
	assertOutboxCount(t, ctx, pool, "invalid-bypass", 0)
	for index, occurredAt := range []string{
		"0001-01-01 00:00:00+00",
		"infinity",
		"-infinity",
	} {
		id := fmt.Sprintf("invalid-occurred-at-%d", index)
		_, err = pool.PGX().Exec(ctx, `
			INSERT INTO outbox_events (
				id, event_type, source, destination, schema_name, occurred_at, payload, metadata
			) VALUES ($1, 't', 's', 'd', 'v', $2::timestamptz,
				convert_to('{}', 'UTF8'), convert_to('{}', 'UTF8'))`, id, occurredAt)
		if err == nil {
			t.Fatalf("direct invalid occurred_at %q insert succeeded, want database constraint rejection", occurredAt)
		}
		assertOutboxCount(t, ctx, pool, id, 0)
	}

	if _, err := pool.PGX().Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata
		) VALUES ('payload-max', 't', 's', 'd', 'v', clock_timestamp(),
			convert_to('"' || repeat('p', 262142) || '"', 'UTF8'), convert_to('{}', 'UTF8'))`); err != nil {
		t.Fatalf("insert maximum payload bytes: %v", err)
	}
	_, err = pool.PGX().Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata
		) VALUES ('payload-too-large', 't', 's', 'd', 'v', clock_timestamp(),
			convert_to('"' || repeat('p', 262143) || '"', 'UTF8'), convert_to('{}', 'UTF8'))`)
	if err == nil {
		t.Fatal("direct oversized payload insert succeeded")
	}
	if _, err := pool.PGX().Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata
		) VALUES ('metadata-max', 't', 's', 'd', 'v', clock_timestamp(),
			convert_to('{}', 'UTF8'), convert_to('{"m":"' || repeat('m', 32760) || '"}', 'UTF8'))`); err != nil {
		t.Fatalf("insert maximum metadata bytes: %v", err)
	}
	_, err = pool.PGX().Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata,
			ordering_key, ordering_sequence
		) VALUES (
			'E' || repeat('e', 255), repeat('t', 256), repeat('s', 256), repeat('d', 256), repeat('v', 256),
			clock_timestamp(), convert_to('"' || repeat('p', 262142) || '"', 'UTF8'),
			convert_to('{"m":"' || repeat('m', 31224) || '"}', 'UTF8'),
			'K' || repeat('k', 255), 1
		)`)
	if err != nil {
		t.Fatalf("insert exact 288 KiB envelope: %v", err)
	}
	_, err = pool.PGX().Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata,
			ordering_key, ordering_sequence
		) VALUES (
			'F' || repeat('f', 255), repeat('t', 256), repeat('s', 256), repeat('d', 256), repeat('v', 256),
			clock_timestamp(), convert_to('"' || repeat('p', 262142) || '"', 'UTF8'),
			convert_to('{"m":"' || repeat('m', 31225) || '"}', 'UTF8'),
			'L' || repeat('l', 255), 1
		)`)
	if err == nil {
		t.Fatal("direct 288 KiB plus one envelope insert succeeded")
	}
	_, err = pool.PGX().Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata
		) VALUES ('metadata-too-large', 't', 's', 'd', 'v', clock_timestamp(),
			convert_to('{}', 'UTF8'), convert_to('{"m":"' || repeat('m', 32761) || '"}', 'UTF8'))`)
	if err == nil {
		t.Fatal("direct oversized metadata insert succeeded")
	}
	// Metadata must be a JSON object, and insignificant leading whitespace does
	// not change that. Every other JSON value is rejected even though it parses.
	if _, err := pool.PGX().Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata
		) VALUES ('metadata-leading-whitespace', 't', 's', 'd', 'v', clock_timestamp(),
			convert_to('{}', 'UTF8'), convert_to(E' \t\r\n{"m":1}', 'UTF8'))`); err != nil {
		t.Fatalf("insert metadata object behind leading whitespace: %v", err)
	}
	for _, metadata := range []string{`[1,2]`, `"text"`, `42`, `null`, `true`} {
		id := "metadata-not-object-" + metadata
		if _, err := pool.PGX().Exec(ctx, `
			INSERT INTO outbox_events (
				id, event_type, source, destination, schema_name, occurred_at, payload, metadata
			) VALUES ($1, 't', 's', 'd', 'v', clock_timestamp(),
				convert_to('{}', 'UTF8'), convert_to($2, 'UTF8'))`, id, metadata); err == nil {
			t.Fatalf("direct non-object metadata %s insert succeeded", metadata)
		}
		assertOutboxCount(t, ctx, pool, id, 0)
	}
	for _, statement := range []string{
		`INSERT INTO outbox_events (id, event_type, source, destination, schema_name, occurred_at, payload, metadata, ordering_key)
		 VALUES ('pair-key-only', 't', 's', 'd', 'v', clock_timestamp(), convert_to('{}','UTF8'), convert_to('{}','UTF8'), 'k')`,
		`INSERT INTO outbox_events (id, event_type, source, destination, schema_name, occurred_at, payload, metadata, ordering_sequence)
		 VALUES ('pair-sequence-only', 't', 's', 'd', 'v', clock_timestamp(), convert_to('{}','UTF8'), convert_to('{}','UTF8'), 1)`,
	} {
		if _, err := pool.PGX().Exec(ctx, statement); err == nil {
			t.Fatalf("asymmetric ordering pair insert succeeded: %s", statement)
		}
	}
}

func TestPostgresOutboxAtomicity(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	if _, err := pool.PGX().Exec(ctx, `
		CREATE TABLE outbox_domain_probe (id text PRIMARY KEY);
		CREATE TABLE outbox_commit_guard (
			value integer UNIQUE DEFERRABLE INITIALLY DEFERRED
		);
		INSERT INTO outbox_commit_guard (value) VALUES (1)`); err != nil {
		t.Fatalf("create atomicity fixtures: %v", err)
	}

	failed := errors.New("feature failed")
	tests := []struct {
		name string
		id   string
		run  func(pgx.Tx) error
	}{
		{name: "before domain write", id: "before", run: func(pgx.Tx) error { return failed }},
		{name: "after domain before append", id: "domain-only", run: func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "INSERT INTO outbox_domain_probe (id) VALUES ($1)", "domain-only"); err != nil {
				return err
			}
			return failed
		}},
		{name: "append validation", id: "invalid-event", run: func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "INSERT INTO outbox_domain_probe (id) VALUES ($1)", "invalid-event"); err != nil {
				return err
			}
			event := outboxEvent("invalid-event")
			event.Payload = []byte("not-json")
			return store.Append(ctx, tx, event)
		}},
		{name: "after append", id: "after-append", run: func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "INSERT INTO outbox_domain_probe (id) VALUES ($1)", "after-append"); err != nil {
				return err
			}
			if err := store.Append(ctx, tx, outboxEvent("after-append")); err != nil {
				return err
			}
			return failed
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := pool.InTx(ctx, pgx.TxOptions{}, test.run)
			if err == nil {
				t.Fatal("InTx() succeeded, want rollback")
			}
			assertAtomicCounts(t, ctx, pool, test.id, 0, 0)
		})
	}

	mustAppendOutbox(t, ctx, pool, store, outboxEvent("duplicate"))
	err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "INSERT INTO outbox_domain_probe (id) VALUES ('insert-failure')"); err != nil {
			return err
		}
		return store.Append(ctx, tx, outboxEvent("duplicate"))
	})
	if err == nil {
		t.Fatal("duplicate outbox insert succeeded")
	}
	assertAtomicCounts(t, ctx, pool, "insert-failure", 0, 0)

	err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "INSERT INTO outbox_domain_probe (id) VALUES ('commit-failure')"); err != nil {
			return err
		}
		if err := store.Append(ctx, tx, outboxEvent("commit-failure")); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, "INSERT INTO outbox_commit_guard (value) VALUES (1)")
		return err
	})
	if !errors.Is(err, postgres.ErrTransaction) || errors.Is(err, postgres.ErrCommitUnknown) {
		t.Fatalf("commit failure = %v, want definite ErrTransaction", err)
	}
	assertAtomicCounts(t, ctx, pool, "commit-failure", 0, 0)

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	err = pool.InTx(canceled, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(canceled, "INSERT INTO outbox_domain_probe (id) VALUES ('canceled')"); err != nil {
			return err
		}
		return store.Append(canceled, tx, outboxEvent("canceled"))
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled transaction error = %v, want context.Canceled", err)
	}
	assertAtomicCounts(t, ctx, pool, "canceled", 0, 0)

	afterAppendCtx, cancelAfterAppend := context.WithCancel(ctx)
	err = pool.InTx(afterAppendCtx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(afterAppendCtx, "INSERT INTO outbox_domain_probe (id) VALUES ('canceled-after-append')"); err != nil {
			return err
		}
		if err := store.Append(afterAppendCtx, tx, outboxEvent("canceled-after-append")); err != nil {
			return err
		}
		cancelAfterAppend()
		return afterAppendCtx.Err()
	})
	cancelAfterAppend()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("post-append cancellation error = %v, want context.Canceled", err)
	}
	assertAtomicCounts(t, ctx, pool, "canceled-after-append", 0, 0)

	err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "INSERT INTO outbox_domain_probe (id) VALUES ('success')"); err != nil {
			return err
		}
		return store.Append(ctx, tx, outboxEvent("success"))
	})
	if err != nil {
		t.Fatalf("successful transaction: %v", err)
	}
	assertAtomicCounts(t, ctx, pool, "success", 1, 1)
}

func TestPostgresOutboxCommitReceiptAtomicityAndLifetime(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	counts := func(id string) (events, receipts int) {
		t.Helper()
		if err := pool.PGX().QueryRow(ctx, `
			SELECT
				(SELECT count(*) FROM outbox_events WHERE id = $1),
				(SELECT count(*) FROM outbox_commit_receipts WHERE event_id = $1)`,
			id,
		).Scan(&events, &receipts); err != nil {
			t.Fatalf("count event and receipt %q: %v", id, err)
		}
		return events, receipts
	}
	assertCounts := func(id string, wantEvents, wantReceipts int) {
		t.Helper()
		events, receipts := counts(id)
		if events != wantEvents || receipts != wantReceipts {
			t.Fatalf("event/receipt counts for %q = %d/%d, want %d/%d", id, events, receipts, wantEvents, wantReceipts)
		}
	}

	plain := outboxEvent("receipt-plain")
	mustAppendOutbox(t, ctx, pool, store, plain)
	ordered := orderedEvent("receipt-ordered", "receipt-key", 2)
	mustAppendOutbox(t, ctx, pool, store, ordered)
	for _, id := range []string{plain.ID, ordered.ID} {
		assertCounts(id, 1, 1)
		var version int16
		var size int
		if err := pool.PGX().QueryRow(ctx, `
			SELECT fingerprint_version, octet_length(envelope_fingerprint)
			FROM outbox_commit_receipts WHERE event_id = $1`, id,
		).Scan(&version, &size); err != nil {
			t.Fatalf("read receipt %q: %v", id, err)
		}
		if version != 1 || size != 32 {
			t.Fatalf("receipt %q version/size = %d/%d, want 1/32", id, version, size)
		}
	}

	rollback := outboxEvent("receipt-rollback")
	rollbackErr := errors.New("rollback receipt transaction")
	err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if err := store.Append(ctx, tx, rollback); err != nil {
			return err
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("receipt rollback error = %v, want %v", err, rollbackErr)
	}
	assertCounts(rollback.ID, 0, 0)

	rejected := orderedEvent("receipt-rejected", "receipt-key", 1)
	err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return store.Append(ctx, tx, rejected)
	})
	if !errors.Is(err, postgresoutbox.ErrOrderingSequence) {
		t.Fatalf("rejected ordered append error = %v, want ErrOrderingSequence", err)
	}
	assertCounts(rejected.ID, 0, 0)

	conflict := plain
	conflict.Payload = []byte(`{"id":"different"}`)
	outcome, err := store.ReconcileCommit(ctx, conflict)
	if outcome != postgresoutbox.CommitStillUnknown || !errors.Is(err, postgresoutbox.ErrReceiptConflict) {
		t.Fatalf("conflicting receipt reconciliation = %v, %v", outcome, err)
	}

	unsupported := outboxEvent("receipt-unsupported-version")
	if _, err := pool.PGX().Exec(ctx, `
		INSERT INTO outbox_commit_receipts (event_id, fingerprint_version, envelope_fingerprint)
		VALUES ($1, 2, $2)`, unsupported.ID, bytes.Repeat([]byte{1}, 32)); err != nil {
		t.Fatalf("seed unsupported receipt: %v", err)
	}
	outcome, err = store.ReconcileCommit(ctx, unsupported)
	if outcome != postgresoutbox.CommitStillUnknown || err == nil {
		t.Fatalf("unsupported receipt reconciliation = %v, %v", outcome, err)
	}

	cleanup := outboxEvent("receipt-survives-cleanup")
	mustAppendOutbox(t, ctx, pool, store, cleanup)
	if _, err := pool.PGX().Exec(ctx, `
		UPDATE outbox_events
		SET published_at = clock_timestamp() - interval '2 hours'
		WHERE id = $1`, cleanup.ID); err != nil {
		t.Fatalf("make receipt event cleanup-eligible: %v", err)
	}
	if deleted, err := store.CleanupPublished(ctx, time.Hour, 10); err != nil || deleted != 1 {
		t.Fatalf("CleanupPublished() = %d, %v; want 1, nil", deleted, err)
	}
	assertCounts(cleanup.ID, 0, 1)
}

func TestPostgresOutboxCommitReconciliationAuthority(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)

	applied := outboxEvent("reconcile-applied")
	mustAppendOutbox(t, ctx, pool, store, applied)
	outcome, err := store.ReconcileCommit(ctx, applied)
	if outcome != postgresoutbox.CommitApplied || err != nil {
		t.Fatalf("ReconcileCommit(applied) = %v, %v", outcome, err)
	}

	notApplied := outboxEvent("reconcile-not-applied")
	outcome, err = store.ReconcileCommit(ctx, notApplied)
	if outcome != postgresoutbox.CommitNotApplied || err != nil {
		t.Fatalf("ReconcileCommit(not applied) = %v, %v", outcome, err)
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	outcome, err = store.ReconcileCommit(canceled, outboxEvent("reconcile-canceled"))
	if outcome != postgresoutbox.CommitStillUnknown || !errors.Is(err, context.Canceled) {
		t.Fatalf("ReconcileCommit(canceled) = %v, %v", outcome, err)
	}

	admin, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire authority fixture connection: %v", err)
	}
	var database string
	if err := admin.QueryRow(ctx, "SELECT current_database()").Scan(&database); err != nil {
		admin.Release()
		t.Fatalf("read fixture database: %v", err)
	}
	readOnlyStatement := fmt.Sprintf(
		"ALTER DATABASE %s SET default_transaction_read_only = on",
		pgx.Identifier{database}.Sanitize(),
	)
	writableStatement := fmt.Sprintf(
		"ALTER DATABASE %s RESET default_transaction_read_only",
		pgx.Identifier{database}.Sanitize(),
	)
	if _, err := admin.Exec(ctx, readOnlyStatement); err != nil {
		admin.Release()
		t.Fatalf("make new fixture sessions read only: %v", err)
	}
	restored := false
	t.Cleanup(func() {
		if restored {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, cleanupErr := admin.Exec(cleanupCtx, writableStatement); cleanupErr != nil {
			t.Errorf("restore fixture writer authority: %v", cleanupErr)
		}
		admin.Release()
		pool.PGX().Reset()
	})
	pool.PGX().Reset()
	outcome, err = store.ReconcileCommit(ctx, outboxEvent("reconcile-read-only"))
	if outcome != postgresoutbox.CommitStillUnknown || err == nil {
		t.Fatalf("ReconcileCommit(read only absence) = %v, %v", outcome, err)
	}
	if _, err := admin.Exec(ctx, writableStatement); err != nil {
		t.Fatalf("restore fixture writer authority: %v", err)
	}
	admin.Release()
	pool.PGX().Reset()
	restored = true

	pool.Close()
	outcome, err = store.ReconcileCommit(ctx, outboxEvent("reconcile-unavailable"))
	if outcome != postgresoutbox.CommitStillUnknown || err == nil {
		t.Fatalf("ReconcileCommit(unavailable) = %v, %v", outcome, err)
	}
}

// One Append call carries a whole business transaction's events. It pipelines
// them rather than taking a round trip each, and the events still see the same
// durable state and report the same outcomes they would one call at a time.
func TestPostgresOutboxBatchedAppend(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)

	// Mixed shapes commit together, and a key's events are ordered by their own
	// sequence rather than by their position in the call. The ordered pair below
	// is passed highest sequence first on purpose: the head still opens at
	// sequence 1, so batch-key-1 is the claimable one and batch-key-2 waits
	// behind it. Passing them in ascending order would leave the two rules
	// indistinguishable, and the ordering contract a feature relies on is that
	// argument order does not matter.
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return store.Append(ctx, tx,
			outboxEvent("batch-plain-1"),
			orderedEvent("batch-key-2", "batch-account", 2),
			outboxEvent("batch-plain-2"),
			orderedEvent("batch-key-1", "batch-account", 1),
		)
	}); err != nil {
		t.Fatalf("Append(batch): %v", err)
	}
	for _, id := range []string{"batch-plain-1", "batch-plain-2", "batch-key-1", "batch-key-2"} {
		assertOutboxCount(t, ctx, pool, id, 1)
	}
	var ready []string
	rows, err := pool.PGX().Query(ctx,
		`SELECT id FROM outbox_events WHERE ordering_ready ORDER BY id`)
	if err != nil {
		t.Fatalf("read ready ordered events: %v", err)
	}
	ready, err = pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("collect ready ordered events: %v", err)
	}
	if len(ready) != 1 || ready[0] != "batch-key-1" {
		t.Fatalf("claimable ordered events = %v, want only batch-key-1", ready)
	}

	// One rejected sequence rejects the whole call, and the message names the
	// event that lost rather than the first one in it.
	err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return store.Append(ctx, tx,
			outboxEvent("batch-rolled-back"),
			orderedEvent("batch-key-replay", "batch-account", 1),
		)
	})
	if rejection := fmt.Sprintf("%v", err); !errors.Is(err, postgresoutbox.ErrOrderingSequence) ||
		!strings.Contains(rejection, `key "batch-account" sequence 1`) {
		t.Fatalf("Append(batch with replayed sequence) error = %v", err)
	}
	assertOutboxCount(t, ctx, pool, "batch-rolled-back", 0)
	assertOutboxCount(t, ctx, pool, "batch-key-replay", 0)

	// A rejected event keeps its whole call off the wire, so the valid events
	// beside it are not stored either.
	invalid := outboxEvent("batch-invalid")
	invalid.Payload = []byte(`not json`)
	err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return store.Append(ctx, tx, outboxEvent("batch-valid-neighbor"), invalid)
	})
	if !errors.Is(err, postgresoutbox.ErrInvalidEvent) {
		t.Fatalf("Append(batch with invalid event) error = %v", err)
	}
	assertOutboxCount(t, ctx, pool, "batch-valid-neighbor", 0)
	assertOutboxCount(t, ctx, pool, "batch-invalid", 0)
}

func TestPostgresOutboxCleanup(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("cleanup-ordered", "cleanup-key", 1))
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("cleanup-unordered"))
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("cleanup-recent"))
	for range 3 {
		claim := mustClaimOutbox(t, ctx, store)
		if err := markOutboxPublished(ctx, store, claim); err != nil {
			t.Fatalf("MarkPublished(): %v", err)
		}
	}
	if _, err := pool.PGX().Exec(ctx, "UPDATE outbox_events SET published_at = clock_timestamp() - interval '2 hours' WHERE id <> 'cleanup-recent'"); err != nil {
		t.Fatalf("backdate published rows: %v", err)
	}
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("cleanup-pending"))
	type cleanupResult struct {
		deleted int
		err     error
	}
	results := make(chan cleanupResult, 2)
	for range 2 {
		go func() {
			deleted, err := store.CleanupPublished(ctx, time.Hour, 1)
			results <- cleanupResult{deleted: deleted, err: err}
		}()
	}
	deletedTotal := 0
	for range 2 {
		result := <-results
		if result.err != nil {
			t.Fatalf("concurrent CleanupPublished(): %v", result.err)
		}
		deletedTotal += result.deleted
	}
	if deletedTotal != 2 {
		t.Fatalf("concurrent deleted rows = %d, want 2", deletedTotal)
	}
	assertTotalOutboxCount(t, ctx, pool, 2)
	deleted, err := store.CleanupPublished(ctx, time.Hour, 1)
	if err != nil || deleted != 0 {
		t.Fatalf("empty CleanupPublished() = %d, %v; want 0, nil", deleted, err)
	}
	assertOutboxCount(t, ctx, pool, "cleanup-recent", 1)
	assertOutboxCount(t, ctx, pool, "cleanup-pending", 1)
	var headCount int
	if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM outbox_ordering_heads WHERE ordering_key = 'cleanup-key'").Scan(&headCount); err != nil {
		t.Fatalf("count retained ordering head: %v", err)
	}
	if headCount != 1 {
		t.Fatalf("retained ordering heads = %d, want 1", headCount)
	}
}

//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/jackc/pgx/v5"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
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

func TestPostgresOutboxOrderingAuthority(t *testing.T) {
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

func TestPostgresOutboxConcurrentClaims(t *testing.T) {
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

func TestPostgresOutboxOrderingHandoffRace(t *testing.T) {
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
		func() string { return "mark did not wait for the ordering head lock" },
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

func TestPostgresOutboxLeaseExpiryAndFence(t *testing.T) {
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
	ctx, pool, store := newOutboxFixture(t)
	event := outboxEvent("poison")
	event.Payload = []byte(" {\n \"redrive\" : true\n} ")
	event.Metadata = []byte(`{"b":2,"a":1}`)
	mustAppendOutbox(t, ctx, pool, store, event)
	if err := store.Redrive(ctx, event.ID, "pending-audit"); !errors.Is(err, postgresoutbox.ErrRedriveRejected) {
		t.Fatalf("pending Redrive() = %v, want ErrRedriveRejected", err)
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
	if err := store.Redrive(ctx, other.Event.ID, "audit-3"); !errors.Is(err, postgresoutbox.ErrRedriveRejected) {
		t.Fatalf("redrive published event = %v, want ErrRedriveRejected", err)
	}
	other = mustClaimOutbox(t, ctx, store)
	if err := poisonOutboxEvent(ctx, store, other.Event.ID, other.Token, "publisher_permanent"); err != nil {
		t.Fatalf("poison other event: %v", err)
	}
	if err := store.Redrive(ctx, other.Event.ID, "audit-2"); !errors.Is(err, postgresoutbox.ErrRedriveConflict) {
		t.Fatalf("cross-event audit reuse = %v, want ErrRedriveConflict", err)
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
	if auditRows != 0 {
		t.Fatalf("redrive audit rows after event cleanup = %d, want 0", auditRows)
	}
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

func TestPostgresOutboxPublishFailure(t *testing.T) {
	tests := []struct {
		name            string
		publisherResult error
		waitForTimeout  bool
		wantClass       string
		wantPoison      bool
	}{
		{name: "temporary before acknowledgement", publisherResult: errors.New("broker unavailable"), wantClass: "publisher_temporary"},
		{name: "timeout before acknowledgement", waitForTimeout: true, wantClass: "publisher_timeout"},
		// A permanent rejection poisons on its first occurrence, which the
		// attempt assertion below is what proves: the row is parked without the
		// relay ever handing the same bytes to the adapter a second time.
		{name: "permanent rejection", publisherResult: postgresoutbox.ErrPermanentPublication, wantClass: "publisher_permanent", wantPoison: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, pool, store := newOutboxFixture(t)
			mustAppendOutbox(t, ctx, pool, store, outboxEvent("publish-failure"))
			entered := make(chan struct{})
			release := make(chan struct{})
			var attempts atomic.Int64
			publisher := testPublisherFunc(func(publishCtx context.Context, _ postgresoutbox.Event) error {
				attempts.Add(1)
				close(entered)
				if test.waitForTimeout {
					<-publishCtx.Done()
					return publishCtx.Err()
				}
				<-release
				return test.publisherResult
			})
			relay := mustNewOutboxRelay(t, store, publisher, nil, testRelayConfig())
			result := runOutboxRelay(ctx, relay)
			<-entered
			relay.StartDrain()
			close(release)
			assertRelayResult(t, result, nil)

			record, err := store.Get(ctx, "publish-failure")
			if err != nil {
				t.Fatalf("Get(): %v", err)
			}
			if record.LastErrorClass != test.wantClass {
				t.Fatalf("error class = %q, want %q", record.LastErrorClass, test.wantClass)
			}
			if gotPoison := !record.PoisonedAt.IsZero(); gotPoison != test.wantPoison {
				t.Fatalf("poisoned = %t, want %t", gotPoison, test.wantPoison)
			}
			if !record.PublishedAt.IsZero() {
				t.Fatal("failed publication was marked published")
			}
			if got := attempts.Load(); got != 1 {
				t.Fatalf("publisher attempts = %d, want 1 before the drain stopped the cycle", got)
			}
		})
	}
}

func TestPostgresOutboxAckCrashDuplicate(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	event := outboxEvent("ack-crash")
	event.Payload = []byte(" {\"same\": true} ")
	mustAppendOutbox(t, ctx, pool, store, event)

	attempts := make(chan postgresoutbox.Event, 2)
	releaseSecond := make(chan struct{})
	var callCount atomic.Int64
	publisher := testPublisherFunc(func(_ context.Context, event postgresoutbox.Event) error {
		copy := event
		copy.Payload = append([]byte(nil), event.Payload...)
		copy.Metadata = append([]byte(nil), event.Metadata...)
		attempts <- copy
		if callCount.Add(1) == 2 {
			<-releaseSecond
		}
		return nil
	})

	first, err := claimOutboxEvent(ctx, store, shortOutboxLease)
	if err != nil {
		t.Fatalf("first Claim(): %v", err)
	}
	if err := publisher.Publish(ctx, first.Event); err != nil {
		t.Fatalf("first durable publish: %v", err)
	}
	firstAttempt := <-attempts
	expireOutboxLease(t, ctx, pool)

	relay := mustNewOutboxRelay(t, store, publisher, nil, testRelayConfig())
	result := runOutboxRelay(ctx, relay)
	secondAttempt := <-attempts
	relay.StartDrain()
	close(releaseSecond)
	assertRelayResult(t, result, nil)
	if !reflect.DeepEqual(firstAttempt, secondAttempt) {
		t.Fatalf("duplicate envelopes differ:\nfirst  %+v\nsecond %+v", firstAttempt, secondAttempt)
	}
	record, err := store.Get(ctx, event.ID)
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if record.PublishedAt.IsZero() || record.TotalAttemptCount != 2 {
		t.Fatalf("final record published=%v attempts=%d, want published and 2", record.PublishedAt, record.TotalAttemptCount)
	}
}

func TestPostgresOutboxRelayReplicas(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	const eventCount = 32
	for i := range eventCount {
		mustAppendOutbox(t, ctx, pool, store, outboxEvent(fmt.Sprintf("replica-%02d", i)))
	}
	attempts := make(chan string, eventCount)
	publisher := testPublisherFunc(func(_ context.Context, event postgresoutbox.Event) error {
		attempts <- event.ID
		return nil
	})

	const replicaCount = 4
	relays := make([]*postgresoutbox.Relay, 0, replicaCount)
	results := make([]<-chan postgresoutbox.RelayResult, 0, replicaCount)
	for range replicaCount {
		relay := mustNewOutboxRelay(t, store, publisher, nil, testRelayConfig())
		relays = append(relays, relay)
		results = append(results, runOutboxRelay(ctx, relay))
	}
	seen := make(map[string]struct{}, eventCount)
	deadline := time.NewTimer(outboxWaitTimeout)
	defer deadline.Stop()
	for len(seen) < eventCount {
		select {
		case id := <-attempts:
			if _, duplicate := seen[id]; duplicate {
				t.Fatalf("event %q published concurrently more than once", id)
			}
			seen[id] = struct{}{}
		case <-deadline.C:
			t.Fatalf("publication attempts = %d, want %d", len(seen), eventCount)
		}
	}
	waitForOutboxCount(t, ctx, pool, "published_at IS NOT NULL", eventCount)
	for _, relay := range relays {
		relay.StartDrain()
	}
	for _, result := range results {
		assertRelayResult(t, result, nil)
	}
}

func TestPostgresOutboxRequestContinuesDuringBrokerOutage(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	if _, err := pool.PGX().Exec(ctx, `CREATE TABLE outbox_http_probe (id text PRIMARY KEY)`); err != nil {
		t.Fatalf("create HTTP mutation probe: %v", err)
	}

	var publisherAttempts atomic.Int64
	relayConfig := testRelayConfig()
	relayConfig.RetryBase = time.Hour
	relayConfig.RetryMax = time.Hour
	relay := mustNewOutboxRelay(t, store, testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
		publisherAttempts.Add(1)
		return errors.New("broker unavailable")
	}), nil, relayConfig)
	relayResult := runOutboxRelay(ctx, relay)

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		const prefix = "/mutations/"
		if request.Method != http.MethodPost || !strings.HasPrefix(request.URL.Path, prefix) || len(request.URL.Path) == len(prefix) {
			http.NotFound(response, request)
			return
		}
		id := strings.TrimPrefix(request.URL.Path, prefix)
		err := pool.InTx(request.Context(), pgx.TxOptions{}, func(tx pgx.Tx) error {
			if _, err := tx.Exec(request.Context(), "INSERT INTO outbox_http_probe (id) VALUES ($1)", id); err != nil {
				return err
			}
			event := outboxEvent(id)
			event.Destination = "events.unavailable"
			return store.Append(request.Context(), tx, event)
		})
		if err != nil {
			http.Error(response, "mutation failed", http.StatusInternalServerError)
			return
		}
		response.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(server.Close)

	const requestCount = 8
	for i := range requestCount {
		id := fmt.Sprintf("outage-%02d", i)
		response, err := http.Post(server.URL+"/mutations/"+id, "application/json", nil)
		if err != nil {
			t.Fatalf("POST mutation %s: %v", id, err)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusCreated {
			t.Fatalf("POST mutation %s status = %d, want %d", id, response.StatusCode, http.StatusCreated)
		}
	}
	waitForOutboxCount(t, ctx, pool, "last_error_class = 'publisher_temporary'", requestCount)
	relay.StartDrain()
	assertRelayResult(t, relayResult, nil)

	var domainRows, outboxRows, publishedRows int
	if err := pool.PGX().QueryRow(ctx, `SELECT
		(SELECT count(*) FROM outbox_http_probe),
		(SELECT count(*) FROM outbox_events),
		(SELECT count(*) FROM outbox_events WHERE published_at IS NOT NULL)
	`).Scan(&domainRows, &outboxRows, &publishedRows); err != nil {
		t.Fatalf("read outage durability counts: %v", err)
	}
	observation, err := store.Observe(ctx)
	if err != nil {
		t.Fatalf("Observe() outage backlog: %v", err)
	}
	if domainRows != requestCount || outboxRows != requestCount || publishedRows != 0 ||
		observation.RetryWaitCount != requestCount || publisherAttempts.Load() != requestCount {
		t.Fatalf(
			"outage state domain=%d outbox=%d published=%d retry_wait=%d attempts=%d, want %d/%d/0/%d/%d",
			domainRows, outboxRows, publishedRows, observation.RetryWaitCount, publisherAttempts.Load(),
			requestCount, requestCount, requestCount, requestCount,
		)
	}
}

// A backlog drains through batched claims and concurrent publication, so
// throughput is not one database round trip per event.
func TestPostgresOutboxRelayPublishesBacklogConcurrently(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	const (
		backlog     = 48
		concurrency = 8
	)
	for index := range backlog {
		mustAppendOutbox(t, ctx, pool, store, outboxEvent(fmt.Sprintf("backlog-%02d", index)))
	}

	var mutex sync.Mutex
	var live, peak int
	var opened sync.Once
	release := make(chan struct{})
	publisher := testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
		mutex.Lock()
		live++
		peak = max(peak, live)
		saturated := live >= concurrency
		mutex.Unlock()
		// A full set of workers must be inside the publisher at once before any
		// of them returns; that is what proves publication is pipelined.
		if saturated {
			opened.Do(func() { close(release) })
		}
		<-release
		mutex.Lock()
		live--
		mutex.Unlock()
		return nil
	})

	config := testRelayConfig()
	config.BatchSize = 24
	config.PublishConcurrency = concurrency
	relay := mustNewOutboxRelay(t, store, publisher, nil, config)
	result := runOutboxRelay(ctx, relay)
	waitForOutboxCount(t, ctx, pool, "published_at IS NOT NULL", backlog)
	relay.StartDrain()
	assertRelayResult(t, result, nil)

	mutex.Lock()
	defer mutex.Unlock()
	if peak != concurrency {
		t.Fatalf("peak concurrent publications = %d, want %d", peak, concurrency)
	}
}

// The claim query hands out at most one ready event per ordering key, which is
// what makes concurrent publication of a batch safe.
func TestPostgresOutboxBatchClaimHoldsOneEventPerOrderingKey(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	for sequence := int64(1); sequence <= 4; sequence++ {
		mustAppendOutbox(t, ctx, pool, store, orderedEvent(fmt.Sprintf("ordered-%d", sequence), "one-key", sequence))
	}
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("unordered"))

	batch, err := store.Claim(ctx, time.Minute, 100)
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
	ctx, pool, store := newOutboxFixture(t)
	const keys = 3
	for key := 1; key <= keys; key++ {
		for sequence := int64(1); sequence <= 2; sequence++ {
			mustAppendOutbox(t, ctx, pool, store, orderedEvent(
				fmt.Sprintf("key-%d-seq-%d", key, sequence), fmt.Sprintf("key-%d", key), sequence))
		}
	}

	batch, err := store.Claim(ctx, time.Minute, 100)
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

	successors, err := store.Claim(ctx, time.Minute, 100)
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

// A committed append wakes an idle relay through PostgreSQL notification
// rather than waiting out the poll interval.
func TestPostgresOutboxRelayWakesOnAppendNotification(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	publisher := testPublisherFunc(func(context.Context, postgresoutbox.Event) error { return nil })
	config := testRelayConfig()
	// Only a notification can deliver this event inside the assertion window.
	config.PollInterval = time.Minute
	relay := mustNewOutboxRelay(t, store, publisher, nil, config)
	result := runOutboxRelay(ctx, relay)
	waitForOutboxReady(t, relay)

	mustAppendOutbox(t, ctx, pool, store, outboxEvent("notified"))
	waitForOutboxCount(t, ctx, pool, "published_at IS NOT NULL", 1)
	relay.StartDrain()
	assertRelayResult(t, result, nil)
}

// Each case here has a synctest twin in internal/infra/postgresoutbox that
// drives the same fault against a stubbed store. The twin owns the relay's
// decision — which store call it makes, and whether it starts another cycle. The
// cases below own the durable consequence: what a real PostgreSQL row looks like
// once the process has stopped, which is what an operator inspects after a
// crash. Both are kept deliberately; a change to the decision belongs in the
// twin, a change to the resulting row state belongs here.
func TestPostgresOutboxRelayLifecycleFaults(t *testing.T) {
	t.Run("process cancellation leaves durable unfinished work", func(t *testing.T) {
		fixtureCtx, pool, store := newOutboxFixture(t)
		mustAppendOutbox(t, fixtureCtx, pool, store, outboxEvent("cancel-current"))
		mustAppendOutbox(t, fixtureCtx, pool, store, outboxEvent("cancel-next"))
		started := make(chan struct{})
		var attempts atomic.Int64
		publisher := testPublisherFunc(func(ctx context.Context, _ postgresoutbox.Event) error {
			attempts.Add(1)
			close(started)
			<-ctx.Done()
			return ctx.Err()
		})
		relayCtx, cancel := context.WithCancel(fixtureCtx)
		relay := mustNewOutboxRelay(t, store, publisher, nil, singleEventRelayConfig())
		result := runOutboxRelay(relayCtx, relay)
		<-started
		if !relay.Ready() {
			t.Fatal("relay not ready during a joined publication attempt")
		}
		cancel()
		assertRelayResult(t, result, nil)
		if relay.Ready() || attempts.Load() != 1 {
			t.Fatalf("after cancellation ready=%t attempts=%d, want false/1", relay.Ready(), attempts.Load())
		}
		assertTotalOutboxCount(t, fixtureCtx, pool, 2)
		waitForOutboxCount(t, fixtureCtx, pool, "published_at IS NULL", 2)
	})

	t.Run("publisher panic is fatal and starts no next claim", func(t *testing.T) {
		ctx, pool, store := newOutboxFixture(t)
		mustAppendOutbox(t, ctx, pool, store, outboxEvent("panic-current"))
		mustAppendOutbox(t, ctx, pool, store, outboxEvent("panic-next"))
		var attempts atomic.Int64
		publisher := testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
			attempts.Add(1)
			panic("publisher detail must remain supervised")
		})
		relay := mustNewOutboxRelay(t, store, publisher, nil, singleEventRelayConfig())
		assertRelayResult(t, runOutboxRelay(ctx, relay), postgresoutbox.ErrPublisherPanic)
		if relay.Ready() || attempts.Load() != 1 {
			t.Fatalf("after panic ready=%t attempts=%d, want false/1", relay.Ready(), attempts.Load())
		}
		waitForOutboxCount(t, ctx, pool, "published_at IS NULL", 2)
	})

	t.Run("stuck publisher is cleanup unsafe and starts no next claim", func(t *testing.T) {
		ctx, pool, store := newOutboxFixture(t)
		mustAppendOutbox(t, ctx, pool, store, outboxEvent("stuck-current"))
		mustAppendOutbox(t, ctx, pool, store, outboxEvent("stuck-next"))
		publisher, started, release, attempts := gatingPublisher()
		config := singleEventRelayConfig()
		config.PublishTimeout = time.Millisecond
		relay := mustNewOutboxRelay(t, store, publisher, nil, config)
		result := runOutboxRelay(ctx, relay)
		<-started
		assertRelayStuckResult(t, result, postgresoutbox.ErrPublisherStuck)
		close(release)
		if relay.Ready() || attempts.Load() != 1 {
			t.Fatalf("after stuck publisher ready=%t attempts=%d, want false/1", relay.Ready(), attempts.Load())
		}
		waitForOutboxCount(t, ctx, pool, "published_at IS NULL", 2)
	})

	t.Run("drain finishes current acknowledgement and starts no next claim", func(t *testing.T) {
		ctx, pool, store := newOutboxFixture(t)
		mustAppendOutbox(t, ctx, pool, store, outboxEvent("drain-current"))
		mustAppendOutbox(t, ctx, pool, store, outboxEvent("drain-next"))
		publisher, started, release, attempts := gatingPublisher()
		relay := mustNewOutboxRelay(t, store, publisher, nil, singleEventRelayConfig())
		result := runOutboxRelay(ctx, relay)
		<-started
		relay.StartDrain()
		close(release)
		assertRelayResult(t, result, nil)
		if relay.Ready() || attempts.Load() != 1 {
			t.Fatalf("after drain ready=%t attempts=%d, want false/1", relay.Ready(), attempts.Load())
		}
		waitForOutboxCount(t, ctx, pool, "published_at IS NOT NULL", 1)
	})
}

func TestPostgresOutboxDrainDuringMaintenanceStartsNoClaim(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("maintenance-drain"))
	if _, err := pool.PGX().Exec(ctx, "UPDATE outbox_events SET available_at = clock_timestamp() + interval '1 hour'"); err != nil {
		t.Fatalf("delay event eligibility: %v", err)
	}

	var attempts atomic.Int64
	publisher := testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
		attempts.Add(1)
		return nil
	})
	config := testRelayConfig()
	config.PollInterval = time.Hour
	config.ObservationInterval = 500 * time.Millisecond
	reader, telemetry := newOutboxTelemetry(t)
	relay := mustNewOutboxRelay(t, store, publisher, telemetry, config)
	result := runOutboxRelay(ctx, relay)
	// The append listener signals one wake-up as soon as it subscribes. Let that
	// wake-up drive a claim to completion first, so the gate below lands while
	// the relay waits for its next observation rather than mid-claim.
	waitForOutboxListener(t, ctx, pool)
	waitForOutboxOperationCount(t, reader, "claim", "empty",
		outboxOperationCount(t, reader, "claim", "empty")+1)

	lockTx, err := pool.PGX().Begin(ctx)
	if err != nil {
		t.Fatalf("begin maintenance gate: %v", err)
	}
	t.Cleanup(func() { _ = lockTx.Rollback(context.Background()) })
	if _, err := lockTx.Exec(ctx, "LOCK TABLE outbox_events IN ACCESS EXCLUSIVE MODE"); err != nil {
		t.Fatalf("lock outbox table: %v", err)
	}
	if _, err := lockTx.Exec(ctx, "UPDATE outbox_events SET available_at = clock_timestamp() - interval '1 second'"); err != nil {
		t.Fatalf("make event eligible behind maintenance gate: %v", err)
	}
	waitForBlockedOutboxObservation(t, ctx, pool)

	relay.StartDrain()
	if err := lockTx.Commit(ctx); err != nil {
		t.Fatalf("release maintenance gate: %v", err)
	}
	assertRelayResult(t, result, nil)
	if attempts.Load() != 0 {
		t.Fatalf("publisher attempts after drain during maintenance = %d, want 0", attempts.Load())
	}
	waitForOutboxCount(t, ctx, pool, "lease_token IS NULL AND published_at IS NULL", 1)
}

func TestPostgresOutboxDrainDuringInitialObservationNeverBecomesReady(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	lockTx, err := pool.PGX().Begin(ctx)
	if err != nil {
		t.Fatalf("begin startup observation gate: %v", err)
	}
	t.Cleanup(func() { _ = lockTx.Rollback(context.Background()) })
	if _, err := lockTx.Exec(ctx, "LOCK TABLE outbox_events IN ACCESS EXCLUSIVE MODE"); err != nil {
		t.Fatalf("lock outbox table: %v", err)
	}

	reader, telemetry := newOutboxTelemetry(t)
	var attempts atomic.Int64
	relay := mustNewOutboxRelay(t, store, testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
		attempts.Add(1)
		return nil
	}), telemetry, testRelayConfig())
	result := runOutboxRelay(ctx, relay)
	waitForBlockedOutboxObservation(t, ctx, pool)

	relay.StartDrain()
	if relay.Ready() {
		t.Fatal("relay became ready while startup observation was blocked and drain had started")
	}
	if err := lockTx.Commit(ctx); err != nil {
		t.Fatalf("release startup observation gate: %v", err)
	}
	assertRelayResult(t, result, nil)
	if relay.Ready() || attempts.Load() != 0 {
		t.Fatalf("startup drain ready=%t attempts=%d, want false/0", relay.Ready(), attempts.Load())
	}
	process := collectOutboxProcessMetrics(t, reader)
	if process.ready != 0 || process.inflight != 0 {
		t.Fatalf("startup drain telemetry ready/inflight = %d/%d, want 0/0", process.ready, process.inflight)
	}
}

// The relay fails closed when any relation it owns is missing, and the gate is
// the startup observation: ObserveOutbox is the one statement that touches all
// three tables. It reaches outbox_redrives only through that relation's
// storage-bytes column, so dropping the column drops the gate — this test is
// what fails, and it says so.
func TestPostgresOutboxStartupRequiresRedriveLedger(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	if _, err := pool.PGX().Exec(ctx, "DROP TABLE outbox_redrives"); err != nil {
		t.Fatalf("drop redrive ledger: %v", err)
	}
	var attempts atomic.Int64
	relay := mustNewOutboxRelay(t, store, testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
		attempts.Add(1)
		return nil
	}), nil, testRelayConfig())
	result := readRelayResult(t, runOutboxRelay(ctx, relay))
	if result.Err == nil {
		t.Error("Run() started without outbox_redrives; the startup observation no longer reads that relation")
	}
	if result.CleanupUnsafe {
		t.Errorf("Run() = %+v, want cleanup to stay safe after a startup failure", result)
	}
	if relay.Ready() {
		t.Error("Ready() = true after a failed startup observation")
	}
	if got := attempts.Load(); got != 0 {
		t.Errorf("publisher attempts = %d, want 0 before the schema gate passes", got)
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

	// An ambiguous failure never proves the broker refused the event, so the
	// attempt cap must keep retrying instead of poisoning a row that may still
	// need delivery.
	t.Run("ambiguous", func(t *testing.T) {
		ctx, pool, store := newOutboxFixture(t)
		mustAppendOutbox(t, ctx, pool, store, outboxEvent("ambiguous"))
		config := testRelayConfig()
		config.MaxAttempts = 2
		config.RetryBase = time.Nanosecond
		config.RetryMax = time.Nanosecond

		var attempts atomic.Int64
		var once sync.Once
		pastThreshold := make(chan struct{})
		publisher := testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
			// Past the cap, not a number that happens to exceed it: an ambiguous
			// failure must keep retrying however the cap is configured.
			if attempts.Add(1) > int64(config.MaxAttempts) {
				once.Do(func() { close(pastThreshold) })
			}
			return errors.New("temporary")
		})
		relay := mustNewOutboxRelay(t, store, publisher, nil, config)
		result := runOutboxRelay(ctx, relay)
		select {
		case <-pastThreshold:
		case <-time.After(outboxWaitTimeout):
			t.Fatalf("relay stopped retrying an ambiguous failure after %d attempts", attempts.Load())
		}
		relay.StartDrain()
		assertRelayResult(t, result, nil)
		waitForOutboxCount(t, ctx, pool, "poisoned_at IS NOT NULL", 0)
		record, err := store.Get(ctx, "ambiguous")
		if err != nil {
			t.Fatalf("Get(): %v", err)
		}
		if !record.PoisonedAt.IsZero() || record.LastErrorClass != "publisher_temporary" {
			t.Fatalf("ambiguous poisoned=%v class=%q, want unpoisoned publisher_temporary", record.PoisonedAt, record.LastErrorClass)
		}
	})

	// A permanent rejection has no loop dynamic to observe, so it belongs with
	// the other single-cycle dispositions in TestPostgresOutboxPublishFailure
	// rather than here. Both subtests above turn on how many cycles the relay
	// runs, which is the thing this test exists to prove.
}

func TestPostgresOutboxObservability(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	states := []string{"eligible", "in-progress", "retry-wait", "recovery-due", "published"}
	for _, state := range states {
		mustAppendOutbox(t, ctx, pool, store, outboxEvent("observe-"+state))
	}
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("observe-poison", "observe-order", 1))
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("observe-ordering-blocked", "observe-order", 2))
	if _, err := pool.PGX().Exec(ctx, `
		UPDATE outbox_events SET lease_token = 'live', lease_expires_at = clock_timestamp() + interval '1 hour'
		WHERE id = 'observe-in-progress';
		UPDATE outbox_events SET available_at = clock_timestamp() + interval '1 hour'
		WHERE id = 'observe-retry-wait';
		UPDATE outbox_events SET lease_token = 'expired', lease_expires_at = clock_timestamp() - interval '1 hour'
		WHERE id = 'observe-recovery-due';
		UPDATE outbox_events SET poisoned_at = clock_timestamp()
		WHERE id = 'observe-poison';
		UPDATE outbox_events SET published_at = clock_timestamp()
		WHERE id = 'observe-published'`); err != nil {
		t.Fatalf("place observation fixtures: %v", err)
	}
	// The retained-published count is the planner's row estimate minus the exact
	// pending count, so the fixture needs current statistics to be comparable.
	if _, err := pool.PGX().Exec(ctx, "ANALYZE outbox_events"); err != nil {
		t.Fatalf("analyze observation fixtures: %v", err)
	}
	observation, err := store.Observe(ctx)
	if err != nil {
		t.Fatalf("Observe(): %v", err)
	}
	if counts := []int64{
		observation.EligibleCount,
		observation.InProgressCount,
		observation.RetryWaitCount,
		observation.RecoveryDueCount,
		observation.OrderingBlockedCount,
		observation.PoisonCount,
		observation.PublishedRetainedEstimate,
	}; !reflect.DeepEqual(counts, []int64{1, 1, 1, 1, 1, 1, 1}) {
		t.Fatalf("state counts = %v, want seven ones", counts)
	}
	assertOutboxObservationMatchesSQL(t, ctx, pool, observation)

	first := collectOutboxStateMetrics(t, observation)
	second := collectOutboxStateMetrics(t, observation)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("replica database-global metrics differ: %v vs %v", first, second)
	}
	for _, state := range []string{"eligible", "in_progress", "retry_wait", "recovery_due", "ordering_blocked", "poison", "published_retained"} {
		if first.counts[state] != 1 {
			t.Fatalf("metric state %q = %d, want 1", state, first.counts[state])
		}
	}
	if first.orderingHeads != observation.OrderingHeadCount || !reflect.DeepEqual(first.storage, map[string]int64{
		"events/total":           observation.EventsBytes,
		"events/indexes":         observation.EventsIndexBytes,
		"ordering_heads/total":   observation.OrderingHeadsBytes,
		"ordering_heads/indexes": observation.OrderingHeadsIndexBytes,
		"redrives/total":         observation.RedrivesBytes,
		"redrives/indexes":       observation.RedrivesIndexBytes,
	}) {
		t.Fatalf("database-global metrics do not match observation: %+v", first)
	}
}

func TestPostgresOutboxTelemetryTransitions(t *testing.T) {
	ctx, pool, store, reader, telemetry := newInstrumentedOutboxFixture(t)

	mustAppendOutbox(t, ctx, pool, store, outboxEvent("telemetry-recovery"))
	first, err := claimOutboxEvent(ctx, store, shortOutboxLease)
	if err != nil {
		t.Fatalf("claim recovery fixture: %v", err)
	}
	expireOutboxLease(t, ctx, pool)
	recovered := mustClaimOutbox(t, ctx, store)
	if recovered.Event.ID != first.Event.ID || !recovered.Recovered {
		t.Fatalf("recovered claim = id %q recovered %t, want %q/true", recovered.Event.ID, recovered.Recovered, first.Event.ID)
	}
	if err := markOutboxPublished(ctx, store, recovered); err != nil {
		t.Fatalf("mark recovered event published: %v", err)
	}

	mustAppendOutbox(t, ctx, pool, store, outboxEvent("telemetry-poison"))
	poison := mustClaimOutbox(t, ctx, store)
	if err := scheduleOutboxRetry(ctx, store, poison.Event.ID, poison.Token, "publisher_temporary", 0); err != nil {
		t.Fatalf("schedule telemetry retry: %v", err)
	}
	poison = mustClaimOutbox(t, ctx, store)
	if err := poisonOutboxEvent(ctx, store, poison.Event.ID, poison.Token, "publisher_permanent"); err != nil {
		t.Fatalf("mark telemetry poison: %v", err)
	}
	if err := store.Redrive(ctx, poison.Event.ID, "telemetry-redrive"); err != nil {
		t.Fatalf("redrive telemetry poison: %v", err)
	}
	poison = mustClaimOutbox(t, ctx, store)
	if err := markOutboxPublished(ctx, store, poison); err != nil {
		t.Fatalf("mark redriven event published: %v", err)
	}
	if _, err := pool.PGX().Exec(ctx, "UPDATE outbox_events SET published_at = clock_timestamp() - interval '2 hours'"); err != nil {
		t.Fatalf("age telemetry cleanup rows: %v", err)
	}
	if deleted, err := store.CleanupPublished(ctx, time.Hour, 10); err != nil || deleted != 2 {
		t.Fatalf("CleanupPublished() = %d, %v; want 2, nil", deleted, err)
	}

	mustAppendOutbox(t, ctx, pool, store, outboxEvent("telemetry-relay"))
	publisher, publishStarted, publishRelease, _ := gatingPublisher()
	relay := mustNewOutboxRelay(t, store, publisher, telemetry, testRelayConfig())
	result := runOutboxRelay(ctx, relay)
	<-publishStarted
	during := collectOutboxProcessMetrics(t, reader)
	if during.ready != 1 || during.inflight != 1 || during.lastProgress != 0 {
		t.Fatalf("during publish ready/inflight/progress = %d/%d/%f, want 1/1/0", during.ready, during.inflight, during.lastProgress)
	}
	close(publishRelease)
	waitForOutboxCount(t, ctx, pool, "published_at IS NOT NULL", 1)
	relay.StartDrain()
	assertRelayResult(t, result, nil)
	after := collectOutboxProcessMetrics(t, reader)
	if after.ready != 0 || after.inflight != 0 || after.lastProgress == 0 {
		t.Fatalf("after durable mark ready/inflight/progress = %d/%d/%f, want 0/0/>0", after.ready, after.inflight, after.lastProgress)
	}

	observation, err := store.Observe(ctx)
	if err != nil {
		t.Fatalf("Observe(): %v", err)
	}
	telemetry.RecordObservation(observation, time.Now())
	operations, durations := collectOutboxOperationMetrics(t, reader)
	// A timed operation measures one statement or one event end to end. A
	// counted one has no span of its own and must stay out of the duration
	// histogram, or the placeholder it would record lands beside the real
	// measurements. mark_published is both: the store times its statement and
	// the relay counts a reconciled verdict under the same label.
	for _, operation := range []string{
		"append", "claim", "publish", "mark_published", "schedule_retry",
		"poison", "redrive", "cleanup", "observe",
	} {
		if !operations[operation] || !durations[operation] {
			t.Errorf("timed operation %q counter/duration present = %t/%t, want true/true",
				operation, operations[operation], durations[operation])
		}
	}
	for _, operation := range []string{"recovery", "drain"} {
		if !operations[operation] || durations[operation] {
			t.Errorf("counted operation %q counter/duration present = %t/%t, want true/false",
				operation, operations[operation], durations[operation])
		}
	}

	// What a second replica would report from this same observation is a property
	// of Telemetry rather than of PostgreSQL, and is proven without a container by
	// TestTelemetryReplicasShareTheObservationButNotOperations.
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
	if _, err := pool.PGX().Exec(ctx, "DROP TABLE outbox_events CASCADE"); err != nil {
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

// newOutboxTelemetry builds outbox telemetry over a manual reader, so a test can
// collect what a scrape would see. The scope is postgresoutbox's own, matching
// production — the collectors below read every scope, so an ad-hoc name works too
// and then reads as if it mattered. The reader and every metricdata accessor come
// from telemetrytest, so only the *Telemetry construction belongs here.
func newOutboxTelemetry(t *testing.T) (*sdkmetric.ManualReader, *postgresoutbox.Telemetry) {
	t.Helper()
	reader, meter := telemetrytest.NewManualMeter(t, postgresoutbox.TelemetryScope)
	telemetry, err := postgresoutbox.NewTelemetry(meter, nil)
	if err != nil {
		t.Fatalf("NewTelemetry(): %v", err)
	}
	t.Cleanup(telemetry.Close)
	return reader, telemetry
}

// newInstrumentedOutboxFixture is newOutboxFixture whose store records into the
// returned reader, for the tests that assert on store operations rather than
// only on relay ones.
func newInstrumentedOutboxFixture(
	t *testing.T,
) (context.Context, *postgres.Pool, *postgresoutbox.Store, *sdkmetric.ManualReader, *postgresoutbox.Telemetry) {
	t.Helper()
	ctx, pool, _ := newOutboxFixture(t)
	reader, telemetry := newOutboxTelemetry(t)
	store, err := postgresoutbox.NewStore(pool, telemetry)
	if err != nil {
		t.Fatalf("NewStore(): %v", err)
	}
	return ctx, pool, store, reader, telemetry
}

func newOutboxFixture(t *testing.T) (context.Context, *postgres.Pool, *postgresoutbox.Store) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     3 * time.Second,
		HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns:       32,
		AcquireTimeout:     time.Second,
		ConnMaxLifetime:    time.Hour,
		StatementTimeout:   10 * time.Second,
	})
	if err != nil {
		t.Fatalf("postgres.New(): %v", err)
	}
	t.Cleanup(pool.Close)
	store, err := postgresoutbox.NewStore(pool, nil)
	if err != nil {
		t.Fatalf("postgresoutbox.NewStore(): %v", err)
	}
	return ctx, pool, store
}

func outboxEvent(id string) postgresoutbox.Event {
	return postgresoutbox.Event{
		ID:          id,
		Type:        "example.changed",
		Source:      "integration-test",
		Destination: "events",
		Schema:      "v1",
		OccurredAt:  time.Now().UTC(),
		Payload:     []byte(`{"id":"` + id + `"}`),
		Metadata:    []byte(`{"test":true}`),
	}
}

func orderedEvent(id, key string, sequence int64) postgresoutbox.Event {
	event := outboxEvent(id)
	event.OrderingKey = key
	event.OrderingSequence = sequence
	return event
}

func mustAppendOutbox(t *testing.T, ctx context.Context, pool *postgres.Pool, store *postgresoutbox.Store, event postgresoutbox.Event) {
	t.Helper()
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return store.Append(ctx, tx, event)
	}); err != nil {
		t.Fatalf("append event %q: %v", event.ID, err)
	}
}

func mustClaimOutbox(t *testing.T, ctx context.Context, store *postgresoutbox.Store) outboxClaim {
	t.Helper()
	claim, err := claimOutboxEvent(ctx, store, time.Minute)
	if err != nil {
		t.Fatalf("Claim(): %v", err)
	}
	return claim
}

// outboxClaim is one leased event plus the token that fences it. The relay
// claims a whole batch under one token; these tests assert per-event
// transitions, so they claim batches of one.
type outboxClaim struct {
	Event             postgresoutbox.Event
	Token             string
	CycleAttemptCount int
	TotalAttemptCount int64
	Recovered         bool
}

// errNoOutboxWork reports an empty claim, which the store returns as an
// ordinary empty batch rather than an error.
var errNoOutboxWork = errors.New("outbox has no eligible work")

func claimOutboxEvent(ctx context.Context, store *postgresoutbox.Store, lease time.Duration) (outboxClaim, error) {
	batch, err := store.Claim(ctx, lease, 1)
	if err != nil {
		return outboxClaim{}, err
	}
	if len(batch.Events) == 0 {
		return outboxClaim{}, errNoOutboxWork
	}
	claimed := batch.Events[0]
	return outboxClaim{
		Event:             claimed.Event,
		Token:             batch.Token,
		CycleAttemptCount: claimed.CycleAttemptCount,
		TotalAttemptCount: claimed.TotalAttemptCount,
		Recovered:         claimed.Recovered,
	}, nil
}

// markOutboxPublished finalizes one claimed event through the statement its
// disposition owns. In production the relay picks that statement from the
// bucket Relay.classify sorted the event into, and never from the event itself;
// these tests claim one event at a time and built it, so branching on the key
// here is reading back what the test already decided rather than re-deriving a
// routing rule.
func markOutboxPublished(ctx context.Context, store *postgresoutbox.Store, claim outboxClaim) error {
	if claim.Event.OrderingKey == "" {
		return store.MarkUnorderedPublished(ctx, claim.Token, claim.Event.ID)
	}
	return store.MarkOrderedPublished(ctx, claim.Token, postgresoutbox.OrderedDirective{
		ID:               claim.Event.ID,
		OrderingKey:      claim.Event.OrderingKey,
		OrderingSequence: claim.Event.OrderingSequence,
	})
}

func scheduleOutboxRetry(
	ctx context.Context,
	store *postgresoutbox.Store,
	id, token, errorClass string,
	delay time.Duration,
) error {
	return store.ScheduleRetryBatch(ctx, token, []postgresoutbox.RetryDirective{
		{ID: id, ErrorClass: errorClass, Delay: delay},
	})
}

func poisonOutboxEvent(ctx context.Context, store *postgresoutbox.Store, id, token, errorClass string) error {
	return store.MarkPoisonedBatch(ctx, token, []postgresoutbox.PoisonDirective{
		{ID: id, ErrorClass: errorClass},
	})
}

func assertAtomicCounts(t *testing.T, ctx context.Context, pool *postgres.Pool, id string, wantDomain, wantOutbox int) {
	t.Helper()
	var domainCount, outboxCount int
	if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM outbox_domain_probe WHERE id = $1", id).Scan(&domainCount); err != nil {
		t.Fatalf("count domain rows for %s: %v", id, err)
	}
	if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM outbox_events WHERE id = $1", id).Scan(&outboxCount); err != nil {
		t.Fatalf("count outbox rows for %s: %v", id, err)
	}
	if domainCount != wantDomain || outboxCount != wantOutbox {
		t.Fatalf("counts for %s = domain %d outbox %d, want %d/%d", id, domainCount, outboxCount, wantDomain, wantOutbox)
	}
}

func assertOutboxCount(t *testing.T, ctx context.Context, pool *postgres.Pool, id string, want int) {
	t.Helper()
	var count int
	if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM outbox_events WHERE id = $1", id).Scan(&count); err != nil {
		t.Fatalf("count outbox rows: %v", err)
	}
	if count != want {
		t.Fatalf("outbox rows for %s = %d, want %d", id, count, want)
	}
}

func assertTotalOutboxCount(t *testing.T, ctx context.Context, pool *postgres.Pool, want int) {
	t.Helper()
	var count int
	if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM outbox_events").Scan(&count); err != nil {
		t.Fatalf("count all outbox rows: %v", err)
	}
	if count != want {
		t.Fatalf("outbox row count = %d, want %d", count, want)
	}
}

func postgresSleep(t *testing.T, ctx context.Context, pool *postgres.Pool, seconds float64) {
	t.Helper()
	if _, err := pool.PGX().Exec(ctx, "SELECT pg_sleep($1)", seconds); err != nil {
		t.Fatalf("PostgreSQL sleep: %v", err)
	}
}

type testPublisherFunc func(context.Context, postgresoutbox.Event) error

func (publish testPublisherFunc) Publish(ctx context.Context, event postgresoutbox.Event) error {
	return publish(ctx, event)
}

func testRelayConfig() postgresoutbox.RelayConfig {
	return postgresoutbox.RelayConfig{
		PollInterval:        time.Millisecond,
		BatchSize:           100,
		PublishConcurrency:  16,
		PublishTimeout:      20 * time.Millisecond,
		LeaseDuration:       2 * time.Second,
		MaxAttempts:         10,
		RetryBase:           time.Second,
		RetryMax:            5 * time.Minute,
		ObservationInterval: time.Hour,
		CleanupInterval:     time.Hour,
		PublishedRetention:  7 * 24 * time.Hour,
		CleanupBatchSize:    1000,
	}
}

// singleEventRelayConfig keeps one event per claim so a test can distinguish
// the attempt in flight from the claim that would follow it.
func singleEventRelayConfig() postgresoutbox.RelayConfig {
	config := testRelayConfig()
	config.BatchSize = 1
	return config
}

// gatingPublisher hands the test control of when one publication completes:
// started closes as the call begins, and the call returns once the test closes
// release.
//
// It publishes exactly once: started is a close, not a send, so a second
// concurrent call panics and surfaces as a confusing publisher_panic. Pair it
// with singleEventRelayConfig and one appended event, or with a test that stops
// the relay before it can claim again.
func gatingPublisher() (publisher testPublisherFunc, started <-chan struct{}, release chan<- struct{}, attempts *atomic.Int64) {
	begun := make(chan struct{})
	gate := make(chan struct{})
	var calls atomic.Int64
	return testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
		calls.Add(1)
		close(begun)
		<-gate
		return nil
	}), begun, gate, &calls
}

func mustNewOutboxRelay(
	t *testing.T,
	store *postgresoutbox.Store,
	publisher postgresoutbox.Publisher,
	telemetry *postgresoutbox.Telemetry,
	config postgresoutbox.RelayConfig,
) *postgresoutbox.Relay {
	t.Helper()
	relay, err := postgresoutbox.NewRelay(store, publisher, telemetry, config)
	if err != nil {
		t.Fatalf("NewRelay(): %v", err)
	}
	return relay
}

func runOutboxRelay(ctx context.Context, relay *postgresoutbox.Relay) <-chan postgresoutbox.RelayResult {
	result := make(chan postgresoutbox.RelayResult, 1)
	go func() { result <- relay.Run(ctx) }()
	return result
}

// assertRelayResult is the ordinary stop: whatever the relay reported, its
// dependencies are still safe to close. Only a stuck publisher says otherwise,
// which assertRelayStuckResult asserts instead.
func assertRelayResult(t *testing.T, result <-chan postgresoutbox.RelayResult, wantErr error) {
	t.Helper()
	got := readRelayResult(t, result)
	if got.CleanupUnsafe {
		t.Fatalf("Relay.Run() = %+v, want cleanup to stay safe", got)
	}
	if !errors.Is(got.Err, wantErr) {
		t.Fatalf("Relay.Run() error = %v, want %v", got.Err, wantErr)
	}
}

func assertRelayStuckResult(t *testing.T, result <-chan postgresoutbox.RelayResult, wantErr error) {
	t.Helper()
	got := readRelayResult(t, result)
	if !got.CleanupUnsafe {
		t.Fatalf("Relay.Run() = %+v, want cleanup reported unsafe", got)
	}
	if !errors.Is(got.Err, wantErr) {
		t.Fatalf("Relay.Run() error = %v, want %v", got.Err, wantErr)
	}
}

func readRelayResult(t *testing.T, result <-chan postgresoutbox.RelayResult) postgresoutbox.RelayResult {
	t.Helper()
	select {
	case got := <-result:
		return got
	case <-time.After(outboxWaitTimeout):
		t.Fatal("Relay.Run() did not stop")
	}
	return postgresoutbox.RelayResult{}
}

// outboxWaitTimeout and outboxPollInterval are the one cadence every wait in
// this file uses — including the two selects that wait on a channel rather than
// poll. A literal duration here is drift: it once reached six copies, one of
// them ticking at 1ms.
//
// The timeout is an outer failure diagnostic, not a synchronization mechanism.
// Every wait is on an owned event or a durable state change; this only bounds
// how long a broken one hangs.
const (
	outboxWaitTimeout  = 10 * time.Second
	outboxPollInterval = 10 * time.Millisecond
)

// shortOutboxLease is the lease a recovery test claims under, and
// expireOutboxLease is how it lets that lease lapse. Kept together because the
// sleep must outlast the lease and the two are otherwise unrelated numbers in
// unrelated units — one a Duration, one a float of PostgreSQL seconds.
const shortOutboxLease = 5 * time.Millisecond

func expireOutboxLease(t *testing.T, ctx context.Context, pool *postgres.Pool) {
	t.Helper()
	postgresSleep(t, ctx, pool, 4*shortOutboxLease.Seconds())
}

// waitForOutbox polls condition until it holds. describe is called only at the
// deadline, so a failure can report the value that was actually last observed
// rather than restating what was wanted.
func waitForOutbox(t *testing.T, describe func() string, condition func() bool) {
	t.Helper()
	deadline := time.NewTimer(outboxWaitTimeout)
	ticker := time.NewTicker(outboxPollInterval)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		if condition() {
			return
		}
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatal(describe())
		}
	}
}

// outboxBlockedBy reports whether any backend is waiting on a lock that the
// given backend holds.
func outboxBlockedBy(t *testing.T, ctx context.Context, pool *postgres.Pool, blockerPID int) bool {
	t.Helper()
	var blocked bool
	if err := pool.PGX().QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_stat_activity AS activity
			WHERE $1 = ANY(pg_blocking_pids(activity.pid))
		)`, blockerPID).Scan(&blocked); err != nil {
		t.Fatalf("observe blocked mark: %v", err)
	}
	return blocked
}

// outboxBackendExists asks pg_stat_activity about the relay's own connections,
// which are always a different backend from the one running the query.
func outboxBackendExists(t *testing.T, ctx context.Context, pool *postgres.Pool, predicate string) bool {
	t.Helper()
	var found bool
	query := "SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pid <> pg_backend_pid() AND " + predicate + ")"
	if err := pool.PGX().QueryRow(ctx, query).Scan(&found); err != nil {
		t.Fatalf("inspect pg_stat_activity for %s: %v", predicate, err)
	}
	return found
}

func waitForOutboxReady(t *testing.T, relay *postgresoutbox.Relay) {
	t.Helper()
	waitForOutbox(t, func() string { return "relay did not become ready" }, relay.Ready)
}

func waitForOutboxCount(t *testing.T, ctx context.Context, pool *postgres.Pool, predicate string, want int) {
	t.Helper()
	var count int
	waitForOutbox(t,
		func() string { return fmt.Sprintf("outbox count for %q = %d, want %d", predicate, count, want) },
		func() bool {
			if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM outbox_events WHERE "+predicate).Scan(&count); err != nil {
				t.Fatalf("count outbox state: %v", err)
			}
			return count == want
		})
}

func waitForOutboxListener(t *testing.T, ctx context.Context, pool *postgres.Pool) {
	t.Helper()
	waitForOutbox(t,
		func() string { return "outbox append listener did not subscribe" },
		func() bool {
			return outboxBackendExists(t, ctx, pool, "query LIKE 'LISTEN %outbox_appended%'")
		})
}

func outboxOperationCount(t *testing.T, reader *sdkmetric.ManualReader, operation, outcome string) int64 {
	t.Helper()
	var count int64
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		if measured.Name != "outbox.relay.operations" {
			return
		}
		for _, point := range telemetrytest.Int64Sum(t, measured).DataPoints {
			if telemetrytest.Attribute(t, point.Attributes, "operation") == operation &&
				telemetrytest.Attribute(t, point.Attributes, "outcome") == outcome {
				count = point.Value
			}
		}
	})
	return count
}

func waitForOutboxOperationCount(t *testing.T, reader *sdkmetric.ManualReader, operation, outcome string, want int64) {
	t.Helper()
	var got int64
	waitForOutbox(t,
		func() string {
			return fmt.Sprintf("outbox operation %s/%s reached %d, want %d", operation, outcome, got, want)
		},
		func() bool {
			got = outboxOperationCount(t, reader, operation, outcome)
			return got >= want
		})
}

// observationStatementProbe identifies the state observation inside
// pg_stat_activity.query, which is the statement text PostgreSQL echoes back.
//
// It matches the sqlc header rather than anything the statement does, because
// pg_stat_activity.query is truncated at track_activity_query_size — 1024 bytes
// by default, and ObserveOutbox is far longer. sqlc keeps the `-- name: X :type`
// header as the literal first line of each generated query constant, which makes
// it the only anchor guaranteed inside that window.
//
// So this stays coupled to the query's sqlc name: renaming ObserveOutbox, or
// configuring sqlc to drop those headers, stops it matching — which is why the
// timeout message below tells the two failures apart. Keep it in step with
// internal/infra/postgres/queries/postgres_outbox.sql.
const observationStatementProbe = "query LIKE '%name: ObserveOutbox :one%'"

// waitForBlockedOutboxObservation waits until the relay's observation is parked
// on a lock.
func waitForBlockedOutboxObservation(t *testing.T, ctx context.Context, pool *postgres.Pool) {
	t.Helper()
	waitForOutbox(t,
		func() string {
			// Evaluated only at the deadline. If nothing anywhere matches the
			// probe, the observation is not what failed to block — the probe no
			// longer names it.
			if !outboxBackendExists(t, ctx, pool, observationStatementProbe) {
				return "no backend is running a statement matching " + observationStatementProbe +
					"; the ObserveOutbox query was probably renamed, so this probe no longer finds it"
			}
			return "relay observation did not block behind the maintenance gate"
		},
		func() bool {
			return outboxBackendExists(t, ctx, pool,
				"wait_event_type = 'Lock' AND "+observationStatementProbe)
		})
}

type outboxProcessMetrics struct {
	ready        int64
	inflight     int64
	observedAt   float64
	lastProgress float64
}

// collectOutboxProcessMetrics reads the four gauges that describe the relay
// process itself. It requires all four rather than returning the zero value for
// a missing one: callers assert ready=0 and inflight=0 to mean the relay
// reported itself stopped and idle, and a gauge that was never published would
// satisfy exactly those assertions while proving nothing.
func collectOutboxProcessMetrics(t *testing.T, reader *sdkmetric.ManualReader) outboxProcessMetrics {
	t.Helper()
	var result outboxProcessMetrics
	seen := map[string]bool{}
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		switch measured.Name {
		case "outbox.relay.readiness":
			result.ready = telemetrytest.SinglePoint(t, measured.Name, telemetrytest.Int64Gauge(t, measured).DataPoints)
		case "outbox.relay.inflight":
			result.inflight = telemetrytest.SinglePoint(t, measured.Name, telemetrytest.Int64Gauge(t, measured).DataPoints)
		case "outbox.relay.observation.timestamp":
			result.observedAt = telemetrytest.SinglePoint(t, measured.Name, telemetrytest.Float64Gauge(t, measured).DataPoints)
		case "outbox.relay.last_progress.timestamp":
			result.lastProgress = telemetrytest.SinglePoint(t, measured.Name, telemetrytest.Float64Gauge(t, measured).DataPoints)
		default:
			return
		}
		seen[measured.Name] = true
	})
	for _, required := range []string{
		"outbox.relay.readiness", "outbox.relay.inflight",
		"outbox.relay.observation.timestamp", "outbox.relay.last_progress.timestamp",
	} {
		if !seen[required] {
			t.Fatalf("process metric %s was not collected; a zero here would read as a real value", required)
		}
	}
	return result
}

func collectOutboxOperationMetrics(t *testing.T, reader *sdkmetric.ManualReader) (map[string]bool, map[string]bool) {
	t.Helper()
	operations := make(map[string]bool)
	durations := make(map[string]bool)
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		switch measured.Name {
		case "outbox.relay.operations":
			for _, point := range telemetrytest.Int64Sum(t, measured).DataPoints {
				if point.Value > 0 {
					operations[telemetrytest.Attribute(t, point.Attributes, "operation")] = true
				}
			}
		case "outbox.relay.operation.duration":
			for _, point := range telemetrytest.Float64Histogram(t, measured).DataPoints {
				if point.Count > 0 {
					durations[telemetrytest.Attribute(t, point.Attributes, "operation")] = true
				}
			}
		}
	})
	return operations, durations
}

func collectOutboxDatabaseMetrics(t *testing.T, reader *sdkmetric.ManualReader) outboxMetricSnapshot {
	t.Helper()
	result := outboxMetricSnapshot{
		counts:  make(map[string]int64),
		oldest:  make(map[string]float64),
		storage: make(map[string]int64),
	}
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		switch measured.Name {
		case "outbox.relay.messages":
			for _, point := range telemetrytest.Int64Gauge(t, measured).DataPoints {
				result.counts[telemetrytest.Attribute(t, point.Attributes, "state")] = point.Value
			}
		case "outbox.relay.oldest.timestamp":
			for _, point := range telemetrytest.Float64Gauge(t, measured).DataPoints {
				result.oldest[telemetrytest.Attribute(t, point.Attributes, "state")] = point.Value
			}
		case "outbox.relay.ordering_heads":
			result.orderingHeads = telemetrytest.SinglePoint(t, measured.Name, telemetrytest.Int64Gauge(t, measured).DataPoints)
		case "outbox.relay.storage.bytes":
			for _, point := range telemetrytest.Int64Gauge(t, measured).DataPoints {
				key := telemetrytest.Attribute(t, point.Attributes, "relation") + "/" + telemetrytest.Attribute(t, point.Attributes, "kind")
				result.storage[key] = point.Value
			}
		}
	})
	return result
}

func assertOutboxObservationMatchesSQL(
	t *testing.T,
	ctx context.Context,
	pool *postgres.Pool,
	observation postgresoutbox.StateObservation,
) {
	t.Helper()
	oldestByID := []struct {
		id  string
		got time.Time
	}{
		{id: "observe-eligible", got: observation.EligibleOldestAt},
		{id: "observe-in-progress", got: observation.InProgressOldestAt},
		{id: "observe-retry-wait", got: observation.RetryWaitOldestAt},
		{id: "observe-recovery-due", got: observation.RecoveryDueOldestAt},
		{id: "observe-ordering-blocked", got: observation.OrderingBlockedOldestAt},
		{id: "observe-poison", got: observation.PoisonOldestAt},
	}
	for _, state := range oldestByID {
		var want time.Time
		if err := pool.PGX().QueryRow(ctx, "SELECT created_at FROM outbox_events WHERE id = $1", state.id).Scan(&want); err != nil {
			t.Fatalf("read direct oldest timestamp for %s: %v", state.id, err)
		}
		if delta := state.got.Sub(want.UTC()); delta < -time.Microsecond || delta > time.Microsecond {
			t.Fatalf("oldest timestamp for %s = %v, direct SQL %v", state.id, state.got, want)
		}
	}
	var publishedAt time.Time
	if err := pool.PGX().QueryRow(ctx, "SELECT published_at FROM outbox_events WHERE id = 'observe-published'").Scan(&publishedAt); err != nil {
		t.Fatalf("read direct published oldest timestamp: %v", err)
	}
	if delta := observation.PublishedRetainedOldestAt.Sub(publishedAt.UTC()); delta < -time.Microsecond || delta > time.Microsecond {
		t.Fatalf("published oldest = %v, direct SQL %v", observation.PublishedRetainedOldestAt, publishedAt)
	}

	var heads, eventsBytes, eventIndexes, headBytes, headIndexes, redriveBytes, redriveIndexes int64
	if err := pool.PGX().QueryRow(ctx, `SELECT
		(SELECT count(*) FROM outbox_ordering_heads),
		pg_total_relation_size('outbox_events'), pg_indexes_size('outbox_events'),
		pg_total_relation_size('outbox_ordering_heads'), pg_indexes_size('outbox_ordering_heads'),
		pg_total_relation_size('outbox_redrives'), pg_indexes_size('outbox_redrives')`).Scan(
		&heads, &eventsBytes, &eventIndexes, &headBytes, &headIndexes, &redriveBytes, &redriveIndexes,
	); err != nil {
		t.Fatalf("read direct outbox storage: %v", err)
	}
	want := []int64{heads, eventsBytes, eventIndexes, headBytes, headIndexes, redriveBytes, redriveIndexes}
	got := []int64{
		observation.OrderingHeadCount,
		observation.EventsBytes,
		observation.EventsIndexBytes,
		observation.OrderingHeadsBytes,
		observation.OrderingHeadsIndexBytes,
		observation.RedrivesBytes,
		observation.RedrivesIndexBytes,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("observation storage = %v, direct SQL %v", got, want)
	}
}

type outboxMetricSnapshot struct {
	counts        map[string]int64
	oldest        map[string]float64
	storage       map[string]int64
	orderingHeads int64
}

func collectOutboxStateMetrics(t *testing.T, observation postgresoutbox.StateObservation) outboxMetricSnapshot {
	t.Helper()
	reader, telemetry := newOutboxTelemetry(t)
	telemetry.RecordObservation(observation, time.Now())
	// The gauges are read the same way whichever recorder produced them, so the
	// extraction is collectOutboxDatabaseMetrics's; what is specific here is the
	// throwaway recorder above and the timestamp check below.
	result := collectOutboxDatabaseMetrics(t, reader)
	wantOldest := map[string]time.Time{
		"eligible":           observation.EligibleOldestAt,
		"in_progress":        observation.InProgressOldestAt,
		"retry_wait":         observation.RetryWaitOldestAt,
		"recovery_due":       observation.RecoveryDueOldestAt,
		"ordering_blocked":   observation.OrderingBlockedOldestAt,
		"poison":             observation.PoisonOldestAt,
		"published_retained": observation.PublishedRetainedOldestAt,
	}
	for state, oldest := range wantOldest {
		want := float64(oldest.UnixNano()) / float64(time.Second)
		if delta := result.oldest[state] - want; delta < -0.000001 || delta > 0.000001 {
			t.Fatalf("oldest metric %s = %f, want %f", state, result.oldest[state], want)
		}
	}
	return result
}

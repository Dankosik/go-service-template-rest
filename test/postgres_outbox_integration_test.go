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
	"testing/fstest"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresmigrate"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
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

func TestPostgresOutboxOrderingReadyMigrationBackfill(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	initialMigration, err := os.ReadFile("../migrations/000001_postgres_outbox.sql")
	if err != nil {
		t.Fatalf("read initial outbox migration: %v", err)
	}
	dsn := pgtest.Migrated(t, fstest.MapFS{
		"migrations/000001_postgres_outbox.sql": {Data: initialMigration},
	}, "migrations")
	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     3 * time.Second,
		HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns:       4,
		AcquireTimeout:     time.Second,
		ConnMaxLifetime:    time.Hour,
		StatementTimeout:   10 * time.Second,
	})
	if err != nil {
		t.Fatalf("postgres.New(): %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.PGX().Exec(ctx, `
		INSERT INTO outbox_ordering_heads (ordering_key, last_sequence)
		VALUES ('existing-head', 3);
		INSERT INTO outbox_ordering_heads (ordering_key, last_sequence, updated_at)
		VALUES
			('historical-head', 9, '2000-01-01 00:00:00+00'),
			('published-only-head', 11, '2001-01-01 00:00:00+00');
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata,
			ordering_key, ordering_sequence, published_at, poisoned_at
		) VALUES
			('migrated-unordered', 't', 's', 'd', 'v', statement_timestamp(),
				convert_to('{}', 'UTF8'), convert_to('{}', 'UTF8'), NULL, NULL, NULL, NULL),
			('migrated-published', 't', 's', 'd', 'v', statement_timestamp(),
				convert_to('{}', 'UTF8'), convert_to('{}', 'UTF8'), 'existing-head', 1, statement_timestamp(), NULL),
			('migrated-poison', 't', 's', 'd', 'v', statement_timestamp(),
				convert_to('{}', 'UTF8'), convert_to('{}', 'UTF8'), 'existing-head', 2, NULL, statement_timestamp()),
			('migrated-blocked', 't', 's', 'd', 'v', statement_timestamp(),
				convert_to('{}', 'UTF8'), convert_to('{}', 'UTF8'), 'existing-head', 3, NULL, NULL),
			('migrated-missing-head', 't', 's', 'd', 'v', statement_timestamp(),
				convert_to('{}', 'UTF8'), convert_to('{}', 'UTF8'), 'missing-head', 7, NULL, NULL),
			('migrated-published-only', 't', 's', 'd', 'v', statement_timestamp(),
				convert_to('{}', 'UTF8'), convert_to('{}', 'UTF8'), 'published-only-head', 11,
				statement_timestamp(), NULL)`); err != nil {
		t.Fatalf("seed pre-remediation outbox rows: %v", err)
	}

	if _, err := postgresmigrate.MigrateUp(ctx, postgresmigrate.MigrationOptions{
		DSN:              dsn,
		SourceFS:         os.DirFS(".."),
		SourcePath:       "migrations",
		ConnectTimeout:   3 * time.Second,
		StatementTimeout: time.Minute,
		LockTimeout:      15 * time.Second,
		CleanupTimeout:   15 * time.Second,
	}); err != nil {
		t.Fatalf("apply ordering-ready migration: %v", err)
	}

	wantReady := map[string]bool{
		"migrated-unordered":      false,
		"migrated-published":      false,
		"migrated-poison":         true,
		"migrated-blocked":        false,
		"migrated-missing-head":   true,
		"migrated-published-only": false,
	}
	rows, err := pool.PGX().Query(ctx, "SELECT id, ordering_ready FROM outbox_events")
	if err != nil {
		t.Fatalf("read migrated readiness: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var ready bool
		if err := rows.Scan(&id, &ready); err != nil {
			t.Fatalf("scan migrated readiness: %v", err)
		}
		if want, ok := wantReady[id]; !ok || ready != want {
			t.Fatalf("migrated readiness %s = %t, want %t", id, ready, want)
		}
		delete(wantReady, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate migrated readiness: %v", err)
	}
	if len(wantReady) != 0 {
		t.Fatalf("missing migrated readiness rows: %v", wantReady)
	}
	for key, want := range map[string]int64{"existing-head": 2, "missing-head": 7} {
		var current int64
		if err := pool.PGX().QueryRow(ctx,
			"SELECT current_sequence FROM outbox_ordering_heads WHERE ordering_key = $1", key,
		).Scan(&current); err != nil {
			t.Fatalf("read migrated head %s: %v", key, err)
		}
		if current != want {
			t.Fatalf("migrated head %s = %d, want %d", key, current, want)
		}
	}
	var historicalCurrent *int64
	var historicalUpdated time.Time
	if err := pool.PGX().QueryRow(ctx, `
		SELECT current_sequence, updated_at
		FROM outbox_ordering_heads
		WHERE ordering_key = 'historical-head'`).Scan(&historicalCurrent, &historicalUpdated); err != nil {
		t.Fatalf("read historical head: %v", err)
	}
	if historicalCurrent != nil || !historicalUpdated.Equal(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("historical head was rewritten: current=%v updated=%s", historicalCurrent, historicalUpdated)
	}
	var publishedOnlyCurrent *int64
	var publishedOnlyUpdated time.Time
	if err := pool.PGX().QueryRow(ctx, `
		SELECT current_sequence, updated_at
		FROM outbox_ordering_heads
		WHERE ordering_key = 'published-only-head'`).Scan(&publishedOnlyCurrent, &publishedOnlyUpdated); err != nil {
		t.Fatalf("read published-only head: %v", err)
	}
	if publishedOnlyCurrent != nil || !publishedOnlyUpdated.Equal(time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("published-only head was rewritten: current=%v updated=%s", publishedOnlyCurrent, publishedOnlyUpdated)
	}

	invariantRows, err := pool.PGX().Query(ctx, `
		WITH expected AS (
			SELECT event.ordering_key, min(event.ordering_sequence) AS current_sequence
			FROM outbox_events AS event
			WHERE event.ordering_key IS NOT NULL AND event.published_at IS NULL
			GROUP BY event.ordering_key
		)
		SELECT coalesce(expected.ordering_key, head.ordering_key) AS ordering_key
		FROM expected
		FULL JOIN outbox_ordering_heads AS head
		  ON head.ordering_key = expected.ordering_key
		LEFT JOIN outbox_events AS event
		  ON event.ordering_key = expected.ordering_key
		 AND event.ordering_sequence = expected.current_sequence
		 AND event.published_at IS NULL
		 AND event.ordering_ready
		WHERE head.current_sequence IS DISTINCT FROM expected.current_sequence
		   OR (expected.ordering_key IS NOT NULL AND event.id IS NULL)`)
	if err != nil {
		t.Fatalf("run ordering-head invariant readback: %v", err)
	}
	if invariantRows.Next() {
		var key string
		if err := invariantRows.Scan(&key); err != nil {
			t.Fatalf("scan ordering-head invariant violation: %v", err)
		}
		t.Fatalf("ordering-head invariant violation for %q", key)
	}
	if err := invariantRows.Err(); err != nil {
		t.Fatalf("iterate ordering-head invariant readback: %v", err)
	}
	invariantRows.Close()

	if _, err := pool.PGX().Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata
		) VALUES (
			'jsonb-incompatible', 't', 's', 'd', 'v', statement_timestamp(),
			convert_to('1e1000000', 'UTF8'), convert_to('{"nul":"\u0000"}', 'UTF8')
		)`); err != nil {
		t.Fatalf("insert jsonb-incompatible event: %v", err)
	}
	if _, err := postgresmigrate.MigrateDown(ctx, postgresmigrate.MigrationOptions{
		DSN:              dsn,
		SourceFS:         os.DirFS(".."),
		SourcePath:       "migrations",
		ConnectTimeout:   3 * time.Second,
		StatementTimeout: time.Minute,
		LockTimeout:      15 * time.Second,
		CleanupTimeout:   15 * time.Second,
	}); err == nil || !strings.Contains(err.Error(), "jsonb-incompatible event bytes exist") {
		t.Fatalf("MigrateDown() error = %v, want jsonb incompatibility fence", err)
	}
	var edgeCount int
	if err := pool.PGX().QueryRow(ctx, `
		SELECT count(*)
		FROM outbox_events
		WHERE id = 'jsonb-incompatible' AND ordering_ready = false`).Scan(&edgeCount); err != nil {
		t.Fatalf("verify failed Down rollback: %v", err)
	}
	if edgeCount != 1 {
		t.Fatalf("failed Down changed edge rows: count = %d, want 1", edgeCount)
	}
}

func TestPostgresOutboxOrderingReadyMigrationHighCardinality(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	initialMigration, err := os.ReadFile("../migrations/000001_postgres_outbox.sql")
	if err != nil {
		t.Fatalf("read initial outbox migration: %v", err)
	}
	dsn := pgtest.Migrated(t, fstest.MapFS{
		"migrations/000001_postgres_outbox.sql": {Data: initialMigration},
	}, "migrations")
	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     3 * time.Second,
		HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns:       2,
		AcquireTimeout:     time.Second,
		ConnMaxLifetime:    time.Hour,
		StatementTimeout:   10 * time.Second,
	})
	if err != nil {
		t.Fatalf("postgres.New(): %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.PGX().Exec(ctx, `
		INSERT INTO outbox_ordering_heads (ordering_key, last_sequence)
		SELECT 'key-' || sequence, 1
		FROM generate_series(1, 3) AS sequence;
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata,
			ordering_key, ordering_sequence
		)
		SELECT 'event-' || sequence, 't', 's', 'd', 'v', statement_timestamp(),
			convert_to('{}', 'UTF8'), convert_to('{}', 'UTF8'), 'key-' || sequence, 1
		FROM generate_series(1, 3) AS sequence`); err != nil {
		t.Fatalf("seed high-cardinality migration rows: %v", err)
	}

	if _, err := postgresmigrate.MigrateUp(ctx, postgresmigrate.MigrationOptions{
		DSN:              dsn,
		SourceFS:         os.DirFS(".."),
		SourcePath:       "migrations",
		ConnectTimeout:   3 * time.Second,
		StatementTimeout: time.Minute,
		LockTimeout:      15 * time.Second,
		CleanupTimeout:   15 * time.Second,
	}); err != nil {
		t.Fatalf("apply high-cardinality ordering-ready migration: %v", err)
	}

	if _, err := pool.PGX().Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata
		) VALUES (
			'post-migration-default', 't', 's', 'd', 'v', statement_timestamp(),
			convert_to('{}', 'UTF8'), convert_to('{}', 'UTF8')
		)`); err != nil {
		t.Fatalf("insert post-migration default row: %v", err)
	}
	var ready, notReady int
	if err := pool.PGX().QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE ordering_ready),
		       count(*) FILTER (WHERE NOT ordering_ready)
		FROM outbox_events`).Scan(&ready, &notReady); err != nil {
		t.Fatalf("read high-cardinality readiness: %v", err)
	}
	if ready != 3 || notReady != 1 {
		t.Fatalf("high-cardinality readiness = %d/%d, want 3/1", ready, notReady)
	}
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
		claim, err := store.Claim(ctx, time.Minute)
		if err != nil {
			t.Fatalf("Claim() for %s: %v", wantID, err)
		}
		if claim.Event.ID != wantID {
			t.Fatalf("claimed %q, want %q", claim.Event.ID, wantID)
		}
		if err := store.MarkPublished(ctx, claim); err != nil {
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

func TestPostgresOutboxConcurrentClaims(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	const eventCount = 24
	for i := range eventCount {
		mustAppendOutbox(t, ctx, pool, store, outboxEvent(fmt.Sprintf("concurrent-%02d", i)))
	}

	start := make(chan struct{})
	results := make(chan postgresoutbox.ClaimedEvent, eventCount)
	errs := make(chan error, eventCount)
	var workers sync.WaitGroup
	for range eventCount {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			claim, err := store.Claim(ctx, time.Minute)
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
	claim, err := store.Claim(ctx, time.Minute)
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
			errs <- store.MarkPublished(ctx, first)
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
		if err := store.MarkPublished(ctx, second); err != nil {
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
	go func() { marked <- store.MarkPublished(ctx, first) }()
	deadline := time.NewTimer(10 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		var blocked bool
		if err := pool.PGX().QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_stat_activity AS activity
				WHERE $1 = ANY(pg_blocking_pids(activity.pid))
			)`, blockerPID).Scan(&blocked); err != nil {
			t.Fatalf("observe blocked mark: %v", err)
		}
		if blocked {
			break
		}
		select {
		case err := <-marked:
			t.Fatalf("mark completed before append commit: %v", err)
		case <-deadline.C:
			t.Fatal("mark did not wait for the ordering head lock")
		case <-ticker.C:
		}
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit successor: %v", err)
	}
	if err := <-marked; err != nil {
		if !errors.Is(err, postgresoutbox.ErrLeaseLost) {
			t.Fatalf("mark after blocked snapshot: %v", err)
		}
		if err := store.MarkPublished(ctx, first); err != nil {
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
	if err := store.MarkPublished(ctx, mutated); !errors.Is(err, postgresoutbox.ErrLeaseLost) {
		t.Fatalf("mutated claim mark = %v, want ErrLeaseLost", err)
	}
	record, err := store.Get(ctx, claim.Event.ID)
	if err != nil {
		t.Fatalf("Get(%s): %v", claim.Event.ID, err)
	}
	if !record.PublishedAt.IsZero() {
		t.Fatal("mutated ordered claim marked the original event published")
	}
	if err := store.MarkPublished(ctx, claim); err != nil {
		t.Fatalf("mark original claim: %v", err)
	}
}

func TestPostgresOutboxOrderingClaims(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("key-1", "key", 1))
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("key-2", "key", 2))
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("key-3", "key", 3))
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("unordered"))

	first := mustClaimOutbox(t, ctx, store)
	if first.Event.ID != "key-1" {
		t.Fatalf("first claim = %q, want key-1", first.Event.ID)
	}
	second := mustClaimOutbox(t, ctx, store)
	if second.Event.ID != "unordered" {
		t.Fatalf("claim while key predecessor leased = %q, want unordered", second.Event.ID)
	}
	if err := store.MarkPublished(ctx, second); err != nil {
		t.Fatalf("publish unordered: %v", err)
	}
	if err := store.MarkPublished(ctx, first); err != nil {
		t.Fatalf("publish key-1: %v", err)
	}
	third := mustClaimOutbox(t, ctx, store)
	if third.Event.ID != "key-2" {
		t.Fatalf("post-predecessor claim = %q, want key-2", third.Event.ID)
	}
	if err := store.ScheduleRetry(ctx, third.Event.ID, third.Token, "publisher_temporary", time.Hour); err != nil {
		t.Fatalf("schedule key-2 retry: %v", err)
	}
	if _, err := store.Claim(ctx, time.Minute); !errors.Is(err, postgresoutbox.ErrNoWork) {
		t.Fatalf("Claim() behind retry-wait predecessor = %v, want ErrNoWork", err)
	}
	if _, err := pool.PGX().Exec(ctx, "UPDATE outbox_events SET available_at = clock_timestamp() WHERE id = 'key-2'"); err != nil {
		t.Fatalf("make key-2 retry eligible: %v", err)
	}
	retryClaim, err := store.Claim(ctx, 5*time.Millisecond)
	if err != nil || retryClaim.Event.ID != "key-2" {
		t.Fatalf("retry claim = %+v, %v; want key-2", retryClaim, err)
	}
	postgresSleep(t, ctx, pool, 0.02)
	lockTx, err := pool.PGX().Begin(ctx)
	if err != nil {
		t.Fatalf("begin recovery predecessor lock: %v", err)
	}
	if _, err := lockTx.Exec(ctx, "SELECT id FROM outbox_events WHERE id = 'key-2' FOR UPDATE"); err != nil {
		_ = lockTx.Rollback(context.WithoutCancel(ctx))
		t.Fatalf("lock recovery predecessor: %v", err)
	}
	if _, err := store.Claim(ctx, time.Minute); !errors.Is(err, postgresoutbox.ErrNoWork) {
		_ = lockTx.Rollback(context.WithoutCancel(ctx))
		t.Fatalf("Claim() around locked recovery predecessor = %v, want ErrNoWork", err)
	}
	if err := lockTx.Rollback(ctx); err != nil {
		t.Fatalf("release recovery predecessor lock: %v", err)
	}
	recoveryClaim, err := store.Claim(ctx, time.Minute)
	if err != nil || recoveryClaim.Event.ID != "key-2" {
		t.Fatalf("recovery claim = %+v, %v; want predecessor key-2", recoveryClaim, err)
	}
	if err := store.MarkPoisoned(ctx, retryClaim.Event.ID, retryClaim.Token, "publisher_permanent"); !errors.Is(err, postgresoutbox.ErrLeaseLost) {
		t.Fatalf("stale poison fence = %v, want ErrLeaseLost", err)
	}
	if err := store.MarkPoisoned(ctx, recoveryClaim.Event.ID, recoveryClaim.Token, "publisher_permanent"); err != nil {
		t.Fatalf("poison recovered key-2: %v", err)
	}
	if _, err := store.Claim(ctx, time.Minute); !errors.Is(err, postgresoutbox.ErrNoWork) {
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
	first, err := store.Claim(ctx, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("first Claim(): %v", err)
	}
	postgresSleep(t, ctx, pool, 0.02)
	second, err := store.Claim(ctx, time.Minute)
	if err != nil {
		t.Fatalf("recovery Claim(): %v", err)
	}
	if second.Event.ID != first.Event.ID || second.Token == first.Token {
		t.Fatalf("recovery claim id/token = %q/%q, first %q/%q", second.Event.ID, second.Token, first.Event.ID, first.Token)
	}
	if second.CycleAttemptCount != 2 || second.TotalAttemptCount != 2 {
		t.Fatalf("recovery attempts = %d/%d, want 2/2", second.CycleAttemptCount, second.TotalAttemptCount)
	}
	if err := store.MarkPublished(ctx, first); !errors.Is(err, postgresoutbox.ErrLeaseLost) {
		t.Fatalf("stale MarkPublished() = %v, want ErrLeaseLost", err)
	}
	if err := store.ScheduleRetry(ctx, first.Event.ID, first.Token, "publisher_temporary", 0); !errors.Is(err, postgresoutbox.ErrLeaseLost) {
		t.Fatalf("stale ScheduleRetry() = %v, want ErrLeaseLost", err)
	}
	if err := store.MarkPoisoned(ctx, first.Event.ID, first.Token, "publisher_permanent"); !errors.Is(err, postgresoutbox.ErrLeaseLost) {
		t.Fatalf("stale MarkPoisoned() = %v, want ErrLeaseLost", err)
	}
	if err := store.MarkPublished(ctx, second); err != nil {
		t.Fatalf("current MarkPublished(): %v", err)
	}
}

func TestPostgresOutboxCrashAfterClaim(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("abandoned"))
	first, err := store.Claim(ctx, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("Claim(): %v", err)
	}
	store = nil
	postgresSleep(t, ctx, pool, 0.02)
	restarted, err := postgresoutbox.NewStore(pool, nil)
	if err != nil {
		t.Fatalf("NewStore() after abandoned claim: %v", err)
	}
	second, err := restarted.Claim(ctx, time.Minute)
	if err != nil {
		t.Fatalf("Claim() after abandoned claim: %v", err)
	}
	if second.Event.ID != first.Event.ID || second.Token == first.Token {
		t.Fatalf("restarted claim id/token = %q/%q, first %q/%q", second.Event.ID, second.Token, first.Event.ID, first.Token)
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
	if err := store.MarkPoisoned(ctx, claim.Event.ID, claim.Token, "publisher_permanent"); err != nil {
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
	if err := store.MarkPoisoned(ctx, claim.Event.ID, claim.Token, "publisher_permanent"); err != nil {
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
	if err := store.MarkPublished(ctx, other); err != nil {
		t.Fatalf("publish redriven event: %v", err)
	}
	if err := store.Redrive(ctx, other.Event.ID, "audit-3"); !errors.Is(err, postgresoutbox.ErrRedriveRejected) {
		t.Fatalf("redrive published event = %v, want ErrRedriveRejected", err)
	}
	other = mustClaimOutbox(t, ctx, store)
	if err := store.MarkPoisoned(ctx, other.Event.ID, other.Token, "publisher_permanent"); err != nil {
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
		if err := store.MarkPublished(ctx, claim); err != nil {
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
		{name: "permanent rejection", publisherResult: postgresoutbox.ErrPermanentPublication, wantClass: "publisher_permanent", wantPoison: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, pool, store := newOutboxFixture(t)
			mustAppendOutbox(t, ctx, pool, store, outboxEvent("publish-failure"))
			entered := make(chan struct{})
			release := make(chan struct{})
			publisher := testPublisherFunc(func(publishCtx context.Context, _ postgresoutbox.Event) error {
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
			assertRelayResult(t, result, true, nil)

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

	first, err := store.Claim(ctx, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("first Claim(): %v", err)
	}
	if err := publisher.Publish(ctx, first.Event); err != nil {
		t.Fatalf("first durable publish: %v", err)
	}
	firstAttempt := <-attempts
	postgresSleep(t, ctx, pool, 0.02)

	relay := mustNewOutboxRelay(t, store, publisher, nil, testRelayConfig())
	result := runOutboxRelay(ctx, relay)
	secondAttempt := <-attempts
	relay.StartDrain()
	close(releaseSecond)
	assertRelayResult(t, result, true, nil)
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
	deadline := time.NewTimer(10 * time.Second)
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
		assertRelayResult(t, result, true, nil)
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
	assertRelayResult(t, relayResult, true, nil)

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
		relay := mustNewOutboxRelay(t, store, publisher, nil, testRelayConfig())
		result := runOutboxRelay(relayCtx, relay)
		<-started
		if !relay.Ready() {
			t.Fatal("relay not ready during a joined publication attempt")
		}
		cancel()
		assertRelayResult(t, result, true, nil)
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
		relay := mustNewOutboxRelay(t, store, publisher, nil, testRelayConfig())
		assertRelayResult(t, runOutboxRelay(ctx, relay), true, postgresoutbox.ErrPublisherPanic)
		if relay.Ready() || attempts.Load() != 1 {
			t.Fatalf("after panic ready=%t attempts=%d, want false/1", relay.Ready(), attempts.Load())
		}
		waitForOutboxCount(t, ctx, pool, "published_at IS NULL", 2)
	})

	t.Run("stuck publisher is cleanup unsafe and starts no next claim", func(t *testing.T) {
		ctx, pool, store := newOutboxFixture(t)
		mustAppendOutbox(t, ctx, pool, store, outboxEvent("stuck-current"))
		mustAppendOutbox(t, ctx, pool, store, outboxEvent("stuck-next"))
		started := make(chan struct{})
		release := make(chan struct{})
		var attempts atomic.Int64
		publisher := testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
			attempts.Add(1)
			close(started)
			<-release
			return nil
		})
		config := testRelayConfig()
		config.PublishTimeout = time.Millisecond
		relay := mustNewOutboxRelay(t, store, publisher, nil, config)
		result := runOutboxRelay(ctx, relay)
		<-started
		assertRelayResult(t, result, false, postgresoutbox.ErrPublisherStuck)
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
		started := make(chan struct{})
		release := make(chan struct{})
		var attempts atomic.Int64
		publisher := testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
			attempts.Add(1)
			close(started)
			<-release
			return nil
		})
		relay := mustNewOutboxRelay(t, store, publisher, nil, testRelayConfig())
		result := runOutboxRelay(ctx, relay)
		<-started
		relay.StartDrain()
		close(release)
		assertRelayResult(t, result, true, nil)
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
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	telemetry, err := postgresoutbox.NewTelemetry(provider.Meter("maintenance-drain"), nil)
	if err != nil {
		t.Fatalf("NewTelemetry(): %v", err)
	}
	t.Cleanup(telemetry.Close)
	relay := mustNewOutboxRelay(t, store, publisher, telemetry, config)
	result := runOutboxRelay(ctx, relay)
	waitForOutboxOperation(t, reader, "claim", "empty")

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
	assertRelayResult(t, result, true, nil)
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

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	telemetry, err := postgresoutbox.NewTelemetry(provider.Meter("startup-drain"), nil)
	if err != nil {
		t.Fatalf("NewTelemetry(): %v", err)
	}
	t.Cleanup(telemetry.Close)
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
	assertRelayResult(t, result, true, nil)
	if relay.Ready() || attempts.Load() != 0 {
		t.Fatalf("startup drain ready=%t attempts=%d, want false/0", relay.Ready(), attempts.Load())
	}
	process := collectOutboxProcessMetrics(t, reader)
	if process.ready != 0 || process.inflight != 0 {
		t.Fatalf("startup drain telemetry ready/inflight = %d/%d, want 0/0", process.ready, process.inflight)
	}
}

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
	if result.Err == nil || !result.CleanupSafe || relay.Ready() || attempts.Load() != 0 {
		t.Fatalf("startup without redrive ledger result=%+v ready=%t attempts=%d", result, relay.Ready(), attempts.Load())
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
		assertRelayResult(t, result, true, nil)
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
		var attempts atomic.Int64
		var once sync.Once
		pastThreshold := make(chan struct{})
		publisher := testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
			if attempts.Add(1) > 3 {
				once.Do(func() { close(pastThreshold) })
			}
			return errors.New("temporary")
		})
		config := testRelayConfig()
		config.MaxAttempts = 2
		config.RetryBase = time.Nanosecond
		config.RetryMax = time.Nanosecond
		relay := mustNewOutboxRelay(t, store, publisher, nil, config)
		result := runOutboxRelay(ctx, relay)
		select {
		case <-pastThreshold:
		case <-time.After(10 * time.Second):
			t.Fatalf("relay stopped retrying an ambiguous failure after %d attempts", attempts.Load())
		}
		relay.StartDrain()
		assertRelayResult(t, result, true, nil)
		waitForOutboxCount(t, ctx, pool, "poisoned_at IS NOT NULL", 0)
		record, err := store.Get(ctx, "ambiguous")
		if err != nil {
			t.Fatalf("Get(): %v", err)
		}
		if !record.PoisonedAt.IsZero() || record.LastErrorClass != "publisher_temporary" {
			t.Fatalf("ambiguous poisoned=%v class=%q, want unpoisoned publisher_temporary", record.PoisonedAt, record.LastErrorClass)
		}
	})

	t.Run("permanent", func(t *testing.T) {
		ctx, pool, store := newOutboxFixture(t)
		mustAppendOutbox(t, ctx, pool, store, outboxEvent("permanent"))
		var attempts atomic.Int64
		publisher := testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
			attempts.Add(1)
			return postgresoutbox.ErrPermanentPublication
		})
		relay := mustNewOutboxRelay(t, store, publisher, nil, testRelayConfig())
		result := runOutboxRelay(ctx, relay)
		waitForOutboxCount(t, ctx, pool, "poisoned_at IS NOT NULL", 1)
		relay.StartDrain()
		assertRelayResult(t, result, true, nil)
		if attempts.Load() != 1 {
			t.Fatalf("permanent attempts = %d, want 1", attempts.Load())
		}
	})
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
		observation.PublishedRetainedCount,
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
	ctx, pool, _ := newOutboxFixture(t)
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	telemetry, err := postgresoutbox.NewTelemetry(provider.Meter("outbox-transitions"), nil)
	if err != nil {
		t.Fatalf("NewTelemetry(): %v", err)
	}
	t.Cleanup(telemetry.Close)
	store, err := postgresoutbox.NewStore(pool, telemetry)
	if err != nil {
		t.Fatalf("NewStore(): %v", err)
	}

	mustAppendOutbox(t, ctx, pool, store, outboxEvent("telemetry-recovery"))
	first, err := store.Claim(ctx, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("claim recovery fixture: %v", err)
	}
	postgresSleep(t, ctx, pool, 0.02)
	recovered := mustClaimOutbox(t, ctx, store)
	if recovered.Event.ID != first.Event.ID || !recovered.Recovered {
		t.Fatalf("recovered claim = id %q recovered %t, want %q/true", recovered.Event.ID, recovered.Recovered, first.Event.ID)
	}
	if err := store.MarkPublished(ctx, recovered); err != nil {
		t.Fatalf("mark recovered event published: %v", err)
	}

	mustAppendOutbox(t, ctx, pool, store, outboxEvent("telemetry-poison"))
	poison := mustClaimOutbox(t, ctx, store)
	if err := store.ScheduleRetry(ctx, poison.Event.ID, poison.Token, "publisher_temporary", 0); err != nil {
		t.Fatalf("schedule telemetry retry: %v", err)
	}
	poison = mustClaimOutbox(t, ctx, store)
	if err := store.MarkPoisoned(ctx, poison.Event.ID, poison.Token, "publisher_permanent"); err != nil {
		t.Fatalf("mark telemetry poison: %v", err)
	}
	if err := store.Redrive(ctx, poison.Event.ID, "telemetry-redrive"); err != nil {
		t.Fatalf("redrive telemetry poison: %v", err)
	}
	poison = mustClaimOutbox(t, ctx, store)
	if err := store.MarkPublished(ctx, poison); err != nil {
		t.Fatalf("mark redriven event published: %v", err)
	}
	if _, err := pool.PGX().Exec(ctx, "UPDATE outbox_events SET published_at = clock_timestamp() - interval '2 hours'"); err != nil {
		t.Fatalf("age telemetry cleanup rows: %v", err)
	}
	if deleted, err := store.CleanupPublished(ctx, time.Hour, 10); err != nil || deleted != 2 {
		t.Fatalf("CleanupPublished() = %d, %v; want 2, nil", deleted, err)
	}

	mustAppendOutbox(t, ctx, pool, store, outboxEvent("telemetry-relay"))
	publishStarted := make(chan struct{})
	publishRelease := make(chan struct{})
	publisher := testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
		close(publishStarted)
		<-publishRelease
		return nil
	})
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
	assertRelayResult(t, result, true, nil)
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
	wantOperations := []string{
		"append", "claim", "recovery", "publish", "mark_published", "schedule_retry",
		"poison", "redrive", "cleanup", "observe", "drain",
	}
	for _, operation := range wantOperations {
		if !operations[operation] || !durations[operation] {
			t.Fatalf("operation %q counter/duration present = %t/%t", operation, operations[operation], durations[operation])
		}
	}

	secondReader := sdkmetric.NewManualReader()
	secondProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(secondReader))
	t.Cleanup(func() { _ = secondProvider.Shutdown(context.Background()) })
	secondTelemetry, err := postgresoutbox.NewTelemetry(secondProvider.Meter("outbox-second-replica"), nil)
	if err != nil {
		t.Fatalf("second NewTelemetry(): %v", err)
	}
	t.Cleanup(secondTelemetry.Close)
	secondTelemetry.RecordObservation(observation, time.Now())
	firstDatabase := collectOutboxDatabaseMetrics(t, reader)
	secondDatabase := collectOutboxDatabaseMetrics(t, secondReader)
	if !reflect.DeepEqual(firstDatabase, secondDatabase) {
		t.Fatalf("replica database metrics differ: %+v vs %+v", firstDatabase, secondDatabase)
	}
	secondOperations, secondDurations := collectOutboxOperationMetrics(t, secondReader)
	if len(secondOperations) != 0 || len(secondDurations) != 0 {
		t.Fatalf("second replica inherited process operations: %v %v", secondOperations, secondDurations)
	}
}

func TestPostgresOutboxFailedObservationStaysStaleAndUnready(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	telemetry, err := postgresoutbox.NewTelemetry(provider.Meter("outbox-stale"), nil)
	if err != nil {
		t.Fatalf("NewTelemetry(): %v", err)
	}
	t.Cleanup(telemetry.Close)
	config := testRelayConfig()
	config.PollInterval = time.Hour
	config.ObservationInterval = 50 * time.Millisecond
	relay := mustNewOutboxRelay(t, store, testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
		return nil
	}), telemetry, config)
	result := runOutboxRelay(ctx, relay)
	waitForOutboxOperation(t, reader, "claim", "empty")
	before := collectOutboxProcessMetrics(t, reader)
	if before.ready != 1 || before.observedAt == 0 {
		t.Fatalf("initial ready/observation = %d/%f, want 1/>0", before.ready, before.observedAt)
	}
	if _, err := pool.PGX().Exec(ctx, "DROP TABLE outbox_events CASCADE"); err != nil {
		t.Fatalf("remove schema for fatal observation: %v", err)
	}
	failed := readRelayResult(t, result)
	if failed.Err == nil || !failed.CleanupSafe || relay.Ready() {
		t.Fatalf("failed observation result=%+v ready=%t", failed, relay.Ready())
	}
	after := collectOutboxProcessMetrics(t, reader)
	if after.ready != 0 || after.observedAt != before.observedAt {
		t.Fatalf("failed observation ready/timestamp = %d/%f, want 0/%f", after.ready, after.observedAt, before.observedAt)
	}
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

func mustClaimOutbox(t *testing.T, ctx context.Context, store *postgresoutbox.Store) postgresoutbox.ClaimedEvent {
	t.Helper()
	claim, err := store.Claim(ctx, time.Minute)
	if err != nil {
		t.Fatalf("Claim(): %v", err)
	}
	return claim
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

func assertRelayResult(t *testing.T, result <-chan postgresoutbox.RelayResult, wantCleanupSafe bool, wantErr error) {
	t.Helper()
	got := readRelayResult(t, result)
	if got.CleanupSafe != wantCleanupSafe || !errors.Is(got.Err, wantErr) {
		t.Fatalf("Relay.Run() = %+v, want cleanupSafe=%t error=%v", got, wantCleanupSafe, wantErr)
	}
}

func readRelayResult(t *testing.T, result <-chan postgresoutbox.RelayResult) postgresoutbox.RelayResult {
	t.Helper()
	select {
	case got := <-result:
		return got
	case <-time.After(10 * time.Second):
		t.Fatal("Relay.Run() did not stop")
	}
	return postgresoutbox.RelayResult{}
}

func waitForOutboxCount(t *testing.T, ctx context.Context, pool *postgres.Pool, predicate string, want int) {
	t.Helper()
	deadline := time.NewTimer(10 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		var count int
		if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM outbox_events WHERE "+predicate).Scan(&count); err != nil {
			t.Fatalf("count outbox state: %v", err)
		}
		if count == want {
			return
		}
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatalf("outbox count for %q = %d, want %d", predicate, count, want)
		}
	}
}

func waitForOutboxOperation(t *testing.T, reader *sdkmetric.ManualReader, operation, outcome string) {
	t.Helper()
	deadline := time.NewTimer(10 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		var collected metricdata.ResourceMetrics
		if err := reader.Collect(t.Context(), &collected); err != nil {
			t.Fatalf("collect outbox operations: %v", err)
		}
		for _, scope := range collected.ScopeMetrics {
			for _, measured := range scope.Metrics {
				if measured.Name != "outbox.relay.operations" {
					continue
				}
				for _, point := range measured.Data.(metricdata.Sum[int64]).DataPoints {
					if metricAttribute(t, point.Attributes, "operation") == operation &&
						metricAttribute(t, point.Attributes, "outcome") == outcome && point.Value > 0 {
						return
					}
				}
			}
		}
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatalf("outbox operation %s/%s was not recorded", operation, outcome)
		}
	}
}

func waitForBlockedOutboxObservation(t *testing.T, ctx context.Context, pool *postgres.Pool) {
	t.Helper()
	deadline := time.NewTimer(10 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		var blocked bool
		if err := pool.PGX().QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE pid <> pg_backend_pid()
				  AND wait_event_type = 'Lock'
				  AND query LIKE '%name: ObserveOutbox :one%'
			)`).Scan(&blocked); err != nil {
			t.Fatalf("inspect blocked observation: %v", err)
		}
		if blocked {
			return
		}
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatal("relay observation did not block behind the maintenance gate")
		}
	}
}

type outboxProcessMetrics struct {
	ready        int64
	inflight     int64
	observedAt   float64
	lastProgress float64
}

func collectOutboxProcessMetrics(t *testing.T, reader *sdkmetric.ManualReader) outboxProcessMetrics {
	t.Helper()
	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("Collect(): %v", err)
	}
	var result outboxProcessMetrics
	for _, scope := range collected.ScopeMetrics {
		for _, measured := range scope.Metrics {
			switch measured.Name {
			case "outbox.relay.readiness":
				result.ready = measured.Data.(metricdata.Gauge[int64]).DataPoints[0].Value
			case "outbox.relay.inflight":
				result.inflight = measured.Data.(metricdata.Gauge[int64]).DataPoints[0].Value
			case "outbox.relay.observation.timestamp":
				result.observedAt = measured.Data.(metricdata.Gauge[float64]).DataPoints[0].Value
			case "outbox.relay.last_progress.timestamp":
				result.lastProgress = measured.Data.(metricdata.Gauge[float64]).DataPoints[0].Value
			}
		}
	}
	return result
}

func collectOutboxOperationMetrics(t *testing.T, reader *sdkmetric.ManualReader) (map[string]bool, map[string]bool) {
	t.Helper()
	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("Collect(): %v", err)
	}
	operations := make(map[string]bool)
	durations := make(map[string]bool)
	for _, scope := range collected.ScopeMetrics {
		for _, measured := range scope.Metrics {
			switch measured.Name {
			case "outbox.relay.operations":
				for _, point := range measured.Data.(metricdata.Sum[int64]).DataPoints {
					if point.Value > 0 {
						operations[metricAttribute(t, point.Attributes, "operation")] = true
					}
				}
			case "outbox.relay.operation.duration":
				for _, point := range measured.Data.(metricdata.Histogram[float64]).DataPoints {
					if point.Count > 0 {
						durations[metricAttribute(t, point.Attributes, "operation")] = true
					}
				}
			}
		}
	}
	return operations, durations
}

func collectOutboxDatabaseMetrics(t *testing.T, reader *sdkmetric.ManualReader) outboxMetricSnapshot {
	t.Helper()
	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("Collect(): %v", err)
	}
	result := outboxMetricSnapshot{
		counts:  make(map[string]int64),
		oldest:  make(map[string]float64),
		storage: make(map[string]int64),
	}
	for _, scope := range collected.ScopeMetrics {
		for _, measured := range scope.Metrics {
			switch measured.Name {
			case "outbox.relay.messages":
				for _, point := range measured.Data.(metricdata.Gauge[int64]).DataPoints {
					result.counts[metricAttribute(t, point.Attributes, "state")] = point.Value
				}
			case "outbox.relay.oldest.timestamp":
				for _, point := range measured.Data.(metricdata.Gauge[float64]).DataPoints {
					result.oldest[metricAttribute(t, point.Attributes, "state")] = point.Value
				}
			case "outbox.relay.ordering_heads":
				result.orderingHeads = measured.Data.(metricdata.Gauge[int64]).DataPoints[0].Value
			case "outbox.relay.storage.bytes":
				for _, point := range measured.Data.(metricdata.Gauge[int64]).DataPoints {
					key := metricAttribute(t, point.Attributes, "relation") + "/" + metricAttribute(t, point.Attributes, "kind")
					result.storage[key] = point.Value
				}
			}
		}
	}
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
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	telemetry, err := postgresoutbox.NewTelemetry(provider.Meter("outbox-integration"), nil)
	if err != nil {
		t.Fatalf("NewTelemetry(): %v", err)
	}
	t.Cleanup(telemetry.Close)
	telemetry.RecordObservation(observation, time.Now())
	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("Collect(): %v", err)
	}
	result := outboxMetricSnapshot{
		counts:  make(map[string]int64),
		oldest:  make(map[string]float64),
		storage: make(map[string]int64),
	}
	for _, scope := range collected.ScopeMetrics {
		for _, measured := range scope.Metrics {
			switch measured.Name {
			case "outbox.relay.messages":
				gauge := measured.Data.(metricdata.Gauge[int64])
				for _, point := range gauge.DataPoints {
					result.counts[metricAttribute(t, point.Attributes, "state")] = point.Value
				}
			case "outbox.relay.oldest.timestamp":
				gauge := measured.Data.(metricdata.Gauge[float64])
				for _, point := range gauge.DataPoints {
					result.oldest[metricAttribute(t, point.Attributes, "state")] = point.Value
				}
			case "outbox.relay.ordering_heads":
				result.orderingHeads = measured.Data.(metricdata.Gauge[int64]).DataPoints[0].Value
			case "outbox.relay.storage.bytes":
				gauge := measured.Data.(metricdata.Gauge[int64])
				for _, point := range gauge.DataPoints {
					key := metricAttribute(t, point.Attributes, "relation") + "/" + metricAttribute(t, point.Attributes, "kind")
					result.storage[key] = point.Value
				}
			}
		}
	}
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

func metricAttribute(t *testing.T, attributes attribute.Set, name string) string {
	t.Helper()
	value, present := attributes.Value(attribute.Key(name))
	if !present {
		t.Fatalf("metric point has no %s attribute", name)
	}
	return value.AsString()
}

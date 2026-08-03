//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/jackc/pgx/v5"
)

// BenchmarkOutboxRelayCycle measures the database cost of one relay cycle:
// leasing a batch and writing that batch's outcome. Those two round trips are
// what batching amortizes, so the per-event metric across batch sizes is the
// direct measure of the claim/finalize design. Batch size 1 is the
// one-event-per-round-trip shape.
//
// Each cycle releases its batch for immediate re-claim instead of publishing
// it, so the seeded backlog stays constant and no fixture work enters the timed
// interval. Publisher concurrency is deliberately outside this boundary: it
// measures a broker, not this repository.
func BenchmarkOutboxRelayCycle(b *testing.B) {
	// Every case shares one backlog size so the per-event metric compares batch
	// sizes rather than fixture shapes, and so no case rewrites a large share of
	// the claim index on every cycle.
	const backlog = 5000

	for _, batchSize := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("batch-%d", batchSize), func(b *testing.B) {
			ctx, pool, store := newOutboxBenchmarkFixture(b, backlog)
			var events int64

			b.ReportAllocs()
			for b.Loop() {
				batch, err := store.Claim(ctx, time.Minute, batchSize)
				if err != nil {
					b.Fatalf("Claim(): %v", err)
				}
				if len(batch.Events) != batchSize {
					b.Fatalf("claimed %d events, want %d", len(batch.Events), batchSize)
				}
				releases := make([]postgresoutbox.RetryDirective, len(batch.Events))
				for index, claimed := range batch.Events {
					releases[index] = postgresoutbox.RetryDirective{
						ID: claimed.Event.ID, ErrorClass: "publisher_temporary",
					}
				}
				if err := store.ScheduleRetryBatch(ctx, batch.Token, releases); err != nil {
					b.Fatalf("ScheduleRetryBatch(): %v", err)
				}
				events += int64(len(batch.Events))
			}
			b.StopTimer()

			reportOutboxEventCost(b, events)
			if _, err := pool.PGX().Exec(ctx, "SELECT 1"); err != nil {
				b.Fatalf("database unusable after benchmark: %v", err)
			}
		})
	}
}

// BenchmarkOutboxRelayPublishCycle measures the cycle a relay actually runs
// against a healthy broker: leasing a batch and finalizing it as published.
// Unlike BenchmarkOutboxRelayCycle the batch leaves the backlog, so this
// includes the index and heap work a real publication costs. The backlog is
// refilled outside the timed interval whenever a claim comes up short.
func BenchmarkOutboxRelayPublishCycle(b *testing.B) {
	const backlog = 5000

	for _, batchSize := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("batch-%d", batchSize), func(b *testing.B) {
			ctx, pool, store := newOutboxBenchmarkFixture(b, backlog)
			var events int64

			b.ReportAllocs()
			for b.Loop() {
				batch, err := store.Claim(ctx, time.Minute, batchSize)
				if err != nil {
					b.Fatalf("Claim(): %v", err)
				}
				if len(batch.Events) < batchSize {
					b.StopTimer()
					refillOutboxBacklog(b, ctx, pool, func() { seedOutboxBacklog(b, ctx, pool, backlog) })
					b.StartTimer()
					continue
				}
				ids := make([]string, len(batch.Events))
				for index, claimed := range batch.Events {
					ids[index] = claimed.Event.ID
				}
				marked, err := store.MarkPublishedBatch(ctx, batch.Token, ids)
				if err != nil || len(marked) != len(ids) {
					b.Fatalf("MarkPublishedBatch() = %d of %d, %v", len(marked), len(ids), err)
				}
				events += int64(len(batch.Events))
			}
			b.StopTimer()
			reportOutboxEventCost(b, events)
		})
	}
}

// BenchmarkOutboxRelayOrderedPublishCycle measures the same cycle for ordered
// events, where finalizing also advances each key's head and unblocks that
// key's successor. Every case drains the same backlog through a different
// number of ordering keys, so the per-event metric is the direct measure of
// finalizing a whole lease in one statement rather than one key at a time.
func BenchmarkOutboxRelayOrderedPublishCycle(b *testing.B) {
	const backlog = 5000

	for _, keys := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("batch-%d", keys), func(b *testing.B) {
			ctx, pool, store := newOutboxBenchmarkFixture(b, 0)
			seed := func() { seedOutboxOrderedBacklog(b, ctx, pool, keys, backlog/keys) }
			seed()
			analyzeOutboxFixture(b, ctx, pool)
			var events int64

			b.ReportAllocs()
			for b.Loop() {
				batch, err := store.Claim(ctx, time.Minute, keys)
				if err != nil {
					b.Fatalf("Claim(): %v", err)
				}
				if len(batch.Events) < keys {
					b.StopTimer()
					refillOutboxBacklog(b, ctx, pool, seed)
					b.StartTimer()
					continue
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
				if err != nil || len(marked) != len(directives) {
					b.Fatalf("MarkOrderedPublishedBatch() = %d of %d, %v", len(marked), len(directives), err)
				}
				events += int64(len(batch.Events))
			}
			b.StopTimer()
			reportOutboxEventCost(b, events)
		})
	}
}

// BenchmarkOutboxAppend measures the request-path cost a feature pays to emit
// one event inside its own transaction: envelope validation plus the insert
// that commits with the domain mutation. The ordered case also advances its
// key's retained high-water mark, which is the same statement.
func BenchmarkOutboxAppend(b *testing.B) {
	for _, ordered := range []bool{false, true} {
		name := "unordered"
		if ordered {
			name = "ordered"
		}
		b.Run(name, func(b *testing.B) {
			ctx, pool, store := newOutboxBenchmarkFixture(b, 0)
			event := outboxEvent("append-bench")
			if ordered {
				event.OrderingKey = "append-bench-key"
			}
			sequence := int64(0)

			b.ReportAllocs()
			for b.Loop() {
				sequence++
				event.ID = fmt.Sprintf("append-bench-%d", sequence)
				if ordered {
					event.OrderingSequence = sequence
				}
				if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
					return store.Append(ctx, tx, event)
				}); err != nil {
					b.Fatalf("Append(): %v", err)
				}
			}
			b.StopTimer()

			var stored int64
			if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM outbox_events").Scan(&stored); err != nil {
				b.Fatalf("count appended events: %v", err)
			}
			if stored != sequence {
				b.Fatalf("stored events = %d, want %d", stored, sequence)
			}
		})
	}
}

func newOutboxBenchmarkFixture(b *testing.B, backlog int) (context.Context, *postgres.Pool, *postgresoutbox.Store) {
	b.Helper()

	ctx, cancel := context.WithTimeout(b.Context(), 10*time.Minute)
	b.Cleanup(cancel)
	dsn := pgtest.Migrated(b, os.DirFS(".."), "migrations")
	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     3 * time.Second,
		HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns:       4,
		AcquireTimeout:     time.Second,
		ConnMaxLifetime:    time.Hour,
		StatementTimeout:   30 * time.Second,
	})
	if err != nil {
		b.Fatalf("postgres.New(): %v", err)
	}
	b.Cleanup(pool.Close)
	store, err := postgresoutbox.NewStore(pool, nil)
	if err != nil {
		b.Fatalf("postgresoutbox.NewStore(): %v", err)
	}
	if backlog > 0 {
		seedOutboxBacklog(b, ctx, pool, backlog)
	}
	analyzeOutboxFixture(b, ctx, pool)
	return ctx, pool, store
}

// analyzeOutboxFixture refreshes statistics so the planner chooses the claim
// plan a warmed deployment would rather than one derived from an empty table.
func analyzeOutboxFixture(b *testing.B, ctx context.Context, pool *postgres.Pool) {
	b.Helper()
	if _, err := pool.PGX().Exec(ctx, "ANALYZE outbox_events, outbox_ordering_heads"); err != nil {
		b.Fatalf("analyze benchmark fixture: %v", err)
	}
}

// refillOutboxBacklog restores the declared fixture after a case has drained
// it. It runs outside the timed interval and discards leftover leases, so every
// measured cycle sees the same backlog shape.
func refillOutboxBacklog(b *testing.B, ctx context.Context, pool *postgres.Pool, seed func()) {
	b.Helper()
	if _, err := pool.PGX().Exec(ctx,
		"TRUNCATE outbox_events, outbox_ordering_heads, outbox_redrives"); err != nil {
		b.Fatalf("reset outbox backlog: %v", err)
	}
	seed()
	analyzeOutboxFixture(b, ctx, pool)
}

func reportOutboxEventCost(b *testing.B, events int64) {
	b.Helper()
	if events == 0 {
		b.Fatal("no events cycled")
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(events), "ns/event")
}

func seedOutboxOrderedBacklog(b *testing.B, ctx context.Context, pool *postgres.Pool, keys, depth int) {
	b.Helper()
	if _, err := pool.PGX().Exec(ctx, `
		INSERT INTO outbox_ordering_heads (ordering_key, last_sequence, current_sequence)
		SELECT 'bench-key-' || key, $2, 1
		FROM generate_series(1, $1) AS key`, keys, depth); err != nil {
		b.Fatalf("seed outbox ordering heads: %v", err)
	}
	if _, err := pool.PGX().Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata,
			ordering_key, ordering_sequence, ordering_ready
		)
		SELECT
			'bench-' || key || '-' || sequence,
			'bench.event',
			'bench',
			'events',
			'v1',
			clock_timestamp(),
			convert_to('{"sequence":' || sequence || '}', 'UTF8'),
			convert_to('{}', 'UTF8'),
			'bench-key-' || key,
			sequence,
			sequence = 1
		FROM generate_series(1, $1) AS key, generate_series(1, $2) AS sequence`, keys, depth); err != nil {
		b.Fatalf("seed ordered outbox backlog: %v", err)
	}
}

func seedOutboxBacklog(b *testing.B, ctx context.Context, pool *postgres.Pool, count int) {
	b.Helper()
	if _, err := pool.PGX().Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata
		)
		SELECT
			'bench-' || generated,
			'bench.event',
			'bench',
			'events',
			'v1',
			clock_timestamp(),
			convert_to('{"sequence":' || generated || '}', 'UTF8'),
			convert_to('{}', 'UTF8')
		FROM generate_series(1, $1) AS generated`, count); err != nil {
		b.Fatalf("seed outbox backlog: %v", err)
	}
}

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

			if events == 0 {
				b.Fatal("no events cycled")
			}
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(events), "ns/event")
			if _, err := pool.PGX().Exec(ctx, "SELECT 1"); err != nil {
				b.Fatalf("database unusable after benchmark: %v", err)
			}
		})
	}
}

// BenchmarkOutboxAppend measures the request-path cost a feature pays to emit
// one event inside its own transaction: envelope validation plus the insert
// that commits with the domain mutation.
func BenchmarkOutboxAppend(b *testing.B) {
	ctx, pool, store := newOutboxBenchmarkFixture(b, 0)
	event := outboxEvent("append-bench")
	sequence := 0

	b.ReportAllocs()
	for b.Loop() {
		sequence++
		event.ID = fmt.Sprintf("append-bench-%d", sequence)
		if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
			return store.Append(ctx, tx, event)
		}); err != nil {
			b.Fatalf("Append(): %v", err)
		}
	}
	b.StopTimer()

	var stored int
	if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM outbox_events").Scan(&stored); err != nil {
		b.Fatalf("count appended events: %v", err)
	}
	if stored != sequence {
		b.Fatalf("stored events = %d, want %d", stored, sequence)
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
	// Statistics must reflect the seeded fixture before the timed interval, or
	// the planner chooses a different claim plan than a warmed deployment would.
	if _, err := pool.PGX().Exec(ctx, "ANALYZE outbox_events"); err != nil {
		b.Fatalf("analyze benchmark fixture: %v", err)
	}
	return ctx, pool, store
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

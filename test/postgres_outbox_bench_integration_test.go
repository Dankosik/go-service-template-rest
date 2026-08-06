//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"sync/atomic"
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
			seed := func() { seedOutboxBacklog(b, ctx, pool, backlog) }
			runOutboxPublishCycle(b, ctx, pool, store, batchSize, seed)
		})
	}
}

// BenchmarkOutboxRelayPublishCycleRetained measures that same cycle against the
// retention window a running service carries. Published rows are kept for seven
// days by default, so a deployed outbox spends nearly all its life with a table
// that is mostly rows the claim must not look at, and a fixture holding only the
// backlog measures a state the service leaves after its first week.
//
// The retained counts are therefore the direct measure of the claim and
// finalization design's one scale question: does a cycle cost what the batch
// costs, or what the table costs?
func BenchmarkOutboxRelayPublishCycleRetained(b *testing.B) {
	const (
		backlog   = 5000
		batchSize = 100
	)

	for _, retained := range []int{50_000, 200_000} {
		b.Run(fmt.Sprintf("retained-%d", retained), func(b *testing.B) {
			ctx, pool, store := newOutboxBenchmarkFixture(b, 0)
			seed := func() {
				seedOutboxBacklog(b, ctx, pool, backlog)
				seedOutboxRetained(b, ctx, pool, retained)
			}
			seed()
			analyzeOutboxFixture(b, ctx, pool)
			runOutboxPublishCycle(b, ctx, pool, store, batchSize, seed)
		})
	}
}

// BenchmarkOutboxCleanup measures the retention path: deleting one bounded batch
// of published rows out of the window the service keeps. The retained counts
// answer the same question the claim faces, because a delete that locates its
// rows through the whole window costs more the better retention is working.
//
// Every window here is larger than the measured interval can drain at this batch
// size, so no case reseeds inside its own loop. A window that runs out mid-run
// measures a truncated and refilled table instead of a retained one.
func BenchmarkOutboxCleanup(b *testing.B) {
	const batchSize = 1000

	for _, retained := range []int{200_000, 400_000} {
		b.Run(fmt.Sprintf("retained-%d", retained), func(b *testing.B) {
			ctx, pool, store := newOutboxBenchmarkFixture(b, 0)
			seed := func() { seedOutboxRetained(b, ctx, pool, retained) }
			seed()
			analyzeOutboxFixture(b, ctx, pool)
			runOutboxCleanupCycle(b, ctx, pool, store, batchSize, seed)
		})
	}
}

// BenchmarkOutboxRelayPublishCycleBatch sweeps the claim batch size past the
// shipped default. `batch_size` defaults to 100, and that number was chosen from
// peak relay memory — the envelope limit makes 100 worth about 29 MiB — rather
// than from where the per-event curve stops falling. These cases are that curve:
// batching amortizes two round trips and two commits, so the question is how
// much of the default's per-event cost is still round trip rather than work.
func BenchmarkOutboxRelayPublishCycleBatch(b *testing.B) {
	// Large enough that the widest batch does not drain it inside the measured
	// interval, so no case reseeds mid-run.
	const backlog = 60000

	for _, batchSize := range []int{100, 250, 500, 1000} {
		b.Run(fmt.Sprintf("batch-%d", batchSize), func(b *testing.B) {
			ctx, pool, store := newOutboxBenchmarkFixture(b, backlog)
			seed := func() { seedOutboxBacklog(b, ctx, pool, backlog) }
			runOutboxPublishCycle(b, ctx, pool, store, batchSize, seed)
		})
	}
}

// BenchmarkOutboxCleanupBatch sweeps the retention batch size the same way.
// `cleanup_batch_size` defaults to 1,000 and validation allows up to 10,000; a
// larger transaction deletes more rows per round trip but holds its locks longer.
//
// Run this group at a short benchtime: the widest batch removes 5,000 rows per
// operation, so a full second of iterations would drain any affordable window
// and spend the run reseeding instead of deleting.
func BenchmarkOutboxCleanupBatch(b *testing.B) {
	const retained = 400_000

	for _, batchSize := range []int{1000, 2500, 5000} {
		b.Run(fmt.Sprintf("batch-%d", batchSize), func(b *testing.B) {
			ctx, pool, store := newOutboxBenchmarkFixture(b, 0)
			seed := func() { seedOutboxRetained(b, ctx, pool, retained) }
			seed()
			analyzeOutboxFixture(b, ctx, pool)
			runOutboxCleanupCycle(b, ctx, pool, store, batchSize, seed)
		})
	}
}

// BenchmarkOutboxRelayPublishCycleDurable is the same publish cycle against a
// server that actually flushes WAL.
//
// Every other case here runs on the shared test fixture, which starts PostgreSQL
// with `fsync=off` so a database per test stays affordable. That makes commit
// look free, and commit is most of what a small batch costs — so the default
// fixture cannot see this axis at all. These cases turn durability back on,
// which is the only place a claim about commit cost can be made.
func BenchmarkOutboxRelayPublishCycleDurable(b *testing.B) {
	const backlog = 5000

	for _, batchSize := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("batch-%d", batchSize), func(b *testing.B) {
			ctx, pool, store := newOutboxBenchmarkFixture(b, backlog)
			setOutboxDurability(b, ctx, pool, true)
			seed := func() { seedOutboxBacklog(b, ctx, pool, backlog) }
			runOutboxPublishCycle(b, ctx, pool, store, batchSize, seed)
		})
	}
}

// BenchmarkOutboxAppendDurable is the request-path append against that same
// durable server, because an append commits inside the caller's transaction and
// is where a WAL flush is most visible.
func BenchmarkOutboxAppendDurable(b *testing.B) {
	for _, perTransaction := range []int{1, 4, 16} {
		b.Run(fmt.Sprintf("unordered-%d", perTransaction), func(b *testing.B) {
			ctx, pool, store := newOutboxBenchmarkFixture(b, 0)
			setOutboxDurability(b, ctx, pool, true)
			events := make([]postgresoutbox.Event, perTransaction)
			sequence := int64(0)

			b.ReportAllocs()
			for b.Loop() {
				for index := range events {
					sequence++
					events[index] = outboxEvent(fmt.Sprintf("durable-bench-%d", sequence))
				}
				if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
					return store.Append(ctx, tx, events...)
				}); err != nil {
					b.Fatalf("Append(): %v", err)
				}
			}
			b.StopTimer()
			reportOutboxEventCost(b, sequence)
		})
	}
}

// setOutboxDurability flips `fsync` for the shared fixture server. It is a
// SIGHUP setting, so a reload is enough and no container restart is needed. The
// value is restored afterwards because the container is shared by the whole
// test binary.
func setOutboxDurability(b *testing.B, ctx context.Context, pool *postgres.Pool, durable bool) {
	b.Helper()
	apply := func(value string) {
		if _, err := pool.PGX().Exec(ctx, "ALTER SYSTEM SET fsync = "+value); err != nil {
			b.Fatalf("set fsync=%s: %v", value, err)
		}
		if _, err := pool.PGX().Exec(ctx, "SELECT pg_reload_conf()"); err != nil {
			b.Fatalf("reload configuration: %v", err)
		}
	}
	setting := "off"
	if durable {
		setting = "on"
	}
	apply(setting)
	b.Cleanup(func() {
		if _, err := pool.PGX().Exec(context.WithoutCancel(ctx), "ALTER SYSTEM RESET fsync"); err == nil {
			_, _ = pool.PGX().Exec(context.WithoutCancel(ctx), "SELECT pg_reload_conf()")
		}
	})
	var effective string
	if err := pool.PGX().QueryRow(ctx, "SHOW fsync").Scan(&effective); err != nil {
		b.Fatalf("read fsync: %v", err)
	}
	if effective != setting {
		b.Fatalf("fsync = %q after reload, want %q", effective, setting)
	}
}

// runOutboxPublishCycle is the measured loop shared by the publish-cycle cases:
// lease a full batch and finalize it as published. A short claim means the case
// drained its backlog, so the fixture is restored outside the timed interval.
func runOutboxPublishCycle(
	b *testing.B,
	ctx context.Context,
	pool *postgres.Pool,
	store *postgresoutbox.Store,
	batchSize int,
	seed func(),
) {
	b.Helper()
	var events int64

	b.ReportAllocs()
	for b.Loop() {
		batch, err := store.Claim(ctx, time.Minute, batchSize)
		if err != nil {
			b.Fatalf("Claim(): %v", err)
		}
		if len(batch.Events) < batchSize {
			b.StopTimer()
			refillOutboxBacklog(b, ctx, pool, seed)
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
}

// runOutboxCleanupCycle is the measured loop shared by the retention cases:
// delete one full batch of retained published rows. A short delete means the
// case drained its backlog, so the fixture is restored outside the timed
// interval, exactly as runOutboxPublishCycle does for a short claim.
func runOutboxCleanupCycle(
	b *testing.B,
	ctx context.Context,
	pool *postgres.Pool,
	store *postgresoutbox.Store,
	batchSize int,
	seed func(),
) {
	b.Helper()
	var events int64

	b.ReportAllocs()
	for b.Loop() {
		deleted, err := store.CleanupPublished(ctx, time.Minute, batchSize)
		if err != nil {
			b.Fatalf("CleanupPublished(): %v", err)
		}
		if deleted < batchSize {
			b.StopTimer()
			refillOutboxBacklog(b, ctx, pool, seed)
			b.StartTimer()
			continue
		}
		events += int64(deleted)
	}
	b.StopTimer()
	reportOutboxEventCost(b, events)
}

// BenchmarkOutboxRelayOrderedPublishCycle measures the same cycle for ordered
// events, where finalizing also advances each key's head and unblocks that
// key's successor. Every case drains the same backlog through a different
// number of ordering keys, so the per-event metric is the direct measure of
// finalizing a whole lease in one statement rather than one key at a time.
// BenchmarkOutboxOrderedSuccessorCost isolates what handing the baton to the
// next event of an ordering key costs.
//
// Finalizing an ordered event also looks up that key's successor, advances the
// head, and marks the successor claimable — and that last update is not
// heap-only, because `ordering_ready` sits in two partial index predicates. Both
// cases here publish the same number of events in the same batch size through
// the same statement; they differ only in whether a successor exists. A key one
// event deep has none, so the lookup finds nothing and the readiness update
// touches no row. The difference is therefore the whole cost of the baton, and
// the upper bound on what a claim redesign that removed it could win.
func BenchmarkOutboxOrderedSuccessorCost(b *testing.B) {
	const (
		events    = 5000
		batchSize = 100
	)

	for _, depth := range []int{1, 50} {
		b.Run(fmt.Sprintf("depth-%d", depth), func(b *testing.B) {
			ctx, pool, store := newOutboxBenchmarkFixture(b, 0)
			seed := func() { seedOutboxOrderedBacklog(b, ctx, pool, events/depth, depth) }
			seed()
			analyzeOutboxFixture(b, ctx, pool)
			runOutboxOrderedPublishCycle(b, ctx, pool, store, batchSize, seed)
		})
	}
}

// BenchmarkOutboxClaimRetryWait claims against a backlog that is mostly not yet
// due.
//
// This is the shape `available_at` is indexed for. Every other case here leaves
// the whole backlog immediately claimable, which lets any candidate that drops
// the column from the claim index look free — nothing is ever filtered out. A
// broker outage produces the opposite: most of the backlog sits in retry-wait
// with a future `available_at`, and a claim that cannot seek past those rows has
// to walk them.
//
// The waiting rows are the *oldest* ones, and that detail decides the answer.
// Seed them newer than the due rows and a claim ordered on creation reaches
// everything it wants before it meets one it must skip — which measured dropping
// `available_at` from the index as a 21% improvement. Age them past the due rows,
// as a real outage does because the longest-waiting event has backed off
// furthest, and the same change measures 77% worse. Keep this ordering when
// changing the fixture.
func BenchmarkOutboxClaimRetryWait(b *testing.B) {
	const (
		due       = 2000
		waiting   = 18000
		batchSize = 100
	)

	ctx, pool, store := newOutboxBenchmarkFixture(b, 0)
	seed := func() {
		seedOutboxBacklog(b, ctx, pool, due)
		seedOutboxRetryWaiting(b, ctx, pool, waiting)
	}
	seed()
	analyzeOutboxFixture(b, ctx, pool)
	var events int64

	b.ReportAllocs()
	for b.Loop() {
		batch, err := store.Claim(ctx, time.Minute, batchSize)
		if err != nil {
			b.Fatalf("Claim(): %v", err)
		}
		if len(batch.Events) < batchSize {
			b.StopTimer()
			refillOutboxBacklog(b, ctx, pool, seed)
			b.StartTimer()
			continue
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
}

// BenchmarkOutboxObserveSplit asks whether the periodic observation is worth
// splitting. The full observation classifies every unpublished row into six
// mutually exclusive states; the alert-critical signal is much smaller — how
// much is pending and how old the oldest is. If the narrow query is far cheaper,
// running it at the current interval and the full breakdown rarely would keep
// the operator signal while dropping the cost that grows with the backlog.
func BenchmarkOutboxObserveSplit(b *testing.B) {
	const pending = 100_000

	b.Run("full", func(b *testing.B) {
		ctx, _, store := newOutboxBenchmarkFixture(b, pending)
		var observations int64
		b.ReportAllocs()
		for b.Loop() {
			if _, err := store.Observe(ctx); err != nil {
				b.Fatalf("Observe(): %v", err)
			}
			observations++
		}
		b.StopTimer()
		reportOutboxEventCost(b, observations)
	})

	b.Run("backlog-only", func(b *testing.B) {
		ctx, pool, _ := newOutboxBenchmarkFixture(b, pending)
		var observations int64
		b.ReportAllocs()
		for b.Loop() {
			var count int64
			var oldest float64
			if err := pool.PGX().QueryRow(ctx, `
				SELECT count(*)::bigint,
				       coalesce(extract(epoch FROM min(created_at)), 0)::double precision
				FROM outbox_events
				WHERE published_at IS NULL`).Scan(&count, &oldest); err != nil {
				b.Fatalf("observe backlog: %v", err)
			}
			observations++
		}
		b.StopTimer()
		reportOutboxEventCost(b, observations)
	})
}

// BenchmarkOutboxHeapGrowth measures storage rather than latency: how many bytes
// of table and index the queue carries per event it has cycled through.
//
// The pack sets churn-proportional autovacuum thresholds instead of PostgreSQL's
// size-proportional defaults, and the migration says outright that those numbers
// are a starting point rather than a measured optimum. This is the check that
// makes them falsifiable — run it against the shipped settings and against the
// defaults and compare the bytes each leaves behind for the same work.
func BenchmarkOutboxHeapGrowth(b *testing.B) {
	const (
		backlog   = 20000
		batchSize = 500
	)

	ctx, pool, store := newOutboxBenchmarkFixture(b, backlog)
	seed := func() { seedOutboxBacklog(b, ctx, pool, backlog) }
	var events int64

	for b.Loop() {
		batch, err := store.Claim(ctx, time.Minute, batchSize)
		if err != nil {
			b.Fatalf("Claim(): %v", err)
		}
		if len(batch.Events) < batchSize {
			b.StopTimer()
			refillOutboxBacklog(b, ctx, pool, seed)
			b.StartTimer()
			continue
		}
		ids := make([]string, len(batch.Events))
		for index, claimed := range batch.Events {
			ids[index] = claimed.Event.ID
		}
		if _, err := store.MarkPublishedBatch(ctx, batch.Token, ids); err != nil {
			b.Fatalf("MarkPublishedBatch(): %v", err)
		}
		events += int64(len(batch.Events))
	}
	b.StopTimer()

	if events == 0 {
		b.Fatal("no events cycled")
	}
	var total, dead int64
	if err := pool.PGX().QueryRow(ctx, `
		SELECT pg_total_relation_size('outbox_events'),
		       coalesce((SELECT n_dead_tup FROM pg_stat_user_tables
		                 WHERE relname = 'outbox_events'), 0)`).Scan(&total, &dead); err != nil {
		b.Fatalf("read relation size: %v", err)
	}
	b.ReportMetric(float64(total)/float64(events), "bytes/event")
	b.ReportMetric(float64(dead), "dead-tuples")
}

// runOutboxOrderedPublishCycle is the ordered publish loop shared by the ordered
// cases: lease a full batch and finalize it, which also advances each key's head.
func runOutboxOrderedPublishCycle(
	b *testing.B,
	ctx context.Context,
	pool *postgres.Pool,
	store *postgresoutbox.Store,
	batchSize int,
	seed func(),
) {
	b.Helper()
	var events int64

	b.ReportAllocs()
	for b.Loop() {
		batch, err := store.Claim(ctx, time.Minute, batchSize)
		if err != nil {
			b.Fatalf("Claim(): %v", err)
		}
		if len(batch.Events) < batchSize {
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
}

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
// events inside its own transaction: envelope validation plus the inserts that
// commit with the domain mutation. The ordered case also advances each key's
// retained high-water mark, which is the same statement.
//
// The event-count cases are the direct measure of pipelining the append: a
// business transaction that emits several events pays one round trip for all of
// them, so the per-event metric falls as the count rises. Every case appends
// through one key or none, which is the shape that shares the most contention.
func BenchmarkOutboxAppend(b *testing.B) {
	for _, ordered := range []bool{false, true} {
		shape := "unordered"
		if ordered {
			shape = "ordered"
		}
		for _, perTransaction := range []int{1, 4, 16} {
			b.Run(fmt.Sprintf("%s-%d", shape, perTransaction), func(b *testing.B) {
				ctx, pool, store := newOutboxBenchmarkFixture(b, 0)
				events := make([]postgresoutbox.Event, perTransaction)
				sequence := int64(0)

				b.ReportAllocs()
				for b.Loop() {
					for index := range events {
						sequence++
						events[index] = outboxEvent(fmt.Sprintf("append-bench-%d", sequence))
						if ordered {
							events[index].OrderingKey = "append-bench-key"
							events[index].OrderingSequence = sequence
						}
					}
					if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
						return store.Append(ctx, tx, events...)
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
				reportOutboxEventCost(b, sequence)
			})
		}
	}
}

// BenchmarkOutboxAppendPayload measures the request-path append across the
// payload sizes real events carry. Every other case here uses a 30-byte payload,
// which exercises none of what a payload costs: PostgreSQL only considers
// compressing or moving a value out of line once the row passes its 2 KiB TOAST
// target, and both the stored `IS JSON` check and the WAL record scale with the
// bytes. 64 KiB is a quarter of the accepted payload limit.
func BenchmarkOutboxAppendPayload(b *testing.B) {
	const perTransaction = 4

	for _, payloadBytes := range []int{256, 4096, 65536} {
		b.Run(fmt.Sprintf("bytes-%d", payloadBytes), func(b *testing.B) {
			ctx, pool, store := newOutboxBenchmarkFixture(b, 0)
			payload := outboxBenchmarkPayload(payloadBytes)
			events := make([]postgresoutbox.Event, perTransaction)
			sequence := int64(0)

			b.ReportAllocs()
			for b.Loop() {
				for index := range events {
					sequence++
					events[index] = outboxEvent(fmt.Sprintf("payload-bench-%d", sequence))
					events[index].Payload = payload
				}
				if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
					return store.Append(ctx, tx, events...)
				}); err != nil {
					b.Fatalf("Append(): %v", err)
				}
			}
			b.StopTimer()
			reportOutboxEventCost(b, sequence)
		})
	}
}

// BenchmarkOutboxRelayPublishCyclePayload measures the relay cycle at the same
// sizes. The claim projects the envelope, so payload bytes cross the wire and
// are decoded once per delivery attempt.
func BenchmarkOutboxRelayPublishCyclePayload(b *testing.B) {
	const (
		backlog   = 2000
		batchSize = 100
	)

	for _, payloadBytes := range []int{256, 4096, 65536} {
		b.Run(fmt.Sprintf("bytes-%d", payloadBytes), func(b *testing.B) {
			ctx, pool, store := newOutboxBenchmarkFixture(b, 0)
			seed := func() { seedOutboxSizedBacklog(b, ctx, pool, backlog, payloadBytes) }
			seed()
			analyzeOutboxFixture(b, ctx, pool)
			runOutboxPublishCycle(b, ctx, pool, store, batchSize, seed)
		})
	}
}

// BenchmarkOutboxAppendConcurrent measures what no other case here can:
// concurrent writers. A deployed API has many request goroutines appending at
// once, and that is where heap pages, the primary key, the append notification,
// and the ordering heads are actually contended.
//
// Only two shapes are valid, and the missing third one is the point. Several
// unsynchronized writers sharing one ordering key is not a supported shape: a
// key's sequences must reach PostgreSQL in order, and the retained high-water
// mark rejects one that does not, so such a call fails rather than storing a
// reordered event. A caller that needs ordering takes the sequence from its
// aggregate's own revision, under the same lock that serializes the domain
// mutation — which is one writer per key, the shape measured here.
func BenchmarkOutboxAppendConcurrent(b *testing.B) {
	const writers = 16

	for _, ordered := range []bool{false, true} {
		shape := "unordered"
		if ordered {
			shape = "ordered-one-writer-per-key"
		}
		b.Run(shape, func(b *testing.B) {
			ctx, pool, store := newOutboxBenchmarkFixtureWithConns(b, 0, writers)
			var writer, appended atomic.Int64

			b.SetParallelism(writers)
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				key := fmt.Sprintf("key-%d", writer.Add(1))
				sequence := int64(0)
				for pb.Next() {
					sequence++
					event := outboxEvent(fmt.Sprintf("%s-%d", key, sequence))
					if ordered {
						event.OrderingKey = key
						event.OrderingSequence = sequence
					}
					if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
						return store.Append(ctx, tx, event)
					}); err != nil {
						b.Fatalf("Append(): %v", err)
					}
					appended.Add(1)
				}
			})
			b.StopTimer()
			reportOutboxEventCost(b, appended.Load())
		})
	}
}

// BenchmarkOutboxClaimConcurrent measures several relay replicas claiming from
// one backlog. `SKIP LOCKED` is what makes that safe, and the question it has to
// answer is whether replicas step over each other's locked rows cheaply or walk
// the same index prefix repeatedly.
func BenchmarkOutboxClaimConcurrent(b *testing.B) {
	const (
		backlog   = 20000
		batchSize = 100
	)

	for _, relays := range []int{2, 8} {
		b.Run(fmt.Sprintf("relays-%d", relays), func(b *testing.B) {
			ctx, _, store := newOutboxBenchmarkFixtureWithConns(b, backlog, relays)
			var claimed atomic.Int64

			b.SetParallelism(relays)
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					batch, err := store.Claim(ctx, time.Minute, batchSize)
					if err != nil {
						b.Fatalf("Claim(): %v", err)
					}
					if len(batch.Events) == 0 {
						continue
					}
					releases := make([]postgresoutbox.RetryDirective, len(batch.Events))
					for index, claim := range batch.Events {
						releases[index] = postgresoutbox.RetryDirective{
							ID: claim.Event.ID, ErrorClass: "publisher_temporary",
						}
					}
					if err := store.ScheduleRetryBatch(ctx, batch.Token, releases); err != nil {
						b.Fatalf("ScheduleRetryBatch(): %v", err)
					}
					claimed.Add(int64(len(batch.Events)))
				}
			})
			b.StopTimer()
			reportOutboxEventCost(b, claimed.Load())
		})
	}
}

// BenchmarkOutboxObserve measures the periodic state observation against the
// backlogs it has to survive. Observation reads every unpublished row, so its
// cost tracks backlog — and a backlog is largest exactly when the broker is down
// and the operator most needs the signal.
func BenchmarkOutboxObserve(b *testing.B) {
	for _, pending := range []int{5_000, 100_000} {
		b.Run(fmt.Sprintf("pending-%d", pending), func(b *testing.B) {
			ctx, _, store := newOutboxBenchmarkFixture(b, pending)
			var observations int64

			b.ReportAllocs()
			for b.Loop() {
				if _, err := store.Observe(ctx); err != nil {
					b.Fatalf("Observe(): %v", err)
				}
				observations++
			}
			b.StopTimer()
			reportOutboxEventCost(b, observations)
		})
	}
}

// BenchmarkOutboxIdentifierShape measures what the shape of an event id costs
// the primary key it is stored in.
//
// `NewID` returns 26 random base32 characters, so every append lands on a random
// leaf of a primary key that already holds the whole retention window, and
// retention deletes the oldest rows, whose keys are scattered through that same
// index. A time-ordered identifier of the same length and alphabet turns both
// into work at one edge of the tree. Every other case in this file seeds
// sequential ids, so none of them can see this.
//
// Both cases seed their existing rows with the generator the case then uses,
// because the question is about the distribution of the whole index rather than
// about one insert.
func BenchmarkOutboxIdentifierShape(b *testing.B) {
	const existing = 200_000

	for _, shape := range []string{"random", "time-ordered"} {
		b.Run("append-"+shape, func(b *testing.B) {
			ctx, pool, store := newOutboxBenchmarkFixture(b, 0)
			nextID := outboxIdentifier(shape)
			seedOutboxIdentifiers(b, ctx, pool, existing, nextID, false)
			analyzeOutboxFixture(b, ctx, pool)
			var appended int64

			b.ReportAllocs()
			for b.Loop() {
				event := outboxEvent(nextID())
				if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
					return store.Append(ctx, tx, event)
				}); err != nil {
					b.Fatalf("Append(): %v", err)
				}
				appended++
			}
			b.StopTimer()
			reportOutboxEventCost(b, appended)
		})

		b.Run("cleanup-"+shape, func(b *testing.B) {
			const batchSize = 1000
			ctx, pool, store := newOutboxBenchmarkFixture(b, 0)
			nextID := outboxIdentifier(shape)
			seed := func() { seedOutboxIdentifiers(b, ctx, pool, existing, nextID, true) }
			seed()
			analyzeOutboxFixture(b, ctx, pool)
			runOutboxCleanupCycle(b, ctx, pool, store, batchSize, seed)
		})
	}
}

// outboxIdentifier returns the id generator for one case. "random" is the
// package's own generator, so that case measures the shipped default;
// "time-ordered" is the same length and alphabet with a monotonic value, which
// is what a ULID-style identifier would give.
func outboxIdentifier(shape string) func() string {
	if shape == "random" {
		return postgresoutbox.NewID
	}
	var counter atomic.Int64
	return func() string {
		// Base32 over the alphabet NewID uses, most significant digit first, so
		// ASCII order and issue order agree.
		const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
		value := counter.Add(1)
		id := make([]byte, 26)
		for index := len(id) - 1; index >= 0; index-- {
			id[index] = alphabet[value&31]
			value >>= 5
		}
		return string(id)
	}
}

// seedOutboxIdentifiers fills the table the case appends into or trims, using
// that case's generator. The ids travel as one array so a 200,000-row fixture is
// one statement rather than one round trip per row.
func seedOutboxIdentifiers(
	b *testing.B,
	ctx context.Context,
	pool *postgres.Pool,
	count int,
	nextID func() string,
	published bool,
) {
	b.Helper()
	ids := make([]string, count)
	for index := range ids {
		ids[index] = nextID()
	}
	publishedAt := "NULL::timestamptz"
	if published {
		publishedAt = "clock_timestamp() - interval '1 hour'"
	}
	if _, err := pool.PGX().Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata,
			published_at
		)
		SELECT
			seeded,
			'bench.event', 'bench', 'events', 'v1', clock_timestamp(),
			convert_to('{"sequence":1}', 'UTF8'), convert_to('{}', 'UTF8'),
			`+publishedAt+`
		FROM unnest($1::text[]) AS seeded`, ids); err != nil {
		b.Fatalf("seed outbox identifiers: %v", err)
	}
}

// outboxBenchmarkPayload builds one JSON object of the requested size whose
// value bytes are pseudo-random hex.
//
// The randomness is the point. PostgreSQL compresses a value only once the row
// passes its TOAST target, and a filler of one repeated byte compresses to
// nothing — which would measure a best case no real event reaches and would hide
// whatever compression actually costs. Hex halves under the default compressor,
// which is close to what a business payload does. The seed is fixed so every
// case and every side of a comparison stores identical bytes.
func outboxBenchmarkPayload(size int) []byte {
	const envelope = `{"data":""}`
	if size <= len(envelope) {
		size = len(envelope) + 1
	}
	random := rand.New(rand.NewPCG(0x6f7574626f78, 0x62656e6368))
	value := make([]byte, size-len(envelope))
	const hex = "0123456789abcdef"
	for index := range value {
		value[index] = hex[random.UintN(16)]
	}
	return []byte(`{"data":"` + string(value) + `"}`)
}

func newOutboxBenchmarkFixture(b *testing.B, backlog int) (context.Context, *postgres.Pool, *postgresoutbox.Store) {
	b.Helper()
	return newOutboxBenchmarkFixtureWithConns(b, backlog, 4)
}

// newOutboxBenchmarkFixtureWithConns sizes the pool for cases that measure
// concurrent callers. A relay runs one statement at a time, so four connections
// are enough for every sequential case; a contention case needs one per caller
// or it measures the pool's acquire queue instead of PostgreSQL.
func newOutboxBenchmarkFixtureWithConns(
	b *testing.B,
	backlog, maxConns int,
) (context.Context, *postgres.Pool, *postgresoutbox.Store) {
	b.Helper()

	ctx, cancel := context.WithTimeout(b.Context(), 10*time.Minute)
	b.Cleanup(cancel)
	dsn := pgtest.Migrated(b, os.DirFS(".."), "migrations")
	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     3 * time.Second,
		HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns:       maxConns,
		AcquireTimeout:     10 * time.Second,
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

// seedOutboxRetained fills the published rows a service is holding for its
// retention window. They are already an hour old, so the retention path treats
// them as expired at any realistic setting.
func seedOutboxRetained(b *testing.B, ctx context.Context, pool *postgres.Pool, count int) {
	b.Helper()
	if count == 0 {
		return
	}
	if _, err := pool.PGX().Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata,
			published_at
		)
		SELECT
			'kept-' || generated,
			'bench.event',
			'bench',
			'events',
			'v1',
			clock_timestamp(),
			convert_to('{"sequence":' || generated || '}', 'UTF8'),
			convert_to('{}', 'UTF8'),
			clock_timestamp() - interval '1 hour'
		FROM generate_series(1, $1) AS generated`, count); err != nil {
		b.Fatalf("seed retained outbox rows: %v", err)
	}
}

// seedOutboxSizedBacklog is seedOutboxBacklog with the payload size a case
// declares, so claim decoding and TOAST behave as they would for that event.
func seedOutboxSizedBacklog(b *testing.B, ctx context.Context, pool *postgres.Pool, count, payloadBytes int) {
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
			$2::bytea,
			convert_to('{}', 'UTF8')
		FROM generate_series(1, $1) AS generated`, count, outboxBenchmarkPayload(payloadBytes)); err != nil {
		b.Fatalf("seed sized outbox backlog: %v", err)
	}
}

// seedOutboxRetryWaiting fills rows that are pending but not yet due, which is
// what a broker outage leaves behind once the relay has backed them off.
func seedOutboxRetryWaiting(b *testing.B, ctx context.Context, pool *postgres.Pool, count int) {
	b.Helper()
	if _, err := pool.PGX().Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, source, destination, schema_name, occurred_at, payload, metadata,
			created_at, available_at, cycle_attempt_count, total_attempt_count, last_error_class
		)
		SELECT
			'wait-' || generated,
			'bench.event', 'bench', 'events', 'v1', clock_timestamp(),
			convert_to('{"sequence":' || generated || '}', 'UTF8'), convert_to('{}', 'UTF8'),
			clock_timestamp() - interval '1 hour',
			clock_timestamp() + interval '1 hour', 3, 3, 'publisher_temporary'
		FROM generate_series(1, $1) AS generated`, count); err != nil {
		b.Fatalf("seed retry-waiting outbox rows: %v", err)
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

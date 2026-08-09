//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/jackc/pgx/v5"
)

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
	batch, err := store.Claim(ctx, lease, 1, 5)
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

func poisonOutcomeUnknown(t *testing.T, ctx context.Context, store *postgresoutbox.Store, claim outboxClaim) {
	t.Helper()
	if err := store.MarkPoisonedBatch(ctx, claim.Token, []postgresoutbox.PoisonDirective{{
		ID: claim.Event.ID, ErrorClass: "outcome_unknown", PublicationUncertain: true,
	}}); err != nil {
		t.Fatalf("mark outcome unknown: %v", err)
	}
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

// outboxWaitTimeout and outboxPollInterval are the one cadence every wait in
// this file uses — including the two selects that wait on a channel rather than
// poll. A literal duration here is drift: it once reached six copies, one of
// them ticking at 1ms.
//
// The timeout is an outer failure diagnostic, not a synchronization mechanism.
// Every wait is on an owned event or a durable state change; this only bounds
// how long a broken one hangs.
const outboxWaitTimeout = 10 * time.Second

// shortOutboxLease is the lease a recovery test claims under, and
// expireOutboxLease is how it lets that lease lapse. Kept together because the
// sleep must outlast the lease and the two are otherwise unrelated numbers in
// unrelated units — one a Duration, one a float of PostgreSQL seconds.
const shortOutboxLease = 5 * time.Millisecond

func expireOutboxLease(t *testing.T, ctx context.Context, pool *postgres.Pool) {
	t.Helper()
	postgresSleep(t, ctx, pool, 4*shortOutboxLease.Seconds())
}

// waitForOutbox polls condition until it holds, at this file's one timeout.
//
// describe is called only at the deadline, so a failure can report the value
// that was actually last observed rather than restating what was wanted — which
// is why this routes through waittest.UntilFunc rather than waittest.Until.
// Each description completes "timed out waiting for ...".
func waitForOutbox(t *testing.T, describe func() string, condition func() bool) {
	t.Helper()
	waittest.UntilFunc(t, outboxWaitTimeout, condition, describe)
}

// outboxBlockedBy reports whether any backend is waiting on a lock that the
// given backend holds.
//
// The inbox suite asks the same question with its own copy of this statement, and
// that duplication is required rather than missed: OUTBOX=none removes this file
// while INBOX=postgres keeps that one, so a shared helper declared here would not
// exist in every profile that needs it.
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

func outboxBackendCount(t *testing.T, ctx context.Context, pool *postgres.Pool, predicate string) int {
	t.Helper()
	var count int
	query := "SELECT count(*) FROM pg_stat_activity WHERE pid <> pg_backend_pid() AND " + predicate
	if err := pool.PGX().QueryRow(ctx, query).Scan(&count); err != nil {
		t.Fatalf("count outbox backends for %s: %v", predicate, err)
	}
	return count
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

func waitForOutboxCount(t *testing.T, ctx context.Context, pool *postgres.Pool, predicate string, want int) {
	t.Helper()
	var count int
	waitForOutbox(t,
		func() string {
			return fmt.Sprintf("outbox count for %q to reach %d, last seen %d", predicate, want, count)
		},
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
		func() string { return "the outbox append listener to subscribe" },
		func() bool {
			return outboxBackendExists(t, ctx, pool, "query LIKE 'LISTEN %outbox_appended%'")
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
				return "a blocked observation, but no backend is running a statement matching " +
					observationStatementProbe +
					"; the ObserveOutbox query was probably renamed, so this probe no longer finds it"
			}
			return "the relay observation to block behind the maintenance gate"
		},
		func() bool {
			return outboxBackendExists(t, ctx, pool,
				"wait_event_type = 'Lock' AND "+observationStatementProbe)
		})
}

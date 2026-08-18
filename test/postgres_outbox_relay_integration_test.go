//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

func TestPostgresOutboxRelayListenerStop(t *testing.T) {
	for _, stopRelay := range []struct {
		name string
		stop func(*postgresoutbox.Relay, context.CancelFunc)
	}{
		{name: "drain", stop: func(relay *postgresoutbox.Relay, _ context.CancelFunc) { relay.StartDrain() }},
		{name: "cancellation", stop: func(_ *postgresoutbox.Relay, cancel context.CancelFunc) { cancel() }},
	} {
		t.Run(stopRelay.name, func(t *testing.T) {
			ctx, pool, store := newOutboxFixture(t)
			relayCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			config := testRelayConfig()
			config.PollInterval = time.Hour
			relay := mustNewOutboxRelay(t, store, testPublisherFunc(func(context.Context, postgresoutbox.Event) error { return nil }), nil, config)
			result := runOutboxRelay(relayCtx, relay)

			listenerPID := outboxListenerPID(t, ctx, pool)
			stopRelay.stop(relay, cancel)
			assertRelayResult(t, result, nil)
			if relay.Ready() {
				t.Fatal("relay remained ready after listener stop")
			}
			waitForOutbox(t, func() string { return "the stopped listener backend to disappear" }, func() bool {
				return !outboxBackendExists(t, ctx, pool, fmt.Sprintf("pid = %d", listenerPID))
			})
			if _, err := store.Observe(ctx); err != nil {
				t.Fatalf("Observe() after listener stop: %v", err)
			}
		})
	}
}

func outboxListenerPID(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int {
	t.Helper()
	var pid int
	waitForOutbox(t, func() string { return "an idle outbox listener in ClientRead" }, func() bool {
		if err := pool.QueryRow(ctx, `
			SELECT COALESCE((
				SELECT pid FROM pg_stat_activity
				WHERE pid <> pg_backend_pid()
				  AND query LIKE 'LISTEN %outbox_appended%'
				  AND state = 'idle'
				  AND wait_event_type = 'Client'
				  AND wait_event = 'ClientRead'
				LIMIT 1
			), 0)`).Scan(&pid); err != nil {
			t.Fatalf("read listener backend: %v", err)
		}
		return pid != 0
	})
	return pid
}

func TestPostgresOutboxRequestContinuesDuringBrokerOutage(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	if _, err := pool.Exec(ctx, `CREATE TABLE outbox_http_probe (id text PRIMARY KEY)`); err != nil {
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
		err := postgres.InTx(request.Context(), pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
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
	if err := pool.QueryRow(ctx, `SELECT
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
	if _, err := pool.Exec(ctx, "UPDATE outbox_events SET available_at = clock_timestamp() + interval '1 hour'"); err != nil {
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

	lockTx, err := pool.Begin(ctx)
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
	lockTx, err := pool.Begin(ctx)
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
// four tables. It reaches outbox_redrives only through that relation's
// storage-bytes column, so dropping the column drops that share of the gate — this test is
// what fails, and it says so.
func TestPostgresOutboxStartupRequiresRedriveLedger(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	if _, err := pool.Exec(ctx, "DROP TABLE outbox_redrives"); err != nil {
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

func TestPostgresOutboxStartupRequiresReceiptLedger(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	if _, err := pool.Exec(ctx, "DROP TABLE outbox_commit_receipts"); err != nil {
		t.Fatalf("drop commit receipt ledger: %v", err)
	}
	var attempts atomic.Int64
	relay := mustNewOutboxRelay(t, store, testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
		attempts.Add(1)
		return nil
	}), nil, testRelayConfig())
	result := readRelayResult(t, runOutboxRelay(ctx, relay))
	if result.Err == nil || result.CleanupUnsafe || relay.Ready() || attempts.Load() != 0 {
		t.Fatalf("Run() without receipt ledger = %+v ready=%t attempts=%d",
			result, relay.Ready(), attempts.Load())
	}
}

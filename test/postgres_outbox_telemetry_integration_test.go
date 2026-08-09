//go:build integration

package integration_test

import (
	"reflect"
	"testing"
	"time"
)

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
		"receipts/total":         observation.ReceiptsBytes,
		"receipts/indexes":       observation.ReceiptsIndexBytes,
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

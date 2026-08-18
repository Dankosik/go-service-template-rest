//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/jackc/pgx/v5/pgxpool"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

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
) (context.Context, *pgxpool.Pool, *postgresoutbox.Store, *sdkmetric.ManualReader, *postgresoutbox.Telemetry) {
	t.Helper()
	ctx, pool, _ := newOutboxFixture(t)
	reader, telemetry := newOutboxTelemetry(t)
	store, err := postgresoutbox.NewStore(pool, telemetry)
	if err != nil {
		t.Fatalf("NewStore(): %v", err)
	}
	return ctx, pool, store, reader, telemetry
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
			return fmt.Sprintf("outbox operation %s/%s to reach %d, last seen %d", operation, outcome, want, got)
		},
		func() bool {
			got = outboxOperationCount(t, reader, operation, outcome)
			return got >= want
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
	pool *pgxpool.Pool,
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
		if err := pool.QueryRow(ctx, "SELECT created_at FROM outbox_events WHERE id = $1", state.id).Scan(&want); err != nil {
			t.Fatalf("read direct oldest timestamp for %s: %v", state.id, err)
		}
		if delta := state.got.Sub(want.UTC()); delta < -time.Microsecond || delta > time.Microsecond {
			t.Fatalf("oldest timestamp for %s = %v, direct SQL %v", state.id, state.got, want)
		}
	}
	var publishedAt time.Time
	if err := pool.QueryRow(ctx, "SELECT published_at FROM outbox_events WHERE id = 'observe-published'").Scan(&publishedAt); err != nil {
		t.Fatalf("read direct published oldest timestamp: %v", err)
	}
	if delta := observation.PublishedRetainedOldestAt.Sub(publishedAt.UTC()); delta < -time.Microsecond || delta > time.Microsecond {
		t.Fatalf("published oldest = %v, direct SQL %v", observation.PublishedRetainedOldestAt, publishedAt)
	}

	var heads, eventsBytes, eventIndexes, headBytes, headIndexes, redriveBytes, redriveIndexes,
		receiptBytes, receiptIndexes int64
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM outbox_ordering_heads),
		pg_total_relation_size('outbox_events'), pg_indexes_size('outbox_events'),
		pg_total_relation_size('outbox_ordering_heads'), pg_indexes_size('outbox_ordering_heads'),
		pg_total_relation_size('outbox_redrives'), pg_indexes_size('outbox_redrives'),
		pg_total_relation_size('outbox_commit_receipts'), pg_indexes_size('outbox_commit_receipts')`).Scan(
		&heads, &eventsBytes, &eventIndexes, &headBytes, &headIndexes, &redriveBytes, &redriveIndexes,
		&receiptBytes, &receiptIndexes,
	); err != nil {
		t.Fatalf("read direct outbox storage: %v", err)
	}
	want := []int64{
		heads, eventsBytes, eventIndexes, headBytes, headIndexes, redriveBytes, redriveIndexes,
		receiptBytes, receiptIndexes,
	}
	got := []int64{
		observation.OrderingHeadCount,
		observation.EventsBytes,
		observation.EventsIndexBytes,
		observation.OrderingHeadsBytes,
		observation.OrderingHeadsIndexBytes,
		observation.RedrivesBytes,
		observation.RedrivesIndexBytes,
		observation.ReceiptsBytes,
		observation.ReceiptsIndexBytes,
	}
	if !slices.Equal(got, want) {
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

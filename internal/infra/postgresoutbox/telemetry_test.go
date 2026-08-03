package postgresoutbox

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestTelemetryBoundedContract(t *testing.T) {
	const payloadCanary = "PAYLOAD_CANARY"

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	var logs bytes.Buffer
	telemetry, err := NewTelemetry(
		provider.Meter(telemetryScope),
		slog.New(slog.NewJSONHandler(&logs, nil)),
	)
	if err != nil {
		t.Fatalf("NewTelemetry() error = %v", err)
	}
	t.Cleanup(telemetry.Close)

	now := time.Unix(100, 0).UTC()
	telemetry.RecordObservation(StateObservation{
		EligibleCount:             1,
		EligibleOldestAt:          time.Unix(1, 0).UTC(),
		InProgressCount:           2,
		InProgressOldestAt:        time.Unix(2, 0).UTC(),
		RetryWaitCount:            3,
		RetryWaitOldestAt:         time.Unix(3, 0).UTC(),
		RecoveryDueCount:          4,
		RecoveryDueOldestAt:       time.Unix(4, 0).UTC(),
		OrderingBlockedCount:      5,
		OrderingBlockedOldestAt:   time.Unix(5, 0).UTC(),
		PoisonCount:               6,
		PoisonOldestAt:            time.Unix(6, 0).UTC(),
		PublishedRetainedEstimate: 7,
		PublishedRetainedOldestAt: time.Unix(7, 0).UTC(),
		OrderingHeadCount:         8,
		EventsBytes:               9,
		EventsIndexBytes:          10,
		OrderingHeadsBytes:        11,
		OrderingHeadsIndexBytes:   12,
	}, now)
	telemetry.RecordProgress(now.Add(time.Second))
	telemetry.SetProcessState(true, 1, time.Hour)
	operations := []struct {
		operation string
		outcome   string
		errorType string
	}{
		{operation: "append", outcome: "success", errorType: "none"},
		{operation: "claim", outcome: "empty", errorType: "none"},
		{operation: "recovery", outcome: "success", errorType: "none"},
		{operation: "publish", outcome: "error", errorType: "publisher_temporary"},
		{operation: "mark_published", outcome: "reconciled", errorType: "none"},
		{operation: "schedule_retry", outcome: "success", errorType: "none"},
		{operation: "poison", outcome: "success", errorType: "publisher_permanent"},
		{operation: "redrive", outcome: "rejected", errorType: "validation"},
		{operation: "cleanup", outcome: "success", errorType: "none"},
		{operation: "observe", outcome: "success", errorType: "none"},
		{operation: "drain", outcome: "started", errorType: "none"},
		{operation: "unknown", outcome: "unknown", errorType: "unknown"},
	}
	for _, operation := range operations {
		telemetry.RecordOperation(t.Context(), operation.operation, operation.outcome, operation.errorType, time.Millisecond)
	}
	telemetry.LogPoison(t.Context(), "event-1", "attempt_exhausted", 10)
	telemetry.LogPublisherStuck(t.Context())
	telemetry.LogRecovery(t.Context(), "event-1", 2)

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	wantPoints := map[string]int{
		"outbox.relay.messages":                7,
		"outbox.relay.oldest.timestamp":        7,
		"outbox.relay.observation.timestamp":   1,
		"outbox.relay.last_progress.timestamp": 1,
		"outbox.relay.ordering_heads":          1,
		"outbox.relay.storage.bytes":           6,
		"outbox.relay.inflight":                1,
		"outbox.relay.readiness":               1,
		"outbox.relay.operations":              len(operations),
		"outbox.relay.operation.duration":      len(operations),
	}
	for _, scope := range collected.ScopeMetrics {
		for _, measured := range scope.Metrics {
			want, required := wantPoints[measured.Name]
			if !required {
				continue
			}
			sets := metricAttributeSets(t, measured.Data)
			if len(sets) != want {
				t.Fatalf("%s points = %d, want %d", measured.Name, len(sets), want)
			}
			for _, set := range sets {
				for _, pair := range set.ToSlice() {
					switch string(pair.Key) {
					case "state", "relation", "kind", "operation", "outcome", "error.type":
					default:
						t.Fatalf("%s has unbounded attribute %q", measured.Name, pair.Key)
					}
				}
			}
			delete(wantPoints, measured.Name)
		}
	}
	if len(wantPoints) != 0 {
		t.Fatalf("missing outbox metrics: %v", wantPoints)
	}
	if strings.Contains(logs.String(), payloadCanary) {
		t.Fatalf("outbox log leaked payload canary: %s", logs.String())
	}
	for _, required := range []string{"outbox_event_poisoned", "event-1", "attempt_exhausted"} {
		if !strings.Contains(logs.String(), required) {
			t.Fatalf("outbox poison log missing %q: %s", required, logs.String())
		}
	}
}

func TestTelemetryReadinessRequiresFreshObservation(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	telemetry, err := NewTelemetry(provider.Meter(telemetryScope), nil)
	if err != nil {
		t.Fatalf("NewTelemetry() error = %v", err)
	}
	t.Cleanup(telemetry.Close)

	telemetry.RecordObservation(StateObservation{}, time.Now())
	telemetry.SetProcessState(true, 0, time.Minute)
	if got := collectInt64Gauge(t, reader, "outbox.relay.readiness"); got != 1 {
		t.Fatalf("fresh readiness gauge = %d, want 1", got)
	}
	telemetry.RecordObservation(StateObservation{}, time.Now().Add(-2*time.Minute))
	if got := collectInt64Gauge(t, reader, "outbox.relay.readiness"); got != 0 {
		t.Fatalf("stale readiness gauge = %d, want 0", got)
	}
}

func TestRelayMarkPublishedReconciliationRecordsDurableProgress(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	telemetry, err := NewTelemetry(provider.Meter(telemetryScope), nil)
	if err != nil {
		t.Fatalf("NewTelemetry() error = %v", err)
	}
	t.Cleanup(telemetry.Close)

	relay := &Relay{
		telemetry: telemetry,
		markEvent: func(context.Context, string, ClaimedEvent) error { return errors.New("ambiguous result") },
		getEvent: func(context.Context, string) (Record, error) {
			return Record{PublishedAt: time.Now()}, nil
		},
	}
	if err := relay.reconcilePublished(t.Context(), "token", ClaimedEvent{Event: Event{ID: "event"}}); err != nil {
		t.Fatalf("reconcilePublished() error = %v", err)
	}

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	var reconciled bool
	for _, scope := range collected.ScopeMetrics {
		for _, measured := range scope.Metrics {
			if measured.Name == "outbox.relay.operations" {
				sum, ok := measured.Data.(metricdata.Sum[int64])
				if !ok {
					t.Fatalf("outbox.relay.operations data type = %T, want metricdata.Sum[int64]", measured.Data)
				}
				for _, point := range sum.DataPoints {
					operation, _ := point.Attributes.Value(attribute.Key("operation"))
					outcome, _ := point.Attributes.Value(attribute.Key("outcome"))
					reconciled = reconciled || operation.AsString() == "mark_published" && outcome.AsString() == "reconciled"
				}
			}
			if measured.Name == "outbox.relay.last_progress.timestamp" {
				gauge, ok := measured.Data.(metricdata.Gauge[float64])
				if !ok {
					t.Fatalf("outbox.relay.last_progress.timestamp data type = %T, want metricdata.Gauge[float64]", measured.Data)
				}
				if len(gauge.DataPoints) == 0 || gauge.DataPoints[0].Value == 0 {
					t.Fatal("reconciled durable publication did not record progress")
				}
			}
		}
	}
	if !reconciled {
		t.Fatal("reconciled mark_published operation was not recorded")
	}
}

func collectInt64Gauge(t *testing.T, reader *sdkmetric.ManualReader, name string) int64 {
	t.Helper()
	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	for _, scope := range collected.ScopeMetrics {
		for _, measured := range scope.Metrics {
			if measured.Name != name {
				continue
			}
			gauge, ok := measured.Data.(metricdata.Gauge[int64])
			if !ok || len(gauge.DataPoints) != 1 {
				t.Fatalf("%s aggregation = %T with unexpected points", name, measured.Data)
			}
			return gauge.DataPoints[0].Value
		}
	}
	t.Fatalf("metric %s was not collected", name)
	return 0
}

func metricAttributeSets(t *testing.T, aggregation metricdata.Aggregation) []attribute.Set {
	t.Helper()
	sets := make([]attribute.Set, 0)
	switch data := aggregation.(type) {
	case metricdata.Gauge[int64]:
		for _, point := range data.DataPoints {
			sets = append(sets, point.Attributes)
		}
	case metricdata.Gauge[float64]:
		for _, point := range data.DataPoints {
			sets = append(sets, point.Attributes)
		}
	case metricdata.Sum[int64]:
		for _, point := range data.DataPoints {
			sets = append(sets, point.Attributes)
		}
	case metricdata.Histogram[float64]:
		for _, point := range data.DataPoints {
			sets = append(sets, point.Attributes)
		}
	default:
		t.Fatalf("unexpected metric aggregation %T", aggregation)
	}
	return sets
}

// Every telemetry entry point is nil-safe, so callers that never built a
// recorder do not guard each call site.
func TestTelemetryNilReceiverIsSafe(t *testing.T) {
	t.Parallel()

	var telemetry *Telemetry
	telemetry.Close()
	telemetry.RecordObservation(StateObservation{}, time.Now())
	telemetry.RecordProgress(time.Now())
	telemetry.SetProcessState(true, 1, time.Hour)
	telemetry.RecordOperation(t.Context(), "claim", "success", "none", time.Second)
	telemetry.LogPoison(t.Context(), "event", "publisher_permanent", 1)
	telemetry.LogPublisherStuck(t.Context())
	telemetry.LogRecovery(t.Context(), "event", 1)
	telemetry.LogListenerRetry(t.Context(), errors.New("listener"))
}

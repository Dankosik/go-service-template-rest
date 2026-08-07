package postgresoutbox

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"maps"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// error.type names a failure in three places: this metric attribute, the
// LogPoison log field, and the stored last_error_class column. Only
// boundedErrorType bounds it, so a class the package produces but that list
// forgot would report "other" on the metric while the log and the row report
// the real value — an operator correlating the two would find no matching
// series. This drives the real producers rather than restating the list.
func TestErrorClassVocabularyIsBounded(t *testing.T) {
	t.Parallel()

	produced := map[string]string{
		"publisher panic":     publicationErrorClass(ErrPublisherPanic),
		"permanent rejection": publicationErrorClass(fmtPermanent()),
		"not accepted":        publicationErrorClass(ErrPublicationNotAccepted),
		"publish deadline":    publicationErrorClass(context.DeadlineExceeded),
		"publish canceled":    publicationErrorClass(context.Canceled),
		"unclassified":        publicationErrorClass(errors.New("adapter detail")),
	}
	// storeOutcome is the only producer of the store's four classes, so drive it
	// rather than restating them. Its outcome half must also stay inside
	// boundedOutcome, which is the same failure mode one attribute over.
	for name, storeErr := range map[string]error{
		"store success":     nil,
		"store lost lease":  ErrLeaseLost,
		"store rejection":   ErrConfig,
		"store event fault": ErrInvalidEvent,
		"store failure":     errors.New("database detail"),
	} {
		outcome, class := storeOutcome(storeErr)
		produced[name] = class
		if bounded := boundedOutcome(outcome); bounded != outcome {
			t.Errorf("boundedOutcome(%q) from %s = %q, want the outcome unchanged", outcome, name, bounded)
		}
	}

	// classify owns the two classes that reach the poison row and its log line;
	// neither travels through publicationErrorClass.
	relay := newUnitRelay(&relayStoreStub{}, publisherFunc(func(context.Context, Event) error { return nil }))
	exhausted := unitClaim(1)
	exhausted.CycleAttemptCount = relay.config.MaxAttempts
	outcomes := relay.classify(
		[]ClaimedEvent{unitClaim(2), exhausted},
		[]error{fmtPermanent(), ErrPublicationNotAccepted},
	)
	if len(outcomes.poisoned) != 2 {
		t.Fatalf("classify() poisoned = %d, want 2", len(outcomes.poisoned))
	}
	for _, poisoned := range outcomes.poisoned {
		produced["classify "+poisoned.errorClass] = poisoned.errorClass
	}
	// classStuck has no producing function: publishBatch and LogPublisherStuck
	// both name the constant directly, so the constant is the producer and this
	// is the one class this test states rather than drives.
	produced["publisher stuck"] = classStuck

	for source, class := range produced {
		if bounded := boundedErrorType(class); bounded != class {
			t.Errorf("boundedErrorType(%q) from %s = %q, want the class unchanged", class, source, bounded)
		}
	}
}

// Attribute values are a fixed enum, so an unrecognized operation, outcome, or
// error type must collapse to "other" rather than mint a new time series.
// Payload and metadata cannot reach telemetry at all: no entry point on
// Telemetry accepts them.
func TestTelemetryBoundedContract(t *testing.T) {
	var logs bytes.Buffer
	reader, telemetry := newTestTelemetry(t, slog.New(slog.NewJSONHandler(&logs, nil)))

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
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		want, required := wantPoints[measured.Name]
		if !required {
			return
		}
		sets := telemetrytest.AttributeSets(t, measured.Data)
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
				if value := pair.Value.AsString(); value == "unknown" {
					t.Fatalf("%s kept unrecognized attribute value %q=%q instead of collapsing it",
						measured.Name, pair.Key, value)
				}
			}
		}
		delete(wantPoints, measured.Name)
	})
	if len(wantPoints) != 0 {
		t.Fatalf("missing outbox metrics: %v", wantPoints)
	}
	for _, required := range []string{"outbox_event_poisoned", "event-1", "attempt_exhausted"} {
		if !strings.Contains(logs.String(), required) {
			t.Fatalf("outbox poison log missing %q: %s", required, logs.String())
		}
	}
}

func TestTelemetryReadinessRequiresFreshObservation(t *testing.T) {
	reader, telemetry := newTestTelemetry(t, nil)

	telemetry.RecordObservation(StateObservation{}, time.Now())
	telemetry.SetProcessState(true, 0, time.Minute)
	if got := telemetrytest.Int64GaugeValue(t, reader, "outbox.relay.readiness"); got != 1 {
		t.Fatalf("fresh readiness gauge = %d, want 1", got)
	}
	telemetry.RecordObservation(StateObservation{}, time.Now().Add(-2*time.Minute))
	if got := telemetrytest.Int64GaugeValue(t, reader, "outbox.relay.readiness"); got != 0 {
		t.Fatalf("stale readiness gauge = %d, want 0", got)
	}
}

func TestRelayMarkPublishedReconciliationRecordsDurableProgress(t *testing.T) {
	reader, telemetry := newTestTelemetry(t, nil)

	relay := newUnitRelay(&relayStoreStub{
		markUnorderedPublished: func(context.Context, string, string) error {
			return errors.New("ambiguous result")
		},
		get: func(context.Context, string) (Record, error) {
			return Record{PublishedAt: time.Now()}, nil
		},
	}, noopPublisher())
	relay.telemetry = telemetry
	if err := relay.reconcilePublished(
		t.Context(), "token", ClaimedEvent{Event: Event{ID: "event"}}, relay.markOneUnordered,
	); err != nil {
		t.Fatalf("reconcilePublished() error = %v", err)
	}

	var reconciled bool
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		switch measured.Name {
		case "outbox.relay.operations":
			for _, point := range telemetrytest.Int64Sum(t, measured).DataPoints {
				reconciled = reconciled ||
					telemetrytest.Attribute(t, point.Attributes, "operation") == "mark_published" &&
						telemetrytest.Attribute(t, point.Attributes, "outcome") == "reconciled"
			}
		case "outbox.relay.last_progress.timestamp":
			if telemetrytest.SinglePoint(
				t, measured.Name, telemetrytest.Float64Gauge(t, measured).DataPoints,
			) == 0 {
				t.Fatal("reconciled durable publication did not record progress")
			}
		}
	})
	if !reconciled {
		t.Fatal("reconciled mark_published operation was not recorded")
	}
}

// Two relay replicas observing one database report identical database-global
// gauges, because the whole reading travels in the StateObservation and nothing
// else. Process-local operation counts do not travel with it: a replica that has
// run nothing reports no operation series at all, so an operator summing across
// replicas gets one database's backlog and each replica's own work.
//
// This is a property of Telemetry alone. It reaches no database and belongs here
// rather than beside the integration test whose observation it once borrowed.
func TestTelemetryReplicasShareTheObservationButNotOperations(t *testing.T) {
	t.Parallel()

	observation := StateObservation{
		EligibleCount: 3, PoisonCount: 1, OrderingHeadCount: 2,
		EventsBytes: 11, EventsIndexBytes: 12,
		OrderingHeadsBytes: 13, OrderingHeadsIndexBytes: 14,
		RedrivesBytes: 15, RedrivesIndexBytes: 16,
	}
	observedAt := time.Unix(100, 0).UTC()

	firstReader, first := newTestTelemetry(t, nil)
	first.RecordObservation(observation, observedAt)
	first.RecordOperation(t.Context(), "claim", outcomeSuccess, "none", time.Millisecond)

	secondReader, second := newTestTelemetry(t, nil)
	second.RecordObservation(observation, observedAt)

	for _, gauge := range []string{
		"outbox.relay.messages", "outbox.relay.storage.bytes", "outbox.relay.ordering_heads",
	} {
		if got, want := gaugePoints(t, secondReader, gauge), gaugePoints(t, firstReader, gauge); !maps.Equal(got, want) {
			t.Errorf("%s differs between replicas: %v vs %v", gauge, got, want)
		}
	}
	if points := gaugePoints(t, secondReader, "outbox.relay.operations"); len(points) != 0 {
		t.Errorf("a replica that ran no operations reported %v", points)
	}
}

// gaugePoints keys one instrument's points by their attributes, so two readers
// can be compared without depending on collection order.
func gaugePoints(t *testing.T, reader *sdkmetric.ManualReader, name string) map[string]int64 {
	t.Helper()
	points := map[string]int64{}
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		if measured.Name != name {
			return
		}
		for _, point := range telemetrytest.Int64Gauge(t, measured).DataPoints {
			points[point.Attributes.Encoded(attribute.DefaultEncoder())] = point.Value
		}
	})
	return points
}

// newTestTelemetry builds a Telemetry over a manual reader, so a test can drive
// the real instruments and collect what a scrape would see. Pass a logger to
// assert on operator log records too, or nil when only metrics matter —
// NewTelemetry already reads nil as slog.Default().
//
// telemetrytest owns the reader and every metricdata accessor below, so only the
// *Telemetry construction belongs here.
func newTestTelemetry(t *testing.T, logger *slog.Logger) (*sdkmetric.ManualReader, *Telemetry) {
	t.Helper()
	reader, meter := telemetrytest.NewManualMeter(t, TelemetryScope)
	telemetry, err := NewTelemetry(meter, logger)
	if err != nil {
		t.Fatalf("NewTelemetry() error = %v", err)
	}
	t.Cleanup(telemetry.Close)
	return reader, telemetry
}

// outbox.relay.inflight is the whole claimed batch for as long as the relay owns
// it, not the events inside Publish — that is what a crash mid-batch would
// redeliver. It falls back to zero once the batch is finalized.
func TestRelayInflightReportsTheClaimedBatch(t *testing.T) {
	t.Parallel()

	reader, telemetry := newTestTelemetry(t, nil)
	const batchSize = 7
	inside, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	relay := newUnitRelay(&relayStoreStub{}, publisherFunc(func(context.Context, Event) error {
		once.Do(func() { close(inside) })
		<-release
		return nil
	}))
	relay.telemetry = telemetry
	// One worker, so the batch is held at its first event until this test
	// releases it. The publish budget is only an outer diagnostic.
	relay.config.PublishConcurrency = 1
	relay.config.PublishTimeout = time.Minute
	relay.config.LeaseDuration = time.Hour

	go func() {
		defer close(finished)
		if _, stop := relay.publishBatch(context.Background(), unitBatch(batchSize), time.Now().Add(time.Hour)); stop {
			t.Error("publishBatch() stopped on a batch every publication acknowledged")
		}
	}()

	<-inside
	if got := telemetrytest.Int64GaugeValue(t, reader, "outbox.relay.inflight"); got != batchSize {
		t.Errorf("inflight while one event is in Publish = %d, want the whole claimed batch of %d", got, batchSize)
	}
	close(release)
	<-finished
	if got := telemetrytest.Int64GaugeValue(t, reader, "outbox.relay.inflight"); got != 0 {
		t.Errorf("inflight after finalization = %d, want 0", got)
	}
}

// Every count StateObservation carries has to reach a metric. The callback
// reads the observation through a hand-written table, so a state added to the
// struct and to Store.Observe but forgotten there is simply invisible: the
// backlog it counts never appears on a dashboard, and nothing else fails.
//
// TestTelemetryBoundedContract pins how many points each instrument emits,
// which catches a row deleted from the table but not the case this exists for:
// adding a field and never wiring it leaves every one of those counts exactly
// as it was, so that pin still passes. This drives every int64 field to a
// distinct value and requires each to show up in a published point, matching by
// value rather than by name — a field the callback never reads takes its value
// with it.
func TestTelemetryObservesEveryObservedCount(t *testing.T) {
	reader, telemetry := newTestTelemetry(t, nil)

	var observation StateObservation
	fields := reflect.ValueOf(&observation).Elem()
	// Spaced well clear of the readiness and inflight gauges, whose 0/1 values
	// would otherwise look like a field that was observed.
	const spacing = 1000
	unobserved := map[int64]string{}
	for index := range fields.NumField() {
		field := fields.Field(index)
		if field.Kind() != reflect.Int64 {
			continue
		}
		value := int64(index+1) * spacing
		field.SetInt(value)
		unobserved[value] = fields.Type().Field(index).Name
	}
	if len(unobserved) == 0 {
		t.Fatal("StateObservation exposes no counts; this test is asserting nothing")
	}
	telemetry.RecordObservation(observation, time.Now())

	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		// Not telemetrytest.Int64Gauge: every other aggregation is simply not a
		// count this test is looking for, so it skips rather than fails.
		gauge, ok := measured.Data.(metricdata.Gauge[int64])
		if !ok {
			return
		}
		for _, point := range gauge.DataPoints {
			delete(unobserved, point.Value)
		}
	})
	if len(unobserved) != 0 {
		t.Fatalf("StateObservation counts never reached a metric: %v", unobserved)
	}
}

// An operation with no span of its own reaches the counter and leaves the
// latency histogram alone. Recording a placeholder duration for it instead —
// a zero span, or the enclosing operation's elapsed time — is invisible in a
// counter assertion and ruins the histogram it lands in.
func TestCountOperationLeavesTheDurationHistogramAlone(t *testing.T) {
	reader, telemetry := newTestTelemetry(t, slog.New(slog.DiscardHandler))

	telemetry.CountOperation(t.Context(), "drain", "started", "none")
	telemetry.CountOperation(t.Context(), "recovery", "success", "none")

	counted := 0
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		switch measured.Name {
		case "outbox.relay.operations":
			counted = len(telemetrytest.AttributeSets(t, measured.Data))
		case "outbox.relay.operation.duration":
			t.Errorf("CountOperation() recorded %d duration points, want none",
				len(telemetrytest.AttributeSets(t, measured.Data)))
		}
	})
	if counted != 2 {
		t.Errorf("outbox.relay.operations points = %d, want 2", counted)
	}
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
	telemetry.CountOperation(t.Context(), "drain", "started", "none")
	telemetry.LogPoison(t.Context(), "event", "publisher_permanent", 1)
	telemetry.LogPublisherStuck(t.Context())
	telemetry.LogRecovery(t.Context(), "event", 1)
	telemetry.LogListenerRetry(t.Context(), "connect")
}

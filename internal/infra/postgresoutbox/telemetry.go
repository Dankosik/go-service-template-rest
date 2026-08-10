package postgresoutbox

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	infratelemetry "github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// TelemetryScope names this package's instrumentation. Callers that build the
// meter themselves pass it to Meter so the scope stays one string; NewTelemetry
// uses it for the global-provider fallback.
const TelemetryScope = "service.outbox.postgres"

// This file owns the instruments, the snapshot the scrape callback reads, and
// the recorders that write it. The closed vocabularies every attribute here is
// bounded through are vocabulary.go.

// Telemetry owns this package's metrics, publish tracer, and operator logger.
// Metrics live in this file, publish spans in telemetry_publish.go, and operator
// records in telemetry_log.go. A nil *Telemetry is a working no-op because
// telemetry is optional at both NewStore and NewRelay.
type Telemetry struct {
	log                  *slog.Logger
	tracer               trace.Tracer
	messages             metric.Int64ObservableGauge
	oldestTimestamp      metric.Float64ObservableGauge
	observationTimestamp metric.Float64ObservableGauge
	lastProgress         metric.Float64ObservableGauge
	orderingHeads        metric.Int64ObservableGauge
	storageBytes         metric.Int64ObservableGauge
	inflight             metric.Int64ObservableGauge
	readiness            metric.Int64ObservableGauge
	operations           metric.Int64Counter
	duration             metric.Float64Histogram
	registration         metric.Registration
	closeOnce            sync.Once
	mu                   sync.RWMutex
	snapshot             telemetrySnapshot
}

type telemetrySnapshot struct {
	observation  StateObservation
	observedAt   time.Time
	lastProgress time.Time
	running      bool
	inflight     int64
	staleAfter   time.Duration
}

func NewTelemetry(meter metric.Meter, logger *slog.Logger) (*Telemetry, error) {
	meter = infratelemetry.MeterOrGlobal(meter, TelemetryScope)
	if logger == nil {
		logger = slog.Default()
	}
	// The tracer comes from the global provider rather than a parameter, because
	// the only process that publishes — cmd/outbox-relay — already installs one
	// and the global W3C propagator during bootstrap. A second constructor
	// argument would let a caller wire a meter and forget the tracer, and there
	// is no composition in this repository that wants them from different
	// providers.
	telemetry := &Telemetry{log: logger, tracer: otel.GetTracerProvider().Tracer(TelemetryScope)}
	set := infratelemetry.NewInstrumentSet(meter)
	set.Int64ObservableGauge(&telemetry.messages, "outbox.relay.messages")
	set.Float64ObservableGauge(&telemetry.oldestTimestamp, "outbox.relay.oldest.timestamp", metric.WithUnit("s"))
	set.Float64ObservableGauge(&telemetry.observationTimestamp, "outbox.relay.observation.timestamp", metric.WithUnit("s"))
	set.Float64ObservableGauge(&telemetry.lastProgress, "outbox.relay.last_progress.timestamp", metric.WithUnit("s"))
	set.Int64ObservableGauge(&telemetry.orderingHeads, "outbox.relay.ordering_heads")
	set.Int64ObservableGauge(&telemetry.storageBytes, "outbox.relay.storage.bytes", metric.WithUnit("By"))
	set.Int64ObservableGauge(&telemetry.inflight, "outbox.relay.inflight")
	set.Int64ObservableGauge(&telemetry.readiness, "outbox.relay.readiness")
	set.Int64Counter(&telemetry.operations, "outbox.relay.operations")
	set.Float64Histogram(&telemetry.duration, "outbox.relay.operation.duration", metric.WithUnit("s"))
	if err := set.Err(); err != nil {
		return nil, err
	}

	registration, err := meter.RegisterCallback(telemetry.collect,
		telemetry.messages,
		telemetry.oldestTimestamp,
		telemetry.observationTimestamp,
		telemetry.lastProgress,
		telemetry.orderingHeads,
		telemetry.storageBytes,
		telemetry.inflight,
		telemetry.readiness,
	)
	if err != nil {
		return nil, fmt.Errorf("register outbox metrics callback: %w", err)
	}
	telemetry.registration = registration
	return telemetry, nil
}

func (t *Telemetry) Close() {
	if t == nil || t.registration == nil {
		return
	}
	t.closeOnce.Do(func() { _ = t.registration.Unregister() })
}

func (t *Telemetry) RecordObservation(observation StateObservation, observedAt time.Time) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.snapshot.observation = observation
	t.snapshot.observedAt = observedAt.UTC()
	t.mu.Unlock()
}

func (t *Telemetry) RecordProgress(at time.Time) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.snapshot.lastProgress = at.UTC()
	t.mu.Unlock()
}

// SetProcessState publishes the relay's lifecycle state for the metric
// callback. running is the lifecycle half of readiness, not the whole verdict:
// the callback combines it with observation freshness on every scrape, because
// the relay only calls this at lifecycle edges and an idle relay can go stale
// between two of them.
func (t *Telemetry) SetProcessState(running bool, inflight int64, staleAfter time.Duration) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.snapshot.running = running
	t.snapshot.inflight = inflight
	t.snapshot.staleAfter = staleAfter
	t.mu.Unlock()
}

// RecordOperation counts one operation and records how long it took. Use it
// only where duration measures the labeled operation end to end; anything
// counted without a span of its own belongs in CountOperation.
func (t *Telemetry) RecordOperation(
	ctx context.Context,
	operation string,
	outcome string,
	errorType string,
	duration time.Duration,
) {
	if t == nil {
		return
	}
	attributes := operationAttributes(operation, outcome, errorType)
	t.operations.Add(ctx, 1, attributes)
	t.duration.Record(ctx, duration.Seconds(), attributes)
}

// CountOperation counts one operation that has no duration of its own: a state
// transition, or a verdict reached partway through a longer operation. Such a
// site has nothing honest to pass as a duration, and the placeholders it would
// otherwise pass — a zero span, or the enclosing operation's elapsed time —
// land in the same histogram as the real measurements and make it unreadable.
// They only ever reach the counter.
func (t *Telemetry) CountOperation(ctx context.Context, operation, outcome, errorType string) {
	if t == nil {
		return
	}
	t.operations.Add(ctx, 1, operationAttributes(operation, outcome, errorType))
}

//nolint:ireturn // metric.WithAttributes returns OTel's own option interface.
func operationAttributes(operation, outcome, errorType string) metric.MeasurementOption {
	return metric.WithAttributes(
		attribute.String("operation", boundedOperation(operation)),
		attribute.String("outcome", boundedOutcome(outcome)),
		attribute.String("error.type", boundedErrorType(errorType)),
	)
}

// collect answers one scrape from the published snapshot. It is the callback
// registered in NewTelemetry, and OTel's own term for that role — the two verbs
// nearby belong elsewhere: Store.Observe runs the observation statement and
// Relay.sampleState runs it and records the result.
func (t *Telemetry) collect(_ context.Context, observer metric.Observer) error {
	t.mu.RLock()
	snapshot := t.snapshot
	t.mu.RUnlock()

	states := []struct {
		name   string
		count  int64
		oldest time.Time
	}{
		{name: "eligible", count: snapshot.observation.EligibleCount, oldest: snapshot.observation.EligibleOldestAt},
		{name: "in_progress", count: snapshot.observation.InProgressCount, oldest: snapshot.observation.InProgressOldestAt},
		{name: "retry_wait", count: snapshot.observation.RetryWaitCount, oldest: snapshot.observation.RetryWaitOldestAt},
		{name: "recovery_due", count: snapshot.observation.RecoveryDueCount, oldest: snapshot.observation.RecoveryDueOldestAt},
		{name: "ordering_blocked", count: snapshot.observation.OrderingBlockedCount, oldest: snapshot.observation.OrderingBlockedOldestAt},
		{name: "poison", count: snapshot.observation.PoisonCount, oldest: snapshot.observation.PoisonOldestAt},
		{name: "outcome_unknown", count: snapshot.observation.OutcomeUnknownCount, oldest: snapshot.observation.OutcomeUnknownOldestAt},
		{name: "published_retained", count: snapshot.observation.PublishedRetainedEstimate, oldest: snapshot.observation.PublishedRetainedOldestAt},
	}
	for _, state := range states {
		attributes := metric.WithAttributes(attribute.String("state", state.name))
		observer.ObserveInt64(t.messages, state.count, attributes)
		observer.ObserveFloat64(t.oldestTimestamp, unixSecondsFromTime(state.oldest), attributes)
	}
	observer.ObserveFloat64(t.observationTimestamp, unixSecondsFromTime(snapshot.observedAt))
	observer.ObserveFloat64(t.lastProgress, unixSecondsFromTime(snapshot.lastProgress))
	observer.ObserveInt64(t.orderingHeads, snapshot.observation.OrderingHeadCount)

	for _, relation := range []struct {
		name         string
		total, index int64
	}{
		{name: "events", total: snapshot.observation.EventsBytes, index: snapshot.observation.EventsIndexBytes},
		{
			name:  "ordering_heads",
			total: snapshot.observation.OrderingHeadsBytes,
			index: snapshot.observation.OrderingHeadsIndexBytes,
		},
		{name: "redrives", total: snapshot.observation.RedrivesBytes, index: snapshot.observation.RedrivesIndexBytes},
		{name: "receipts", total: snapshot.observation.ReceiptsBytes, index: snapshot.observation.ReceiptsIndexBytes},
	} {
		name := attribute.String("relation", relation.name)
		observer.ObserveInt64(t.storageBytes, relation.total,
			metric.WithAttributes(name, attribute.String("kind", "total")))
		observer.ObserveInt64(t.storageBytes, relation.index,
			metric.WithAttributes(name, attribute.String("kind", "indexes")))
	}

	observer.ObserveInt64(t.inflight, snapshot.inflight)
	// The snapshot carries the relay's lifecycle state as of its last
	// SetProcessState, which only happens at lifecycle edges. This callback runs
	// on scrape, so it asks the shared readiness predicate again against the
	// current clock; without that, a relay that stopped observing between two
	// edges would keep reporting ready=1 until its next edge.
	ready := int64(0)
	if readyWithFreshObservation(snapshot.running, snapshot.observedAt, snapshot.staleAfter) {
		ready = 1
	}
	observer.ObserveInt64(t.readiness, ready)
	return nil
}

// unixSecondsFromTime is the inverse of timeFromUnixSeconds in store_rows.go,
// which reads the same values off the observation statement.
func unixSecondsFromTime(value time.Time) float64 {
	if value.IsZero() {
		return 0
	}
	return float64(value.UnixNano()) / float64(time.Second)
}

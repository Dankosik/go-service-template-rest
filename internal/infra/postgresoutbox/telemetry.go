package postgresoutbox

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// TelemetryScope names this package's instrumentation. Callers that build the
// meter themselves pass it to Meter so the scope stays one string; NewTelemetry
// uses it for the global-provider fallback.
const TelemetryScope = "service.outbox.postgres"

// The outcome labels every operation site chooses between. Attribute values are
// a fixed enum, so they are named rather than repeated.
const (
	outcomeSuccess  = "success"
	outcomeError    = "error"
	outcomeRejected = "rejected"
	// boundedOther is what every closed vocabulary in this package collapses an
	// unrecognized value to, so an unexpected string cannot mint a time series.
	boundedOther = "other"
)

// The closed error.type vocabulary. Every class this package produces is named
// here and used at its producing site, so the producer and boundedErrorType
// below reference one identifier rather than two literals that can drift apart.
// A class added at a call site without a constant here is what
// TestErrorClassVocabularyIsBounded fails on.
const (
	classNone               = "none"
	classDatabase           = "database"
	classLostLease          = "lost_lease"
	classValidation         = "validation"
	classStuck              = "stuck"
	classPanic              = "panic"
	classPublisherPermanent = "publisher_permanent"
	classPublisherRejected  = "publisher_rejected"
	classPublisherTimeout   = "publisher_timeout"
	classPublisherCanceled  = "publisher_canceled"
	classPublisherTemporary = "publisher_temporary"
	classAttemptExhausted   = "attempt_exhausted"
)

// Telemetry records this package's metrics and operator logs. A nil *Telemetry
// is a working no-op: every method below returns on a nil receiver, because
// telemetry is optional at both NewStore and NewRelay. Call these methods
// directly rather than guarding at the call site — a second nil check reads as
// if one of the two were load-bearing, and neither is. A method added here
// carries the same guard.
type Telemetry struct {
	log                  *slog.Logger
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
	if meter == nil {
		meter = otel.GetMeterProvider().Meter(TelemetryScope)
	}
	if logger == nil {
		logger = slog.Default()
	}
	telemetry := &Telemetry{log: logger}
	var err error
	if telemetry.messages, err = meter.Int64ObservableGauge("outbox.relay.messages"); err != nil {
		return nil, fmt.Errorf("create outbox messages metric: %w", err)
	}
	if telemetry.oldestTimestamp, err = meter.Float64ObservableGauge("outbox.relay.oldest.timestamp", metric.WithUnit("s")); err != nil {
		return nil, fmt.Errorf("create outbox oldest timestamp metric: %w", err)
	}
	if telemetry.observationTimestamp, err = meter.Float64ObservableGauge("outbox.relay.observation.timestamp", metric.WithUnit("s")); err != nil {
		return nil, fmt.Errorf("create outbox observation timestamp metric: %w", err)
	}
	if telemetry.lastProgress, err = meter.Float64ObservableGauge("outbox.relay.last_progress.timestamp", metric.WithUnit("s")); err != nil {
		return nil, fmt.Errorf("create outbox last progress metric: %w", err)
	}
	if telemetry.orderingHeads, err = meter.Int64ObservableGauge("outbox.relay.ordering_heads"); err != nil {
		return nil, fmt.Errorf("create outbox ordering heads metric: %w", err)
	}
	if telemetry.storageBytes, err = meter.Int64ObservableGauge("outbox.relay.storage.bytes", metric.WithUnit("By")); err != nil {
		return nil, fmt.Errorf("create outbox storage metric: %w", err)
	}
	if telemetry.inflight, err = meter.Int64ObservableGauge("outbox.relay.inflight"); err != nil {
		return nil, fmt.Errorf("create outbox inflight metric: %w", err)
	}
	if telemetry.readiness, err = meter.Int64ObservableGauge("outbox.relay.readiness"); err != nil {
		return nil, fmt.Errorf("create outbox readiness metric: %w", err)
	}
	if telemetry.operations, err = meter.Int64Counter("outbox.relay.operations"); err != nil {
		return nil, fmt.Errorf("create outbox operations metric: %w", err)
	}
	if telemetry.duration, err = meter.Float64Histogram("outbox.relay.operation.duration", metric.WithUnit("s")); err != nil {
		return nil, fmt.Errorf("create outbox operation duration metric: %w", err)
	}
	telemetry.registration, err = meter.RegisterCallback(telemetry.observe,
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

func (t *Telemetry) LogPoison(ctx context.Context, id, errorClass string, attempt int) {
	if t != nil {
		t.log.ErrorContext(ctx, "outbox_event_poisoned", "event_id", id, "error.type", errorClass, "attempt", attempt)
	}
}

func (t *Telemetry) LogPublisherStuck(ctx context.Context) {
	if t != nil {
		t.log.ErrorContext(ctx, "outbox_publisher_stuck", "error.type", classStuck)
	}
}

func (t *Telemetry) LogRecovery(ctx context.Context, id string, attempt int) {
	if t != nil {
		t.log.WarnContext(ctx, "outbox_lease_recovered", "event_id", id, "attempt", attempt)
	}
}

// LogListenerRetry reports that wake-up notifications are unavailable. Pickup
// latency falls back to the poll interval until the listener reconnects.
//
// stage is a bounded class from listenerStage, never the driver's error text.
// A pgx connect failure formats the DSN's user, database, and host into its
// message, and this package promises never to log DSN material. Losing that
// detail costs little: the listener shares the pool's DSN, so a real
// connectivity fault also fails the pool, and that path exits the process with
// a postgres_unavailable class instead of degrading quietly.
func (t *Telemetry) LogListenerRetry(ctx context.Context, stage string) {
	if t != nil {
		t.log.WarnContext(ctx, "outbox_listener_retry", "error.type", classDatabase, "stage", stage)
	}
}

func (t *Telemetry) observe(_ context.Context, observer metric.Observer) error {
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
		{name: "published_retained", count: snapshot.observation.PublishedRetainedEstimate, oldest: snapshot.observation.PublishedRetainedOldestAt},
	}
	for _, state := range states {
		attributes := metric.WithAttributes(attribute.String("state", state.name))
		observer.ObserveInt64(t.messages, state.count, attributes)
		observer.ObserveFloat64(t.oldestTimestamp, unixSeconds(state.oldest), attributes)
	}
	observer.ObserveFloat64(t.observationTimestamp, unixSeconds(snapshot.observedAt))
	observer.ObserveFloat64(t.lastProgress, unixSeconds(snapshot.lastProgress))
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

func unixSeconds(value time.Time) float64 {
	if value.IsZero() {
		return 0
	}
	return float64(value.UnixNano()) / float64(time.Second)
}

// boundedOperation is the closed vocabulary for the operation attribute. The
// unit differs by operation, and the duration histogram is only meaningful per
// operation because of it:
//
//   - append, claim, mark_published, schedule_retry, poison, redrive, cleanup,
//     and observe are one statement each, and carry that statement's duration.
//   - publish is one event, not one batch.
//   - recovery and drain, and the reconciled outcome of mark_published, are
//     counted through CountOperation and carry no duration.
//
// A new operation states its unit here and picks the matching recorder.
func boundedOperation(value string) string {
	switch value {
	case "append", "claim", "recovery", "publish", "mark_published", "schedule_retry", "poison", "redrive", "cleanup", "observe", "drain":
		return value
	default:
		return boundedOther
	}
}

func boundedOutcome(value string) string {
	switch value {
	case outcomeSuccess, outcomeError, outcomeRejected, "empty", "reconciled", "started":
		return value
	default:
		return boundedOther
	}
}

// boundedErrorType is the closed vocabulary for error.type. One class describes
// a failure in three places — this metric attribute, the error.type field of
// LogPoison, and the stored last_error_class column — and only this function
// bounds it, because the column's CHECK constraint bounds length alone. Any
// class the package produces must therefore appear here, or the metric silently
// reports "other" while the log and the row report the real value.
// TestErrorClassVocabularyIsBounded drives the producers and fails on a class
// this list forgot.
func boundedErrorType(value string) string {
	switch value {
	case classNone, classDatabase, classLostLease, classValidation, classStuck, classPanic,
		classPublisherPermanent, classPublisherRejected, classPublisherTimeout,
		classPublisherCanceled, classPublisherTemporary, classAttemptExhausted:
		return value
	default:
		return boundedOther
	}
}

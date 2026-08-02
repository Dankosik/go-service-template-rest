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

const telemetryScope = "service.outbox.postgres"

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
	ready        bool
	inflight     int64
	staleAfter   time.Duration
}

func NewTelemetry(meter metric.Meter, logger *slog.Logger) (*Telemetry, error) {
	if meter == nil {
		meter = otel.GetMeterProvider().Meter(telemetryScope)
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

func (t *Telemetry) SetProcessState(ready bool, inflight int64, staleAfter time.Duration) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.snapshot.ready = ready
	t.snapshot.inflight = inflight
	t.snapshot.staleAfter = staleAfter
	t.mu.Unlock()
}

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
	operation = boundedOperation(operation)
	outcome = boundedOutcome(outcome)
	errorType = boundedErrorType(errorType)
	attributes := metric.WithAttributes(
		attribute.String("operation", operation),
		attribute.String("outcome", outcome),
		attribute.String("error.type", errorType),
	)
	t.operations.Add(ctx, 1, attributes)
	t.duration.Record(ctx, duration.Seconds(), attributes)
}

func (t *Telemetry) LogPoison(ctx context.Context, id, errorClass string, attempt int) {
	if t != nil {
		t.log.ErrorContext(ctx, "outbox_event_poisoned", "event_id", id, "error.type", errorClass, "attempt", attempt)
	}
}

func (t *Telemetry) LogPublisherStuck(ctx context.Context) {
	if t != nil {
		t.log.ErrorContext(ctx, "outbox_publisher_stuck", "error.type", "stuck")
	}
}

func (t *Telemetry) LogRecovery(ctx context.Context, id string, attempt int) {
	if t != nil {
		t.log.WarnContext(ctx, "outbox_lease_recovered", "event_id", id, "attempt", attempt)
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
		{name: "published_retained", count: snapshot.observation.PublishedRetainedCount, oldest: snapshot.observation.PublishedRetainedOldestAt},
	}
	for _, state := range states {
		attributes := metric.WithAttributes(attribute.String("state", state.name))
		observer.ObserveInt64(t.messages, state.count, attributes)
		observer.ObserveFloat64(t.oldestTimestamp, unixSeconds(state.oldest), attributes)
	}
	observer.ObserveFloat64(t.observationTimestamp, unixSeconds(snapshot.observedAt))
	observer.ObserveFloat64(t.lastProgress, unixSeconds(snapshot.lastProgress))
	observer.ObserveInt64(t.orderingHeads, snapshot.observation.OrderingHeadCount)
	observer.ObserveInt64(t.storageBytes, snapshot.observation.EventsBytes,
		metric.WithAttributes(attribute.String("relation", "events"), attribute.String("kind", "total")))
	observer.ObserveInt64(t.storageBytes, snapshot.observation.EventsIndexBytes,
		metric.WithAttributes(attribute.String("relation", "events"), attribute.String("kind", "indexes")))
	observer.ObserveInt64(t.storageBytes, snapshot.observation.OrderingHeadsBytes,
		metric.WithAttributes(attribute.String("relation", "ordering_heads"), attribute.String("kind", "total")))
	observer.ObserveInt64(t.storageBytes, snapshot.observation.OrderingHeadsIndexBytes,
		metric.WithAttributes(attribute.String("relation", "ordering_heads"), attribute.String("kind", "indexes")))
	observer.ObserveInt64(t.inflight, snapshot.inflight)
	ready := int64(0)
	if snapshot.ready && snapshot.staleAfter > 0 && time.Since(snapshot.observedAt) <= snapshot.staleAfter {
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

func boundedOperation(value string) string {
	switch value {
	case "append", "claim", "recovery", "publish", "mark_published", "schedule_retry", "poison", "redrive", "cleanup", "observe", "drain":
		return value
	default:
		return "other"
	}
}

func boundedOutcome(value string) string {
	switch value {
	case "success", "error", "empty", "reconciled", "rejected", "started":
		return value
	default:
		return "other"
	}
}

func boundedErrorType(value string) string {
	switch value {
	case "none", "database", "lost_lease", "validation", "stuck", "panic",
		"publisher_permanent", "publisher_rejected", "publisher_timeout", "publisher_canceled", "publisher_temporary":
		return value
	default:
		return "other"
	}
}

package postgreswebhook

import (
	"context"
	"fmt"
	"sync"
	"time"

	infratelemetry "github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const webhookMeterName = "service.postgres.webhooks"

type telemetrySnapshot struct {
	observation  Observation
	ready        bool
	fresh        bool
	lastClaim    time.Time
	lastMaintain time.Time
	lastObserve  time.Time
}

type Telemetry struct {
	mu           sync.RWMutex
	snapshot     telemetrySnapshot
	events       metric.Int64Counter
	duration     metric.Float64Histogram
	depth        metric.Int64ObservableGauge
	capacity     metric.Int64ObservableGauge
	readiness    metric.Int64ObservableGauge
	clock        metric.Int64ObservableGauge
	age          metric.Int64ObservableGauge
	freshness    metric.Int64ObservableGauge
	outcomes     metric.Int64ObservableGauge
	exhaustion   metric.Int64ObservableGauge
	registration metric.Registration
}

func NewTelemetry(meter metric.Meter) (*Telemetry, error) {
	meter = infratelemetry.MeterOrGlobal(meter, webhookMeterName)
	telemetry := &Telemetry{}
	set := infratelemetry.NewInstrumentSet(meter)
	set.Int64Counter(&telemetry.events, "postgres.webhooks.events")
	set.Float64Histogram(&telemetry.duration, "postgres.webhooks.operation.duration", metric.WithUnit("s"))
	set.Int64ObservableGauge(&telemetry.depth, "postgres.webhooks.depth")
	set.Int64ObservableGauge(&telemetry.capacity, "postgres.webhooks.capacity.available")
	set.Int64ObservableGauge(&telemetry.readiness, "postgres.webhooks.component.ready")
	set.Int64ObservableGauge(&telemetry.clock, "postgres.webhooks.clock.regression")
	set.Int64ObservableGauge(&telemetry.age, "postgres.webhooks.oldest_due.age")
	set.Int64ObservableGauge(&telemetry.freshness, "postgres.webhooks.success.age")
	set.Int64ObservableGauge(&telemetry.outcomes, "postgres.webhooks.outcomes")
	set.Int64ObservableGauge(&telemetry.exhaustion, "postgres.webhooks.exhaustion")
	if err := set.Err(); err != nil {
		return nil, fmt.Errorf("create PostgreSQL webhook telemetry: %w", err)
	}
	registration, err := meter.RegisterCallback(telemetry.collect, telemetry.depth, telemetry.capacity, telemetry.readiness, telemetry.clock, telemetry.age, telemetry.freshness, telemetry.outcomes, telemetry.exhaustion)
	if err != nil {
		return nil, fmt.Errorf("register PostgreSQL webhook telemetry: %w", err)
	}
	telemetry.registration = registration
	return telemetry, nil
}

func (telemetry *Telemetry) Update(observation Observation, ready bool) {
	if telemetry == nil {
		return
	}
	telemetry.mu.Lock()
	telemetry.snapshot.observation = observation
	telemetry.snapshot.ready = ready
	telemetry.snapshot.fresh = true
	telemetry.snapshot.lastObserve = time.Now()
	telemetry.mu.Unlock()
}

func (telemetry *Telemetry) MarkClaim() {
	if telemetry == nil {
		return
	}
	telemetry.mu.Lock()
	telemetry.snapshot.lastClaim = time.Now()
	telemetry.mu.Unlock()
}

func (telemetry *Telemetry) MarkMaintenance() {
	if telemetry == nil {
		return
	}
	telemetry.mu.Lock()
	telemetry.snapshot.lastMaintain = time.Now()
	telemetry.mu.Unlock()
}

func (telemetry *Telemetry) MarkStale() {
	if telemetry == nil {
		return
	}
	telemetry.mu.Lock()
	telemetry.snapshot.fresh = false
	telemetry.snapshot.ready = false
	telemetry.mu.Unlock()
}

func (telemetry *Telemetry) Record(ctx context.Context, event string, outcome OutcomeClass) {
	telemetry.recordN(ctx, event, outcome, failureNone, 1)
}

func (telemetry *Telemetry) RecordN(ctx context.Context, event string, outcome OutcomeClass, count int64) {
	telemetry.recordN(ctx, event, outcome, failureNone, count)
}

func (telemetry *Telemetry) RecordFailure(ctx context.Context, event string, outcome OutcomeClass, failure string) {
	telemetry.recordN(ctx, event, outcome, failure, 1)
}

func (telemetry *Telemetry) RecordDuration(ctx context.Context, event string, duration time.Duration) {
	if telemetry != nil && duration >= 0 {
		telemetry.duration.Record(ctx, duration.Seconds(), metric.WithAttributes(attribute.String("event", boundedValue(boundedEvents, event))))
	}
}

func (telemetry *Telemetry) recordN(ctx context.Context, event string, outcome OutcomeClass, failure string, count int64) {
	if telemetry != nil && count > 0 {
		telemetry.events.Add(ctx, count, metric.WithAttributes(
			attribute.String("event", boundedValue(boundedEvents, event)),
			attribute.String("outcome", boundedValue(boundedOutcomes, string(outcome))),
			attribute.String("error_class", boundedValue(boundedFailures, failure)),
		))
	}
}

func (telemetry *Telemetry) collect(_ context.Context, observer metric.Observer) error {
	telemetry.mu.RLock()
	snapshot := telemetry.snapshot
	telemetry.mu.RUnlock()
	states := []struct {
		name  string
		count int64
	}{{"ready", snapshot.observation.Ready}, {"scheduled", snapshot.observation.Scheduled}, {"in_flight", snapshot.observation.InFlight}, {"terminal", snapshot.observation.Terminal}, {"suspended", snapshot.observation.Suspended}, {"quarantined", snapshot.observation.Quarantined}, {"disabled", snapshot.observation.Disabled}, {"privacy_pending", snapshot.observation.PrivacyPending}}
	for _, state := range states {
		observer.ObserveInt64(telemetry.depth, state.count, metric.WithAttributes(attribute.String("state", state.name)))
	}
	observer.ObserveInt64(telemetry.capacity, max(0, snapshot.observation.TotalSlots-snapshot.observation.LeasedSlots))
	ready := int64(0)
	if snapshot.fresh && snapshot.ready {
		ready = 1
	}
	observer.ObserveInt64(telemetry.readiness, ready)
	regression := int64(0)
	if snapshot.observation.ClockRegression {
		regression = 1
	}
	observer.ObserveInt64(telemetry.clock, regression)
	observer.ObserveInt64(telemetry.age, max(0, snapshot.observation.OldestDueAge.Milliseconds()), metric.WithAttributes(attribute.String("kind", "due")))
	now := time.Now()
	for _, value := range []struct {
		kind string
		at   time.Time
	}{{"claim", snapshot.lastClaim}, {"maintenance", snapshot.lastMaintain}, {"observation", snapshot.lastObserve}} {
		age := int64(-1)
		if !value.at.IsZero() {
			age = max(0, now.Sub(value.at).Milliseconds())
		}
		observer.ObserveInt64(telemetry.freshness, age, metric.WithAttributes(attribute.String("kind", value.kind)))
	}
	for _, value := range []struct {
		outcome string
		count   int64
	}{{"http_accepted", snapshot.observation.HTTPAccepted}, {"http_rejected", snapshot.observation.HTTPRejected}, {"locally_denied", snapshot.observation.LocallyDenied}, {"outcome_unknown", snapshot.observation.OutcomeUnknown}} {
		observer.ObserveInt64(telemetry.outcomes, value.count, metric.WithAttributes(attribute.String("outcome", value.outcome)))
	}
	observer.ObserveInt64(telemetry.exhaustion, snapshot.observation.AttemptsExhausted, metric.WithAttributes(attribute.String("kind", "automatic")))
	observer.ObserveInt64(telemetry.exhaustion, snapshot.observation.RedriveExhausted, metric.WithAttributes(attribute.String("kind", "redrive")))
	return nil
}

func (telemetry *Telemetry) Unregister() error {
	if telemetry == nil || telemetry.registration == nil {
		return nil
	}
	if err := telemetry.registration.Unregister(); err != nil {
		return fmt.Errorf("unregister PostgreSQL webhook telemetry: %w", err)
	}
	return nil
}

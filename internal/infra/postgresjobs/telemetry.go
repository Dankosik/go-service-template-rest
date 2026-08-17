package postgresjobs

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	infratelemetry "github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/jobs"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const jobsMeterName = "service.postgres.jobs"

type telemetrySnapshot struct {
	observation Observation
	facts       EngineFacts
	freshUntil  time.Time
}

// Telemetry owns bounded job metrics. Callbacks only read this cached snapshot.
type Telemetry struct {
	mu           sync.RWMutex
	snapshot     telemetrySnapshot
	renewals     metric.Int64Counter
	events       metric.Int64Counter
	queueDelay   metric.Float64Histogram
	attemptTime  metric.Float64Histogram
	depth        metric.Int64ObservableGauge
	oldest       metric.Int64ObservableGauge
	capacity     metric.Int64ObservableGauge
	freshness    metric.Int64ObservableGauge
	readiness    metric.Int64ObservableGauge
	registration metric.Registration
}

func NewTelemetry(meter metric.Meter) (*Telemetry, error) {
	meter = infratelemetry.MeterOrGlobal(meter, jobsMeterName)
	t := &Telemetry{}
	set := infratelemetry.NewInstrumentSet(meter)
	set.Int64Counter(&t.renewals, "postgres.jobs.lease.renewals")
	set.Int64Counter(&t.events, "postgres.jobs.events")
	set.Float64Histogram(&t.queueDelay, "postgres.jobs.queue.delay", metric.WithUnit("s"))
	set.Float64Histogram(&t.attemptTime, "postgres.jobs.attempt.duration", metric.WithUnit("s"))
	set.Int64ObservableGauge(&t.depth, "postgres.jobs.depth")
	set.Int64ObservableGauge(&t.oldest, "postgres.jobs.oldest_available.timestamp")
	set.Int64ObservableGauge(&t.capacity, "postgres.jobs.capacity.available")
	set.Int64ObservableGauge(&t.freshness, "postgres.jobs.observation.fresh")
	set.Int64ObservableGauge(&t.readiness, "postgres.jobs.component.ready")
	if err := set.Err(); err != nil {
		return nil, fmt.Errorf("create PostgreSQL jobs telemetry: %w", err)
	}
	registration, err := meter.RegisterCallback(t.collect, t.depth, t.oldest, t.capacity, t.freshness, t.readiness)
	if err != nil {
		return nil, fmt.Errorf("register PostgreSQL jobs telemetry: %w", err)
	}
	t.registration = registration
	return t, nil
}

func (t *Telemetry) Update(observation Observation, facts EngineFacts, freshUntil time.Time) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.snapshot = telemetrySnapshot{observation: observation, facts: facts, freshUntil: freshUntil}
	t.mu.Unlock()
}

func (t *Telemetry) MarkStale() {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.snapshot.freshUntil = time.Time{}
	t.mu.Unlock()
}

func (t *Telemetry) UpdateFacts(facts EngineFacts) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.snapshot.facts = facts
	t.mu.Unlock()
}

func (t *Telemetry) RecordRenewal(ctx context.Context) {
	if t != nil {
		t.renewals.Add(ctx, 1)
	}
}

func (t *Telemetry) RecordRescue(ctx context.Context, outcome jobs.OutcomeClass) {
	t.record(ctx, "rescue", metricOutcome(outcome))
}

func (t *Telemetry) RecordClaim(ctx context.Context, outcome jobs.OutcomeClass, delay time.Duration) {
	t.record(ctx, "claim", metricOutcome(outcome))
	if t != nil {
		t.queueDelay.Record(ctx, delay.Seconds())
	}
}

func (t *Telemetry) RecordAttempt(ctx context.Context, outcome jobs.OutcomeClass, duration time.Duration) {
	t.record(ctx, "attempt", metricOutcome(outcome))
	if t != nil {
		t.attemptTime.Record(ctx, duration.Seconds(), metric.WithAttributes(attribute.String("outcome", metricOutcome(outcome))))
	}
}

func (t *Telemetry) RecordRetry(ctx context.Context, outcome jobs.OutcomeClass) {
	t.record(ctx, "retry", metricOutcome(outcome))
}

func (t *Telemetry) RecordCancellation(ctx context.Context, outcome jobs.OutcomeClass) {
	t.record(ctx, "cancellation", metricOutcome(outcome))
}

func (t *Telemetry) RecordDrain(ctx context.Context, outcome jobs.OutcomeClass) {
	t.record(ctx, "drain", metricOutcome(outcome))
}

func (t *Telemetry) RecordTerminalFailure(ctx context.Context) {
	t.record(ctx, "terminal_failure", metricOutcome(jobs.OutcomeUnknown))
}

func (t *Telemetry) record(ctx context.Context, event, outcome string) {
	if t != nil {
		t.events.Add(ctx, 1, metric.WithAttributes(attribute.String("event", metricEvent(event)), attribute.String("outcome", outcome)))
	}
}

func (t *Telemetry) collect(_ context.Context, observer metric.Observer) error {
	t.mu.RLock()
	snapshot := t.snapshot
	t.mu.RUnlock()
	fresh := !snapshot.freshUntil.IsZero() && time.Now().Before(snapshot.freshUntil)
	if fresh {
		for _, state := range snapshot.observation.States {
			options := metric.WithAttributes(attribute.String("state", metricState(state.State)))
			observer.ObserveInt64(t.depth, observationCount(state.Count), options)
			observer.ObserveInt64(t.oldest, state.OldestAvailableAt.UTC().Unix(), options)
		}
		observer.ObserveInt64(t.freshness, 1)
	} else {
		observer.ObserveInt64(t.freshness, 0)
	}
	observer.ObserveInt64(t.capacity, int64(max(0, snapshot.facts.Capacity-snapshot.facts.InFlight)))
	ready := int64(0)
	if fresh && snapshot.facts.ClaimAdmissionOpen && snapshot.facts.Compatible {
		ready = 1
	}
	observer.ObserveInt64(t.readiness, ready)
	return nil
}

func observationCount(count uint64) int64 {
	if count > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(count) // #nosec G115 -- count is at most math.MaxInt64 after saturation.
}

func (t *Telemetry) Unregister() error {
	if t == nil || t.registration == nil {
		return nil
	}
	if err := t.registration.Unregister(); err != nil {
		return fmt.Errorf("unregister PostgreSQL jobs telemetry: %w", err)
	}
	return nil
}

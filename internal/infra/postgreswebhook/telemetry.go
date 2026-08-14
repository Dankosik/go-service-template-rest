package postgreswebhook

import (
	"context"
	"fmt"
	"sync"

	infratelemetry "github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const webhookMeterName = "service.postgres.webhooks"

type telemetrySnapshot struct {
	observation Observation
	ready       bool
	fresh       bool
}

type Telemetry struct {
	mu           sync.RWMutex
	snapshot     telemetrySnapshot
	events       metric.Int64Counter
	depth        metric.Int64ObservableGauge
	capacity     metric.Int64ObservableGauge
	readiness    metric.Int64ObservableGauge
	clock        metric.Int64ObservableGauge
	registration metric.Registration
}

func NewTelemetry(meter metric.Meter) (*Telemetry, error) {
	meter = infratelemetry.MeterOrGlobal(meter, webhookMeterName)
	telemetry := &Telemetry{}
	set := infratelemetry.NewInstrumentSet(meter)
	set.Int64Counter(&telemetry.events, "postgres.webhooks.events")
	set.Int64ObservableGauge(&telemetry.depth, "postgres.webhooks.depth")
	set.Int64ObservableGauge(&telemetry.capacity, "postgres.webhooks.capacity.available")
	set.Int64ObservableGauge(&telemetry.readiness, "postgres.webhooks.component.ready")
	set.Int64ObservableGauge(&telemetry.clock, "postgres.webhooks.clock.regression")
	if err := set.Err(); err != nil {
		return nil, fmt.Errorf("create PostgreSQL webhook telemetry: %w", err)
	}
	registration, err := meter.RegisterCallback(telemetry.collect, telemetry.depth, telemetry.capacity, telemetry.readiness, telemetry.clock)
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
	telemetry.snapshot = telemetrySnapshot{observation: observation, ready: ready, fresh: true}
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
	if telemetry != nil {
		telemetry.events.Add(ctx, 1, metric.WithAttributes(attribute.String("event", boundedValue(event)), attribute.String("outcome", boundedValue(string(outcome)))))
	}
}

func (telemetry *Telemetry) collect(_ context.Context, observer metric.Observer) error {
	telemetry.mu.RLock()
	snapshot := telemetry.snapshot
	telemetry.mu.RUnlock()
	states := []struct {
		name  string
		count int64
	}{{"scheduled", snapshot.observation.Scheduled}, {"in_flight", snapshot.observation.InFlight}, {"terminal", snapshot.observation.Terminal}, {"disabled", snapshot.observation.Disabled}}
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

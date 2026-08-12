package postgresjobs

import (
	"context"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/jobs"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestTelemetryExportsCachedObservationWithoutStoreCalls(t *testing.T) {
	reader := telemetrytest.InstallManualReader(t)
	telemetry, err := NewTelemetry(nil)
	if err != nil {
		t.Fatal(err)
	}
	telemetry.Update(Observation{ObservedAt: time.Now(), Compatible: true, States: []StateObservation{{State: jobs.StateReady, Count: 2}}}, EngineFacts{ClaimAdmissionOpen: true, Compatible: true})
	telemetry.RecordRenewal(context.Background())
	telemetry.RecordRescue(context.Background(), jobs.OutcomeLost)
	telemetry.RecordAcceptance(context.Background(), jobs.OutcomeSuccess)
	telemetry.RecordClaim(context.Background(), jobs.OutcomeSuccess)
	telemetry.RecordAttempt(context.Background(), jobs.OutcomeSuccess)
	telemetry.RecordRetry(context.Background(), jobs.OutcomeRetryable)
	telemetry.RecordCancellation(context.Background(), jobs.OutcomeCancelled)
	telemetry.RecordRecovery(context.Background(), jobs.OutcomeLost)
	telemetry.RecordAction(context.Background(), jobs.OutcomeUnknown)
	telemetry.RecordDrain(context.Background(), jobs.OutcomeSuccess)
	telemetry.RecordRescue(context.Background(), jobs.OutcomeClass("sentinel"))
	if got := telemetrytest.Int64GaugeValue(t, reader, "postgres.jobs.observation.fresh"); got != 1 {
		t.Fatalf("fresh gauge = %d, want 1", got)
	}
	if got := telemetrytest.Int64GaugeValue(t, reader, "postgres.jobs.component.ready"); got != 1 {
		t.Fatalf("ready gauge = %d, want 1", got)
	}
	for _, event := range []struct {
		name    string
		outcome jobs.OutcomeClass
	}{
		{"rescue", jobs.OutcomeLost}, {"rescue", jobs.OutcomeClass("other")}, {"acceptance", jobs.OutcomeSuccess},
		{"claim", jobs.OutcomeSuccess}, {"attempt", jobs.OutcomeSuccess}, {"retry", jobs.OutcomeRetryable},
		{"cancellation", jobs.OutcomeCancelled}, {"recovery", jobs.OutcomeLost}, {"action", jobs.OutcomeUnknown}, {"drain", jobs.OutcomeSuccess},
	} {
		assertJobsEvent(t, reader, event.name, event.outcome)
	}
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		if measured.Name != "postgres.jobs.events" {
			return
		}
		for _, attributes := range telemetrytest.AttributeSets(t, measured.Data) {
			if len(attributes.ToSlice()) != 2 || telemetrytest.Attribute(t, attributes, "event") == "" || telemetrytest.Attribute(t, attributes, "outcome") == "" {
				t.Fatalf("event metric attributes = %v, want bounded event/outcome pair", attributes)
			}
		}
	})
	telemetrytest.AssertNoAttributeContains(t, reader, "sentinel")
	if err := telemetry.Unregister(); err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}
	telemetry.Update(Observation{ObservedAt: time.Now(), Compatible: true, States: []StateObservation{{State: jobs.StateScheduled, Count: 9}}}, EngineFacts{ClaimAdmissionOpen: true, Compatible: true})
	if got := jobDepthValue(t, reader, jobs.StateScheduled); got != 0 {
		t.Fatalf("unregistered callback exported updated scheduled depth = %d, want 0", got)
	}
}

func jobDepthValue(t *testing.T, reader *sdkmetric.ManualReader, state jobs.State) int64 {
	t.Helper()
	var value int64
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		if measured.Name != "postgres.jobs.depth" {
			return
		}
		for _, point := range telemetrytest.Int64Gauge(t, measured).DataPoints {
			if telemetrytest.Attribute(t, point.Attributes, "state") == string(state) {
				value += point.Value
			}
		}
	})
	return value
}

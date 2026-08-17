package postgresjobs

import (
	"context"
	"errors"
	"math"
	"testing"
	"testing/synctest"
	"time"

	infratelemetry "github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/jobs"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestTelemetryExportsCachedObservationWithoutStoreCalls(t *testing.T) {
	t.Parallel()
	reader := telemetrytest.InstallManualReader(t)
	telemetry, err := NewTelemetry(nil)
	if err != nil {
		t.Fatal(err)
	}
	telemetry.Update(Observation{ObservedAt: time.Now(), Compatible: true, States: []StateObservation{{State: jobs.StateReady, Count: math.MaxUint64}}}, EngineFacts{ClaimAdmissionOpen: true, Compatible: true}, time.Now().Add(time.Minute))
	telemetry.RecordRenewal(context.Background())
	telemetry.RecordRescue(context.Background(), jobs.OutcomeLost)
	telemetry.RecordClaim(context.Background(), jobs.OutcomeSuccess, 2*time.Second)
	telemetry.RecordAttempt(context.Background(), jobs.OutcomeSuccess, 3*time.Second)
	telemetry.RecordRetry(context.Background(), jobs.OutcomeRetryable)
	telemetry.RecordCancellation(context.Background(), jobs.OutcomeCancelled)
	telemetry.RecordDrain(context.Background(), jobs.OutcomeSuccess)
	telemetry.RecordTerminalFailure(context.Background())
	telemetry.RecordRescue(context.Background(), jobs.OutcomeClass("sentinel"))
	if got := telemetrytest.Int64GaugeValue(t, reader, "postgres.jobs.observation.fresh"); got != 1 {
		t.Fatalf("fresh gauge = %d, want 1", got)
	}
	if got := telemetrytest.Int64GaugeValue(t, reader, "postgres.jobs.component.ready"); got != 1 {
		t.Fatalf("ready gauge = %d, want 1", got)
	}
	if got := jobDepthValue(t, reader, jobs.StateReady); got != math.MaxInt64 {
		t.Fatalf("ready depth = %d, want %d", got, math.MaxInt64)
	}
	for _, event := range []struct {
		name    string
		outcome jobs.OutcomeClass
	}{
		{"rescue", jobs.OutcomeLost},
		{"rescue", jobs.OutcomeClass("other")},
		{"claim", jobs.OutcomeSuccess},
		{"attempt", jobs.OutcomeSuccess},
		{"retry", jobs.OutcomeRetryable},
		{"cancellation", jobs.OutcomeCancelled},
		{"drain", jobs.OutcomeSuccess},
		{"terminal_failure", jobs.OutcomeUnknown},
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
	assertJobsHistogram(t, reader, "postgres.jobs.queue.delay", 2, "")
	assertJobsHistogram(t, reader, "postgres.jobs.attempt.duration", 3, string(jobs.OutcomeSuccess))
	if err := telemetry.Unregister(); err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}
	telemetry.Update(Observation{ObservedAt: time.Now(), Compatible: true, States: []StateObservation{{State: jobs.StateScheduled, Count: 9}}}, EngineFacts{ClaimAdmissionOpen: true, Compatible: true}, time.Now().Add(time.Minute))
	if got := jobDepthValue(t, reader, jobs.StateScheduled); got != 0 {
		t.Fatalf("unregistered callback exported updated scheduled depth = %d, want 0", got)
	}
}

func TestStoreRecordsAcceptanceOutcomes(t *testing.T) {
	t.Parallel()
	reader := telemetrytest.InstallManualReader(t)
	meter := infratelemetry.MeterOrGlobal(nil, jobsMeterName)
	events, err := meter.Int64Counter("postgres.jobs.events")
	if err != nil {
		t.Fatal(err)
	}
	operationTime, err := meter.Float64Histogram("postgres.jobs.store.operation.duration", metric.WithUnit("s"))
	if err != nil {
		t.Fatal(err)
	}
	store := &Store{events: events, operationTime: operationTime}
	store.recordAcceptance(context.Background(), jobs.StageResult{Outcome: jobs.StageNew}, nil)
	store.recordAcceptance(context.Background(), jobs.StageResult{Outcome: jobs.StageExisting}, nil)
	store.recordAcceptance(context.Background(), jobs.StageResult{Outcome: jobs.StageConflict}, nil)
	store.recordAcceptance(context.Background(), jobs.StageResult{Outcome: jobs.StageRejected}, errors.New("database failed"))
	assertJobsEvent(t, reader, "acceptance", jobs.OutcomeSuccess)
	assertJobsEvent(t, reader, "acceptance_duplicate", jobs.OutcomeSuccess)
	assertJobsEvent(t, reader, "acceptance_conflict", jobs.OutcomePermanent)
	assertJobsEvent(t, reader, "acceptance_rejected", jobs.OutcomeUnknown)
	store.recordOperation(context.Background(), "claim", nil, 2*time.Second)
	store.recordOperation(context.Background(), "sentinel", errors.New("failed"), 3*time.Second)
	assertJobsStoreOperations(t, reader)
	telemetrytest.AssertNoAttributeContains(t, reader, "sentinel")
}

func TestTelemetryExpiresObservationOnLocalClock(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		reader := telemetrytest.InstallManualReader(t)
		telemetry, err := NewTelemetry(nil)
		if err != nil {
			t.Fatal(err)
		}
		telemetry.Update(Observation{ObservedAt: time.Date(2200, time.January, 1, 0, 0, 0, 0, time.UTC), Compatible: true}, EngineFacts{ClaimAdmissionOpen: true, Compatible: true}, time.Now().Add(time.Minute))
		if got := telemetrytest.Int64GaugeValue(t, reader, "postgres.jobs.observation.fresh"); got != 1 {
			t.Fatalf("fresh gauge = %d, want 1", got)
		}
		time.Sleep(time.Minute)
		synctest.Wait()
		if got := telemetrytest.Int64GaugeValue(t, reader, "postgres.jobs.observation.fresh"); got != 0 {
			t.Fatalf("expired fresh gauge = %d, want 0", got)
		}
		if got := telemetrytest.Int64GaugeValue(t, reader, "postgres.jobs.component.ready"); got != 0 {
			t.Fatalf("expired ready gauge = %d, want 0", got)
		}
	})
}

func assertJobsHistogram(t *testing.T, reader *sdkmetric.ManualReader, name string, want float64, outcome string) {
	t.Helper()
	found := false
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		if measured.Name != name {
			return
		}
		found = true
		points := telemetrytest.Float64Histogram(t, measured).DataPoints
		if measured.Unit != "s" || len(points) != 1 || points[0].Count != 1 || points[0].Sum != want {
			t.Fatalf("%s = unit %q points %#v, want one %g-second sample", name, measured.Unit, points, want)
		}
		attributes := points[0].Attributes
		if outcome == "" && attributes.Len() != 0 {
			t.Fatalf("%s attributes = %v, want none", name, attributes.ToSlice())
		}
		if outcome != "" && (attributes.Len() != 1 || telemetrytest.Attribute(t, attributes, "outcome") != outcome) {
			t.Fatalf("%s attributes = %v, want outcome %q", name, attributes.ToSlice(), outcome)
		}
	})
	if !found {
		t.Fatalf("metric %s not exported", name)
	}
}

func assertJobsStoreOperations(t *testing.T, reader *sdkmetric.ManualReader) {
	t.Helper()
	found := map[string]float64{}
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		if measured.Name != "postgres.jobs.store.operation.duration" {
			return
		}
		if measured.Unit != "s" {
			t.Fatalf("store operation unit = %q, want seconds", measured.Unit)
		}
		for _, point := range telemetrytest.Float64Histogram(t, measured).DataPoints {
			if point.Count != 1 || point.Attributes.Len() != 2 {
				t.Fatalf("store operation point = %#v, want one sample and two attributes", point)
			}
			operation := telemetrytest.Attribute(t, point.Attributes, "operation")
			outcome := telemetrytest.Attribute(t, point.Attributes, "outcome")
			found[operation+"/"+outcome] = point.Sum
		}
	})
	if found["claim/success"] != 2 || found["other/unknown"] != 3 {
		t.Fatalf("store operations = %v, want claim/success=2 and other/unknown=3", found)
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

package postgreswebhook

import (
	"context"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestWebhookTelemetryPrivacyAndVocabulary(t *testing.T) {
	if got := boundedValue(boundedEvents, "https://secret.example/path"); got != "other" {
		t.Fatalf("unbounded label = %q", got)
	}
	for _, value := range []string{"attempt", "claim_progress", "maintenance"} {
		if got := boundedValue(boundedEvents, value); got != value {
			t.Fatalf("boundedValue(%q) = %q", value, got)
		}
	}
	if got := boundedValue(boundedEvents, string(OutcomeHTTPAccepted)); got != "other" {
		t.Fatalf("outcome accepted as event = %q", got)
	}
	for _, test := range []struct {
		err  error
		want string
	}{
		{nil, failureNone},
		{ErrDestinationDenied, failureSSRFDenied},
		{ErrSecretUnavailable, failureSecretRotation},
		{ErrResponseLimit, failureResponseBound},
		{ErrConflict, failureReconciliationConflict},
		{context.DeadlineExceeded, failureTimeout},
		{context.Canceled, failureCanceled},
	} {
		if got := failureClass(test.err); got != test.want {
			t.Fatalf("failureClass(%v) = %q, want %q", test.err, got, test.want)
		}
	}
}

func TestWebhookTelemetryExportsOperatorSignals(t *testing.T) {
	reader, meter := telemetrytest.NewManualMeter(t, webhookMeterName)
	telemetry, err := NewTelemetry(meter)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := telemetry.Unregister(); err != nil {
			t.Errorf("Unregister() error = %v", err)
		}
	})
	telemetry.Update(Observation{
		Ready: 1, Scheduled: 2, InFlight: 3, Terminal: 4, Suspended: 5, Quarantined: 6, Disabled: 7,
		PrivacyPending: 8, OldestDueAge: 8 * time.Second, HTTPAccepted: 9, HTTPRejected: 10, LocallyDenied: 11, OutcomeUnknown: 12,
		AttemptsExhausted: 12, RedriveExhausted: 13, LeasedSlots: 14, TotalSlots: 20, ClockRegression: true,
	}, true)
	telemetry.MarkClaim()
	telemetry.MarkMaintenance()
	telemetry.Record(context.Background(), "claim_progress", OutcomeHTTPAccepted)
	telemetry.RecordDuration(context.Background(), "reconciliation", 25*time.Millisecond)
	for _, failure := range []string{failureSSRFDenied, failureSecretRotation, failureResponseBound, failureReconciliationConflict} {
		telemetry.RecordFailure(context.Background(), "attempt", OutcomeUnknown, failure)
	}
	telemetry.RecordFailure(context.Background(), "https://forbidden.example/path", OutcomeClass("credential-canary-outcome"), "credential-canary-error")

	want := map[string]map[string]int64{
		"postgres.webhooks.depth":      {"ready": 1, "scheduled": 2, "in_flight": 3, "terminal": 4, "suspended": 5, "quarantined": 6, "disabled": 7, "privacy_pending": 8},
		"postgres.webhooks.outcomes":   {"http_accepted": 9, "http_rejected": 10, "locally_denied": 11, "outcome_unknown": 12},
		"postgres.webhooks.exhaustion": {"automatic": 12, "redrive": 13},
	}
	for name, expected := range want {
		if got := webhookGaugeValues(t, reader, name); !equalWebhookValues(got, expected) {
			t.Fatalf("%s = %#v, want %#v", name, got, expected)
		}
	}
	for name, expected := range map[string]int64{
		"postgres.webhooks.capacity.available": 6,
		"postgres.webhooks.component.ready":    1,
		"postgres.webhooks.clock.regression":   1,
	} {
		if got := telemetrytest.Int64GaugeValue(t, reader, name); got != expected {
			t.Fatalf("%s = %d, want %d", name, got, expected)
		}
	}
	if got := webhookGaugeValues(t, reader, "postgres.webhooks.oldest_due.age")["due"]; got != 8000 {
		t.Fatalf("oldest due age = %d, want 8000", got)
	}
	ages := webhookGaugeValues(t, reader, "postgres.webhooks.success.age")
	if len(ages) != 3 {
		t.Fatalf("success ages = %#v, want claim, maintenance, and observation", ages)
	}
	for kind, age := range ages {
		if (kind != "claim" && kind != "maintenance" && kind != "observation") || age < 0 {
			t.Fatalf("success age %s = %d", kind, age)
		}
	}
	if got := webhookDurationEvents(t, reader)["reconciliation"]; got != 1 {
		t.Fatalf("reconciliation duration count = %d, want 1", got)
	}
	failures := webhookEventFailures(t, reader)
	for _, failure := range []string{failureNone, failureSSRFDenied, failureSecretRotation, failureResponseBound, failureReconciliationConflict, "other"} {
		if failures[failure] == 0 {
			t.Fatalf("event error class %q was not exported: %#v", failure, failures)
		}
	}
	telemetrytest.AssertNoAttributeContains(t, reader, "forbidden.example", "credential-canary")
}

func webhookDurationEvents(t *testing.T, reader interface {
	Collect(ctx context.Context, metrics *metricdata.ResourceMetrics) error
},
) map[string]uint64 {
	t.Helper()
	values := make(map[string]uint64)
	var collected metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &collected); err != nil {
		t.Fatal(err)
	}
	for _, scope := range collected.ScopeMetrics {
		for _, measured := range scope.Metrics {
			if measured.Name != "postgres.webhooks.operation.duration" {
				continue
			}
			for _, point := range telemetrytest.Float64Histogram(t, measured).DataPoints {
				attributes := point.Attributes.ToSlice()
				if len(attributes) != 1 || attributes[0].Key != "event" {
					t.Fatalf("duration attributes = %v, want event", attributes)
				}
				values[attributes[0].Value.AsString()] += point.Count
			}
		}
	}
	return values
}

func webhookEventFailures(t *testing.T, reader interface {
	Collect(ctx context.Context, metrics *metricdata.ResourceMetrics) error
},
) map[string]int64 {
	t.Helper()
	values := make(map[string]int64)
	var collected metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &collected); err != nil {
		t.Fatal(err)
	}
	for _, scope := range collected.ScopeMetrics {
		for _, measured := range scope.Metrics {
			if measured.Name != "postgres.webhooks.events" {
				continue
			}
			for _, point := range telemetrytest.Int64Sum(t, measured).DataPoints {
				attributes := point.Attributes.ToSlice()
				if len(attributes) != 3 {
					t.Fatalf("event attributes = %v, want event/outcome/error_class", attributes)
				}
				for _, item := range attributes {
					if item.Key == "error_class" {
						values[item.Value.AsString()] += point.Value
					}
				}
			}
		}
	}
	return values
}

func webhookGaugeValues(t *testing.T, reader interface {
	Collect(ctx context.Context, metrics *metricdata.ResourceMetrics) error
}, name string,
) map[string]int64 {
	t.Helper()
	values := make(map[string]int64)
	var collected metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &collected); err != nil {
		t.Fatal(err)
	}
	for _, scope := range collected.ScopeMetrics {
		for _, measured := range scope.Metrics {
			if measured.Name != name {
				continue
			}
			for _, point := range telemetrytest.Int64Gauge(t, measured).DataPoints {
				attributes := point.Attributes.ToSlice()
				if len(attributes) != 1 {
					t.Fatalf("%s attributes = %v, want one", name, attributes)
				}
				values[attributes[0].Value.AsString()] = point.Value
			}
		}
	}
	return values
}

func equalWebhookValues(left, right map[string]int64) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range right {
		if left[key] != value {
			return false
		}
	}
	return true
}

package postgreswebhook

import (
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
)

func TestWebhookTelemetryPrivacyAndVocabulary(t *testing.T) {
	if got := boundedValue("https://secret.example/path"); got != "other" {
		t.Fatalf("unbounded label = %q", got)
	}
	for _, value := range []string{"ready", "scheduled", "http_accepted", "outcome_unknown"} {
		if got := boundedValue(value); got != value {
			t.Fatalf("boundedValue(%q) = %q", value, got)
		}
	}
}

func TestWebhookTelemetryReadinessExpiresAndExposesQueueAgeInputs(t *testing.T) {
	reader := telemetrytest.InstallManualReader(t)
	telemetry, err := NewTelemetry(nil, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = telemetry.Unregister() })
	telemetry.Update(Observation{OldestDueTimestamp: 10, ObservationTimestamp: 20, OutcomeUnknown: 2}, true)
	if got := telemetrytest.Int64GaugeValue(t, reader, "postgres.webhooks.component.ready"); got != 1 {
		t.Fatalf("fresh readiness = %d", got)
	}
	if got := telemetrytest.Int64GaugeValue(t, reader, "postgres.webhooks.oldest.due.timestamp"); got != 10 {
		t.Fatalf("oldest due timestamp = %d", got)
	}
	telemetry.mu.Lock()
	telemetry.snapshot.updatedAt = time.Now().Add(-2 * time.Minute)
	telemetry.mu.Unlock()
	if got := telemetrytest.Int64GaugeValue(t, reader, "postgres.webhooks.component.ready"); got != 0 {
		t.Fatalf("stale readiness = %d", got)
	}
}

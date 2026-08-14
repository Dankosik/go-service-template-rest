package postgreswebhook

import "testing"

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

package config

// profile:webhooks-durable:start

import (
	"context"
	"os"
	"testing"
)

func TestWebhookWorkerLoaderIgnoresForeignProfiles(t *testing.T) {
	setWebhookWorkerConfigEnv(t)
	t.Setenv("APP__AUTHN__ISSUER", "")
	t.Setenv("APP__OUTBOUND_AUTH__DEPENDENCY", "not a dependency")
	t.Setenv("APP__OUTBOUND_AUTH__ACQUISITION_TIMEOUT", "not a duration")
	t.Setenv("APP__OBJECT_STORAGE__PROVIDER", "not a provider")
	t.Setenv("APP__JOBS__POLL_INTERVAL", "not a duration")

	if _, _, err := LoadWebhookWorkerDetailedWithContext(context.Background(), LoadOptions{}); err != nil {
		t.Fatalf("LoadWebhookWorkerDetailedWithContext() error = %v", err)
	}
}

func setWebhookWorkerConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range configEnvResetKeys(t) {
		if value, ok := os.LookupEnv(key); ok {
			t.Setenv(key, value)
		}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("os.Unsetenv(%q) error = %v", key, err)
		}
	}
	for key, value := range map[string]string{
		"APP__APP__ENV":                          "local",
		"APP__POSTGRES__ENABLED":                 "true",
		"APP__POSTGRES__DSN":                     "postgres://webhook:password@localhost:5432/webhook?sslmode=disable",
		"APP__POSTGRES__MAX_OPEN_CONNS":          "2",
		"APP__HTTP__GRACE_PERIOD":                "20s",
		"APP__HTTP__SHUTDOWN_TIMEOUT":            "3s",
		"APP__HTTP__READINESS_PROPAGATION_DELAY": "0s",
		"APP__HTTP__REQUEST_TIMEOUT":             "2s",
		"APP__HTTP__WRITE_TIMEOUT":               "2s",
		"APP__HTTP__READINESS_TIMEOUT":           "1s",
		"APP__WEBHOOKS__ENABLED":                 "true",
		"APP__WEBHOOKS__CAPACITY_REVISION":       "1",
		"APP__WEBHOOKS__GLOBAL_CONCURRENCY":      "1",
		"APP__WEBHOOKS__CLAIM_SCAN_PAGE":         "1",
		"APP__WEBHOOKS__POLL_INTERVAL":           "1s",
		"APP__WEBHOOKS__OBSERVATION_INTERVAL":    "1s",
		"APP__WEBHOOKS__STORE_OPERATION_TIMEOUT": "200ms",
		"APP__WEBHOOKS__ATTEMPT_TIMEOUT":         "5s",
		"APP__WEBHOOKS__RESPONSE_HEADER_TIMEOUT": "2s",
		"APP__WEBHOOKS__RESPONSE_HEADER_BYTES":   "4096",
		"APP__WEBHOOKS__RESPONSE_BODY_BYTES":     "4096",
		"APP__WEBHOOKS__DRAIN_TIMEOUT":           "10s",
		"APP__WEBHOOKS__MAINTENANCE_INTERVAL":    "1s",
		"APP__WEBHOOKS__MAINTENANCE_BATCH":       "10",
		"APP__WEBHOOKS__STATIC_SECRETS":          `{"revision":1,"entries":[]}`,
		"APP__OBSERVABILITY__METRICS__ADDR":      "127.0.0.1:9090",
	} {
		t.Setenv(key, value)
	}
}

// profile:webhooks-durable:end

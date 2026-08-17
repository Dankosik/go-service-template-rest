package config

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

func TestWebhooksConfigContract(t *testing.T) {
	t.Parallel()
	valid := WebhooksConfig{Enabled: true, CapacityRevision: 1, GlobalConcurrency: 4, ClaimScanPage: 4, PollInterval: time.Second, ObservationInterval: time.Second, StoreOperationTimeout: time.Second, AttemptTimeout: 5 * time.Second, ResponseHeaderTimeout: 2 * time.Second, ResponseHeaderBytes: 4096, ResponseBodyBytes: 4096, DrainTimeout: 10 * time.Second, MaintenanceInterval: time.Second, MaintenanceBatch: 10, StaticSecrets: `{"revision":1,"entries":[]}`}
	postgres := PostgresConfig{Enabled: true, StatementTimeout: 5 * time.Second}
	http := HTTPConfig{GracePeriod: 45 * time.Second}
	if err := validateWebhooks(valid, postgres, http); err != nil {
		t.Fatalf("validateWebhooks(valid) error = %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*WebhooksConfig, *PostgresConfig, *HTTPConfig)
	}{
		{"postgres disabled", func(_ *WebhooksConfig, p *PostgresConfig, _ *HTTPConfig) { p.Enabled = false }},
		{"capacity revision", func(w *WebhooksConfig, _ *PostgresConfig, _ *HTTPConfig) { w.CapacityRevision = 0 }},
		{"claim page", func(w *WebhooksConfig, _ *PostgresConfig, _ *HTTPConfig) { w.ClaimScanPage = 257 }},
		{"store timeout", func(w *WebhooksConfig, _ *PostgresConfig, _ *HTTPConfig) { w.StoreOperationTimeout = 6 * time.Second }},
		{"budget nesting", func(w *WebhooksConfig, _ *PostgresConfig, _ *HTTPConfig) { w.ResponseHeaderTimeout = 6 * time.Second }},
		{"attempt ceiling", func(w *WebhooksConfig, _ *PostgresConfig, h *HTTPConfig) {
			w.AttemptTimeout = 10*time.Minute + time.Nanosecond
			w.DrainTimeout = w.AttemptTimeout + time.Second
			h.GracePeriod = w.DrainTimeout + time.Second
		}},
		{"drain ceiling", func(w *WebhooksConfig, _ *PostgresConfig, h *HTTPConfig) {
			w.DrainTimeout = 30*time.Minute + time.Nanosecond
			h.GracePeriod = w.DrainTimeout + time.Second
		}},
		{"grace", func(w *WebhooksConfig, _ *PostgresConfig, h *HTTPConfig) { h.GracePeriod = w.DrainTimeout }},
		{"secret", func(w *WebhooksConfig, _ *PostgresConfig, _ *HTTPConfig) { w.StaticSecrets = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			w, p, h := valid, postgres, http
			test.mutate(&w, &p, &h)
			if err := validateWebhooks(w, p, h); !errors.Is(err, ErrValidate) {
				t.Fatalf("error = %v, want ErrValidate", err)
			}
		})
	}
}

func TestWebhooksConfigDefaultsDisabled(t *testing.T) {
	t.Parallel()
	resetConfigEnv(t)
	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Webhooks != (WebhooksConfig{}) {
		t.Fatalf("default webhooks = %+v", cfg.Webhooks)
	}
}

//nolint:paralleltest // This test mutates process-global environment or working directory.
func TestWebhookWorkerProcessEnvironment(t *testing.T) {
	resetConfigEnv(t)
	secret := "whsec_" + base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	for key, value := range map[string]string{
		"APP__APP__ENV": "integration", "APP__OBSERVABILITY__METRICS__ADDR": ":9090",
		"APP__POSTGRES__ENABLED": "true", "APP__POSTGRES__DSN": "postgres://app:app@127.0.0.1:5432/app?sslmode=disable",
		"APP__POSTGRES__MAX_OPEN_CONNS": "4", "APP__POSTGRES__MIN_IDLE_CONNS": "0",
		"APP__POSTGRES__CONNECT_TIMEOUT": "1s", "APP__POSTGRES__HEALTHCHECK_TIMEOUT": "1s",
		"APP__POSTGRES__ACQUIRE_TIMEOUT": "100ms", "APP__POSTGRES__STATEMENT_TIMEOUT": "500ms",
		"APP__HTTP__REQUEST_TIMEOUT": "2s", "APP__HTTP__GRACE_PERIOD": "12s", "APP__HTTP__SHUTDOWN_TIMEOUT": "3s",
		"APP__HTTP__WRITE_TIMEOUT": "2s", "APP__HTTP__READINESS_TIMEOUT": "1s", "APP__HTTP__READINESS_PROPAGATION_DELAY": "0s",
		"APP__WEBHOOKS__ENABLED": "true", "APP__WEBHOOKS__CAPACITY_REVISION": "1", "APP__WEBHOOKS__GLOBAL_CONCURRENCY": "2",
		"APP__WEBHOOKS__CLAIM_SCAN_PAGE": "4", "APP__WEBHOOKS__POLL_INTERVAL": "50ms", "APP__WEBHOOKS__OBSERVATION_INTERVAL": "100ms",
		"APP__WEBHOOKS__STORE_OPERATION_TIMEOUT": "200ms", "APP__WEBHOOKS__ATTEMPT_TIMEOUT": "500ms",
		"APP__WEBHOOKS__RESPONSE_HEADER_TIMEOUT": "200ms", "APP__WEBHOOKS__RESPONSE_HEADER_BYTES": "4096",
		"APP__WEBHOOKS__RESPONSE_BODY_BYTES": "4096", "APP__WEBHOOKS__DRAIN_TIMEOUT": "1s",
		"APP__WEBHOOKS__MAINTENANCE_INTERVAL": "100ms", "APP__WEBHOOKS__MAINTENANCE_BATCH": "10",
		"APP__WEBHOOKS__STATIC_SECRETS": `{"revision":1,"entries":[{"owner_scope":"owner-a","destination_id":"dest-a","key_reference":"key-a","secret":"` + secret + `"}]}`,
	} {
		t.Setenv(key, value)
	}
	if _, _, err := LoadDetailed(LoadOptions{}); err != nil {
		t.Fatal(err)
	}
}

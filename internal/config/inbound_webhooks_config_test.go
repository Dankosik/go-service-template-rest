// profile:inbound-webhooks-standard:start
package config

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

const inboundWebhookTestCanary = "inbound-secret-canary"

func inboundWebhookSecretJSON(t *testing.T, endpoint, reference string, key []byte) string {
	t.Helper()
	return `{"entries":[{"endpoint_id":"` + endpoint + `","key_reference":"` + reference + `","secret":"whsec_` + base64.StdEncoding.EncodeToString(key) + `"}]}`
}

func TestInboundWebhooksConfigBoundary(t *testing.T) {
	key32 := []byte("0123456789abcdef0123456789abcdef")
	endpoints := `{"endpoints":[{"endpoint_id":"orders","active_key_reference":"key-v1"}]}`

	t.Run("service snapshot keeps both leaves", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__INBOUND_WEBHOOKS__ENDPOINTS", endpoints)
		t.Setenv("APP__INBOUND_WEBHOOKS__STATIC_SECRETS", inboundWebhookSecretJSON(t, "orders", "key-v1", key32))
		cfg, _, err := LoadDetailed(LoadOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if cfg.InboundWebhooks.Endpoints != endpoints {
			t.Fatalf("endpoints = %q", cfg.InboundWebhooks.Endpoints)
		}
		if !strings.Contains(cfg.InboundWebhooks.StaticSecrets, "whsec_") {
			t.Fatal("service snapshot lost secret leaf")
		}
	})

	t.Run("partial leaves are rejected without canary", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__INBOUND_WEBHOOKS__ENDPOINTS", endpoints)
		t.Setenv("APP__INBOUND_WEBHOOKS__STATIC_SECRETS", "")
		_, _, err := LoadDetailed(LoadOptions{})
		if !errors.Is(err, ErrValidate) || strings.Contains(err.Error(), inboundWebhookTestCanary) {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("worker projects endpoints only", func(t *testing.T) {
		setJobsWorkerConfigEnv(t)
		t.Setenv("APP__INBOUND_WEBHOOKS__ENDPOINTS", endpoints)
		cfg, _, err := LoadJobsWorkerDetailedWithContext(context.Background(), LoadOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if cfg.InboundWebhooks.Endpoints != endpoints || cfg.InboundWebhooks.StaticSecrets != "" {
			t.Fatalf("worker snapshot = %+v", cfg.InboundWebhooks)
		}
	})

	t.Run("worker rejects static secrets", func(t *testing.T) {
		setJobsWorkerConfigEnv(t)
		t.Setenv("APP__INBOUND_WEBHOOKS__ENDPOINTS", endpoints)
		t.Setenv("APP__INBOUND_WEBHOOKS__STATIC_SECRETS", inboundWebhookTestCanary)
		_, _, err := LoadJobsWorkerDetailedWithContext(context.Background(), LoadOptions{})
		if !errors.Is(err, ErrValidate) || !strings.Contains(err.Error(), "inbound_webhooks.static_secrets") {
			t.Fatalf("error = %v", err)
		}
		if strings.Contains(err.Error(), inboundWebhookTestCanary) {
			t.Fatalf("error disclosed canary: %v", err)
		}
	})
}

// profile:inbound-webhooks-standard:end

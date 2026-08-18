package main

// profile:webhooks-durable:start

import (
	"context"
	"encoding/base64"
	"log/slog"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
)

func TestBuildWebhookRegistry(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	cfg := config.Config{Webhooks: config.WebhooksConfig{
		Enabled:       true,
		StaticSecrets: `{"entries":[{"owner_scope":"orders","receiver_id":"alpha","key_reference":"key-v1","secret":"whsec_` + secret + `"}]}`,
	}}
	registry, cleanup, err := buildWebhookRegistry(context.Background(), cfg, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if cleanup != nil || registry == nil || len(registry.Keys()) != 1 || registry.Keys()[0].Kind != "outbound_webhook" {
		t.Fatalf("registry = %+v, cleanup nil = %t", registry, cleanup == nil)
	}
}

// profile:webhooks-durable:end

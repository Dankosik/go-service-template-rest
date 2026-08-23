package main

import (
	"context"
	"encoding/base64"
	"log/slog"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
)

func TestBuildWebhookWorkers(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	cfg := config.Config{Webhooks: config.WebhooksConfig{
		Enabled:       true,
		StaticSecrets: `{"entries":[{"owner_scope":"orders","receiver_id":"alpha","key_reference":"key-v1","secret":"whsec_` + secret + `"}]}`,
	}}
	runtime, err := buildWebhookWorkers(context.Background(), cfg, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if runtime.Workers == nil {
		t.Fatal("workers are nil")
	}
	workers := runtime.Workers
	secrets, err := postgreswebhook.ParseSecretManifest(cfg.Webhooks.StaticSecrets)
	if err != nil {
		t.Fatal(err)
	}
	if err := postgreswebhook.AddWorker(workers, secrets); err == nil {
		t.Fatal("duplicate webhook worker registration succeeded")
	}
}

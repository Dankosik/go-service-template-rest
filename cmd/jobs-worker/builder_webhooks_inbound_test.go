//go:build !jobs_test_worker && !inbound_webhook_test_worker

// profile:inbound-webhooks-standard:start
package main

import (
	"context"
	"log/slog"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
)

func TestInboundWebhookWorkerRegistrationBindIsPresent(t *testing.T) {
	registration, err := buildWebhookWorkers(context.Background(), config.Config{}, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if registration.Workers == nil || registration.Bind == nil {
		t.Fatalf("registration workers=%v bind=%v", registration.Workers == nil, registration.Bind == nil)
	}
}

// profile:inbound-webhooks-standard:end

//go:build !jobs_test_worker && !inbound_webhook_test_worker

// profile:inbound-webhooks-standard:start
package main

import (
	"context"
	"log/slog"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
)

func TestInboundWebhookWorkerRuntimeBindIsPresent(t *testing.T) {
	runtime, err := buildWebhookWorkers(context.Background(), config.Config{}, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if runtime.Workers == nil || runtime.Bind == nil {
		t.Fatalf("runtime workers=%v bind=%v", runtime.Workers == nil, runtime.Bind == nil)
	}
}

// profile:inbound-webhooks-standard:end

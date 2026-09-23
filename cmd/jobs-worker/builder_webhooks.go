//go:build !jobs_test_worker && !inbound_webhook_test_worker

package main

import (
	"context"
	"errors"

	// profile:webhooks-durable:start
	"fmt"
	// profile:webhooks-durable:end
	"log/slog"

	"github.com/example/go-service-template-rest/cmd/jobs-worker/internal/bootstrap"
	"github.com/example/go-service-template-rest/internal/config"

	// profile:webhooks-durable:start
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	// profile:webhooks-durable:end
	// profile:inbound-webhooks-standard:start
	"github.com/jackc/pgx/v5/pgxpool"
	// profile:inbound-webhooks-standard:end
	"github.com/riverqueue/river"
	// profile:inbound-webhooks-standard:start
	"go.opentelemetry.io/otel/metric"
	// profile:inbound-webhooks-standard:end
)

//nolint:gochecknoinits // Profile composition installs the only shipped business worker.
func init() {
	buildWorkers = buildWebhookWorkers
}

func buildWebhookWorkers(
	_ context.Context,
	cfg config.Config,
	log *slog.Logger,
) (bootstrap.WorkerRegistration, error) {
	workers := river.NewWorkers()
	var registered bool
	// profile:webhooks-durable:start
	if cfg.OutboundWebhooks.Enabled {
		secrets, err := postgreswebhook.ParseSecretManifest(cfg.OutboundWebhooks.StaticSecrets)
		if err != nil {
			return bootstrap.WorkerRegistration{}, fmt.Errorf("parse webhook worker secrets: %w", err)
		}
		if err := postgreswebhook.AddWorker(workers, secrets); err != nil {
			return bootstrap.WorkerRegistration{}, fmt.Errorf("register webhook worker: %w", err)
		}
		registered = true //nolint:ineffassign,wastedassign // Kept when inbound is stripped.
	}
	// profile:webhooks-durable:end
	registration := bootstrap.WorkerRegistration{Workers: workers}
	// profile:inbound-webhooks-standard:start
	registration.Bind = func(_ context.Context, pool *pgxpool.Pool, meter metric.MeterProvider) error {
		return bindInboundWebhookWorkers(cfg, workers, pool, meter, log)
	}
	registered = true
	// profile:inbound-webhooks-standard:end
	if !registered {
		return bootstrap.WorkerRegistration{}, errors.New("no webhook workers are configured")
	}
	return registration, nil
}

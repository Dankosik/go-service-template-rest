//go:build !jobs_test_worker && !inbound_webhook_test_worker

// profile:inbound-webhooks-standard:start
package main

import (
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	"github.com/example/go-service-template-rest/internal/infra/postgresinboundwebhook"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/metric"
)

func bindInboundWebhookWorkers(
	cfg config.Config,
	workers *river.Workers,
	pool *pgxpool.Pool,
	meter metric.MeterProvider,
	log *slog.Logger,
) error {
	endpoints, err := postgresinboundwebhook.ParseEndpointManifest(cfg.InboundWebhooks.Endpoints)
	if err != nil {
		return fmt.Errorf("parse inbound webhook endpoints: %w", err)
	}
	registry := inboundwebhook.NewRegistry()
	if err := registry.RequireExact(endpoints.IDs()); err != nil {
		return fmt.Errorf("bind inbound webhook handlers: %w", err)
	}
	if err := postgresinboundwebhook.AddWorker(workers, pool, registry, meter, log); err != nil {
		return fmt.Errorf("register inbound webhook worker: %w", err)
	}
	return nil
}

// profile:inbound-webhooks-standard:end

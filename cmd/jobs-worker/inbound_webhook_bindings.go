//go:build !jobs_test_worker

// profile:inbound-webhooks-standard:start
package main

import (
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	inboundmanifest "github.com/example/go-service-template-rest/internal/inboundwebhook/manifest"
	"github.com/example/go-service-template-rest/internal/infra/postgresinboundwebhook"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/metric"
)

// bindInboundWebhookWorkers adds the inbound webhook worker once register has
// bound a handler for every configured endpoint; an endpoint left without one
// fails startup.
func bindInboundWebhookWorkers(
	cfg config.Config,
	workers *river.Workers,
	pool *pgxpool.Pool,
	meter metric.MeterProvider,
	log *slog.Logger,
	register func(*inboundwebhook.Registry) error,
) error {
	endpoints, err := inboundmanifest.ParseEndpoints(cfg.InboundWebhooks.Endpoints)
	if err != nil {
		return fmt.Errorf("parse inbound webhook endpoints: %w", err)
	}
	// The registry starts empty; register is the only place handlers join it.
	registry := inboundwebhook.NewRegistry()
	if err := register(registry); err != nil {
		return fmt.Errorf("bind inbound webhook handlers: %w", err)
	}
	if err := registry.RequireExact(endpoints.IDs()); err != nil {
		return fmt.Errorf("bind inbound webhook handlers: %w", err)
	}
	if err := postgresinboundwebhook.AddWorker(workers, pool, registry, meter, log); err != nil {
		return fmt.Errorf("register inbound webhook worker: %w", err)
	}
	return nil
}

// profile:inbound-webhooks-standard:end

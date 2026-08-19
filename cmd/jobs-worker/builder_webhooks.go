//go:build !jobs_test_worker

package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	"github.com/riverqueue/river"
)

//nolint:gochecknoinits // Profile composition installs the only shipped business worker.
func init() {
	buildWorkers = buildWebhookWorkers
}

func buildWebhookWorkers(
	_ context.Context,
	cfg config.Config,
	_ *slog.Logger,
) (*river.Workers, func(context.Context), error) {
	if !cfg.Webhooks.Enabled {
		return nil, nil, fmt.Errorf("%w: webhooks must be enabled for the template jobs worker", postgreswebhook.ErrConfig)
	}
	secrets, err := postgreswebhook.ParseSecretManifest(cfg.Webhooks.StaticSecrets)
	if err != nil {
		return nil, nil, fmt.Errorf("parse webhook worker secrets: %w", err)
	}
	workers := river.NewWorkers()
	if err := postgreswebhook.AddWorker(workers, secrets); err != nil {
		return nil, nil, fmt.Errorf("register webhook worker: %w", err)
	}
	return workers, nil, nil
}

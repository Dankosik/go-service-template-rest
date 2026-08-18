//go:build !jobs_test_worker

package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/cmd/jobs-worker/internal/bootstrap"
	// profile:webhooks-durable:start

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	"github.com/riverqueue/river"
	// profile:webhooks-durable:end
)

// A concrete feature replaces this nil builder in its selected binary. The
// generic pack has no default kind and must therefore fail before database I/O.
var buildWorkers = func() (builder bootstrap.WorkersBuilder) {
	// profile:webhooks-durable:start
	builder = buildWebhookWorkers
	// profile:webhooks-durable:end
	return builder
}()

// profile:webhooks-durable:start
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

// profile:webhooks-durable:end

package bootstrap

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
)

const (
	startupTimeout   = 15 * time.Second
	diagnosticsClose = 2 * time.Second
	telemetryClose   = 5 * time.Second
	databaseClose    = 2 * time.Second
)

func parseLoadOptions(args []string) (config.LoadOptions, error) {
	return config.ParseLoadOptions("webhook-worker", args, nil)
}

func validateRuntimeConfig(cfg config.Config) error {
	if !cfg.Webhooks.Enabled || !cfg.Postgres.Enabled {
		return fmt.Errorf("%w: webhooks and postgres must be enabled for webhook-worker", postgreswebhook.ErrConfig)
	}
	if strings.TrimSpace(cfg.Observability.Metrics.Addr) == "" {
		return fmt.Errorf("%w: webhook worker diagnostics address is required", postgreswebhook.ErrConfig)
	}
	return runtimeopts.ValidateGracePeriod(cfg.HTTP.GracePeriod, "webhooks.drain_timeout", cfg.Webhooks.DrainTimeout, diagnosticsClose+telemetryClose+databaseClose)
}

func workerConfig(cfg config.Config) (postgreswebhook.WorkerConfig, error) {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		return postgreswebhook.WorkerConfig{}, fmt.Errorf("resolve webhook worker identity: %w", err)
	}
	return postgreswebhook.WorkerConfig{
		WorkerID: host, ClaimScanPage: cfg.Webhooks.ClaimScanPage, PollInterval: cfg.Webhooks.PollInterval,
		ObservationInterval: cfg.Webhooks.ObservationInterval, AttemptTimeout: cfg.Webhooks.AttemptTimeout,
		StoreOperationTimeout: cfg.Webhooks.StoreOperationTimeout, DrainTimeout: cfg.Webhooks.DrainTimeout,
		MaintenanceInterval: cfg.Webhooks.MaintenanceInterval, MaintenanceBatch: cfg.Webhooks.MaintenanceBatch,
	}, nil
}

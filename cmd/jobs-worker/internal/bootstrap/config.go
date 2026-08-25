package bootstrap

import (
	"fmt"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
)

const (
	startupTimeout     = 15 * time.Second
	diagnosticsClose   = 2 * time.Second
	telemetryClose     = 5 * time.Second
	riverHardStopClose = 5 * time.Second
	jobsTailBudget     = riverHardStopClose + diagnosticsClose + telemetryClose
)

func parseLoadOptions(args []string) (config.LoadOptions, error) {
	return config.ParseLoadOptions(args)
}

func validateRuntimeConfig(cfg config.Config) error {
	if !cfg.Postgres.Enabled {
		return fmt.Errorf("%w: jobs worker requires postgres.enabled", config.ErrValidate)
	}
	if cfg.Jobs.MaxWorkers < 1 {
		return fmt.Errorf("%w: jobs.max_workers must be positive for jobs-worker", config.ErrValidate)
	}
	if strings.TrimSpace(cfg.Observability.Metrics.Addr) == "" {
		return fmt.Errorf("%w: jobs worker diagnostics address is required", config.ErrValidate)
	}
	return runtimeopts.ValidateGracePeriod(
		cfg.HTTP.GracePeriod,
		"http.shutdown_timeout",
		cfg.HTTP.ShutdownTimeout,
		jobsTailBudget,
	)
}

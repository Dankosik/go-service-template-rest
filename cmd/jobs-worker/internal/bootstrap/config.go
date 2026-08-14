package bootstrap

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
)

const (
	startupTimeout   = 15 * time.Second
	diagnosticsClose = 2 * time.Second
	telemetryClose   = 5 * time.Second
)

func parseLoadOptions(args []string) (config.LoadOptions, error) {
	return config.ParseLoadOptions("jobs-worker", args, nil)
}

func engineConfig(cfg config.JobsConfig) (postgresjobs.EngineConfig, error) {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		return postgresjobs.EngineConfig{}, fmt.Errorf("resolve jobs worker identity: %w", err)
	}
	return postgresjobs.EngineConfig{
		WorkerID: host, MaxConcurrency: cfg.MaxConcurrency, LeaseDuration: cfg.LeaseDuration,
		ObservationInterval: cfg.ObservationInterval, DrainTimeout: cfg.DrainTimeout,
	}, nil
}

func validateRuntimeConfig(cfg config.Config) error {
	if !cfg.Jobs.Enabled || !cfg.Postgres.Enabled {
		return fmt.Errorf("%w: jobs and postgres must be enabled for jobs-worker", postgresjobs.ErrConfig)
	}
	if strings.TrimSpace(cfg.Observability.Metrics.Addr) == "" {
		return fmt.Errorf("%w: jobs worker diagnostics address is required", postgresjobs.ErrConfig)
	}
	return runtimeopts.ValidateGracePeriod(cfg.HTTP.GracePeriod, "jobs.drain_timeout", cfg.Jobs.DrainTimeout, diagnosticsClose+telemetryClose)
}

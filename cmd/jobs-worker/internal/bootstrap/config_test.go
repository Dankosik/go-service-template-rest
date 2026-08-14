package bootstrap

import (
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
)

func TestJobsWorkerConfigMapsEngineFields(t *testing.T) {
	cfg, err := engineConfig(config.JobsConfig{MaxConcurrency: 3, LeaseDuration: time.Minute, ObservationInterval: 10 * time.Second, DrainTimeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorkerID == "" || cfg.MaxConcurrency != 3 || cfg.LeaseDuration != time.Minute || cfg.ObservationInterval != 10*time.Second || cfg.DrainTimeout != 5*time.Second {
		t.Fatalf("engineConfig() = %+v, want mapped jobs configuration", cfg)
	}
}

func TestJobsWorkerRuntimeConfigRejectsMissingRequirements(t *testing.T) {
	valid := config.Config{}
	valid.Jobs.Enabled = true
	valid.Jobs.DrainTimeout = time.Second
	valid.Postgres.Enabled = true
	valid.Observability.Metrics.Addr = ":9090"
	valid.HTTP.GracePeriod = 8 * time.Second
	if err := validateRuntimeConfig(valid); err != nil {
		t.Fatalf("validateRuntimeConfig(valid) error = %v", err)
	}

	for _, test := range []struct {
		name     string
		mutate   func(*config.Config)
		contains string
	}{
		{name: "jobs disabled", mutate: func(cfg *config.Config) { cfg.Jobs.Enabled = false }, contains: "jobs and postgres"},
		{name: "postgres disabled", mutate: func(cfg *config.Config) { cfg.Postgres.Enabled = false }, contains: "jobs and postgres"},
		{name: "diagnostics disabled", mutate: func(cfg *config.Config) { cfg.Observability.Metrics.Addr = "" }, contains: "diagnostics address"},
		{name: "grace too short", mutate: func(cfg *config.Config) { cfg.HTTP.GracePeriod = 7 * time.Second }, contains: "http.grace_period"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := valid
			test.mutate(&cfg)
			if err := validateRuntimeConfig(cfg); err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("validateRuntimeConfig() error = %v, want %q", err, test.contains)
			}
		})
	}
}

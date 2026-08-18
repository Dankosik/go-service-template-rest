package bootstrap

import (
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
)

func TestJobsWorkerRuntimeConfig(t *testing.T) {
	valid := config.Config{}
	valid.Postgres.Enabled = true
	valid.Jobs.MaxWorkers = 1
	valid.Observability.Metrics.Addr = ":9090"
	valid.HTTP.GracePeriod = 10 * time.Second
	valid.HTTP.ShutdownTimeout = 5 * time.Second
	if err := validateRuntimeConfig(valid); err != nil {
		t.Fatalf("validateRuntimeConfig(valid) error = %v", err)
	}

	for _, test := range []struct {
		name     string
		mutate   func(*config.Config)
		contains string
	}{
		{name: "postgres disabled", mutate: func(cfg *config.Config) { cfg.Postgres.Enabled = false }, contains: "postgres.enabled"},
		{name: "workers disabled", mutate: func(cfg *config.Config) { cfg.Jobs.MaxWorkers = 0 }, contains: "jobs.max_workers"},
		{name: "diagnostics disabled", mutate: func(cfg *config.Config) { cfg.Observability.Metrics.Addr = "" }, contains: "diagnostics address"},
		{name: "grace too short", mutate: func(cfg *config.Config) { cfg.HTTP.GracePeriod = 9 * time.Second }, contains: "http.grace_period"},
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

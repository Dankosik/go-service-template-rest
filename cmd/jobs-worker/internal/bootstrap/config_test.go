package bootstrap

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestJobsWorkerConfigMapsEngineFields(t *testing.T) {
	jobsConfig := config.JobsConfig{PollInterval: time.Second, MaxConcurrency: 3, LeaseDuration: time.Minute, StoreOperationTimeout: 2 * time.Second, ObservationInterval: 10 * time.Second, DrainTimeout: 5 * time.Second}
	cfg, err := engineConfig(jobsConfig, "pod-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := engineConfig(jobsConfig, "pod-1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(cfg.WorkerID, "pod-1/") || cfg.WorkerID == second.WorkerID || cfg.MaxConcurrency != 3 || cfg.LeaseDuration != time.Minute || cfg.ObservationInterval != 10*time.Second || cfg.ObservationMaxAge != 13*time.Second || cfg.DrainTimeout != 5*time.Second {
		t.Fatalf("engineConfig() = %+v, want mapped jobs configuration", cfg)
	}
}

func TestJobsWorkerConfigRejectsObservationFreshnessOverflow(t *testing.T) {
	_, err := engineConfig(config.JobsConfig{PollInterval: time.Nanosecond, StoreOperationTimeout: time.Nanosecond, ObservationInterval: time.Duration(math.MaxInt64)}, "pod-1")
	if err == nil || !strings.Contains(err.Error(), "freshness envelope overflows") {
		t.Fatalf("engineConfig() error = %v, want observation freshness overflow", err)
	}
}

func TestJobsWorkerConfigRejectsInvalidInstanceIdentity(t *testing.T) {
	_, err := engineConfig(config.JobsConfig{}, strings.Repeat("x", jobs.MaxIdentityBytes))
	if err == nil || !strings.Contains(err.Error(), "instance identity") {
		t.Fatalf("engineConfig() error = %v, want invalid instance identity", err)
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

func TestJobsWorkerRejectsDefinitionOutsideTerminationEnvelope(t *testing.T) {
	if err := validateTerminationEnvelope(8*time.Second, 8*time.Second); err != nil {
		t.Fatalf("validateTerminationEnvelope(equal) error = %v", err)
	}
	if err := validateTerminationEnvelope(8*time.Second, 9*time.Second); err == nil || !strings.Contains(err.Error(), "termination envelope") {
		t.Fatalf("validateTerminationEnvelope() error = %v, want termination envelope mismatch", err)
	}
}

package config

// profile:jobs-postgres:start

import (
	"errors"
	"strings"
	"testing"
)

func TestJobsConfigDefaultsDisabled(t *testing.T) {
	resetConfigEnv(t)
	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Jobs != (JobsConfig{}) {
		t.Fatalf("default jobs config = %+v, want zero workers", cfg.Jobs)
	}
}

func TestJobsConfigValidation(t *testing.T) {
	if err := validateJobs(JobsConfig{MaxWorkers: 1}, PostgresConfig{Enabled: true}); err != nil {
		t.Fatalf("validateJobs(valid) error = %v", err)
	}

	for _, test := range []struct {
		name     string
		jobs     JobsConfig
		postgres PostgresConfig
		contains string
	}{
		{name: "disabled", jobs: JobsConfig{}, postgres: PostgresConfig{}},
		{name: "postgres disabled", jobs: JobsConfig{MaxWorkers: 1}, contains: "requires postgres.enabled"},
		{name: "negative", jobs: JobsConfig{MaxWorkers: -1}, postgres: PostgresConfig{Enabled: true}, contains: "jobs.max_workers"},
		{name: "too many", jobs: JobsConfig{MaxWorkers: 501}, postgres: PostgresConfig{Enabled: true}, contains: "jobs.max_workers"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateJobs(test.jobs, test.postgres)
			if test.contains == "" {
				if err != nil {
					t.Fatalf("validateJobs() error = %v", err)
				}
				return
			}
			if !errors.Is(err, ErrValidate) || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("validateJobs() error = %v, want ErrValidate containing %q", err, test.contains)
			}
		})
	}
}

// profile:jobs-postgres:end

package config

// profile:jobs-postgres:start

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func TestJobsConfigDefaultsDisabled(t *testing.T) {
	t.Parallel()
	resetConfigEnv(t)
	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Jobs != (JobsConfig{}) {
		t.Fatalf("default jobs config = %+v, want disabled zero value", cfg.Jobs)
	}
}

func TestJobsConfigValidation(t *testing.T) {
	t.Parallel()
	validJobs := JobsConfig{
		Enabled:               true,
		PollInterval:          time.Second,
		MaxConcurrency:        4,
		LeaseDuration:         30 * time.Second,
		StoreOperationTimeout: 5 * time.Second,
		ObservationInterval:   2 * time.Second,
		DrainTimeout:          20 * time.Second,
	}
	validPostgres := PostgresConfig{Enabled: true, MaxOpenConns: 5}
	if err := validateJobs(validJobs, validPostgres); err != nil {
		t.Fatalf("validateJobs(valid) error = %v", err)
	}
	aboveMinimumLease := validJobs
	aboveMinimumLease.LeaseDuration++
	if err := validateJobs(aboveMinimumLease, validPostgres); err != nil {
		t.Fatalf("validateJobs(above minimum lease) error = %v", err)
	}

	for _, test := range []struct {
		name     string
		mutate   func(*JobsConfig, *PostgresConfig)
		contains string
	}{
		{name: "postgres disabled", mutate: func(_ *JobsConfig, postgres *PostgresConfig) { postgres.Enabled = false }, contains: "requires postgres.enabled"},
		{name: "poll interval missing", mutate: func(jobs *JobsConfig, _ *PostgresConfig) { jobs.PollInterval = 0 }, contains: "jobs.poll_interval"},
		{name: "max concurrency missing", mutate: func(jobs *JobsConfig, _ *PostgresConfig) { jobs.MaxConcurrency = 0 }, contains: "jobs.max_concurrency"},
		{name: "lease duration missing", mutate: func(jobs *JobsConfig, _ *PostgresConfig) { jobs.LeaseDuration = 0 }, contains: "jobs.lease_duration"},
		{name: "store timeout missing", mutate: func(jobs *JobsConfig, _ *PostgresConfig) { jobs.StoreOperationTimeout = 0 }, contains: "jobs.store_operation_timeout"},
		{name: "store timeout below timer floor", mutate: func(jobs *JobsConfig, _ *PostgresConfig) { jobs.StoreOperationTimeout = 99 * time.Millisecond }, contains: "at least 100ms"},
		{name: "observation interval missing", mutate: func(jobs *JobsConfig, _ *PostgresConfig) { jobs.ObservationInterval = 0 }, contains: "jobs.observation_interval"},
		{name: "drain timeout missing", mutate: func(jobs *JobsConfig, _ *PostgresConfig) { jobs.DrainTimeout = 0 }, contains: "jobs.drain_timeout"},
		{name: "pool has no reserved connection", mutate: func(jobs *JobsConfig, postgres *PostgresConfig) { postgres.MaxOpenConns = jobs.MaxConcurrency }, contains: "reserve the control Session"},
		{name: "pool below worker concurrency", mutate: func(jobs *JobsConfig, postgres *PostgresConfig) { postgres.MaxOpenConns = jobs.MaxConcurrency - 1 }, contains: "reserve the control Session"},
		{name: "lease ratio below six", mutate: func(jobs *JobsConfig, _ *PostgresConfig) {
			jobs.LeaseDuration = 6*jobs.StoreOperationTimeout - time.Nanosecond
		}, contains: "at least 6 times"},
		{name: "poll cannot renew lease in time", mutate: func(jobs *JobsConfig, _ *PostgresConfig) {
			jobs.PollInterval = jobs.LeaseDuration/3 - jobs.StoreOperationTimeout + time.Nanosecond
		}, contains: "jobs.poll_interval"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			jobs, postgres := validJobs, validPostgres
			test.mutate(&jobs, &postgres)
			err := validateJobs(jobs, postgres)
			if !errors.Is(err, ErrValidate) || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("validateJobs() error = %v, want ErrValidate containing %q", err, test.contains)
			}
		})
	}
}

func TestJobsConfigLeaseRatioAvoidsOverflow(t *testing.T) {
	t.Parallel()
	postgres := PostgresConfig{Enabled: true, MaxOpenConns: 2}
	jobs := JobsConfig{
		Enabled:               true,
		PollInterval:          time.Nanosecond,
		MaxConcurrency:        1,
		LeaseDuration:         time.Duration(math.MaxInt64),
		StoreOperationTimeout: time.Duration(math.MaxInt64 / 6),
		ObservationInterval:   time.Nanosecond,
		DrainTimeout:          time.Nanosecond,
	}
	if err := validateJobs(jobs, postgres); err != nil {
		t.Fatalf("validateJobs(max valid ratio) error = %v", err)
	}
	jobs.StoreOperationTimeout++
	if err := validateJobs(jobs, postgres); !errors.Is(err, ErrValidate) {
		t.Fatalf("validateJobs(overflow edge) error = %v, want ErrValidate", err)
	}
}

// profile:jobs-postgres:end

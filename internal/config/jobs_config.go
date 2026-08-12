package config

// profile:jobs-postgres:start

import (
	"fmt"
	"time"
)

type JobsConfig struct {
	Enabled               bool          `koanf:"enabled"`
	PollInterval          time.Duration `koanf:"poll_interval"`
	MaxConcurrency        int           `koanf:"max_concurrency"`
	LeaseDuration         time.Duration `koanf:"lease_duration"`
	StoreOperationTimeout time.Duration `koanf:"store_operation_timeout"`
	ObservationInterval   time.Duration `koanf:"observation_interval"`
	DrainTimeout          time.Duration `koanf:"drain_timeout"`
}

func jobsDefaults() map[string]any {
	return map[string]any{
		"jobs.enabled":                 false,
		"jobs.poll_interval":           "0s",
		"jobs.max_concurrency":         0,
		"jobs.lease_duration":          "0s",
		"jobs.store_operation_timeout": "0s",
		"jobs.observation_interval":    "0s",
		"jobs.drain_timeout":           "0s",
	}
}

func validateJobs(cfg JobsConfig, postgres PostgresConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if !postgres.Enabled {
		return fmt.Errorf("%w: jobs.enabled requires postgres.enabled", ErrValidate)
	}
	for _, duration := range []struct {
		name  string
		value time.Duration
	}{
		{name: "jobs.poll_interval", value: cfg.PollInterval},
		{name: "jobs.lease_duration", value: cfg.LeaseDuration},
		{name: "jobs.store_operation_timeout", value: cfg.StoreOperationTimeout},
		{name: "jobs.observation_interval", value: cfg.ObservationInterval},
		{name: "jobs.drain_timeout", value: cfg.DrainTimeout},
	} {
		if duration.value <= 0 {
			return fmt.Errorf("%w: %s must be positive", ErrValidate, duration.name)
		}
	}
	if err := validateIntRange("jobs.max_concurrency", cfg.MaxConcurrency, 1, 500); err != nil {
		return err
	}
	if postgres.MaxOpenConns <= cfg.MaxConcurrency {
		return fmt.Errorf(
			"%w: postgres.max_open_conns must exceed jobs.max_concurrency to reserve the control Session",
			ErrValidate,
		)
	}
	if cfg.StoreOperationTimeout > cfg.LeaseDuration/6 {
		return fmt.Errorf(
			"%w: jobs.lease_duration must be at least 6 times jobs.store_operation_timeout",
			ErrValidate,
		)
	}
	return nil
}

// profile:jobs-postgres:end

package config

// profile:jobs-postgres:start

import "fmt"

type JobsConfig struct {
	MaxWorkers int `koanf:"max_workers"`
}

func jobsDefaults() map[string]any {
	return map[string]any{"jobs.max_workers": 0}
}

func validateJobs(cfg JobsConfig, postgres PostgresConfig) error {
	if cfg.MaxWorkers == 0 {
		return nil
	}
	if !postgres.Enabled {
		return fmt.Errorf("%w: positive jobs.max_workers requires postgres.enabled", ErrValidate)
	}
	return validateIntRange("jobs.max_workers", cfg.MaxWorkers, 1, 500)
}

// profile:jobs-postgres:end

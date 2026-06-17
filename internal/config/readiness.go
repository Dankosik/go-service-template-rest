package config

import "time"

// ReadinessProbeBudget describes one enabled readiness probe budget from config.
type ReadinessProbeBudget struct {
	// ConfigKey is the config key that owns this probe budget.
	ConfigKey string
	// Budget is the probe duration budget.
	Budget time.Duration
}

// PostgresReadinessProbeRequired reports whether Postgres participates in runtime readiness.
func (cfg Config) PostgresReadinessProbeRequired() bool {
	return cfg.Postgres.Enabled && cfg.FeatureFlags.PostgresReadinessProbe
}

// ReadinessProbeBudgets returns the enabled runtime readiness probe budgets.
func (cfg Config) ReadinessProbeBudgets() []ReadinessProbeBudget {
	budgets := make([]ReadinessProbeBudget, 0, 1)
	if cfg.PostgresReadinessProbeRequired() {
		budgets = append(budgets, ReadinessProbeBudget{
			ConfigKey: "postgres.healthcheck_timeout",
			Budget:    cfg.Postgres.HealthcheckTimeout,
		})
	}
	return budgets
}

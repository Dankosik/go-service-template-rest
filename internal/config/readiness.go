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

// RedisReadinessProbeRequired reports whether Redis participates in runtime readiness.
func (cfg Config) RedisReadinessProbeRequired() bool {
	return cfg.Redis.Enabled && (cfg.FeatureFlags.RedisReadinessProbe || cfg.Redis.StoreMode())
}

// MongoReadinessProbeRequired reports whether Mongo participates in runtime readiness.
func (cfg Config) MongoReadinessProbeRequired() bool {
	return cfg.Mongo.Enabled && cfg.FeatureFlags.MongoReadinessProbe
}

// ReadinessProbeBudgets returns the enabled runtime readiness probe budgets.
func (cfg Config) ReadinessProbeBudgets() []ReadinessProbeBudget {
	budgets := make([]ReadinessProbeBudget, 0, 3)
	if cfg.PostgresReadinessProbeRequired() {
		budgets = append(budgets, ReadinessProbeBudget{
			ConfigKey: "postgres.healthcheck_timeout",
			Budget:    cfg.Postgres.HealthcheckTimeout,
		})
	}
	if cfg.RedisReadinessProbeRequired() {
		budgets = append(budgets, ReadinessProbeBudget{
			ConfigKey: "redis.dial_timeout",
			Budget:    cfg.Redis.DialTimeout,
		})
	}
	if cfg.MongoReadinessProbeRequired() {
		budgets = append(budgets, ReadinessProbeBudget{
			ConfigKey: "mongo.connect_timeout",
			Budget:    cfg.Mongo.ConnectTimeout,
		})
	}
	return budgets
}

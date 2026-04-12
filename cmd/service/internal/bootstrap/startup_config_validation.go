package bootstrap

import (
	"fmt"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
)

func validateStartupBudgetCompatibility(cfg config.Config) error {
	if cfg.Postgres.Enabled {
		if err := validateStartupTimeoutBudget("postgres.connect_timeout", cfg.Postgres.ConnectTimeout, postgresProbeBudget); err != nil {
			return err
		}
		if err := validateStartupTimeoutBudget("postgres.healthcheck_timeout", cfg.Postgres.HealthcheckTimeout, postgresProbeBudget); err != nil {
			return err
		}
	}
	if cfg.Redis.Enabled {
		if err := validateStartupTimeoutBudget("redis.dial_timeout", cfg.Redis.DialTimeout, redisProbeBudget); err != nil {
			return err
		}
	}
	if cfg.Mongo.Enabled {
		if err := validateStartupTimeoutBudget("mongo.connect_timeout", cfg.Mongo.ConnectTimeout, mongoProbeBudget); err != nil {
			return err
		}
	}
	if err := validateStartupReadinessHeadroom(cfg); err != nil {
		return err
	}
	return nil
}

func validateStartupTimeoutBudget(name string, value time.Duration, budget time.Duration) error {
	if value <= budget {
		return nil
	}
	return fmt.Errorf(
		"%w: %s must be <= startup probe budget %s",
		config.ErrValidate,
		name,
		budget,
	)
}

func validateStartupReadinessHeadroom(cfg config.Config) error {
	probes := startupReadinessProbeBudgets(cfg)
	if len(probes) == 0 {
		return nil
	}

	var aggregate time.Duration
	names := make([]string, 0, len(probes))
	for _, probe := range probes {
		aggregate += probe.budget
		names = append(names, probe.name)
	}
	required := aggregate + startupReadinessHeadroom
	if cfg.HTTP.ReadinessTimeout >= required {
		return nil
	}
	return fmt.Errorf(
		"%w: http.readiness_timeout must be >= aggregate sequential readiness probe budget plus startup headroom (%s + %s = %s; probes: %s)",
		config.ErrValidate,
		aggregate,
		startupReadinessHeadroom,
		required,
		strings.Join(names, " + "),
	)
}

func startupReadinessProbeBudgets(cfg config.Config) []startupReadinessProbeBudget {
	budgets := make([]startupReadinessProbeBudget, 0, 3)
	if cfg.PostgresReadinessProbeRequired() {
		budgets = append(budgets, startupReadinessProbeBudget{
			name:   "postgres.healthcheck_timeout",
			budget: cfg.Postgres.HealthcheckTimeout,
		})
	}
	if cfg.RedisReadinessProbeRequired() {
		budgets = append(budgets, startupReadinessProbeBudget{
			name:   "redis.dial_timeout",
			budget: cfg.Redis.DialTimeout,
		})
	}
	if cfg.MongoReadinessProbeRequired() {
		budgets = append(budgets, startupReadinessProbeBudget{
			name:   "mongo.connect_timeout",
			budget: cfg.Mongo.ConnectTimeout,
		})
	}
	return budgets
}

type startupReadinessProbeBudget struct {
	name   string
	budget time.Duration
}

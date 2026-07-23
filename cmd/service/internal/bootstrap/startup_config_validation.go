package bootstrap

import (
	"fmt"
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
	if !cfg.Postgres.Enabled {
		return nil
	}

	required := cfg.Postgres.HealthcheckTimeout + startupReadinessHeadroom
	if cfg.HTTP.ReadinessTimeout >= required {
		return nil
	}
	return fmt.Errorf(
		"%w: http.readiness_timeout must be >= postgres.healthcheck_timeout readiness budget plus startup headroom (%s + %s = %s)",
		config.ErrValidate,
		cfg.Postgres.HealthcheckTimeout,
		startupReadinessHeadroom,
		required,
	)
}

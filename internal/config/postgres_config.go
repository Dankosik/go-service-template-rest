package config

// profile:database-postgres:start

import (
	"fmt"
	"strings"
	"time"
)

func validatePostgres(cfg PostgresConfig) error {
	if cfg.Enabled && strings.TrimSpace(cfg.DSN) == "" {
		return fmt.Errorf("%w: postgres.dsn is required when postgres.enabled=true", ErrSecretPolicy)
	}

	if err := validateDurationRange("postgres.connect_timeout", cfg.ConnectTimeout, 100*time.Millisecond, 10*time.Second); err != nil {
		return err
	}
	if err := validateDurationRange("postgres.healthcheck_timeout", cfg.HealthcheckTimeout, 100*time.Millisecond, 10*time.Second); err != nil {
		return err
	}
	if err := validateDurationRange("postgres.migration_timeout", cfg.MigrationTimeout, time.Second, time.Hour); err != nil {
		return err
	}
	if err := validateDurationRange(
		"postgres.migration_statement_timeout",
		cfg.MigrationStatementTimeout,
		100*time.Millisecond,
		cfg.MigrationTimeout,
	); err != nil {
		return err
	}
	if err := validateDurationRange(
		"postgres.migration_lock_timeout",
		cfg.MigrationLockTimeout,
		100*time.Millisecond,
		cfg.MigrationTimeout,
	); err != nil {
		return err
	}
	if cfg.MigrationLockTimeout >= cfg.MigrationTimeout {
		return fmt.Errorf(
			"%w: postgres.migration_lock_timeout must be less than postgres.migration_timeout to reserve cleanup time",
			ErrValidate,
		)
	}
	if err := validateDurationRange("postgres.acquire_timeout", cfg.AcquireTimeout, 10*time.Millisecond, 30*time.Second); err != nil {
		return err
	}
	if err := validateIntRange("postgres.max_open_conns", cfg.MaxOpenConns, 1, 500); err != nil {
		return err
	}
	if cfg.MinIdleConns < 0 || cfg.MinIdleConns > cfg.MaxOpenConns {
		return fmt.Errorf(
			"%w: postgres.min_idle_conns must be in range [0,postgres.max_open_conns] (%d)",
			ErrValidate,
			cfg.MaxOpenConns,
		)
	}
	if err := validateDurationRange("postgres.conn_max_lifetime", cfg.ConnMaxLifetime, time.Minute, 24*time.Hour); err != nil {
		return err
	}
	if err := validateDurationRange("postgres.statement_timeout", cfg.StatementTimeout, 100*time.Millisecond, 10*time.Minute); err != nil {
		return err
	}

	return nil
}

// profile:database-postgres:end

package config

// profile:database-postgres:start

import (
	"fmt"
	"strings"
)

// PostgresConfig contains the profile switch, secret connection source, and
// the one deployment capacity value that has no universal safe answer.
type PostgresConfig struct {
	Enabled      bool   `koanf:"enabled"`
	DSN          string `koanf:"dsn"`
	MaxOpenConns int    `koanf:"max_open_conns"`
}

func postgresDefaults() map[string]any {
	return map[string]any{
		"postgres.enabled":        false,
		"postgres.dsn":            "",
		"postgres.max_open_conns": 4,
	}
}

func validatePostgres(cfg PostgresConfig, _ HTTPConfig) error {
	if cfg.Enabled && strings.TrimSpace(cfg.DSN) == "" {
		return fmt.Errorf("%w: postgres.dsn is required when postgres.enabled=true", ErrSecretPolicy)
	}
	if err := validateIntRange("postgres.max_open_conns", cfg.MaxOpenConns, 1, 500); err != nil {
		return err
	}
	return nil
}

// profile:database-postgres:end

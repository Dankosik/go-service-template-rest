package config

// profile:database-postgres:start

import (
	"fmt"
	"strings"
	"time"
)

// PostgresConfig defines PostgreSQL connection, migration, and pool settings.
type PostgresConfig struct {
	Enabled                   bool          `koanf:"enabled"`
	DSN                       string        `koanf:"dsn"`
	ConnectTimeout            time.Duration `koanf:"connect_timeout"`
	HealthcheckTimeout        time.Duration `koanf:"healthcheck_timeout"`
	MigrationTimeout          time.Duration `koanf:"migration_timeout"`
	MigrationStatementTimeout time.Duration `koanf:"migration_statement_timeout"`
	MigrationLockTimeout      time.Duration `koanf:"migration_lock_timeout"`
	MaxOpenConns              int           `koanf:"max_open_conns"`
	// MinIdleConns keeps connections open through quiet periods. At zero, pgx
	// closes every idle connection after its idle timeout, so a low-traffic
	// service pays TCP, TLS, and authentication on the first request of each
	// spike — once per connection the spike needs, all at the same moment.
	MinIdleConns int `koanf:"min_idle_conns"`
	// AcquireTimeout bounds waiting for a pooled connection. Nothing else does:
	// an acquire waits out whatever remains of the request budget, so a slow
	// database does not shed, it queues — and once every in-flight slot is held
	// by a connection waiter, requests that touch no database at all are shed
	// for capacity the database took. Validated against http.request_timeout so
	// a caller that waits still has budget left to run its query.
	AcquireTimeout  time.Duration `koanf:"acquire_timeout"`
	ConnMaxLifetime time.Duration `koanf:"conn_max_lifetime"`
	// StatementTimeout bounds every runtime statement server-side, and bounds
	// how long a session may sit idle inside a transaction. The request budget
	// only cancels client-side: if the cancel request itself is lost, which is
	// the failure a slow dependency tends to produce, an abandoned query keeps
	// running and keeps its locks. Validated against http.request_timeout so the
	// two are one budget rather than two numbers.
	StatementTimeout time.Duration `koanf:"statement_timeout"`
}

func postgresDefaults() map[string]any {
	return map[string]any{
		"postgres.enabled":                     false,
		"postgres.dsn":                         "",
		"postgres.connect_timeout":             "3s",
		"postgres.healthcheck_timeout":         "3s",
		"postgres.migration_timeout":           "5m",
		"postgres.migration_statement_timeout": "2m",
		"postgres.migration_lock_timeout":      "15s",
		// max_open_conns sits deliberately below http.max_in_flight, so shedding
		// engages after the pool saturates rather than before it is used. See
		// http.max_in_flight, which states the same relation from its side.
		"postgres.max_open_conns": 25,
		// min_idle_conns keeps a small warm floor so a low-traffic service does
		// not pay a full connect handshake on the first request after a quiet
		// period, once per connection the arriving burst needs.
		"postgres.min_idle_conns": 2,
		// acquire_timeout is a small fraction of http.request_timeout on purpose:
		// a caller that has waited this long is queued behind a saturated pool,
		// and telling it so in a second beats spending the rest of its budget in
		// a queue and answering 504. What is left of the budget is the query's.
		"postgres.acquire_timeout":   "1s",
		"postgres.conn_max_lifetime": "30m",
		"postgres.statement_timeout": "8s",
	}
}

// validatePostgres also holds the two budgets this section shares with HTTP,
// which is why it takes that section too. They live here rather than in a
// central cross-section pass because Postgres is the dependent half: both rules
// bound a Postgres setting against the request budget, the field comments above
// already state the relation from this side, and a section a build profile
// removes then takes its rules with it.
func validatePostgres(cfg PostgresConfig, httpCfg HTTPConfig) error {
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

	if !cfg.Enabled {
		return nil
	}
	if cfg.StatementTimeout > httpCfg.RequestTimeout {
		return fmt.Errorf(
			"%w: postgres.statement_timeout must be <= http.request_timeout (%s)",
			ErrValidate,
			httpCfg.RequestTimeout,
		)
	}
	// Strictly less, not at most: a caller that spends the whole request budget
	// waiting for a connection has nothing left to run a query with, which makes
	// the wait indistinguishable from the unbounded one this budget replaced.
	if cfg.AcquireTimeout >= httpCfg.RequestTimeout {
		return fmt.Errorf(
			"%w: postgres.acquire_timeout must be < http.request_timeout (%s) so a caller that waited still has budget to query",
			ErrValidate,
			httpCfg.RequestTimeout,
		)
	}
	return nil
}

// profile:database-postgres:end

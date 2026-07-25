package config

import (
	"fmt"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/observability/otelconfig"
)

// validateConfig is pure computation over an in-memory snapshot: no I/O, and it
// measures at 0ms in the startup log. Cancellation is observed once by the
// caller before this runs rather than between every rule.
func validateConfig(cfg *Config, unknownKeys []string) error {
	if unknown := findUnknownKeys(unknownKeys); len(unknown) > 0 {
		return fmt.Errorf("%w: unknown keys: %s", ErrUnknownKey, strings.Join(unknown, ", "))
	}

	if err := validateAppConfig(&cfg.App); err != nil {
		return err
	}
	if err := validateHTTPConfig(&cfg.HTTP); err != nil {
		return err
	}
	if err := validateHealthConfig(cfg.Health); err != nil {
		return err
	}
	if err := validateRuntimeConfig(cfg.Runtime); err != nil {
		return err
	}

	// profile:database-postgres:start
	if err := validatePostgres(cfg.Postgres); err != nil {
		return err
	}
	// profile:database-postgres:end

	if err := validateObservabilityConfig(&cfg.Observability); err != nil {
		return err
	}
	return validateCrossSectionBudgets(*cfg)
}

// validateCrossSectionBudgets rejects settings that are each individually legal
// but incoherent together. A budget that only holds inside one section is not a
// budget: it is a number that happens to look like one.
func validateCrossSectionBudgets(cfg Config) error {
	// profile:database-postgres:start
	if cfg.Postgres.Enabled {
		if cfg.Postgres.StatementTimeout > cfg.HTTP.RequestTimeout {
			return fmt.Errorf(
				"%w: postgres.statement_timeout must be <= http.request_timeout (%s)",
				ErrValidate,
				cfg.HTTP.RequestTimeout,
			)
		}
		// Strictly less, not at most: a caller that spends the whole request
		// budget waiting for a connection has nothing left to run a query with,
		// which makes the wait indistinguishable from the unbounded one this
		// budget replaced.
		if cfg.Postgres.AcquireTimeout >= cfg.HTTP.RequestTimeout {
			return fmt.Errorf(
				"%w: postgres.acquire_timeout must be < http.request_timeout (%s) so a caller that waited still has budget to query",
				ErrValidate,
				cfg.HTTP.RequestTimeout,
			)
		}
		if cfg.HTTP.MaxInFlight > 0 && cfg.HTTP.MaxInFlight < cfg.Postgres.MaxOpenConns {
			return fmt.Errorf(
				"%w: http.max_in_flight must be >= postgres.max_open_conns (%d) so shedding cannot be tighter than the pool it protects",
				ErrValidate,
				cfg.Postgres.MaxOpenConns,
			)
		}
	}
	// profile:database-postgres:end

	// There is deliberately no rule tying health.refresh_interval to
	// http.readiness_timeout. The readiness handler answers from cached state and
	// performs no I/O, so its budget bounds nothing the refresher does; the
	// interval only has to be small relative to the orchestrator's own probe
	// period, which this service cannot see.
	if !cfg.Observability.Pprof.Enabled {
		return nil
	}
	if cfg.Observability.Metrics.Addr == "" {
		return fmt.Errorf(
			"%w: observability.pprof.enabled requires observability.metrics.addr, which serves the diagnostics listener",
			ErrValidate,
		)
	}
	return nil
}

func validateHealthConfig(cfg HealthConfig) error {
	if err := validateDurationRange("health.refresh_interval", cfg.RefreshInterval, 100*time.Millisecond, time.Minute); err != nil {
		return err
	}
	if cfg.FailureThreshold < 1 || cfg.FailureThreshold > 100 {
		return fmt.Errorf("%w: health.failure_threshold must be in range [1,100]", ErrValidate)
	}
	return nil
}

func validateRuntimeConfig(cfg RuntimeConfig) error {
	if math.IsNaN(cfg.MemoryLimitRatio) || math.IsInf(cfg.MemoryLimitRatio, 0) {
		return fmt.Errorf("%w: runtime.memory_limit_ratio must be finite", ErrValidate)
	}
	if cfg.MemoryLimitRatio < 0 || cfg.MemoryLimitRatio > 1 {
		return fmt.Errorf("%w: runtime.memory_limit_ratio must be in range [0,1]", ErrValidate)
	}
	return nil
}

func validateAppConfig(cfg *AppConfig) error {
	cfg.Env = strings.TrimSpace(cfg.Env)
	cfg.Version = strings.TrimSpace(cfg.Version)
	cfg.Commit = strings.TrimSpace(cfg.Commit)
	cfg.InstanceID = strings.TrimSpace(cfg.InstanceID)
	if cfg.Env == "" {
		return fmt.Errorf("%w: app.env cannot be empty", ErrValidate)
	}
	if cfg.Version == "" {
		return fmt.Errorf("%w: app.version cannot be empty", ErrValidate)
	}
	// Rejected rather than defaulted, because the honest unknown is already the
	// build default: an empty value would publish a resource attribute and a
	// diagnostic field that read as a lost identifier rather than an unstamped
	// build.
	if cfg.Commit == "" {
		return fmt.Errorf("%w: app.commit cannot be empty", ErrValidate)
	}
	return nil
}

func validateHTTPConfig(cfg *HTTPConfig) error {
	cfg.Addr = strings.TrimSpace(cfg.Addr)
	if cfg.Addr == "" {
		return fmt.Errorf("%w: http.addr cannot be empty", ErrValidate)
	}
	if err := validateDurationRange("http.grace_period", cfg.GracePeriod, time.Second, 10*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.shutdown_timeout", cfg.ShutdownTimeout, time.Second, 10*time.Minute); err != nil {
		return err
	}
	// The drain is one stage inside the grace period, never the whole of it. How
	// much the stages after it need is owned by the composition root, which knows
	// their ceilings; see validateShutdownGraceBudget.
	if cfg.ShutdownTimeout > cfg.GracePeriod {
		return fmt.Errorf(
			"%w: http.shutdown_timeout must be <= http.grace_period (%s)",
			ErrValidate,
			cfg.GracePeriod,
		)
	}
	if err := validateDurationRange("http.readiness_timeout", cfg.ReadinessTimeout, 100*time.Millisecond, 30*time.Second); err != nil {
		return err
	}
	if err := validateDurationRange("http.readiness_propagation_delay", cfg.ReadinessPropagationDelay, 0, cfg.ShutdownTimeout); err != nil {
		return err
	}
	if err := validateDurationRange("http.read_header_timeout", cfg.ReadHeaderTimeout, 100*time.Millisecond, 5*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.read_timeout", cfg.ReadTimeout, 100*time.Millisecond, 5*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.request_timeout", cfg.RequestTimeout, 100*time.Millisecond, 10*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.write_timeout", cfg.WriteTimeout, 100*time.Millisecond, 10*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.idle_timeout", cfg.IdleTimeout, 100*time.Millisecond, 24*time.Hour); err != nil {
		return err
	}
	if err := validateHTTPReadinessWriteTimeout(*cfg); err != nil {
		return err
	}
	if err := validateHTTPRequestWriteTimeout(*cfg); err != nil {
		return err
	}
	if err := validateHTTPShutdownBudget(*cfg); err != nil {
		return err
	}
	if cfg.MaxHeaderBytes <= 0 {
		return fmt.Errorf("%w: http.max_header_bytes must be > 0", ErrValidate)
	}
	if cfg.MaxBodyBytes <= 0 {
		return fmt.Errorf("%w: http.max_body_bytes must be > 0", ErrValidate)
	}
	if cfg.MaxInFlight < 0 || cfg.MaxInFlight > 100_000 {
		return fmt.Errorf("%w: http.max_in_flight must be in range [0,100000]", ErrValidate)
	}
	if err := validateDurationRange(
		"http.idempotency_outcome_timeout",
		cfg.IdempotencyOutcomeTimeout,
		10*time.Millisecond,
		30*time.Second,
	); err != nil {
		return err
	}
	return validateHTTPIdempotencyShutdownBudget(*cfg)
}

// validateHTTPIdempotencyShutdownBudget keeps the outcome bound inside the drain.
//
// A request admitted just before SIGTERM can be inside the outcome call when the
// drain starts, and http.Server.Shutdown waits for it. A bound larger than the
// whole shutdown budget would mean the drain gives up, force-closes the
// connection, and the process then waits on a goroutine that is still writing to a
// store — which is the shutdown this budget exists to keep orderly.
func validateHTTPIdempotencyShutdownBudget(cfg HTTPConfig) error {
	if cfg.IdempotencyOutcomeTimeout <= cfg.ShutdownTimeout {
		return nil
	}
	return fmt.Errorf(
		"%w: http.idempotency_outcome_timeout must be <= http.shutdown_timeout (%s)",
		ErrValidate,
		cfg.ShutdownTimeout,
	)
}

func validateObservabilityConfig(cfg *ObservabilityConfig) error {
	cfg.Metrics.Addr = strings.TrimSpace(cfg.Metrics.Addr)
	if cfg.Metrics.Addr != "" {
		_, rawPort, err := net.SplitHostPort(cfg.Metrics.Addr)
		if err != nil {
			return fmt.Errorf("%w: observability.metrics.addr must be host:port", ErrValidate)
		}
		port, err := strconv.ParseUint(rawPort, 10, 16)
		if err != nil || port == 0 {
			return fmt.Errorf("%w: observability.metrics.addr port must be in range [1,65535]", ErrValidate)
		}
	}
	cfg.OTel.ServiceName = strings.TrimSpace(cfg.OTel.ServiceName)
	cfg.OTel.TracesSampler = strings.TrimSpace(cfg.OTel.TracesSampler)
	cfg.OTel.Exporter.OTLPEndpoint = strings.TrimSpace(cfg.OTel.Exporter.OTLPEndpoint)
	cfg.OTel.Exporter.OTLPMetricsEndpoint = strings.TrimSpace(cfg.OTel.Exporter.OTLPMetricsEndpoint)
	if cfg.OTel.ServiceName == "" {
		return fmt.Errorf("%w: observability.otel.service_name cannot be empty", ErrValidate)
	}
	return validateObservabilitySampler(cfg.OTel.TracesSampler, cfg.OTel.TracesSamplerArg)
}

// validateObservabilitySampler prefixes the shared sampler rules with the
// section that owns them, so a rejection names the setting an operator can edit.
func validateObservabilitySampler(sampler string, samplerArg float64) error {
	if err := otelconfig.ValidateTraceSampler(sampler, samplerArg); err != nil {
		return fmt.Errorf("%w: observability.otel.%w", ErrValidate, err)
	}
	return nil
}

func findUnknownKeys(keys []string) []string {
	unknownSet := make(map[string]struct{})
	unknown := make([]string, 0)
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := unknownSet[key]; ok {
			continue
		}
		unknownSet[key] = struct{}{}
		unknown = append(unknown, key)
	}
	sort.Strings(unknown)
	return unknown
}

// profile:database-postgres:start
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
	if err := validateDurationRange("postgres.acquire_timeout", cfg.AcquireTimeout, 10*time.Millisecond, 30*time.Second); err != nil {
		return err
	}
	if cfg.MaxOpenConns < 1 || cfg.MaxOpenConns > 500 {
		return fmt.Errorf("%w: postgres.max_open_conns must be in range [1,500]", ErrValidate)
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
	if err := validateDurationRange("postgres.idempotency_retention", cfg.IdempotencyRetention, time.Minute, 30*24*time.Hour); err != nil {
		return err
	}
	if err := validateDurationRange(
		"postgres.idempotency_sweep_interval",
		cfg.IdempotencySweepInterval,
		time.Second,
		cfg.IdempotencyRetention,
	); err != nil {
		return err
	}

	return nil
}

// profile:database-postgres:end

func validateHTTPShutdownBudget(cfg HTTPConfig) error {
	effectiveDrainBudget := cfg.ShutdownTimeout - cfg.ReadinessPropagationDelay
	if effectiveDrainBudget <= 0 {
		return fmt.Errorf("%w: http.readiness_propagation_delay must be less than http.shutdown_timeout", ErrValidate)
	}
	if cfg.WriteTimeout > effectiveDrainBudget {
		return fmt.Errorf(
			"%w: http.write_timeout must be <= effective drain budget after readiness propagation (%s)",
			ErrValidate,
			effectiveDrainBudget,
		)
	}
	return nil
}

func validateHTTPReadinessWriteTimeout(cfg HTTPConfig) error {
	if cfg.ReadinessTimeout > cfg.WriteTimeout {
		return fmt.Errorf("%w: http.readiness_timeout must be <= http.write_timeout", ErrValidate)
	}
	return nil
}

// validateHTTPRequestWriteTimeout keeps the handler budget inside the response
// write deadline. A request budget larger than http.write_timeout expires only
// after the connection can no longer carry a response, so the timeout would be
// reported to the client as a dropped connection instead of a 504.
func validateHTTPRequestWriteTimeout(cfg HTTPConfig) error {
	if cfg.RequestTimeout > cfg.WriteTimeout {
		return fmt.Errorf("%w: http.request_timeout must be <= http.write_timeout", ErrValidate)
	}
	return nil
}

func validateDurationRange(name string, value time.Duration, lowerBound time.Duration, upperBound time.Duration) error {
	if value < lowerBound || value > upperBound {
		return fmt.Errorf("%w: %s must be in range [%s,%s]", ErrValidate, name, lowerBound, upperBound)
	}
	return nil
}

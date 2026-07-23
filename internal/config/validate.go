package config

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/observability/otelconfig"
)

func validateConfig(ctx context.Context, cfg *Config, strict bool, unknownKeys []string) ([]string, error) {
	if err := checkValidateContext(ctx); err != nil {
		return nil, err
	}

	unknownKeyWarnings := findUnknownKeys(unknownKeys)
	if strict && len(unknownKeyWarnings) > 0 {
		return nil, fmt.Errorf("%w: unknown keys: %s", ErrStrictUnknownKey, strings.Join(unknownKeyWarnings, ", "))
	}
	if err := checkValidateContext(ctx); err != nil {
		return unknownKeyWarnings, err
	}

	if err := validateAppConfig(cfg.App); err != nil {
		return unknownKeyWarnings, err
	}
	if err := validateHTTPConfig(cfg.HTTP); err != nil {
		return unknownKeyWarnings, err
	}
	if err := checkValidateContext(ctx); err != nil {
		return unknownKeyWarnings, err
	}

	if err := validatePostgres(cfg.Postgres); err != nil {
		return unknownKeyWarnings, err
	}
	if err := validatePostgresReadinessBudget(*cfg); err != nil {
		return unknownKeyWarnings, err
	}
	if err := checkValidateContext(ctx); err != nil {
		return unknownKeyWarnings, err
	}

	if err := validateObservabilityConfig(cfg.Observability); err != nil {
		return unknownKeyWarnings, err
	}

	return unknownKeyWarnings, nil
}

func validateAppConfig(cfg AppConfig) error {
	if strings.TrimSpace(cfg.Env) == "" {
		return fmt.Errorf("%w: app.env cannot be empty", ErrValidate)
	}
	if strings.TrimSpace(cfg.Version) == "" {
		return fmt.Errorf("%w: app.version cannot be empty", ErrValidate)
	}
	return nil
}

func validateHTTPConfig(cfg HTTPConfig) error {
	if strings.TrimSpace(cfg.Addr) == "" {
		return fmt.Errorf("%w: http.addr cannot be empty", ErrValidate)
	}
	if err := validateDurationRange("http.shutdown_timeout", cfg.ShutdownTimeout, time.Second, 10*time.Minute); err != nil {
		return err
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
	if err := validateDurationRange("http.write_timeout", cfg.WriteTimeout, 100*time.Millisecond, 10*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.idle_timeout", cfg.IdleTimeout, 100*time.Millisecond, 24*time.Hour); err != nil {
		return err
	}
	if err := validateHTTPReadinessWriteTimeout(cfg); err != nil {
		return err
	}
	if err := validateHTTPShutdownBudget(cfg); err != nil {
		return err
	}
	if cfg.MaxHeaderBytes <= 0 {
		return fmt.Errorf("%w: http.max_header_bytes must be > 0", ErrValidate)
	}
	if cfg.MaxBodyBytes <= 0 {
		return fmt.Errorf("%w: http.max_body_bytes must be > 0", ErrValidate)
	}
	return nil
}

func validateObservabilityConfig(cfg ObservabilityConfig) error {
	if strings.TrimSpace(cfg.OTel.ServiceName) == "" {
		return fmt.Errorf("%w: observability.otel.service_name cannot be empty", ErrValidate)
	}
	return validateSampler(cfg.OTel.TracesSampler, cfg.OTel.TracesSamplerArg)
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
	if cfg.MaxOpenConns < 1 || cfg.MaxOpenConns > 500 {
		return fmt.Errorf("%w: postgres.max_open_conns must be in range [1,500]", ErrValidate)
	}
	if err := validateDurationRange("postgres.conn_max_lifetime", cfg.ConnMaxLifetime, time.Minute, 24*time.Hour); err != nil {
		return err
	}

	return nil
}

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

func validatePostgresReadinessBudget(cfg Config) error {
	if !cfg.Postgres.Enabled {
		return nil
	}
	if cfg.HTTP.ReadinessTimeout < cfg.Postgres.HealthcheckTimeout {
		return fmt.Errorf(
			"%w: http.readiness_timeout must be >= readiness probe budget (postgres.healthcheck_timeout = %s)",
			ErrValidate,
			cfg.Postgres.HealthcheckTimeout,
		)
	}
	return nil
}

func validateSampler(sampler string, samplerArg float64) error {
	if !otelconfig.TraceSamplerSupported(sampler) {
		return fmt.Errorf("%w: observability.otel.traces_sampler is unsupported", ErrValidate)
	}

	if !otelconfig.TraceSamplerArgFinite(samplerArg) {
		return fmt.Errorf("%w: observability.otel.traces_sampler_arg must be finite", ErrValidate)
	}
	if !otelconfig.TraceSamplerArgInRange(samplerArg) {
		return fmt.Errorf("%w: observability.otel.traces_sampler_arg must be in range [0,1]", ErrValidate)
	}
	return nil
}

func validateDurationRange(name string, value time.Duration, lowerBound time.Duration, upperBound time.Duration) error {
	if value < lowerBound || value > upperBound {
		return fmt.Errorf("%w: %s must be in range [%s,%s]", ErrValidate, name, lowerBound, upperBound)
	}
	return nil
}

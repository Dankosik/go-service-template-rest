package config

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knadh/koanf/v2"
)

type ValidationOptions struct {
	Strict bool
}

type ValidationResult struct {
	UnknownKeyWarnings []string
}

func validateConfig(ctx context.Context, k *koanf.Koanf, cfg *Config, opts ValidationOptions) (ValidationResult, error) {
	result := ValidationResult{}
	if err := checkContext(ctx); err != nil {
		return ValidationResult{}, err
	}

	unknownKeys := findUnknownKeys(k.Keys())
	if len(unknownKeys) > 0 {
		if opts.Strict {
			return ValidationResult{}, fmt.Errorf("%w: unknown keys: %s", ErrStrictUnknownKey, strings.Join(unknownKeys, ", "))
		}
		result.UnknownKeyWarnings = unknownKeys
	}
	if err := checkContext(ctx); err != nil {
		return ValidationResult{}, err
	}

	if strings.TrimSpace(cfg.App.Env) == "" {
		return ValidationResult{}, fmt.Errorf("%w: app.env cannot be empty", ErrValidate)
	}
	if strings.TrimSpace(cfg.HTTP.Addr) == "" {
		return ValidationResult{}, fmt.Errorf("%w: http.addr cannot be empty", ErrValidate)
	}

	if err := validateDurationRange("http.shutdown_timeout", cfg.HTTP.ShutdownTimeout, time.Second, 10*time.Minute); err != nil {
		return ValidationResult{}, err
	}
	if err := validateDurationRange("http.readiness_timeout", cfg.HTTP.ReadinessTimeout, 100*time.Millisecond, 30*time.Second); err != nil {
		return ValidationResult{}, err
	}
	if err := validateDurationRange("http.readiness_propagation_delay", cfg.HTTP.ReadinessPropagationDelay, 0, cfg.HTTP.ShutdownTimeout); err != nil {
		return ValidationResult{}, err
	}
	if err := validateDurationRange("http.read_header_timeout", cfg.HTTP.ReadHeaderTimeout, 100*time.Millisecond, 5*time.Minute); err != nil {
		return ValidationResult{}, err
	}
	if err := validateDurationRange("http.read_timeout", cfg.HTTP.ReadTimeout, 100*time.Millisecond, 5*time.Minute); err != nil {
		return ValidationResult{}, err
	}
	if err := validateDurationRange("http.write_timeout", cfg.HTTP.WriteTimeout, 100*time.Millisecond, 10*time.Minute); err != nil {
		return ValidationResult{}, err
	}
	if err := validateDurationRange("http.idle_timeout", cfg.HTTP.IdleTimeout, 100*time.Millisecond, 24*time.Hour); err != nil {
		return ValidationResult{}, err
	}
	if err := validateHTTPShutdownBudget(cfg.HTTP); err != nil {
		return ValidationResult{}, err
	}
	if cfg.HTTP.MaxHeaderBytes <= 0 {
		return ValidationResult{}, fmt.Errorf("%w: http.max_header_bytes must be > 0", ErrValidate)
	}
	if cfg.HTTP.MaxBodyBytes <= 0 {
		return ValidationResult{}, fmt.Errorf("%w: http.max_body_bytes must be > 0", ErrValidate)
	}
	if err := checkContext(ctx); err != nil {
		return ValidationResult{}, err
	}

	if err := validatePostgres(cfg.Postgres); err != nil {
		return ValidationResult{}, err
	}
	if err := validateRedis(cfg.Redis); err != nil {
		return ValidationResult{}, err
	}
	if err := validateMongo(cfg.Mongo); err != nil {
		return ValidationResult{}, err
	}
	if err := validateReadinessProbeBudgets(*cfg); err != nil {
		return ValidationResult{}, err
	}
	if err := checkContext(ctx); err != nil {
		return ValidationResult{}, err
	}

	if err := validateSampler(cfg.Observability.OTel.TracesSampler, cfg.Observability.OTel.TracesSamplerArg); err != nil {
		return ValidationResult{}, err
	}
	if err := validateOTLPExporter(cfg.Observability.OTel.Exporter); err != nil {
		return ValidationResult{}, err
	}

	return result, nil
}

func findUnknownKeys(allKeys []string) []string {
	knownKeys := knownConfigKeys()
	unknown := make([]string, 0)
	for _, key := range allKeys {
		if _, ok := knownKeys[key]; ok {
			continue
		}

		knownAsPrefix := false
		for known := range knownKeys {
			if strings.HasPrefix(known, key+".") {
				knownAsPrefix = true
				break
			}
		}
		if knownAsPrefix {
			continue
		}

		unknown = append(unknown, key)
	}

	sort.Strings(unknown)
	return unknown
}

func validatePostgres(cfg PostgresConfig) error {
	if cfg.Enabled && strings.TrimSpace(cfg.DSN) == "" {
		return fmt.Errorf("%w: postgres.dsn is required when postgres.enabled=true", ErrSecretPolicy)
	}
	if cfg.Enabled {
		if _, err := pgxpool.ParseConfig(cfg.DSN); err != nil {
			return fmt.Errorf("%w: postgres.dsn must be parseable", ErrValidate)
		}
	}

	if err := validateDurationRange("postgres.connect_timeout", cfg.ConnectTimeout, 100*time.Millisecond, 10*time.Second); err != nil {
		return err
	}
	if err := validateDurationRange("postgres.healthcheck_timeout", cfg.HealthcheckTimeout, 100*time.Millisecond, 10*time.Second); err != nil {
		return err
	}
	if err := validateIntRange("postgres.max_open_conns", cfg.MaxOpenConns, 1, 500); err != nil {
		return err
	}
	if err := validateIntRange("postgres.max_idle_conns", cfg.MaxIdleConns, 0, cfg.MaxOpenConns); err != nil {
		return err
	}
	if err := validateDurationRange("postgres.conn_max_lifetime", cfg.ConnMaxLifetime, time.Minute, 24*time.Hour); err != nil {
		return err
	}

	return nil
}

func validateRedis(cfg RedisConfig) error {
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "" {
		return fmt.Errorf("%w: redis.mode cannot be empty", ErrValidate)
	}
	if mode != "cache" && mode != "store" {
		return fmt.Errorf("%w: redis.mode must be one of [cache,store]", ErrValidate)
	}

	if cfg.Enabled {
		if strings.TrimSpace(cfg.Addr) == "" {
			return fmt.Errorf("%w: redis.addr is required when redis.enabled=true", ErrValidate)
		}
		if _, _, err := net.SplitHostPort(strings.TrimSpace(cfg.Addr)); err != nil {
			return fmt.Errorf("%w: redis.addr must be host:port", ErrValidate)
		}
	}

	if mode == "store" {
		// ARCH-008: v1 only supports guard/reject behavior for store mode.
		if !cfg.AllowStoreMode {
			return fmt.Errorf("%w: redis.mode=store is blocked unless redis.allow_store_mode=true", ErrValidate)
		}
		if cfg.StaleWindow != 0 {
			return fmt.Errorf("%w: redis.stale_window must be 0 when redis.mode=store", ErrValidate)
		}
	}

	if err := validateIntRange("redis.db", cfg.DB, 0, 15); err != nil {
		return err
	}
	if err := validateDurationRange("redis.dial_timeout", cfg.DialTimeout, 50*time.Millisecond, 5*time.Second); err != nil {
		return err
	}
	if err := validateDurationRange("redis.read_timeout", cfg.ReadTimeout, 50*time.Millisecond, 5*time.Second); err != nil {
		return err
	}
	if err := validateDurationRange("redis.write_timeout", cfg.WriteTimeout, 50*time.Millisecond, 5*time.Second); err != nil {
		return err
	}
	if err := validateIntRange("redis.pool_size", cfg.PoolSize, 1, 1000); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.KeyPrefix) == "" {
		return fmt.Errorf("%w: redis.key_prefix cannot be empty", ErrValidate)
	}
	if err := validateDurationRange("redis.fresh_ttl", cfg.FreshTTL, time.Second, 15*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("redis.stale_window", cfg.StaleWindow, 0, 5*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("redis.negative_ttl", cfg.NegativeTTL, time.Second, 60*time.Second); err != nil {
		return err
	}
	if err := validateIntRange("redis.ttl_jitter_percent", cfg.TTLJitterPercent, 0, 30); err != nil {
		return err
	}
	if err := validateIntRange("redis.max_fallback_concurrency", cfg.MaxFallbackConcurrency, 1, 256); err != nil {
		return err
	}

	return nil
}

func validateMongo(cfg MongoConfig) error {
	if cfg.Enabled && strings.TrimSpace(cfg.URI) == "" {
		return fmt.Errorf("%w: mongo.uri is required when mongo.enabled=true", ErrSecretPolicy)
	}
	if cfg.Enabled {
		if _, err := MongoProbeAddress(cfg.URI); err != nil {
			return fmt.Errorf("%w: mongo.uri must be parseable", ErrValidate)
		}
		if strings.TrimSpace(cfg.Database) == "" {
			return fmt.Errorf("%w: mongo.database is required when mongo.enabled=true", ErrValidate)
		}
	}

	if err := validateDurationRange("mongo.connect_timeout", cfg.ConnectTimeout, 100*time.Millisecond, 15*time.Second); err != nil {
		return err
	}
	if err := validateDurationRange("mongo.server_selection_timeout", cfg.ServerSelectionTimeout, 100*time.Millisecond, 10*time.Second); err != nil {
		return err
	}
	if err := validateIntRange("mongo.max_pool_size", cfg.MaxPoolSize, 1, 1000); err != nil {
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

func validateReadinessProbeBudgets(cfg Config) error {
	budgets := make([]readinessProbeBudget, 0, 3)
	if cfg.Postgres.Enabled && cfg.FeatureFlags.PostgresReadinessProbe {
		budgets = append(budgets, readinessProbeBudget{
			name:   "postgres.healthcheck_timeout",
			budget: cfg.Postgres.HealthcheckTimeout,
		})
	}

	redisMode := strings.ToLower(strings.TrimSpace(cfg.Redis.Mode))
	if cfg.Redis.Enabled && (cfg.FeatureFlags.RedisReadinessProbe || redisMode == "store") {
		budgets = append(budgets, readinessProbeBudget{
			name:   "redis.dial_timeout",
			budget: cfg.Redis.DialTimeout,
		})
	}

	if cfg.Mongo.Enabled && cfg.FeatureFlags.MongoReadinessProbe {
		budgets = append(budgets, readinessProbeBudget{
			name:   "mongo.connect_timeout",
			budget: cfg.Mongo.ConnectTimeout,
		})
	}

	var aggregate time.Duration
	names := make([]string, 0, len(budgets))
	for _, probe := range budgets {
		aggregate += probe.budget
		names = append(names, probe.name)
	}
	if cfg.HTTP.ReadinessTimeout < aggregate {
		return fmt.Errorf(
			"%w: http.readiness_timeout must be >= aggregate sequential readiness probe budget (%s = %s)",
			ErrValidate,
			strings.Join(names, " + "),
			aggregate,
		)
	}
	return nil
}

type readinessProbeBudget struct {
	name   string
	budget time.Duration
}

func validateSampler(sampler string, samplerArg float64) error {
	switch strings.ToLower(strings.TrimSpace(sampler)) {
	case "always_on", "always_off", "traceidratio", "parentbased_traceidratio":
	default:
		return fmt.Errorf("%w: observability.otel.traces_sampler is unsupported", ErrValidate)
	}

	if samplerArg < 0 || samplerArg > 1 {
		return fmt.Errorf("%w: observability.otel.traces_sampler_arg must be in range [0,1]", ErrValidate)
	}
	return nil
}

func validateOTLPExporter(cfg OTelExporterConfig) error {
	protocol := strings.ToLower(strings.TrimSpace(cfg.OTLPProtocol))
	if protocol == "" {
		return nil
	}

	if protocol != "http/protobuf" {
		return fmt.Errorf("%w: observability.otel.exporter.otlp_protocol must be http/protobuf", ErrValidate)
	}
	return nil
}

func validateDurationRange(name string, value time.Duration, min time.Duration, max time.Duration) error {
	if value < min || value > max {
		return fmt.Errorf("%w: %s must be in range [%s,%s]", ErrValidate, name, min, max)
	}
	return nil
}

func validateIntRange(name string, value int, min int, max int) error {
	if value < min || value > max {
		return fmt.Errorf("%w: %s must be in range [%d,%d]", ErrValidate, name, min, max)
	}
	return nil
}

const (
	mongodbScheme    = "mongodb://"
	mongodbSRVScheme = "mongodb+srv://"
	defaultMongoPort = "27017"
)

// MongoProbeAddress extracts a probe-ready host:port from a MongoDB URI.
func MongoProbeAddress(rawURI string) (string, error) {
	uri := strings.TrimSpace(rawURI)
	if uri == "" {
		return "", fmt.Errorf("empty mongo uri")
	}

	lower := strings.ToLower(uri)
	var hostPart string
	switch {
	case strings.HasPrefix(lower, mongodbScheme):
		hostPart = uri[len(mongodbScheme):]
	case strings.HasPrefix(lower, mongodbSRVScheme):
		hostPart = uri[len(mongodbSRVScheme):]
	default:
		return "", fmt.Errorf("unsupported mongo uri scheme")
	}

	if hostPart == "" {
		return "", fmt.Errorf("empty mongo host section")
	}
	if slash := strings.Index(hostPart, "/"); slash >= 0 {
		hostPart = hostPart[:slash]
	}
	if at := strings.LastIndex(hostPart, "@"); at >= 0 {
		hostPart = hostPart[at+1:]
	}
	if hostPart == "" {
		return "", fmt.Errorf("empty mongo host section")
	}

	hosts := strings.Split(hostPart, ",")
	firstHost := strings.TrimSpace(hosts[0])
	if firstHost == "" {
		return "", fmt.Errorf("empty mongo host")
	}

	return normalizeMongoProbeAddress(firstHost)
}

func normalizeMongoProbeAddress(host string) (string, error) {
	trimmed := strings.TrimSpace(host)
	if trimmed == "" {
		return "", fmt.Errorf("empty mongo host")
	}

	if _, _, err := net.SplitHostPort(trimmed); err == nil {
		return trimmed, nil
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "?") {
		return "", fmt.Errorf("invalid mongo host %q", host)
	}

	if strings.Count(trimmed, ":") > 1 && !strings.HasPrefix(trimmed, "[") && !strings.HasSuffix(trimmed, "]") {
		return net.JoinHostPort(trimmed, defaultMongoPort), nil
	}
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		return net.JoinHostPort(strings.Trim(trimmed, "[]"), defaultMongoPort), nil
	}

	if strings.Contains(trimmed, ":") {
		return "", fmt.Errorf("invalid mongo host %q", host)
	}

	return net.JoinHostPort(strings.Trim(trimmed, "[]"), defaultMongoPort), nil
}

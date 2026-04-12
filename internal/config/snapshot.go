package config

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/knadh/koanf/v2"
)

func buildSnapshot(k *koanf.Koanf) (Config, error) {
	var cfg Config

	cfg.App.Env = readTrimmedString(k, "app.env")
	cfg.App.Version = readTrimmedString(k, "app.version")

	cfg.HTTP.Addr = readTrimmedString(k, "http.addr")
	if err := readDurationInto(k, "http.shutdown_timeout", &cfg.HTTP.ShutdownTimeout); err != nil {
		return Config{}, err
	}
	if err := readDurationInto(k, "http.readiness_timeout", &cfg.HTTP.ReadinessTimeout); err != nil {
		return Config{}, err
	}
	if err := readDurationInto(k, "http.readiness_propagation_delay", &cfg.HTTP.ReadinessPropagationDelay); err != nil {
		return Config{}, err
	}
	if err := readDurationInto(k, "http.read_header_timeout", &cfg.HTTP.ReadHeaderTimeout); err != nil {
		return Config{}, err
	}
	if err := readDurationInto(k, "http.read_timeout", &cfg.HTTP.ReadTimeout); err != nil {
		return Config{}, err
	}
	if err := readDurationInto(k, "http.write_timeout", &cfg.HTTP.WriteTimeout); err != nil {
		return Config{}, err
	}
	if err := readDurationInto(k, "http.idle_timeout", &cfg.HTTP.IdleTimeout); err != nil {
		return Config{}, err
	}
	if err := readIntInto(k, "http.max_header_bytes", &cfg.HTTP.MaxHeaderBytes); err != nil {
		return Config{}, err
	}
	if err := readInt64Into(k, "http.max_body_bytes", &cfg.HTTP.MaxBodyBytes); err != nil {
		return Config{}, err
	}

	level, err := readLogLevel(k, "log.level")
	if err != nil {
		return Config{}, err
	}
	cfg.Log.Level = level

	cfg.Observability.OTel.ServiceName = readTrimmedString(k, "observability.otel.service_name")
	cfg.Observability.OTel.TracesSampler = readTrimmedString(k, "observability.otel.traces_sampler")
	if err := readFloat64Into(k, "observability.otel.traces_sampler_arg", &cfg.Observability.OTel.TracesSamplerArg); err != nil {
		return Config{}, err
	}
	cfg.Observability.OTel.Exporter.OTLPEndpoint = readTrimmedString(k, "observability.otel.exporter.otlp_endpoint")
	cfg.Observability.OTel.Exporter.OTLPTracesEndpoint = readTrimmedString(k, "observability.otel.exporter.otlp_traces_endpoint")
	cfg.Observability.OTel.Exporter.OTLPHeaders = readRawString(k, "observability.otel.exporter.otlp_headers")
	cfg.Observability.OTel.Exporter.OTLPProtocol = readTrimmedString(k, "observability.otel.exporter.otlp_protocol")

	if err := readBoolInto(k, "postgres.enabled", &cfg.Postgres.Enabled); err != nil {
		return Config{}, err
	}
	cfg.Postgres.DSN = readRawString(k, "postgres.dsn")
	if err := readDurationInto(k, "postgres.connect_timeout", &cfg.Postgres.ConnectTimeout); err != nil {
		return Config{}, err
	}
	if err := readDurationInto(k, "postgres.healthcheck_timeout", &cfg.Postgres.HealthcheckTimeout); err != nil {
		return Config{}, err
	}
	if err := readIntInto(k, "postgres.max_open_conns", &cfg.Postgres.MaxOpenConns); err != nil {
		return Config{}, err
	}
	if err := readIntInto(k, "postgres.max_idle_conns", &cfg.Postgres.MaxIdleConns); err != nil {
		return Config{}, err
	}
	if err := readDurationInto(k, "postgres.conn_max_lifetime", &cfg.Postgres.ConnMaxLifetime); err != nil {
		return Config{}, err
	}

	if err := readBoolInto(k, "redis.enabled", &cfg.Redis.Enabled); err != nil {
		return Config{}, err
	}
	cfg.Redis.Mode = normalizeRedisMode(readTrimmedString(k, "redis.mode"))
	if err := readBoolInto(k, "redis.allow_store_mode", &cfg.Redis.AllowStoreMode); err != nil {
		return Config{}, err
	}
	cfg.Redis.Addr = readTrimmedString(k, "redis.addr")
	cfg.Redis.Username = readRawString(k, "redis.username")
	cfg.Redis.Password = readRawString(k, "redis.password")
	if err := readIntInto(k, "redis.db", &cfg.Redis.DB); err != nil {
		return Config{}, err
	}
	if err := readDurationInto(k, "redis.dial_timeout", &cfg.Redis.DialTimeout); err != nil {
		return Config{}, err
	}
	if err := readDurationInto(k, "redis.read_timeout", &cfg.Redis.ReadTimeout); err != nil {
		return Config{}, err
	}
	if err := readDurationInto(k, "redis.write_timeout", &cfg.Redis.WriteTimeout); err != nil {
		return Config{}, err
	}
	if err := readIntInto(k, "redis.pool_size", &cfg.Redis.PoolSize); err != nil {
		return Config{}, err
	}
	cfg.Redis.KeyPrefix = readTrimmedString(k, "redis.key_prefix")
	if err := readDurationInto(k, "redis.fresh_ttl", &cfg.Redis.FreshTTL); err != nil {
		return Config{}, err
	}
	if err := readDurationInto(k, "redis.stale_window", &cfg.Redis.StaleWindow); err != nil {
		return Config{}, err
	}
	if err := readDurationInto(k, "redis.negative_ttl", &cfg.Redis.NegativeTTL); err != nil {
		return Config{}, err
	}
	if err := readIntInto(k, "redis.ttl_jitter_percent", &cfg.Redis.TTLJitterPercent); err != nil {
		return Config{}, err
	}
	if err := readBoolInto(k, "redis.enable_singleflight", &cfg.Redis.EnableSingleflight); err != nil {
		return Config{}, err
	}
	if err := readIntInto(k, "redis.max_fallback_concurrency", &cfg.Redis.MaxFallbackConcurrency); err != nil {
		return Config{}, err
	}

	if err := readBoolInto(k, "mongo.enabled", &cfg.Mongo.Enabled); err != nil {
		return Config{}, err
	}
	cfg.Mongo.URI = readRawString(k, "mongo.uri")
	cfg.Mongo.Database = readTrimmedString(k, "mongo.database")
	if err := readDurationInto(k, "mongo.connect_timeout", &cfg.Mongo.ConnectTimeout); err != nil {
		return Config{}, err
	}
	if err := readDurationInto(k, "mongo.server_selection_timeout", &cfg.Mongo.ServerSelectionTimeout); err != nil {
		return Config{}, err
	}
	if err := readIntInto(k, "mongo.max_pool_size", &cfg.Mongo.MaxPoolSize); err != nil {
		return Config{}, err
	}

	if err := readBoolInto(k, "feature_flags.postgres_readiness_probe", &cfg.FeatureFlags.PostgresReadinessProbe); err != nil {
		return Config{}, err
	}
	if err := readBoolInto(k, "feature_flags.mongo_readiness_probe", &cfg.FeatureFlags.MongoReadinessProbe); err != nil {
		return Config{}, err
	}
	if err := readBoolInto(k, "feature_flags.redis_readiness_probe", &cfg.FeatureFlags.RedisReadinessProbe); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func readRawString(k *koanf.Koanf, key string) string {
	return k.String(key)
}

func readTrimmedString(k *koanf.Koanf, key string) string {
	return strings.TrimSpace(readRawString(k, key))
}

func readDurationInto(k *koanf.Koanf, key string, dst *time.Duration) error {
	raw := readTrimmedString(k, key)
	if raw == "" {
		return fmt.Errorf("%w: %s is empty", ErrParse, key)
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fmt.Errorf("%w: %s has invalid duration", ErrParse, key)
	}
	*dst = d
	return nil
}

func readIntInto(k *koanf.Koanf, key string, dst *int) error {
	value, err := parseInt(k.Get(key))
	if err != nil {
		return fmt.Errorf("%w: %s has invalid int value", ErrParse, key)
	}
	*dst = value
	return nil
}

func readInt64Into(k *koanf.Koanf, key string, dst *int64) error {
	value, err := parseInt64(k.Get(key))
	if err != nil {
		return fmt.Errorf("%w: %s has invalid int64 value", ErrParse, key)
	}
	*dst = value
	return nil
}

func readFloat64Into(k *koanf.Koanf, key string, dst *float64) error {
	value, err := parseFloat64(k.Get(key))
	if err != nil {
		return fmt.Errorf("%w: %s has invalid float value", ErrParse, key)
	}
	*dst = value
	return nil
}

func readBoolInto(k *koanf.Koanf, key string, dst *bool) error {
	value, err := parseBool(k.Get(key))
	if err != nil {
		return fmt.Errorf("%w: %s has invalid bool value", ErrParse, key)
	}
	*dst = value
	return nil
}

func readLogLevel(k *koanf.Koanf, key string) (slog.Level, error) {
	raw := readTrimmedString(k, key)
	if raw == "" {
		return slog.LevelInfo, fmt.Errorf("%w: %s is empty", ErrParse, key)
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(raw)); err != nil {
		return slog.LevelInfo, fmt.Errorf("%w: %s has invalid log level", ErrParse, key)
	}
	return level, nil
}

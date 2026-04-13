package config

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/knadh/koanf/v2"
)

func buildSnapshot(k *koanf.Koanf) (Config, error) {
	httpCfg, err := readHTTPSnapshot(k)
	if err != nil {
		return Config{}, err
	}
	logCfg, err := readLogSnapshot(k)
	if err != nil {
		return Config{}, err
	}
	observabilityCfg, err := readObservabilitySnapshot(k)
	if err != nil {
		return Config{}, err
	}
	postgresCfg, err := readPostgresSnapshot(k)
	if err != nil {
		return Config{}, err
	}
	redisCfg, err := readRedisSnapshot(k)
	if err != nil {
		return Config{}, err
	}
	mongoCfg, err := readMongoSnapshot(k)
	if err != nil {
		return Config{}, err
	}
	featureFlagsCfg, err := readFeatureFlagsSnapshot(k)
	if err != nil {
		return Config{}, err
	}

	return Config{
		App:           readAppSnapshot(k),
		HTTP:          httpCfg,
		Log:           logCfg,
		Observability: observabilityCfg,
		Postgres:      postgresCfg,
		Redis:         redisCfg,
		Mongo:         mongoCfg,
		FeatureFlags:  featureFlagsCfg,
	}, nil
}

func readAppSnapshot(k *koanf.Koanf) AppConfig {
	return AppConfig{
		Env:     readTrimmedConfigString(k, "app.env"),
		Version: readTrimmedConfigString(k, "app.version"),
	}
}

func readHTTPSnapshot(k *koanf.Koanf) (HTTPConfig, error) {
	addr := readTrimmedConfigString(k, "http.addr")
	var shutdownTimeout time.Duration
	if err := readDurationInto(k, "http.shutdown_timeout", &shutdownTimeout); err != nil {
		return HTTPConfig{}, err
	}
	var readinessTimeout time.Duration
	if err := readDurationInto(k, "http.readiness_timeout", &readinessTimeout); err != nil {
		return HTTPConfig{}, err
	}
	var readinessPropagationDelay time.Duration
	if err := readDurationInto(k, "http.readiness_propagation_delay", &readinessPropagationDelay); err != nil {
		return HTTPConfig{}, err
	}
	var readHeaderTimeout time.Duration
	if err := readDurationInto(k, "http.read_header_timeout", &readHeaderTimeout); err != nil {
		return HTTPConfig{}, err
	}
	var readTimeout time.Duration
	if err := readDurationInto(k, "http.read_timeout", &readTimeout); err != nil {
		return HTTPConfig{}, err
	}
	var writeTimeout time.Duration
	if err := readDurationInto(k, "http.write_timeout", &writeTimeout); err != nil {
		return HTTPConfig{}, err
	}
	var idleTimeout time.Duration
	if err := readDurationInto(k, "http.idle_timeout", &idleTimeout); err != nil {
		return HTTPConfig{}, err
	}
	var maxHeaderBytes int
	if err := readIntInto(k, "http.max_header_bytes", &maxHeaderBytes); err != nil {
		return HTTPConfig{}, err
	}
	var maxBodyBytes int64
	if err := readInt64Into(k, "http.max_body_bytes", &maxBodyBytes); err != nil {
		return HTTPConfig{}, err
	}
	return HTTPConfig{
		Addr:                      addr,
		ShutdownTimeout:           shutdownTimeout,
		ReadinessTimeout:          readinessTimeout,
		ReadinessPropagationDelay: readinessPropagationDelay,
		ReadHeaderTimeout:         readHeaderTimeout,
		ReadTimeout:               readTimeout,
		WriteTimeout:              writeTimeout,
		IdleTimeout:               idleTimeout,
		MaxHeaderBytes:            maxHeaderBytes,
		MaxBodyBytes:              maxBodyBytes,
	}, nil
}

func readLogSnapshot(k *koanf.Koanf) (LogConfig, error) {
	level, err := readLogLevel(k, "log.level")
	if err != nil {
		return LogConfig{}, err
	}
	return LogConfig{Level: level}, nil
}

func readObservabilitySnapshot(k *koanf.Koanf) (ObservabilityConfig, error) {
	var tracesSamplerArg float64
	if err := readFloat64Into(k, "observability.otel.traces_sampler_arg", &tracesSamplerArg); err != nil {
		return ObservabilityConfig{}, err
	}
	return ObservabilityConfig{
		OTel: OTelConfig{
			ServiceName:      readTrimmedConfigString(k, "observability.otel.service_name"),
			TracesSampler:    readTrimmedConfigString(k, "observability.otel.traces_sampler"),
			TracesSamplerArg: tracesSamplerArg,
			Exporter: OTelExporterConfig{
				OTLPEndpoint:       readTrimmedConfigString(k, "observability.otel.exporter.otlp_endpoint"),
				OTLPTracesEndpoint: readTrimmedConfigString(k, "observability.otel.exporter.otlp_traces_endpoint"),
				OTLPHeaders:        readRawConfigString(k, "observability.otel.exporter.otlp_headers"),
				OTLPProtocol:       readTrimmedConfigString(k, "observability.otel.exporter.otlp_protocol"),
			},
		},
	}, nil
}

func readPostgresSnapshot(k *koanf.Koanf) (PostgresConfig, error) {
	var enabled bool
	if err := readBoolInto(k, "postgres.enabled", &enabled); err != nil {
		return PostgresConfig{}, err
	}
	dsn := readRawConfigString(k, "postgres.dsn")
	var connectTimeout time.Duration
	if err := readDurationInto(k, "postgres.connect_timeout", &connectTimeout); err != nil {
		return PostgresConfig{}, err
	}
	var healthcheckTimeout time.Duration
	if err := readDurationInto(k, "postgres.healthcheck_timeout", &healthcheckTimeout); err != nil {
		return PostgresConfig{}, err
	}
	var maxOpenConns int
	if err := readIntInto(k, "postgres.max_open_conns", &maxOpenConns); err != nil {
		return PostgresConfig{}, err
	}
	var maxIdleConns int
	if err := readIntInto(k, "postgres.max_idle_conns", &maxIdleConns); err != nil {
		return PostgresConfig{}, err
	}
	var connMaxLifetime time.Duration
	if err := readDurationInto(k, "postgres.conn_max_lifetime", &connMaxLifetime); err != nil {
		return PostgresConfig{}, err
	}
	return PostgresConfig{
		Enabled:            enabled,
		DSN:                dsn,
		ConnectTimeout:     connectTimeout,
		HealthcheckTimeout: healthcheckTimeout,
		MaxOpenConns:       maxOpenConns,
		MaxIdleConns:       maxIdleConns,
		ConnMaxLifetime:    connMaxLifetime,
	}, nil
}

func readRedisSnapshot(k *koanf.Koanf) (RedisConfig, error) {
	var enabled bool
	if err := readBoolInto(k, "redis.enabled", &enabled); err != nil {
		return RedisConfig{}, err
	}
	mode := normalizeRedisMode(readTrimmedConfigString(k, "redis.mode"))
	var allowStoreMode bool
	if err := readBoolInto(k, "redis.allow_store_mode", &allowStoreMode); err != nil {
		return RedisConfig{}, err
	}
	addr := readTrimmedConfigString(k, "redis.addr")
	username := readRawConfigString(k, "redis.username")
	password := readRawConfigString(k, "redis.password")
	var db int
	if err := readIntInto(k, "redis.db", &db); err != nil {
		return RedisConfig{}, err
	}
	var dialTimeout time.Duration
	if err := readDurationInto(k, "redis.dial_timeout", &dialTimeout); err != nil {
		return RedisConfig{}, err
	}
	var readTimeout time.Duration
	if err := readDurationInto(k, "redis.read_timeout", &readTimeout); err != nil {
		return RedisConfig{}, err
	}
	var writeTimeout time.Duration
	if err := readDurationInto(k, "redis.write_timeout", &writeTimeout); err != nil {
		return RedisConfig{}, err
	}
	var poolSize int
	if err := readIntInto(k, "redis.pool_size", &poolSize); err != nil {
		return RedisConfig{}, err
	}
	keyPrefix := readTrimmedConfigString(k, "redis.key_prefix")
	var freshTTL time.Duration
	if err := readDurationInto(k, "redis.fresh_ttl", &freshTTL); err != nil {
		return RedisConfig{}, err
	}
	var staleWindow time.Duration
	if err := readDurationInto(k, "redis.stale_window", &staleWindow); err != nil {
		return RedisConfig{}, err
	}
	var negativeTTL time.Duration
	if err := readDurationInto(k, "redis.negative_ttl", &negativeTTL); err != nil {
		return RedisConfig{}, err
	}
	var ttlJitterPercent int
	if err := readIntInto(k, "redis.ttl_jitter_percent", &ttlJitterPercent); err != nil {
		return RedisConfig{}, err
	}
	var enableSingleflight bool
	if err := readBoolInto(k, "redis.enable_singleflight", &enableSingleflight); err != nil {
		return RedisConfig{}, err
	}
	var maxFallbackConcurrency int
	if err := readIntInto(k, "redis.max_fallback_concurrency", &maxFallbackConcurrency); err != nil {
		return RedisConfig{}, err
	}
	return RedisConfig{
		Enabled:                enabled,
		Mode:                   mode,
		AllowStoreMode:         allowStoreMode,
		Addr:                   addr,
		Username:               username,
		Password:               password,
		DB:                     db,
		DialTimeout:            dialTimeout,
		ReadTimeout:            readTimeout,
		WriteTimeout:           writeTimeout,
		PoolSize:               poolSize,
		KeyPrefix:              keyPrefix,
		FreshTTL:               freshTTL,
		StaleWindow:            staleWindow,
		NegativeTTL:            negativeTTL,
		TTLJitterPercent:       ttlJitterPercent,
		EnableSingleflight:     enableSingleflight,
		MaxFallbackConcurrency: maxFallbackConcurrency,
	}, nil
}

func readMongoSnapshot(k *koanf.Koanf) (MongoConfig, error) {
	var enabled bool
	if err := readBoolInto(k, "mongo.enabled", &enabled); err != nil {
		return MongoConfig{}, err
	}
	uri := readRawConfigString(k, "mongo.uri")
	database := readTrimmedConfigString(k, "mongo.database")
	var connectTimeout time.Duration
	if err := readDurationInto(k, "mongo.connect_timeout", &connectTimeout); err != nil {
		return MongoConfig{}, err
	}
	var serverSelectionTimeout time.Duration
	if err := readDurationInto(k, "mongo.server_selection_timeout", &serverSelectionTimeout); err != nil {
		return MongoConfig{}, err
	}
	var maxPoolSize int
	if err := readIntInto(k, "mongo.max_pool_size", &maxPoolSize); err != nil {
		return MongoConfig{}, err
	}
	return MongoConfig{
		Enabled:                enabled,
		URI:                    uri,
		Database:               database,
		ConnectTimeout:         connectTimeout,
		ServerSelectionTimeout: serverSelectionTimeout,
		MaxPoolSize:            maxPoolSize,
	}, nil
}

func readFeatureFlagsSnapshot(k *koanf.Koanf) (FeatureFlagsConfig, error) {
	var postgresReadinessProbe bool
	if err := readBoolInto(k, "feature_flags.postgres_readiness_probe", &postgresReadinessProbe); err != nil {
		return FeatureFlagsConfig{}, err
	}
	var mongoReadinessProbe bool
	if err := readBoolInto(k, "feature_flags.mongo_readiness_probe", &mongoReadinessProbe); err != nil {
		return FeatureFlagsConfig{}, err
	}
	var redisReadinessProbe bool
	if err := readBoolInto(k, "feature_flags.redis_readiness_probe", &redisReadinessProbe); err != nil {
		return FeatureFlagsConfig{}, err
	}
	return FeatureFlagsConfig{
		PostgresReadinessProbe: postgresReadinessProbe,
		MongoReadinessProbe:    mongoReadinessProbe,
		RedisReadinessProbe:    redisReadinessProbe,
	}, nil
}

func readRawConfigString(k *koanf.Koanf, key string) string {
	return k.String(key)
}

func readTrimmedConfigString(k *koanf.Koanf, key string) string {
	return strings.TrimSpace(k.String(key))
}

func readDurationInto(k *koanf.Koanf, key string, dst *time.Duration) error {
	raw := readTrimmedConfigString(k, key)
	if raw == "" {
		return fmt.Errorf("%w: %s is empty", ErrParse, key)
	}
	d, err := parseDuration(raw)
	if err != nil {
		return fmt.Errorf("%w: %s has invalid duration: %w", ErrParse, key, err)
	}
	*dst = d
	return nil
}

func readIntInto(k *koanf.Koanf, key string, dst *int) error {
	value, err := parseInt(k.Get(key))
	if err != nil {
		return fmt.Errorf("%w: %s has invalid int value: %w", ErrParse, key, err)
	}
	*dst = value
	return nil
}

func readInt64Into(k *koanf.Koanf, key string, dst *int64) error {
	value, err := parseInt64(k.Get(key))
	if err != nil {
		return fmt.Errorf("%w: %s has invalid int64 value: %w", ErrParse, key, err)
	}
	*dst = value
	return nil
}

func readFloat64Into(k *koanf.Koanf, key string, dst *float64) error {
	value, err := parseFloat64(k.Get(key))
	if err != nil {
		return fmt.Errorf("%w: %s has invalid float value: %w", ErrParse, key, err)
	}
	*dst = value
	return nil
}

func readBoolInto(k *koanf.Koanf, key string, dst *bool) error {
	value, err := parseBool(k.Get(key))
	if err != nil {
		return fmt.Errorf("%w: %s has invalid bool value: %w", ErrParse, key, err)
	}
	*dst = value
	return nil
}

func readLogLevel(k *koanf.Koanf, key string) (slog.Level, error) {
	raw := readTrimmedConfigString(k, key)
	if raw == "" {
		return slog.LevelInfo, fmt.Errorf("%w: %s is empty", ErrParse, key)
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(raw)); err != nil {
		return slog.LevelInfo, fmt.Errorf("%w: %s has invalid log level", ErrParse, key)
	}
	return level, nil
}

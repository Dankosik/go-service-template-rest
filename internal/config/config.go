package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/knadh/koanf/v2"
)

var (
	ErrLoad             = errors.New("config load")
	ErrParse            = errors.New("config parse")
	ErrValidate         = errors.New("config validate")
	ErrStrictUnknownKey = errors.New("config strict unknown key")
	ErrSecretPolicy     = errors.New("config secret policy")
	ErrDependencyInit   = errors.New("dependency init")
)

const (
	StageLoadDefaults = "config.load.defaults"
	StageLoadFile     = "config.load.file"
	StageLoadEnv      = "config.load.env"
	StageParse        = "config.parse"
	StageValidate     = "config.validate"
)

type LoadOptions struct {
	ConfigPath     string
	ConfigOverlays []string
	Strict         bool
	LoadBudget     time.Duration
	ValidateBudget time.Duration
}

type LoadReport struct {
	LoadDuration         time.Duration
	LoadDefaultsDuration time.Duration
	LoadFileDuration     time.Duration
	LoadEnvDuration      time.Duration
	ParseDuration        time.Duration
	ValidateDuration     time.Duration
	UnknownKeyWarnings   []string
	FailedStage          string
	FailedStageDuration  time.Duration
}

func Load() (Config, error) {
	cfg, _, err := LoadDetailed(LoadOptions{})
	return cfg, err
}

func LoadWithOptions(opts LoadOptions) (Config, error) {
	cfg, _, err := LoadDetailed(opts)
	return cfg, err
}

func LoadDetailed(opts LoadOptions) (Config, LoadReport, error) {
	return LoadDetailedWithContext(context.Background(), opts)
}

func LoadDetailedWithContext(ctx context.Context, opts LoadOptions) (Config, LoadReport, error) {
	if err := checkContext(ctx); err != nil {
		return Config{}, LoadReport{}, err
	}

	loadCtx, loadCancel := withContextBudget(ctx, opts.LoadBudget)
	defer loadCancel()

	loadStarted := time.Now()
	k, metadata, err := loadKoanf(loadCtx, opts)
	report := LoadReport{
		LoadDuration:         time.Since(loadStarted),
		LoadDefaultsDuration: metadata.loadDefaultsDuration,
		LoadFileDuration:     metadata.loadFileDuration,
		LoadEnvDuration:      metadata.loadEnvDuration,
		FailedStage:          metadata.failedStage,
		FailedStageDuration:  metadata.failedStageDuration,
	}
	if err != nil {
		if strings.TrimSpace(report.FailedStage) == "" {
			report.FailedStage = StageLoadDefaults
		}
		if report.FailedStageDuration <= 0 {
			report.FailedStageDuration = report.LoadDuration
		}
		return Config{}, report, err
	}
	if err := checkContext(loadCtx); err != nil {
		return Config{}, report, err
	}

	parseStarted := time.Now()
	cfg, err := buildSnapshot(k)
	report.ParseDuration = time.Since(parseStarted)
	if err != nil {
		report.FailedStage = StageParse
		report.FailedStageDuration = report.ParseDuration
		return Config{}, report, err
	}
	if err := checkContext(loadCtx); err != nil {
		return Config{}, report, err
	}

	validateCtx, validateCancel := withContextBudget(ctx, opts.ValidateBudget)
	defer validateCancel()
	if err := checkContext(validateCtx); err != nil {
		report.FailedStage = StageValidate
		return Config{}, report, err
	}

	validateStarted := time.Now()
	validationResult, err := validateConfig(validateCtx, k, &cfg, ValidationOptions{
		Strict: opts.Strict,
	})
	report.ValidateDuration = time.Since(validateStarted)
	report.UnknownKeyWarnings = validationResult.UnknownKeyWarnings
	if err != nil {
		report.FailedStage = StageValidate
		report.FailedStageDuration = report.ValidateDuration
		return Config{}, report, err
	}

	return cfg, report, nil
}

func buildSnapshot(k *koanf.Koanf) (Config, error) {
	var cfg Config

	cfg.App.Env = readString(k, "app.env")
	cfg.App.Version = readString(k, "app.version")

	cfg.HTTP.Addr = readString(k, "http.addr")
	if err := readDurationInto(k, "http.shutdown_timeout", &cfg.HTTP.ShutdownTimeout); err != nil {
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

	cfg.Observability.OTel.ServiceName = readString(k, "observability.otel.service_name")
	cfg.Observability.OTel.TracesSampler = readString(k, "observability.otel.traces_sampler")
	if err := readFloat64Into(k, "observability.otel.traces_sampler_arg", &cfg.Observability.OTel.TracesSamplerArg); err != nil {
		return Config{}, err
	}
	cfg.Observability.OTel.Exporter.OTLPEndpoint = readString(k, "observability.otel.exporter.otlp_endpoint")
	cfg.Observability.OTel.Exporter.OTLPTracesEndpoint = readString(k, "observability.otel.exporter.otlp_traces_endpoint")
	cfg.Observability.OTel.Exporter.OTLPHeaders = readString(k, "observability.otel.exporter.otlp_headers")
	cfg.Observability.OTel.Exporter.OTLPProtocol = readString(k, "observability.otel.exporter.otlp_protocol")

	if err := readBoolInto(k, "observability.metrics.enabled", &cfg.Observability.Metrics.Enabled); err != nil {
		return Config{}, err
	}
	cfg.Observability.Metrics.Path = readString(k, "observability.metrics.path")
	if err := readBoolInto(k, "observability.grafana.enabled", &cfg.Observability.Grafana.Enabled); err != nil {
		return Config{}, err
	}
	cfg.Observability.Grafana.CloudOTLPEndpoint = readString(k, "observability.grafana.cloud_otlp_endpoint")

	if err := readBoolInto(k, "postgres.enabled", &cfg.Postgres.Enabled); err != nil {
		return Config{}, err
	}
	cfg.Postgres.DSN = readString(k, "postgres.dsn")
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
	cfg.Redis.Mode = strings.ToLower(readString(k, "redis.mode"))
	if err := readBoolInto(k, "redis.allow_store_mode", &cfg.Redis.AllowStoreMode); err != nil {
		return Config{}, err
	}
	cfg.Redis.Addr = readString(k, "redis.addr")
	cfg.Redis.Username = readString(k, "redis.username")
	cfg.Redis.Password = readString(k, "redis.password")
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
	cfg.Redis.KeyPrefix = readString(k, "redis.key_prefix")
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
	cfg.Mongo.URI = readString(k, "mongo.uri")
	cfg.Mongo.Database = readString(k, "mongo.database")
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

func readString(k *koanf.Koanf, key string) string {
	return strings.TrimSpace(k.String(key))
}

func readDurationInto(k *koanf.Koanf, key string, dst *time.Duration) error {
	raw := readString(k, key)
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
	raw := readString(k, key)
	if raw == "" {
		return slog.LevelInfo, fmt.Errorf("%w: %s is empty", ErrParse, key)
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(raw)); err != nil {
		return slog.LevelInfo, fmt.Errorf("%w: %s has invalid log level", ErrParse, key)
	}
	return level, nil
}

func parseInt(value any) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case uint:
		return int(v), nil
	case uint8:
		return int(v), nil
	case uint16:
		return int(v), nil
	case uint32:
		return int(v), nil
	case uint64:
		return int(v), nil
	case float32:
		if math.Trunc(float64(v)) != float64(v) {
			return 0, fmt.Errorf("non-integer numeric value")
		}
		return int(v), nil
	case float64:
		if math.Trunc(v) != v {
			return 0, fmt.Errorf("non-integer numeric value")
		}
		return int(v), nil
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, fmt.Errorf("invalid integer format")
		}
		return n, nil
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func parseInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case float32:
		if math.Trunc(float64(v)) != float64(v) {
			return 0, fmt.Errorf("non-integer numeric value")
		}
		return int64(v), nil
	case float64:
		if math.Trunc(v) != v {
			return 0, fmt.Errorf("non-integer numeric value")
		}
		return int64(v), nil
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid integer format")
		}
		return n, nil
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func parseFloat64(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid float format")
		}
		return n, nil
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func parseBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false, fmt.Errorf("invalid boolean format")
		}
		return b, nil
	default:
		return false, fmt.Errorf("unsupported type %T", value)
	}
}

func ErrorType(err error) string {
	switch {
	case errors.Is(err, ErrStrictUnknownKey):
		return "strict_unknown_key"
	case errors.Is(err, ErrSecretPolicy):
		return "secret_policy"
	case errors.Is(err, ErrValidate):
		return "validate"
	case errors.Is(err, ErrParse):
		return "parse"
	case errors.Is(err, ErrDependencyInit):
		return "dependency_init"
	case errors.Is(err, ErrLoad):
		return "load"
	default:
		return "load"
	}
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("%w: nil context", ErrLoad)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %w", ErrLoad, err)
	}
	return nil
}

func withContextBudget(parent context.Context, budget time.Duration) (context.Context, context.CancelFunc) {
	if budget <= 0 {
		return context.WithCancel(parent)
	}
	if deadline, ok := parent.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < budget {
			budget = remaining
		}
	}
	if budget <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, budget)
}

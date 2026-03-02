package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env      string
	HTTP     HTTPConfig
	Log      LogConfig
	OTel     OTelConfig
	Postgres PostgresConfig
}

type HTTPConfig struct {
	Addr              string
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	MaxBodyBytes      int64
}

type LogConfig struct {
	Level slog.Level
}

type OTelConfig struct {
	ServiceName      string
	TracesSampler    string
	TracesSamplerArg float64
}

type PostgresConfig struct {
	DSN string
}

func Load() (Config, error) {
	cfg := Config{
		Env: "local",
		HTTP: HTTPConfig{
			Addr:              ":8080",
			ShutdownTimeout:   10 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    1 << 20,
			MaxBodyBytes:      1 << 20,
		},
		Log: LogConfig{
			Level: slog.LevelInfo,
		},
		OTel: OTelConfig{
			ServiceName:      "service",
			TracesSampler:    "parentbased_traceidratio",
			TracesSamplerArg: 0.10,
		},
	}

	if v, ok := lookupNonEmpty("APP_ENV"); ok {
		cfg.Env = v
	}
	if v, ok := lookupNonEmpty("HTTP_ADDR"); ok {
		cfg.HTTP.Addr = v
	}
	if err := applyDuration("HTTP_SHUTDOWN_TIMEOUT", &cfg.HTTP.ShutdownTimeout); err != nil {
		return Config{}, err
	}
	if err := applyDuration("HTTP_READ_HEADER_TIMEOUT", &cfg.HTTP.ReadHeaderTimeout); err != nil {
		return Config{}, err
	}
	if err := applyDuration("HTTP_READ_TIMEOUT", &cfg.HTTP.ReadTimeout); err != nil {
		return Config{}, err
	}
	if err := applyDuration("HTTP_WRITE_TIMEOUT", &cfg.HTTP.WriteTimeout); err != nil {
		return Config{}, err
	}
	if err := applyDuration("HTTP_IDLE_TIMEOUT", &cfg.HTTP.IdleTimeout); err != nil {
		return Config{}, err
	}
	if err := applyPositiveInt("HTTP_MAX_HEADER_BYTES", &cfg.HTTP.MaxHeaderBytes); err != nil {
		return Config{}, err
	}
	if err := applyPositiveInt64("HTTP_MAX_BODY_BYTES", &cfg.HTTP.MaxBodyBytes); err != nil {
		return Config{}, err
	}
	if err := applyLogLevel("LOG_LEVEL", &cfg.Log.Level); err != nil {
		return Config{}, err
	}
	if v, ok := lookupNonEmpty("OTEL_SERVICE_NAME"); ok {
		cfg.OTel.ServiceName = v
	}
	if err := applyOTelSampler("OTEL_TRACES_SAMPLER", &cfg.OTel.TracesSampler); err != nil {
		return Config{}, err
	}
	if err := applyOTelSamplerArg("OTEL_TRACES_SAMPLER_ARG", &cfg.OTel.TracesSamplerArg); err != nil {
		return Config{}, err
	}
	if v, ok := lookupNonEmpty("POSTGRES_DSN"); ok {
		cfg.Postgres.DSN = v
	}

	if cfg.Env == "" {
		return Config{}, fmt.Errorf("APP_ENV cannot be empty")
	}
	if cfg.HTTP.Addr == "" {
		return Config{}, fmt.Errorf("HTTP_ADDR cannot be empty")
	}

	return cfg, nil
}

func lookupNonEmpty(key string) (string, bool) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}
	return v, true
}

func applyDuration(key string, dst *time.Duration) error {
	v, ok := lookupNonEmpty(key)
	if !ok {
		return nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fmt.Errorf("%s: invalid duration %q: %w", key, v, err)
	}
	*dst = d
	return nil
}

func applyPositiveInt(key string, dst *int) error {
	v, ok := lookupNonEmpty(key)
	if !ok {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("%s: invalid int %q: %w", key, v, err)
	}
	if n <= 0 {
		return fmt.Errorf("%s: value must be > 0", key)
	}
	*dst = n
	return nil
}

func applyPositiveInt64(key string, dst *int64) error {
	v, ok := lookupNonEmpty(key)
	if !ok {
		return nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fmt.Errorf("%s: invalid int %q: %w", key, v, err)
	}
	if n <= 0 {
		return fmt.Errorf("%s: value must be > 0", key)
	}
	*dst = n
	return nil
}

func applyLogLevel(key string, dst *slog.Level) error {
	v, ok := lookupNonEmpty(key)
	if !ok {
		return nil
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(v)); err != nil {
		return fmt.Errorf("%s: invalid log level %q", key, v)
	}
	*dst = level
	return nil
}

func applyOTelSampler(key string, dst *string) error {
	v, ok := lookupNonEmpty(key)
	if !ok {
		return nil
	}
	sampler := strings.ToLower(v)
	switch sampler {
	case "always_on", "always_off", "traceidratio", "parentbased_traceidratio":
		*dst = sampler
		return nil
	default:
		return fmt.Errorf("%s: unsupported sampler %q", key, v)
	}
}

func applyOTelSamplerArg(key string, dst *float64) error {
	v, ok := lookupNonEmpty(key)
	if !ok {
		return nil
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fmt.Errorf("%s: invalid float %q: %w", key, v, err)
	}
	if n < 0 || n > 1 {
		return fmt.Errorf("%s: value must be in range [0,1]", key)
	}
	*dst = n
	return nil
}

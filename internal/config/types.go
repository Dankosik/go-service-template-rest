package config

import (
	"log/slog"
	"time"
)

// Config is the immutable runtime snapshot built during startup.
type Config struct {
	App           AppConfig           `koanf:"app"`
	HTTP          HTTPConfig          `koanf:"http"`
	Log           LogConfig           `koanf:"log"`
	Observability ObservabilityConfig `koanf:"observability"`
	Postgres      PostgresConfig      `koanf:"postgres"`
	Redis         RedisConfig         `koanf:"redis"`
	Mongo         MongoConfig         `koanf:"mongo"`
	FeatureFlags  FeatureFlagsConfig  `koanf:"feature_flags"`
}

type AppConfig struct {
	Env     string `koanf:"env"`
	Version string `koanf:"version"`
}

type HTTPConfig struct {
	Addr              string        `koanf:"addr"`
	ShutdownTimeout   time.Duration `koanf:"shutdown_timeout"`
	ReadHeaderTimeout time.Duration `koanf:"read_header_timeout"`
	ReadTimeout       time.Duration `koanf:"read_timeout"`
	WriteTimeout      time.Duration `koanf:"write_timeout"`
	IdleTimeout       time.Duration `koanf:"idle_timeout"`
	MaxHeaderBytes    int           `koanf:"max_header_bytes"`
	MaxBodyBytes      int64         `koanf:"max_body_bytes"`
}

type LogConfig struct {
	Level slog.Level `koanf:"level"`
}

type ObservabilityConfig struct {
	OTel    OTelConfig    `koanf:"otel"`
	Metrics MetricsConfig `koanf:"metrics"`
	Grafana GrafanaConfig `koanf:"grafana"`
}

type OTelConfig struct {
	ServiceName      string             `koanf:"service_name"`
	TracesSampler    string             `koanf:"traces_sampler"`
	TracesSamplerArg float64            `koanf:"traces_sampler_arg"`
	Exporter         OTelExporterConfig `koanf:"exporter"`
}

type OTelExporterConfig struct {
	OTLPEndpoint       string `koanf:"otlp_endpoint"`
	OTLPTracesEndpoint string `koanf:"otlp_traces_endpoint"`
	OTLPHeaders        string `koanf:"otlp_headers"`
	OTLPProtocol       string `koanf:"otlp_protocol"`
}

type MetricsConfig struct {
	Enabled bool   `koanf:"enabled"`
	Path    string `koanf:"path"`
}

type GrafanaConfig struct {
	Enabled           bool   `koanf:"enabled"`
	CloudOTLPEndpoint string `koanf:"cloud_otlp_endpoint"`
}

type PostgresConfig struct {
	Enabled            bool          `koanf:"enabled"`
	DSN                string        `koanf:"dsn"`
	ConnectTimeout     time.Duration `koanf:"connect_timeout"`
	HealthcheckTimeout time.Duration `koanf:"healthcheck_timeout"`
	MaxOpenConns       int           `koanf:"max_open_conns"`
	MaxIdleConns       int           `koanf:"max_idle_conns"`
	ConnMaxLifetime    time.Duration `koanf:"conn_max_lifetime"`
}

type RedisConfig struct {
	Enabled                bool          `koanf:"enabled"`
	Mode                   string        `koanf:"mode"`
	AllowStoreMode         bool          `koanf:"allow_store_mode"`
	Addr                   string        `koanf:"addr"`
	Username               string        `koanf:"username"`
	Password               string        `koanf:"password"`
	DB                     int           `koanf:"db"`
	DialTimeout            time.Duration `koanf:"dial_timeout"`
	ReadTimeout            time.Duration `koanf:"read_timeout"`
	WriteTimeout           time.Duration `koanf:"write_timeout"`
	PoolSize               int           `koanf:"pool_size"`
	KeyPrefix              string        `koanf:"key_prefix"`
	FreshTTL               time.Duration `koanf:"fresh_ttl"`
	StaleWindow            time.Duration `koanf:"stale_window"`
	NegativeTTL            time.Duration `koanf:"negative_ttl"`
	TTLJitterPercent       int           `koanf:"ttl_jitter_percent"`
	EnableSingleflight     bool          `koanf:"enable_singleflight"`
	MaxFallbackConcurrency int           `koanf:"max_fallback_concurrency"`
}

type MongoConfig struct {
	Enabled                bool          `koanf:"enabled"`
	URI                    string        `koanf:"uri"`
	Database               string        `koanf:"database"`
	ConnectTimeout         time.Duration `koanf:"connect_timeout"`
	ServerSelectionTimeout time.Duration `koanf:"server_selection_timeout"`
	MaxPoolSize            int           `koanf:"max_pool_size"`
}

type FeatureFlagsConfig struct {
	PostgresReadinessProbe bool `koanf:"postgres_readiness_probe"`
	MongoReadinessProbe    bool `koanf:"mongo_readiness_probe"`
	RedisReadinessProbe    bool `koanf:"redis_readiness_probe"`
}

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
}

type AppConfig struct {
	Env     string `koanf:"env"`
	Version string `koanf:"version"`
}

type HTTPConfig struct {
	Addr                      string        `koanf:"addr"`
	ShutdownTimeout           time.Duration `koanf:"shutdown_timeout"`
	ReadinessTimeout          time.Duration `koanf:"readiness_timeout"`
	ReadinessPropagationDelay time.Duration `koanf:"readiness_propagation_delay"`
	ReadHeaderTimeout         time.Duration `koanf:"read_header_timeout"`
	ReadTimeout               time.Duration `koanf:"read_timeout"`
	WriteTimeout              time.Duration `koanf:"write_timeout"`
	IdleTimeout               time.Duration `koanf:"idle_timeout"`
	MaxHeaderBytes            int           `koanf:"max_header_bytes"`
	MaxBodyBytes              int64         `koanf:"max_body_bytes"`
}

type LogConfig struct {
	Level slog.Level `koanf:"level"`
}

type ObservabilityConfig struct {
	OTel OTelConfig `koanf:"otel"`
}

type OTelConfig struct {
	ServiceName      string             `koanf:"service_name"`
	TracesSampler    string             `koanf:"traces_sampler"`
	TracesSamplerArg float64            `koanf:"traces_sampler_arg"`
	Exporter         OTelExporterConfig `koanf:"exporter"`
}

type OTelExporterConfig struct {
	// OTLPEndpoint is the full OTLP HTTP traces endpoint URL (http or https).
	// A missing path defaults to /v1/traces.
	OTLPEndpoint string `koanf:"otlp_endpoint"`
	OTLPHeaders  string `koanf:"otlp_headers"`
}

type PostgresConfig struct {
	Enabled            bool          `koanf:"enabled"`
	DSN                string        `koanf:"dsn"`
	ConnectTimeout     time.Duration `koanf:"connect_timeout"`
	HealthcheckTimeout time.Duration `koanf:"healthcheck_timeout"`
	MaxOpenConns       int           `koanf:"max_open_conns"`
	ConnMaxLifetime    time.Duration `koanf:"conn_max_lifetime"`
}

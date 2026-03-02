package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_VERSION", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "")
	t.Setenv("HTTP_READ_HEADER_TIMEOUT", "")
	t.Setenv("HTTP_READ_TIMEOUT", "")
	t.Setenv("HTTP_WRITE_TIMEOUT", "")
	t.Setenv("HTTP_IDLE_TIMEOUT", "")
	t.Setenv("HTTP_MAX_HEADER_BYTES", "")
	t.Setenv("HTTP_MAX_BODY_BYTES", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("OTEL_SERVICE_NAME", "")
	t.Setenv("OTEL_TRACES_SAMPLER", "")
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "")
	t.Setenv("POSTGRES_DSN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Env != "local" {
		t.Fatalf("Env = %q, want local", cfg.Env)
	}
	if cfg.Version != "dev" {
		t.Fatalf("Version = %q, want dev", cfg.Version)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("Addr = %q, want :8080", cfg.HTTP.Addr)
	}
	if cfg.HTTP.MaxHeaderBytes != 16<<10 {
		t.Fatalf("MaxHeaderBytes = %d, want %d", cfg.HTTP.MaxHeaderBytes, 16<<10)
	}
	if cfg.HTTP.ReadTimeout != 5*time.Second {
		t.Fatalf("ReadTimeout = %s, want 5s", cfg.HTTP.ReadTimeout)
	}
	if cfg.HTTP.WriteTimeout != 10*time.Second {
		t.Fatalf("WriteTimeout = %s, want 10s", cfg.HTTP.WriteTimeout)
	}
	if cfg.HTTP.MaxBodyBytes != 1<<20 {
		t.Fatalf("MaxBodyBytes = %d, want %d", cfg.HTTP.MaxBodyBytes, 1<<20)
	}
	if cfg.OTel.ServiceName != "service" {
		t.Fatalf("OTel.ServiceName = %q, want service", cfg.OTel.ServiceName)
	}
	if cfg.OTel.TracesSampler != "parentbased_traceidratio" {
		t.Fatalf("OTel.TracesSampler = %q, want parentbased_traceidratio", cfg.OTel.TracesSampler)
	}
	if cfg.OTel.TracesSamplerArg != 0.10 {
		t.Fatalf("OTel.TracesSamplerArg = %v, want 0.10", cfg.OTel.TracesSamplerArg)
	}
	if cfg.Postgres.DSN != "" {
		t.Fatalf("Postgres.DSN = %q, want empty by default", cfg.Postgres.DSN)
	}
}

func TestLoadInvalidDuration(t *testing.T) {
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "oops")

	_, err := Load()
	if err == nil {
		t.Fatalf("Load() expected error for invalid duration")
	}
}

func TestLoadInvalidLogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatalf("Load() expected error for invalid log level")
	}
}

func TestLoadInvalidOTelSampler(t *testing.T) {
	t.Setenv("OTEL_TRACES_SAMPLER", "unsupported")

	_, err := Load()
	if err == nil {
		t.Fatalf("Load() expected error for invalid OTel sampler")
	}
}

func TestLoadInvalidOTelSamplerArg(t *testing.T) {
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "2.0")

	_, err := Load()
	if err == nil {
		t.Fatalf("Load() expected error for invalid OTel sampler arg")
	}
}

func TestLoadHTTPTimeoutOverrides(t *testing.T) {
	t.Setenv("HTTP_READ_TIMEOUT", "12s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "18s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.ReadTimeout != 12*time.Second {
		t.Fatalf("ReadTimeout = %s, want 12s", cfg.HTTP.ReadTimeout)
	}
	if cfg.HTTP.WriteTimeout != 18*time.Second {
		t.Fatalf("WriteTimeout = %s, want 18s", cfg.HTTP.WriteTimeout)
	}
}

func TestLoadHTTPBodyLimitOverride(t *testing.T) {
	t.Setenv("HTTP_MAX_BODY_BYTES", "2048")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.MaxBodyBytes != 2048 {
		t.Fatalf("MaxBodyBytes = %d, want 2048", cfg.HTTP.MaxBodyBytes)
	}
}

func TestLoadHTTPHeaderLimitOverride(t *testing.T) {
	t.Setenv("HTTP_MAX_HEADER_BYTES", "32768")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.MaxHeaderBytes != 32768 {
		t.Fatalf("MaxHeaderBytes = %d, want 32768", cfg.HTTP.MaxHeaderBytes)
	}
}

func TestLoadVersionOverride(t *testing.T) {
	t.Setenv("APP_VERSION", "1.2.3")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Version != "1.2.3" {
		t.Fatalf("Version = %q, want %q", cfg.Version, "1.2.3")
	}
}

func TestLoadPostgresDSNOverride(t *testing.T) {
	want := "postgres://app:app@localhost:5432/app?sslmode=disable"
	t.Setenv("POSTGRES_DSN", want)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Postgres.DSN != want {
		t.Fatalf("Postgres.DSN = %q, want %q", cfg.Postgres.DSN, want)
	}
}

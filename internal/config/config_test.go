package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("POSTGRES_DSN", "")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "")
	t.Setenv("HTTP_READ_HEADER_TIMEOUT", "")
	t.Setenv("HTTP_READ_TIMEOUT", "")
	t.Setenv("HTTP_WRITE_TIMEOUT", "")
	t.Setenv("HTTP_IDLE_TIMEOUT", "")
	t.Setenv("HTTP_MAX_HEADER_BYTES", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != "local" {
		t.Fatalf("Env = %q, want local", cfg.Env)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("Addr = %q, want :8080", cfg.HTTP.Addr)
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

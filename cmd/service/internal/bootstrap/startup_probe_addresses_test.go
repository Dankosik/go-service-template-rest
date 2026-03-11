package bootstrap

import (
	"errors"
	"strings"
	"testing"

	"github.com/Dankosik/search-service/internal/config"
)

func TestStartupProbeAddresses(t *testing.T) {
	t.Parallel()

	t.Run("postgres invalid dsn", func(t *testing.T) {
		_, err := postgresStartupProbeAddress(config.PostgresConfig{DSN: "::bad::dsn"})
		if err == nil {
			t.Fatal("postgresStartupProbeAddress() error = nil, want non-nil")
		}
		if !errors.Is(err, config.ErrDependencyInit) {
			t.Fatalf("err = %v, want wrapped %v", err, config.ErrDependencyInit)
		}
	})

	t.Run("postgres valid dsn", func(t *testing.T) {
		address, err := postgresStartupProbeAddress(config.PostgresConfig{DSN: "postgres://user:pass@localhost:5432/app?sslmode=disable"})
		if err != nil {
			t.Fatalf("postgresStartupProbeAddress() error = %v, want nil", err)
		}
		if address != "localhost:5432" {
			t.Fatalf("address = %q, want %q", address, "localhost:5432")
		}
	})

	t.Run("redis empty", func(t *testing.T) {
		_, err := redisStartupProbeAddress(config.RedisConfig{Addr: "   "})
		if err == nil {
			t.Fatal("redisStartupProbeAddress() error = nil, want non-nil")
		}
	})

	t.Run("redis trimmed", func(t *testing.T) {
		address, err := redisStartupProbeAddress(config.RedisConfig{Addr: " 127.0.0.1:6379 "})
		if err != nil {
			t.Fatalf("redisStartupProbeAddress() error = %v, want nil", err)
		}
		if address != "127.0.0.1:6379" {
			t.Fatalf("address = %q, want %q", address, "127.0.0.1:6379")
		}
	})

	t.Run("mongo invalid", func(t *testing.T) {
		_, err := mongoStartupProbeAddress(config.MongoConfig{URI: "::bad-uri"})
		if err == nil {
			t.Fatal("mongoStartupProbeAddress() error = nil, want non-nil")
		}
		if !errors.Is(err, config.ErrDependencyInit) {
			t.Fatalf("err = %v, want wrapped %v", err, config.ErrDependencyInit)
		}
	})

	t.Run("mongo valid", func(t *testing.T) {
		address, err := mongoStartupProbeAddress(config.MongoConfig{URI: "mongodb://localhost:27017/app"})
		if err != nil {
			t.Fatalf("mongoStartupProbeAddress() error = %v, want nil", err)
		}
		if !strings.Contains(address, ":") {
			t.Fatalf("address = %q, want host:port", address)
		}
	})
}

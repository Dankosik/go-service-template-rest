package bootstrap

import (
	"errors"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
)

func TestStartupProbeAddresses(t *testing.T) {
	t.Parallel()

	t.Run("postgres invalid dsn", func(t *testing.T) {
		t.Parallel()

		rawDSN := "postgres://user:top-secret%@localhost:5432/app"
		_, err := postgresStartupProbeAddress(config.PostgresConfig{DSN: rawDSN})
		if err == nil {
			t.Fatal("postgresStartupProbeAddress() error = nil, want non-nil")
		}
		if !errors.Is(err, errDependencyInit) {
			t.Fatalf("err = %v, want wrapped %v", err, errDependencyInit)
		}
		if !errors.Is(err, postgres.ErrConfig) {
			t.Fatalf("err = %v, want wrapped postgres ErrConfig", err)
		}
		if !strings.Contains(err.Error(), "parse postgres dsn") || !strings.Contains(err.Error(), "redacted") {
			t.Fatalf("err = %v, want redacted parse context", err)
		}
		for _, leaked := range []string{rawDSN, "top-secret", "user"} {
			if strings.Contains(err.Error(), leaked) {
				t.Fatalf("err = %v, leaked %q", err, leaked)
			}
		}
	})

	t.Run("postgres valid dsn", func(t *testing.T) {
		t.Parallel()

		address, err := postgresStartupProbeAddress(config.PostgresConfig{DSN: "postgres://user:pass@localhost:5432/app?sslmode=disable"})
		if err != nil {
			t.Fatalf("postgresStartupProbeAddress() error = %v, want nil", err)
		}
		if address != "localhost:5432" {
			t.Fatalf("address = %q, want %q", address, "localhost:5432")
		}
	})

	t.Run("postgres fallback dsn fails before admission", func(t *testing.T) {
		t.Parallel()

		_, err := postgresStartupProbeAddress(config.PostgresConfig{
			DSN: "postgres://user:pass@localhost:5432,api.example.com:5432/app?sslmode=disable",
		})
		if err == nil {
			t.Fatal("postgresStartupProbeAddress() error = nil, want non-nil")
		}
		if !errors.Is(err, errDependencyInit) {
			t.Fatalf("err = %v, want wrapped %v", err, errDependencyInit)
		}
		if !errors.Is(err, postgres.ErrConfig) {
			t.Fatalf("err = %v, want wrapped postgres ErrConfig", err)
		}
		if !strings.Contains(err.Error(), "postgres dsn fallback targets are not supported") {
			t.Fatalf("err = %v, want fallback rejection", err)
		}
	})
}

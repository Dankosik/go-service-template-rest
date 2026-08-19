package postgres

import (
	"context"
	"errors"
	"math"
	"os"
	"strings"
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	restore := clearPostgresEnvForTests()
	goleak.VerifyTestMain(m, goleak.Cleanup(func(exitCode int) {
		restore()
		os.Exit(exitCode)
	}))
}

func clearPostgresEnvForTests() func() {
	type envState struct {
		name  string
		value string
	}

	var states []envState
	for _, name := range ambientPostgresEnvNames {
		value, ok := os.LookupEnv(name)
		if !ok {
			continue
		}
		states = append(states, envState{name: name, value: value})
		_ = os.Unsetenv(name)
	}

	return func() {
		for _, state := range states {
			_ = os.Setenv(state.name, state.value)
		}
	}
}

func TestOpenRejectsEmptyDSN(t *testing.T) {
	t.Parallel()

	_, err := Open(context.Background(), Options{DSN: "   \n\t", MaxOpenConns: 10})
	requirePostgresConfigError(t, err, "postgres dsn is empty")
}

func TestOpenRejectsInvalidPoolSize(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name         string
		maxOpenConns int
	}{
		{name: "zero", maxOpenConns: 0},
		{name: "too large", maxOpenConns: math.MaxInt},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Open(context.Background(), Options{
				DSN:          "postgres://user:pass@localhost:5432/db?sslmode=disable",
				MaxOpenConns: tc.maxOpenConns,
			})
			if !errors.Is(err, ErrConfig) {
				t.Fatalf("Open() error = %v, want ErrConfig", err)
			}
		})
	}
}

func TestOpenInvalidDSNIsRedacted(t *testing.T) {
	t.Parallel()

	rawDSN := "postgres://user:top-secret%@localhost:5432/app"
	_, err := Open(context.Background(), Options{DSN: rawDSN, MaxOpenConns: 10})
	if !errors.Is(err, ErrConfig) {
		t.Fatalf("Open() error = %v, want ErrConfig", err)
	}
	if !strings.Contains(err.Error(), "parse postgres dsn") || !strings.Contains(err.Error(), "redacted") {
		t.Fatalf("Open() error = %v, want redacted parse context", err)
	}
	requireErrorDoesNotContain(t, err, rawDSN, "top-secret", "user")
}

func TestOpenReportsUnavailablePostgresHealthcheck(t *testing.T) {
	t.Parallel()

	pool, err := Open(t.Context(), Options{
		DSN:          "postgres://app:app@127.0.0.1:1/app?sslmode=disable",
		MaxOpenConns: 1,
	})
	if pool != nil || !errors.Is(err, ErrHealthcheck) {
		t.Fatalf("Open() = (%v, %v), want unavailable postgres healthcheck", pool, err)
	}
}

func requirePostgresConfigError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want %q", want)
	}
	if !errors.Is(err, ErrConfig) {
		t.Fatalf("error = %v, want ErrConfig", err)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want to contain %q", err, want)
	}
}

func requireErrorDoesNotContain(t *testing.T, err error, forbidden ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("error = nil, want non-nil")
	}
	for _, value := range forbidden {
		if value != "" && strings.Contains(err.Error(), value) {
			t.Fatalf("error = %v, leaked %q", err, value)
		}
	}
}

func TestPostgresOperationNameSkipsSQLCComment(t *testing.T) {
	t.Parallel()

	statement := `-- name: CurrentTransactionID :one
SELECT pg_current_xact_id()::text AS transaction_id
`
	if got := postgresOperationName(statement); got != "SELECT" {
		t.Fatalf("postgresOperationName() = %q, want SELECT", got)
	}
}

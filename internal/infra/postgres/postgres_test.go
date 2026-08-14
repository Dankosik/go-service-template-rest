package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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

func TestNewRejectsEmptyDSN(t *testing.T) {
	t.Parallel()

	_, err := New(context.Background(), Options{
		DSN:                "   \n\t",
		ConnectTimeout:     time.Second,
		HealthcheckTimeout: time.Second,
		MaxOpenConns:       10,
		AcquireTimeout:     time.Second,
		ConnMaxLifetime:    time.Minute,
		StatementTimeout:   time.Second,
	})
	if err == nil {
		t.Fatal("New() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "postgres dsn is empty") {
		t.Fatalf("New() error = %q, want to contain %q", err.Error(), "postgres dsn is empty")
	}
	if !errors.Is(err, ErrConfig) {
		t.Fatalf("New() error = %v, want ErrConfig", err)
	}
}

func TestNewRejectsInvalidOptions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		opts Options
	}{
		{
			name: "connect timeout",
			opts: Options{
				DSN:                "postgres://user:pass@localhost:5432/db?sslmode=disable",
				HealthcheckTimeout: time.Second,
				MaxOpenConns:       10,
				AcquireTimeout:     time.Second,
				ConnMaxLifetime:    time.Minute,
				StatementTimeout:   time.Second,
			},
		},
		{
			name: "healthcheck timeout",
			opts: Options{
				DSN:              "postgres://user:pass@localhost:5432/db?sslmode=disable",
				ConnectTimeout:   time.Second,
				MaxOpenConns:     10,
				AcquireTimeout:   time.Second,
				ConnMaxLifetime:  time.Minute,
				StatementTimeout: time.Second,
			},
		},
		{
			name: "max open conns",
			opts: Options{
				DSN:                "postgres://user:pass@localhost:5432/db?sslmode=disable",
				ConnectTimeout:     time.Second,
				HealthcheckTimeout: time.Second,
				AcquireTimeout:     time.Second,
				ConnMaxLifetime:    time.Minute,
				StatementTimeout:   time.Second,
			},
		},
		{
			name: "conn max lifetime",
			opts: Options{
				DSN:                "postgres://user:pass@localhost:5432/db?sslmode=disable",
				ConnectTimeout:     time.Second,
				HealthcheckTimeout: time.Second,
				MaxOpenConns:       10,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := New(context.Background(), tc.opts)
			if err == nil {
				t.Fatal("New() error = nil, want non-nil")
			}
			if !errors.Is(err, ErrConfig) {
				t.Fatalf("New() error = %v, want ErrConfig", err)
			}
		})
	}
}

func TestNewInvalidDSNIsRedacted(t *testing.T) {
	t.Parallel()

	rawDSN := "postgres://user:top-secret%@localhost:5432/app"
	_, err := New(context.Background(), Options{
		DSN:                rawDSN,
		ConnectTimeout:     time.Second,
		HealthcheckTimeout: time.Second,
		MaxOpenConns:       10,
		AcquireTimeout:     time.Second,
		ConnMaxLifetime:    time.Minute,
		StatementTimeout:   time.Second,
	})
	if err == nil {
		t.Fatal("New() error = nil, want non-nil")
	}
	if !errors.Is(err, ErrConfig) {
		t.Fatalf("New() error = %v, want ErrConfig", err)
	}
	if !strings.Contains(err.Error(), "parse postgres dsn") || !strings.Contains(err.Error(), "redacted") {
		t.Fatalf("New() error = %v, want redacted parse context", err)
	}
	for _, leaked := range []string{rawDSN, "top-secret", "user"} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("New() error = %v, leaked %q", err, leaked)
		}
	}
}

func TestNewReportsUnavailablePostgresHealthcheck(t *testing.T) {
	t.Parallel()

	pool, err := New(t.Context(), Options{
		DSN:                "postgres://app:app@127.0.0.1:1/app?sslmode=disable",
		ConnectTimeout:     100 * time.Millisecond,
		HealthcheckTimeout: 100 * time.Millisecond,
		MaxOpenConns:       1,
		AcquireTimeout:     time.Second,
		ConnMaxLifetime:    time.Minute,
		StatementTimeout:   time.Second,
	})
	if pool != nil || !errors.Is(err, ErrHealthcheck) {
		t.Fatalf("New() = (%v, %v), want unavailable postgres healthcheck", pool, err)
	}
}

func TestPoolAcquireAndCheckClassifyConnectionFailure(t *testing.T) {
	t.Parallel()

	config, err := parsePoolConfig("postgres://app:app@127.0.0.1:1/app?sslmode=disable")
	if err != nil {
		t.Fatalf("parsePoolConfig() error = %v", err)
	}
	config.ConnConfig.ConnectTimeout = 100 * time.Millisecond
	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatalf("pgxpool.NewWithConfig() error = %v", err)
	}
	p := &Pool{pool: pool, acquireTimeout: 100 * time.Millisecond}
	t.Cleanup(p.Close)

	if _, err := p.Acquire(t.Context()); !errors.Is(err, ErrConnect) {
		t.Fatalf("Acquire() error = %v, want ErrConnect", err)
	}
	if err := p.Check(t.Context()); !errors.Is(err, ErrHealthcheck) || !errors.Is(err, ErrConnect) {
		t.Fatalf("Check() error = %v, want ErrHealthcheck wrapping ErrConnect", err)
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
		if value == "" {
			continue
		}
		if strings.Contains(err.Error(), value) {
			t.Fatalf("error = %v, leaked %q", err, value)
		}
	}
}

func TestPoolHelpersWithoutConnection(t *testing.T) {
	t.Parallel()

	var nilPool *Pool
	nilPool.Close()

	if err := nilPool.Check(context.Background()); err == nil {
		t.Fatal("(*Pool)(nil).Check() error = nil, want non-nil")
	} else if !errors.Is(err, ErrHealthcheck) {
		t.Fatalf("(*Pool)(nil).Check() error = %v, want ErrHealthcheck", err)
	}

	pool := &Pool{}
	if got := pool.Name(); got != "postgres" {
		t.Fatalf("Name() = %q, want %q", got, "postgres")
	}

	pool.Close()
	if err := pool.Check(context.Background()); err == nil {
		t.Fatal("Check() error = nil, want non-nil for nil internal pool")
	} else if !errors.Is(err, ErrHealthcheck) {
		t.Fatalf("Check() error = %v, want ErrHealthcheck", err)
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

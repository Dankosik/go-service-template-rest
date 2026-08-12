//go:build integration

package integration_test

import (
	"context"
	"net"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
)

func newPostgresJobsFixture(t *testing.T) (context.Context, *postgres.Pool, *postgresjobs.Store) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     3 * time.Second,
		HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns:       8,
		AcquireTimeout:     time.Second,
		ConnMaxLifetime:    time.Hour,
		StatementTimeout:   5 * time.Second,
	})
	if err != nil {
		t.Fatalf("postgres.New(): %v", err)
	}
	t.Cleanup(pool.Close)
	store, err := postgresjobs.NewStore(pool, postgresjobs.StoreOptions{
		OperationTimeout: 3 * time.Second,
		StatementTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("postgresjobs.NewStore(): %v", err)
	}
	return ctx, pool, store
}

func newPostgresJobsStore(
	ctx context.Context,
	t *testing.T,
	dsn string,
	maxOpenConns int,
	acquireTimeout time.Duration,
) (*postgres.Pool, *postgresjobs.Store) {
	t.Helper()
	pool, err := postgres.New(ctx, postgres.Options{
		DSN: dsn, ConnectTimeout: time.Second, HealthcheckTimeout: time.Second,
		MaxOpenConns: maxOpenConns, AcquireTimeout: acquireTimeout,
		ConnMaxLifetime: time.Hour, StatementTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("postgres.New(): %v", err)
	}
	store, err := postgresjobs.NewStore(pool, postgresjobs.StoreOptions{
		OperationTimeout: time.Second,
		StatementTimeout: time.Second,
	})
	if err != nil {
		pool.Close()
		t.Fatalf("postgresjobs.NewStore(): %v", err)
	}
	return pool, store
}

func postgresJobsDSNParam(t *testing.T, dsn, key, value string) string {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL DSN: %v", err)
	}
	query := parsed.Query()
	query.Set(key, value)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func postgresJobsDSN(pool *postgres.Pool) string {
	config := pool.PGX().Config().ConnConfig
	query := url.Values{"sslmode": []string{"disable"}}
	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(config.User, config.Password),
		Host:     net.JoinHostPort(config.Host, strconv.Itoa(int(config.Port))),
		Path:     "/" + config.Database,
		RawQuery: query.Encode(),
	}).String()
}

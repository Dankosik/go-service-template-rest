//go:build integration

package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newOutboxFixture(t *testing.T) (context.Context, *pgxpool.Pool, *postgresoutbox.Appender) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	pool, err := postgres.Open(ctx, postgres.Options{DSN: dsn, MaxOpenConns: 32})
	if err != nil {
		t.Fatalf("postgres.Open(): %v", err)
	}
	t.Cleanup(pool.Close)
	appender, err := natsjs.NewOutboxAppender(testMaxPayloadBytes, natsjs.Route{
		Type: "example.changed", Version: 1, Subject: sourceSubject,
	})
	if err != nil {
		t.Fatalf("natsjs.NewOutboxAppender(): %v", err)
	}
	return ctx, pool, appender
}

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
)

func newOutboxFixture(t *testing.T) (context.Context, *postgres.Pool, *postgresoutbox.Appender) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     3 * time.Second,
		HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns:       32,
		AcquireTimeout:     time.Second,
		ConnMaxLifetime:    time.Hour,
		StatementTimeout:   10 * time.Second,
	})
	if err != nil {
		t.Fatalf("postgres.New(): %v", err)
	}
	t.Cleanup(pool.Close)
	appender, err := natsjs.NewOutboxAppender(testMaxPayloadBytes, natsjs.OutboxRoute{
		Type: "example.changed", Version: 1, Subject: sourceSubject,
	})
	if err != nil {
		t.Fatalf("natsjs.NewOutboxAppender(): %v", err)
	}
	return ctx, pool, appender
}

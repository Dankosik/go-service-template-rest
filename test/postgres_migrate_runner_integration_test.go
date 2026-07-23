//go:build integration

package integration_test

import (
	"context"
	"testing"
	"testing/fstest"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresMigrateUpAppliesAndReplaysMigrations(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	dsn := integrationPostgresDSN(t)
	migrationFS := fstest.MapFS{
		"migrations/000001_integration.up.sql":   {Data: []byte("select 1;")},
		"migrations/000001_integration.down.sql": {Data: []byte("select 1;")},
	}

	firstRunChanged, err := postgres.MigrateUp(ctx, postgres.MigrationOptions{
		DSN:        dsn,
		SourceFS:   migrationFS,
		SourcePath: "migrations",
	})
	if err != nil {
		t.Fatalf("MigrateUp(first) error: %v", err)
	}
	if !firstRunChanged {
		t.Fatal("MigrateUp(first) reported no change, want applied migrations")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("create verification pool: %v", err)
	}
	defer pool.Close()

	var version int
	var dirty bool
	if err := pool.QueryRow(ctx, "select version, dirty from schema_migrations").Scan(&version, &dirty); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if version != 1 || dirty {
		t.Fatalf("schema_migrations = version %d dirty %t, want version 1 dirty false", version, dirty)
	}

	secondRunChanged, err := postgres.MigrateUp(ctx, postgres.MigrationOptions{
		DSN:        dsn,
		SourceFS:   migrationFS,
		SourcePath: "migrations",
	})
	if err != nil {
		t.Fatalf("MigrateUp(second) error: %v", err)
	}
	if secondRunChanged {
		t.Fatal("MigrateUp(second) reported schema change, want no change")
	}

	if err := postgres.ValidateMigrations(ctx, postgres.MigrationOptions{
		DSN:        dsn,
		SourceFS:   migrationFS,
		SourcePath: "migrations",
	}); err != nil {
		t.Fatalf("ValidateMigrations() error: %v", err)
	}

	if err := pool.QueryRow(ctx, "select version, dirty from schema_migrations").Scan(&version, &dirty); err != nil {
		t.Fatalf("query schema_migrations after validation: %v", err)
	}
	if version != 1 || dirty {
		t.Fatalf("schema_migrations after validation = version %d dirty %t, want version 1 dirty false", version, dirty)
	}
}

//go:build integration

package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresMigrateUpAppliesAndReplaysMigrations(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	dsn := postgresTestDSN(t, ctx)

	firstRun, err := postgres.MigrateUp(ctx, postgres.MigrationOptions{
		DSN:        dsn,
		SourceFS:   os.DirFS(".."),
		SourcePath: "env/migrations",
	})
	if err != nil {
		t.Fatalf("MigrateUp(first) error: %v", err)
	}
	if !firstRun.Changed {
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

	secondRun, err := postgres.MigrateUp(ctx, postgres.MigrationOptions{
		DSN:        dsn,
		SourceFS:   os.DirFS(".."),
		SourcePath: "env/migrations",
	})
	if err != nil {
		t.Fatalf("MigrateUp(second) error: %v", err)
	}
	if secondRun.Changed {
		t.Fatal("MigrateUp(second) reported schema change, want no change")
	}

	if err := postgres.ValidateMigrations(ctx, postgres.MigrationOptions{
		DSN:        dsn,
		SourceFS:   os.DirFS(".."),
		SourcePath: "env/migrations",
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

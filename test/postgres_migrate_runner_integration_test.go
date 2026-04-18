//go:build integration

package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
)

func TestPostgresMigrateUpAppliesAndReplaysMigrations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := runPostgresContainer(ctx)
	if err != nil {
		if isDockerUnavailable(err) {
			if requireDockerForIntegration() {
				t.Fatalf("docker is required for integration tests: %v", err)
			}
			t.Skipf("docker is unavailable: %v", err)
		}
		t.Fatalf("start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if termErr := testcontainers.TerminateContainer(container); termErr != nil {
			t.Errorf("terminate postgres container: %v", termErr)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("build postgres dsn: %v", err)
	}

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

	var tableName string
	if err := pool.QueryRow(ctx, "select coalesce(to_regclass('public.ping_history')::text, '')").Scan(&tableName); err != nil {
		t.Fatalf("query ping_history table: %v", err)
	}
	if tableName != "ping_history" {
		t.Fatalf("ping_history table = %q, want ping_history", tableName)
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
}

//go:build integration

package integration_test

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresmigrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresMigrateUpAppliesAndReplaysMigrations(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	dsn := pgtest.DSN(t)
	migrationFS := fstest.MapFS{
		"migrations/000001_integration.up.sql":   {Data: []byte("select 1;")},
		"migrations/000001_integration.down.sql": {Data: []byte("select 1;")},
		"migrations/000002_integration.up.sql":   {Data: []byte("select 1;")},
		"migrations/000002_integration.down.sql": {Data: []byte("select 1;")},
	}

	firstRunChanged, err := postgresmigrate.MigrateUp(ctx, postgresmigrate.MigrationOptions{
		DSN:              dsn,
		SourceFS:         migrationFS,
		SourcePath:       "migrations",
		StatementTimeout: time.Minute,
		LockTimeout:      15 * time.Second,
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
	if version != 2 || dirty {
		t.Fatalf("schema_migrations = version %d dirty %t, want version 2 dirty false", version, dirty)
	}

	secondRunChanged, err := postgresmigrate.MigrateUp(ctx, postgresmigrate.MigrationOptions{
		DSN:              dsn,
		SourceFS:         migrationFS,
		SourcePath:       "migrations",
		StatementTimeout: time.Minute,
		LockTimeout:      15 * time.Second,
	})
	if err != nil {
		t.Fatalf("MigrateUp(second) error: %v", err)
	}
	if secondRunChanged {
		t.Fatal("MigrateUp(second) reported schema change, want no change")
	}

	// The up/down/up rehearsal that used to be a `validate` subcommand of the
	// production migrate binary. It belongs here: a throwaway database proves the
	// same property without shipping a schema-dropping command in the image that
	// runs migrations against production.
	options := postgresmigrate.MigrationOptions{
		DSN:              dsn,
		SourceFS:         migrationFS,
		SourcePath:       "migrations",
		StatementTimeout: time.Minute,
		LockTimeout:      15 * time.Second,
	}
	if err := postgresmigrate.MigrateDown(ctx, options); err != nil {
		t.Fatalf("MigrateDown() error: %v", err)
	}
	if err := pool.QueryRow(ctx, "select count(*) from schema_migrations").Scan(&version); err != nil {
		t.Fatalf("query schema_migrations after rollback: %v", err)
	}
	if version != 0 {
		t.Fatalf("schema_migrations after full rollback holds %d rows, want 0", version)
	}

	reappliedChanged, err := postgresmigrate.MigrateUp(ctx, options)
	if err != nil {
		t.Fatalf("MigrateUp(reapply) error: %v", err)
	}
	if !reappliedChanged {
		t.Fatal("MigrateUp(reapply) reported no change, want the migrations applied again")
	}
	if err := pool.QueryRow(ctx, "select version, dirty from schema_migrations").Scan(&version, &dirty); err != nil {
		t.Fatalf("query schema_migrations after reapply: %v", err)
	}
	if version != 2 || dirty {
		t.Fatalf("schema_migrations after reapply = version %d dirty %t, want version 2 dirty false", version, dirty)
	}
}

// TestMigrateDownExercisesEveryDownMigration is what the rehearsal was for: a down
// migration that is never run is a rollback plan nobody has tested.
func TestMigrateDownExercisesEveryDownMigration(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	dsn := pgtest.DSN(t)
	migrationFS := fstest.MapFS{
		"migrations/000001_integration.up.sql": {
			Data: []byte("select 1;"),
		},
		"migrations/000001_integration.down.sql": {
			Data: []byte("select * from migration_down_must_reach_missing_relation;"),
		},
		"migrations/000002_integration.up.sql": {
			Data: []byte("select 1;"),
		},
		"migrations/000002_integration.down.sql": {
			Data: []byte("select 1;"),
		},
	}

	options := postgresmigrate.MigrationOptions{
		DSN:              dsn,
		SourceFS:         migrationFS,
		SourcePath:       "migrations",
		StatementTimeout: time.Minute,
		LockTimeout:      15 * time.Second,
	}
	if _, err := postgresmigrate.MigrateUp(ctx, options); err != nil {
		t.Fatalf("MigrateUp() error: %v", err)
	}

	err := postgresmigrate.MigrateDown(ctx, options)
	if err == nil {
		t.Fatal("MigrateDown() error = nil, want the earlier down migration to fail")
	}
	if !strings.Contains(err.Error(), "roll back all postgres migrations") {
		t.Fatalf("MigrateDown() error = %q, want full rollback context", err.Error())
	}
}

func TestPostgresMigrateBoundsLongStatement(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	dsn := pgtest.DSN(t)
	migrationFS := fstest.MapFS{
		"migrations/000001_slow.up.sql":   {Data: []byte("select pg_sleep(5);")},
		"migrations/000001_slow.down.sql": {Data: []byte("select 1;")},
	}

	started := time.Now()
	_, err := postgresmigrate.MigrateUp(ctx, postgresmigrate.MigrationOptions{
		DSN:              dsn,
		SourceFS:         migrationFS,
		SourcePath:       "migrations",
		StatementTimeout: 100 * time.Millisecond,
		LockTimeout:      time.Second,
	})
	elapsed := time.Since(started)
	if err == nil {
		t.Fatal("MigrateUp() error = nil, want statement timeout")
	}
	if elapsed >= 3*time.Second {
		t.Fatalf("MigrateUp() elapsed = %s, want bounded well below 5s statement", elapsed)
	}
}

func TestPostgresMigrateReportsDirtyVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	dsn := pgtest.DSN(t)
	migrationFS := fstest.MapFS{
		"migrations/000001_fail.up.sql": {
			Data: []byte("select * from migration_failure_missing_relation;"),
		},
		"migrations/000001_fail.down.sql": {Data: []byte("select 1;")},
	}
	options := postgresmigrate.MigrationOptions{
		DSN:              dsn,
		SourceFS:         migrationFS,
		SourcePath:       "migrations",
		StatementTimeout: time.Second,
		LockTimeout:      time.Second,
	}

	if _, err := postgresmigrate.MigrateUp(ctx, options); err == nil {
		t.Fatal("MigrateUp(first) error = nil, want failed migration")
	}
	_, err := postgresmigrate.MigrateUp(ctx, options)
	if err == nil {
		t.Fatal("MigrateUp(second) error = nil, want dirty version")
	}
	for _, want := range []string{
		"dirty at version 1",
		"automatic force is disabled",
		"docs/railway-deployment-profile.md",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("MigrateUp(second) error = %q, want %q", err, want)
		}
	}
}

//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresmigrate"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	gooselock "github.com/pressly/goose/v3/lock"
)

func TestPostgresMigrateUpNoopDownUp(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	dsn := pgtest.DSN(t)
	source := fstest.MapFS{
		"migrations/000001_create_items.sql": {
			Data: migrationFile(
				"CREATE TABLE migration_items (id bigint PRIMARY KEY);",
				"DROP TABLE migration_items;",
			),
		},
		"migrations/000002_add_name.sql": {
			Data: migrationFile(
				"ALTER TABLE migration_items ADD COLUMN name text;",
				"ALTER TABLE migration_items DROP COLUMN name;",
			),
		},
	}
	options := migrationOptions(dsn, source)

	first, err := postgresmigrate.MigrateUp(ctx, options)
	if err != nil {
		t.Fatalf("MigrateUp(first) error: %v", err)
	}
	if first.Before != 0 || first.Target != 2 || first.After != 2 || len(first.Migrations) != 2 {
		t.Fatalf("MigrateUp(first) result = %+v", first)
	}

	second, err := postgresmigrate.MigrateUp(ctx, options)
	if err != nil {
		t.Fatalf("MigrateUp(second) error: %v", err)
	}
	if second.Before != 2 || second.After != 2 || len(second.Migrations) != 0 {
		t.Fatalf("MigrateUp(second) result = %+v, want no change", second)
	}

	down, err := postgresmigrate.MigrateDown(ctx, options)
	if err != nil {
		t.Fatalf("MigrateDown() error: %v", err)
	}
	if down.Before != 2 || down.Target != 0 || down.After != 0 || len(down.Migrations) != 2 {
		t.Fatalf("MigrateDown() result = %+v", down)
	}

	pool := openVerificationPool(t, ctx, dsn)
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM goose_db_version").Scan(&count); err != nil {
		t.Fatalf("query goose version rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("goose_db_version row count = %d, want bootstrap only", count)
	}
	if relationExists(t, ctx, pool, "migration_items") {
		t.Fatal("migration_items exists after DownTo(0)")
	}

	reapplied, err := postgresmigrate.MigrateUp(ctx, options)
	if err != nil {
		t.Fatalf("MigrateUp(reapply) error: %v", err)
	}
	if reapplied.Before != 0 || reapplied.After != 2 || len(reapplied.Migrations) != 2 {
		t.Fatalf("MigrateUp(reapply) result = %+v", reapplied)
	}
}

func TestPostgresMigrateRepositorySourceRehearsal(t *testing.T) {
	if _, err := os.Stat("../migrations"); errors.Is(err, fs.ErrNotExist) {
		t.Skip("repository has no owned migration corpus yet")
	} else if err != nil {
		t.Fatalf("inspect repository migration source: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	dsn := pgtest.DSN(t)
	options := postgresmigrate.MigrationOptions{
		DSN:              dsn,
		SourceFS:         os.DirFS(".."),
		SourcePath:       "migrations",
		ConnectTimeout:   5 * time.Second,
		StatementTimeout: time.Minute,
		LockTimeout:      15 * time.Second,
		CleanupTimeout:   15 * time.Second,
	}

	first, err := postgresmigrate.MigrateUp(ctx, options)
	if err != nil {
		t.Fatalf("repository MigrateUp(first) error: %v", err)
	}
	second, err := postgresmigrate.MigrateUp(ctx, options)
	if err != nil {
		t.Fatalf("repository MigrateUp(second) error: %v", err)
	}
	if len(second.Migrations) != 0 || second.After != first.After {
		t.Fatalf("repository second Up = %+v, first = %+v", second, first)
	}
	down, err := postgresmigrate.MigrateDown(ctx, options)
	if err != nil {
		t.Fatalf("repository MigrateDown() error: %v", err)
	}
	if down.After != 0 {
		t.Fatalf("repository Down after = %d, want 0", down.After)
	}
	reapplied, err := postgresmigrate.MigrateUp(ctx, options)
	if err != nil {
		t.Fatalf("repository MigrateUp(reapply) error: %v", err)
	}
	if reapplied.After != first.Target {
		t.Fatalf("repository reapply after = %d, want target %d", reapplied.After, first.Target)
	}
}

func TestPostgresMigratePreservesCommittedPrefixAndRollsBackFailedFile(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	dsn := pgtest.DSN(t)
	source := fstest.MapFS{
		"migrations/000001_committed.sql": {
			Data: migrationFile(
				"CREATE TABLE migration_committed (id bigint);",
				"DROP TABLE migration_committed;",
			),
		},
		"migrations/000002_fails.sql": {
			Data: migrationFile(
				"CREATE TABLE migration_transient (id bigint); SELECT * FROM migration_missing_relation;",
				"DROP TABLE migration_transient;",
			),
		},
	}

	result, err := postgresmigrate.MigrateUp(ctx, migrationOptions(dsn, source))
	if err == nil {
		t.Fatal("MigrateUp() error = nil, want failing second migration")
	}
	if postgresmigrate.FailureStageOf(err) != postgresmigrate.FailureExecute {
		t.Fatalf("failure stage = %q, want execute; error = %v", postgresmigrate.FailureStageOf(err), err)
	}
	if result.After != 1 || len(result.Migrations) != 1 || result.Failed == nil ||
		result.Failed.Version != 2 {
		t.Fatalf("partial result = %+v", result)
	}

	pool := openVerificationPool(t, ctx, dsn)
	if !relationExists(t, ctx, pool, "migration_committed") {
		t.Fatal("committed first migration is absent")
	}
	if relationExists(t, ctx, pool, "migration_transient") {
		t.Fatal("failed migration left transactional DDL behind")
	}
	var versions []int64
	rows, queryErr := pool.Query(ctx, "SELECT version_id FROM goose_db_version ORDER BY id")
	if queryErr != nil {
		t.Fatalf("query goose history: %v", queryErr)
	}
	defer rows.Close()
	for rows.Next() {
		var version int64
		if scanErr := rows.Scan(&version); scanErr != nil {
			t.Fatalf("scan goose history: %v", scanErr)
		}
		versions = append(versions, version)
	}
	if rows.Err() != nil {
		t.Fatalf("iterate goose history: %v", rows.Err())
	}
	if fmt.Sprint(versions) != "[0 1]" {
		t.Fatalf("goose versions = %v, want [0 1]", versions)
	}
}

func TestPostgresMigrateRejectsSourceBehindAppliedHistory(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	dsn := pgtest.DSN(t)
	source := fstest.MapFS{
		"migrations/000001_create.sql": {
			Data: migrationFile("CREATE TABLE migration_owned (id bigint);", "DROP TABLE migration_owned;"),
		},
	}
	if _, err := postgresmigrate.MigrateUp(ctx, migrationOptions(dsn, source)); err != nil {
		t.Fatalf("MigrateUp(seed) error: %v", err)
	}

	empty := fstest.MapFS{"migrations": {Mode: fs.ModeDir}}
	_, err := postgresmigrate.MigrateUp(ctx, migrationOptions(dsn, empty))
	if err == nil {
		t.Fatal("MigrateUp(empty) error = nil, want source-behind failure")
	}
	if postgresmigrate.FailureStageOf(err) != postgresmigrate.FailureState {
		t.Fatalf("failure stage = %q, want state; error = %v", postgresmigrate.FailureStageOf(err), err)
	}
}

func TestPostgresMigrateBoundsStatementAndDDLLockWait(t *testing.T) {
	t.Run("statement timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()

		dsn := pgtest.DSN(t)
		source := fstest.MapFS{
			"migrations/000001_slow.sql": {
				Data: migrationFile("SELECT pg_sleep(5);", "SELECT 1;"),
			},
		}
		options := migrationOptions(dsn, source)
		options.StatementTimeout = 100 * time.Millisecond

		started := time.Now()
		_, err := postgresmigrate.MigrateUp(ctx, options)
		if err == nil {
			t.Fatal("MigrateUp() error = nil, want statement timeout")
		}
		if elapsed := time.Since(started); elapsed >= 3*time.Second {
			t.Fatalf("MigrateUp() elapsed = %s, want bounded below slow statement", elapsed)
		}
		if postgresmigrate.SQLStateOf(err) != "57014" {
			t.Fatalf("SQLSTATE = %q, want 57014; error = %v", postgresmigrate.SQLStateOf(err), err)
		}
	})

	t.Run("ddl lock timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()

		dsn := pgtest.DSN(t)
		pool := openVerificationPool(t, ctx, dsn)
		if _, err := pool.Exec(ctx, "CREATE TABLE migration_locked (id bigint)"); err != nil {
			t.Fatalf("create lock fixture: %v", err)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin lock holder: %v", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, "LOCK TABLE migration_locked IN ACCESS EXCLUSIVE MODE"); err != nil {
			t.Fatalf("lock fixture table: %v", err)
		}

		source := fstest.MapFS{
			"migrations/000001_alter.sql": {
				Data: migrationFile(
					"ALTER TABLE migration_locked ADD COLUMN name text;",
					"ALTER TABLE migration_locked DROP COLUMN name;",
				),
			},
		}
		options := migrationOptions(dsn, source)
		options.LockTimeout = 100 * time.Millisecond
		options.CleanupTimeout = time.Second

		_, err = postgresmigrate.MigrateUp(ctx, options)
		if err == nil {
			t.Fatal("MigrateUp() error = nil, want DDL lock timeout")
		}
		if postgresmigrate.SQLStateOf(err) != "55P03" {
			t.Fatalf("SQLSTATE = %q, want 55P03; error = %v", postgresmigrate.SQLStateOf(err), err)
		}
	})
}

func TestPostgresMigrateSessionLockSerializesConcurrentRunners(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()

	dsn := pgtest.DSN(t)
	source := fstest.MapFS{
		"migrations/000001_once.sql": {
			Data: migrationFile(
				"SELECT pg_sleep(0.2); CREATE TABLE migration_once (id bigint);",
				"DROP TABLE migration_once;",
			),
		},
	}
	options := migrationOptions(dsn, source)
	options.LockTimeout = 3 * time.Second
	options.CleanupTimeout = 3 * time.Second

	start := make(chan struct{})
	results := make(chan postgresmigrate.RunResult, 2)
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			result, err := postgresmigrate.MigrateUp(ctx, options)
			results <- result
			errs <- err
		}()
	}
	close(start)

	totalApplied := 0
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent MigrateUp() error: %v", err)
		}
		totalApplied += len((<-results).Migrations)
	}
	if totalApplied != 1 {
		t.Fatalf("concurrent applied count = %d, want exactly one", totalApplied)
	}
}

func TestPostgresMigrateAdvisoryLockTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	dsn := pgtest.DSN(t)
	holder, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect advisory lock holder: %v", err)
	}
	defer func() { _ = holder.Close(ctx) }()
	if _, err := holder.Exec(ctx, "SELECT pg_advisory_lock($1)", gooselock.DefaultLockID); err != nil {
		t.Fatalf("acquire advisory lock fixture: %v", err)
	}
	defer func() {
		_, _ = holder.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", gooselock.DefaultLockID)
	}()

	source := fstest.MapFS{
		"migrations/000001_create.sql": {
			Data: migrationFile("CREATE TABLE migration_waited (id bigint);", "DROP TABLE migration_waited;"),
		},
	}
	options := migrationOptions(dsn, source)
	options.LockTimeout = 200 * time.Millisecond
	options.CleanupTimeout = time.Second

	_, err = postgresmigrate.MigrateUp(ctx, options)
	if err == nil {
		t.Fatal("MigrateUp() error = nil, want advisory lock timeout")
	}
	if postgresmigrate.FailureStageOf(err) != postgresmigrate.FailureLock {
		t.Fatalf("failure stage = %q, want lock; error = %v", postgresmigrate.FailureStageOf(err), err)
	}
}

func TestPostgresMigrateCancellationStopsLaterMigration(t *testing.T) {
	outer, cancelOuter := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancelOuter()

	dsn := pgtest.DSN(t)
	source := fstest.MapFS{
		"migrations/000001_wait.sql": {
			Data: migrationFile("SELECT pg_sleep(30);", "SELECT 1;"),
		},
		"migrations/000002_later.sql": {
			Data: migrationFile("CREATE TABLE migration_later (id bigint);", "DROP TABLE migration_later;"),
		},
	}
	runCtx, cancelRun := context.WithCancel(outer)
	resultCh := make(chan postgresmigrate.RunResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := postgresmigrate.MigrateUp(runCtx, migrationOptions(dsn, source))
		resultCh <- result
		errCh <- err
	}()

	pool := openVerificationPool(t, outer, dsn)
	waitForMigrationQuery(t, outer, pool, "pg_sleep")
	cancelRun()

	result := <-resultCh
	err := <-errCh
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("MigrateUp() error = %v, want context.Canceled", err)
	}
	if result.After != 0 || len(result.Migrations) != 0 {
		t.Fatalf("canceled result = %+v, want no committed migration", result)
	}
	if relationExists(t, outer, pool, "migration_later") {
		t.Fatal("later migration started after cancellation")
	}
}

func migrationOptions(dsn string, source fstest.MapFS) postgresmigrate.MigrationOptions {
	return postgresmigrate.MigrationOptions{
		DSN:              dsn,
		SourceFS:         source,
		SourcePath:       "migrations",
		ConnectTimeout:   5 * time.Second,
		StatementTimeout: 5 * time.Second,
		LockTimeout:      time.Second,
		CleanupTimeout:   time.Second,
	}
}

func migrationFile(up, down string) []byte {
	return []byte("-- +goose Up\n" + up + "\n-- +goose Down\n" + down + "\n")
}

func openVerificationPool(t *testing.T, ctx context.Context, dsn string) *pgxpool.Pool {
	t.Helper()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open verification pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func relationExists(t *testing.T, ctx context.Context, pool *pgxpool.Pool, relation string) bool {
	t.Helper()

	var exists bool
	if err := pool.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", relation).Scan(&exists); err != nil {
		t.Fatalf("inspect relation %s: %v", relation, err)
	}
	return exists
}

func waitForMigrationQuery(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	queryFragment string,
) {
	t.Helper()

	// ctx still bounds the observation itself: when the caller's deadline passes
	// first, the query below fails and names that, rather than this wait running
	// on past the test that owns it.
	waittest.Until(t, migrationQueryWait, func() bool {
		var count int
		err := pool.QueryRow(
			ctx,
			`SELECT count(*)
			   FROM pg_stat_activity
			  WHERE datname = current_database()
			    AND application_name = 'goose-migrate'
			    AND state = 'active'
			    AND query ILIKE '%' || $1 || '%'`,
			queryFragment,
		).Scan(&count)
		if err != nil {
			t.Fatalf("observe migration query: %v", err)
		}
		return count > 0
	}, "an active migration query matching "+queryFragment)
}

// migrationQueryWait bounds the wait above. It sits under the 20s budget each
// caller gives its context, so a broken wait fails here with what it was looking
// for rather than as an expired context somewhere further along.
const migrationQueryWait = 10 * time.Second

func TestPostgresMigrateFailureDoesNotCreateDirtyState(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()

	dsn := pgtest.DSN(t)
	source := fstest.MapFS{
		"migrations/000001_fail.sql": {
			Data: migrationFile("SELECT * FROM migration_missing_forever;", "SELECT 1;"),
		},
	}
	options := migrationOptions(dsn, source)

	for attempt := 1; attempt <= 2; attempt++ {
		result, err := postgresmigrate.MigrateUp(ctx, options)
		if err == nil {
			t.Fatalf("MigrateUp(attempt %d) error = nil", attempt)
		}
		if postgresmigrate.FailureStageOf(err) != postgresmigrate.FailureExecute {
			t.Fatalf("attempt %d stage = %q; error = %v", attempt, postgresmigrate.FailureStageOf(err), err)
		}
		if result.Before != 0 || result.After != 0 {
			t.Fatalf("attempt %d result = %+v", attempt, result)
		}
		if strings.Contains(err.Error(), "dirty") {
			t.Fatalf("Goose transactional failure reported legacy dirty state: %v", err)
		}
	}
}

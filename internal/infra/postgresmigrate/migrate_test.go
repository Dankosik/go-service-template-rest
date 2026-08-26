package postgresmigrate

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pressly/goose/v3"
)

const unreachableMigrationDSN = "postgres://user:secret@127.0.0.1:1/app?sslmode=disable" //nolint:gosec // Deliberately unreachable test DSN.

func TestRootMigrationSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    fs.FS
		path      string
		wantError string
	}{
		{
			name: "canonical migration",
			source: fstest.MapFS{
				"migrations":                   {Mode: fs.ModeDir},
				"migrations/000001_create.sql": {Data: []byte("-- +goose Up\nSELECT 1;\n-- +goose Down\nSELECT 1;\n")},
			},
			path: "migrations",
		},
		{
			name:   "empty directory",
			source: fstest.MapFS{"migrations": {Mode: fs.ModeDir}},
			path:   "migrations",
		},
		{
			name:      "filesystem is required",
			path:      "migrations",
			wantError: "filesystem is required",
		},
		{
			name:      "root path is rejected",
			source:    fstest.MapFS{"migrations": {Mode: fs.ModeDir}},
			path:      ".",
			wantError: "invalid",
		},
		{
			name:      "traversal is rejected",
			source:    fstest.MapFS{"migrations": {Mode: fs.ModeDir}},
			path:      "../migrations",
			wantError: "invalid",
		},
		{
			name:      "missing directory",
			source:    fstest.MapFS{},
			path:      "migrations",
			wantError: "file does not exist",
		},
		{
			name: "source symlink is rejected",
			source: fstest.MapFS{
				"migrations": {Mode: fs.ModeSymlink, Data: []byte("real")},
				"real":       {Mode: fs.ModeDir},
			},
			path:      "migrations",
			wantError: "symbolic link",
		},
		{
			name: "nested directory is rejected",
			source: fstest.MapFS{
				"migrations":        {Mode: fs.ModeDir},
				"migrations/nested": {Mode: fs.ModeDir},
			},
			path:      "migrations",
			wantError: "nested directory",
		},
		{
			name: "migration symlink is rejected",
			source: fstest.MapFS{
				"migrations":                   {Mode: fs.ModeDir},
				"migrations/000001_create.sql": {Mode: fs.ModeSymlink, Data: []byte("../outside.sql")},
			},
			path:      "migrations",
			wantError: "symbolic link",
		},
		{
			name: "Go migration is rejected",
			source: fstest.MapFS{
				"migrations":                  {Mode: fs.ModeDir},
				"migrations/000001_create.go": {Data: []byte("package migrations")},
			},
			path:      "migrations",
			wantError: "not canonical",
		},
		{
			name: "non-canonical filename is rejected",
			source: fstest.MapFS{
				"migrations":                   {Mode: fs.ModeDir},
				"migrations/000001-create.sql": {Data: []byte("-- +goose Up\nSELECT 1;\n")},
			},
			path:      "migrations",
			wantError: "not canonical",
		},
		{
			name: "zero migration version is rejected",
			source: fstest.MapFS{
				"migrations":                   {Mode: fs.ModeDir},
				"migrations/000000_create.sql": {Data: []byte("-- +goose Up\nSELECT 1;\n")},
			},
			path:      "migrations",
			wantError: "not canonical",
		},
		{
			name: "non-transactional migration is rejected case-insensitively",
			source: fstest.MapFS{
				"migrations":                   {Mode: fs.ModeDir},
				"migrations/000001_create.sql": {Data: []byte("-- +GoOsE no transaction\n-- +goose Up\nSELECT 1;\n")},
			},
			path:      "migrations",
			wantError: "disables transactions",
		},
		{
			name: "environment substitution is rejected case-insensitively",
			source: fstest.MapFS{
				"migrations":                   {Mode: fs.ModeDir},
				"migrations/000001_create.sql": {Data: []byte("-- +gOoSe EnVsUb On\n-- +goose Up\nSELECT 1;\n")},
			},
			path:      "migrations",
			wantError: "environment substitution",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := rootMigrationSource(tc.source, tc.path)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("rootMigrationSource() error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("rootMigrationSource() error = %v", err)
			}
		})
	}
}

func TestMigrateUpRejectsSourceAndConfigBeforeConnecting(t *testing.T) {
	t.Parallel()

	_, err := MigrateUp(context.Background(), MigrationOptions{
		DSN:        unreachableMigrationDSN,
		SourceFS:   fstest.MapFS{},
		SourcePath: "missing",
	})
	if FailureStageOf(err) != FailureSource {
		t.Fatalf("MigrateUp() stage = %q, want %q; error = %v", FailureStageOf(err), FailureSource, err)
	}

	_, err = MigrateUp(context.Background(), MigrationOptions{
		DSN:        unreachableMigrationDSN,
		SourceFS:   fstest.MapFS{},
		SourcePath: ".",
	})
	if FailureStageOf(err) != FailureSource {
		t.Fatalf("MigrateUp() empty path stage = %q, want %q; error = %v", FailureStageOf(err), FailureSource, err)
	}

	_, err = MigrateUp(context.Background(), MigrationOptions{
		DSN: unreachableMigrationDSN,
		SourceFS: fstest.MapFS{
			"migrations":                   {Mode: fs.ModeDir},
			"migrations/000001_create.sql": {Data: []byte("-- +goose NO TRANSACTION\n-- +goose Up\nSELECT 1;\n")},
		},
		SourcePath: "migrations",
	})
	if FailureStageOf(err) != FailureSource {
		t.Fatalf("MigrateUp() forbidden directive stage = %q, want %q; error = %v", FailureStageOf(err), FailureSource, err)
	}

	_, err = MigrateUp(context.Background(), MigrationOptions{
		DSN:        unreachableMigrationDSN,
		SourceFS:   fstest.MapFS{"migrations": {Mode: fs.ModeDir}},
		SourcePath: "migrations",
	})
	if FailureStageOf(err) != FailureConfig {
		t.Fatalf("MigrateUp() invalid options stage = %q, want %q; error = %v", FailureStageOf(err), FailureConfig, err)
	}
}

func TestEmptySourceStillConnects(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	_, err := MigrateUp(ctx, MigrationOptions{
		DSN:              unreachableMigrationDSN,
		SourceFS:         fstest.MapFS{"migrations": {Mode: fs.ModeDir}},
		SourcePath:       "migrations",
		ConnectTimeout:   50 * time.Millisecond,
		StatementTimeout: time.Second,
		LockTimeout:      50 * time.Millisecond,
		CleanupTimeout:   50 * time.Millisecond,
	})
	if FailureStageOf(err) != FailureConnect {
		t.Fatalf("MigrateUp() stage = %q, want %q; error = %v", FailureStageOf(err), FailureConnect, err)
	}
}

func TestExecutionContextReservesCleanupBudget(t *testing.T) {
	t.Parallel()

	outer, cancelOuter := context.WithTimeout(context.Background(), time.Minute)
	defer cancelOuter()
	outerDeadline, _ := outer.Deadline()

	executionCtx, cancelExecution, err := executionContext(outer, 10*time.Second)
	if err != nil {
		t.Fatalf("executionContext() error = %v", err)
	}
	defer cancelExecution()
	executionDeadline, ok := executionCtx.Deadline()
	if !ok {
		t.Fatal("execution context has no deadline")
	}
	if difference := outerDeadline.Sub(executionDeadline); difference != 10*time.Second {
		t.Fatalf("reserved duration = %s, want 10s", difference)
	}
}

func TestRunErrorClassification(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("sentinel")
	err := stageError(FailureState, sentinel)
	if FailureStageOf(err) != FailureState {
		t.Fatalf("FailureStageOf() = %q, want %q", FailureStageOf(err), FailureState)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("RunError does not wrap sentinel: %v", err)
	}

	pgErr := &pgconn.PgError{Code: "55P03"}
	if got := SQLStateOf(stageError(FailureExecute, pgErr)); got != "55P03" {
		t.Fatalf("SQLStateOf() = %q, want 55P03", got)
	}
}

func TestMigrationCleanupStageWinsAndPreservesCauses(t *testing.T) {
	t.Parallel()

	primaryCause := errors.New("primary")
	cleanupCause := errors.New("cleanup")
	primary := stageError(FailureState, primaryCause)

	if got := withMigrationCleanup(primary, nil); FailureStageOf(got) != FailureState || !errors.Is(got, primaryCause) {
		t.Fatalf("withMigrationCleanup(primary, nil) = %v, want original stage and cause", got)
	}

	err := withMigrationCleanup(primary, cleanupCause)
	if FailureStageOf(err) != FailureCleanup {
		t.Fatalf("withMigrationCleanup() stage = %q, want %q", FailureStageOf(err), FailureCleanup)
	}
	for _, want := range []error{primaryCause, cleanupCause} {
		if !errors.Is(err, want) {
			t.Fatalf("withMigrationCleanup() error = %v, want wrapped %v", err, want)
		}
	}

	cleanupOnly := withMigrationCleanup(nil, cleanupCause)
	if FailureStageOf(cleanupOnly) != FailureCleanup || !errors.Is(cleanupOnly, cleanupCause) {
		t.Fatalf("withMigrationCleanup(nil, cleanup) = %v, want cleanup stage and cause", cleanupOnly)
	}
}

func TestSetAfterFromApplied(t *testing.T) {
	t.Parallel()

	sources := []*goose.Source{{Version: 1}, {Version: 2}, {Version: 3}}
	up := RunResult{
		Before: 1,
		After:  1,
	}
	setAfterFromApplied(&up, directionUp, sources, []*goose.MigrationResult{{Source: sources[1]}})
	if up.After != 2 || up.AppliedCount != 1 {
		t.Fatalf("up = %+v, want after 2 and applied count 1", up)
	}

	down := RunResult{
		Before: 3,
		After:  3,
	}
	setAfterFromApplied(&down, directionDown, sources, []*goose.MigrationResult{
		{Source: sources[2]},
		{Source: sources[1]},
	})
	if down.After != 1 || down.AppliedCount != 2 {
		t.Fatalf("down = %+v, want after 1 and applied count 2", down)
	}
}

func TestLogMigrationResultUsesSafeMetadata(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	logMigrationResult(t.Context(), logger, &goose.MigrationResult{
		Source:    &goose.Source{Path: "nested/000001_create.sql", Version: 1},
		Direction: "up",
	}, true)

	record := output.String()
	for _, field := range []string{`"migration.version":1`, `"migration.filename":"000001_create.sql"`, `"outcome":"error"`} {
		if !strings.Contains(record, field) {
			t.Fatalf("migration result log %q does not contain %q", record, field)
		}
	}
	if strings.Contains(record, "nested/") {
		t.Fatalf("migration result log disclosed source path: %s", record)
	}
}

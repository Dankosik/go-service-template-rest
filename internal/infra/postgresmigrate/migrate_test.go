package postgresmigrate

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
)

func TestAdmitSource(t *testing.T) {
	t.Parallel()

	valid := migrationSQL("CREATE TABLE example (id bigint);", "DROP TABLE example;")
	tests := []struct {
		name      string
		source    fstest.MapFS
		wantFiles []string
		wantError string
	}{
		{
			name:   "empty source is valid",
			source: fstest.MapFS{"migrations": {Mode: fs.ModeDir}},
		},
		{
			name: "canonical migrations are sorted",
			source: fstest.MapFS{
				"migrations/000002_add_name.sql": {Data: valid},
				"migrations/000001_create.sql":   {Data: valid},
			},
			wantFiles: []string{"000001_create.sql", "000002_add_name.sql"},
		},
		{
			name: "legacy pair is rejected",
			source: fstest.MapFS{
				"migrations/000001_create.up.sql": {Data: valid},
			},
			wantError: "not canonical",
		},
		{
			name: "nested directory is rejected",
			source: fstest.MapFS{
				"migrations/nested/000001_create.sql": {Data: valid},
			},
			wantError: "nested directory",
		},
		{
			name: "zero version is rejected",
			source: fstest.MapFS{
				"migrations/000000_create.sql": {Data: valid},
			},
			wantError: "invalid version",
		},
		{
			name: "duplicate version is rejected",
			source: fstest.MapFS{
				"migrations/000001_create.sql": {Data: valid},
				"migrations/000001_rename.sql": {Data: valid},
			},
			wantError: "duplicated",
		},
		{
			name: "missing direction is rejected",
			source: fstest.MapFS{
				"migrations/000001_create.sql": {Data: []byte("-- +goose Up\nSELECT 1;\n")},
			},
			wantError: "exactly one",
		},
		{
			name: "directions must be ordered",
			source: fstest.MapFS{
				"migrations/000001_create.sql": {
					Data: []byte("-- +goose Down\nSELECT 1;\n-- +goose Up\nSELECT 1;\n"),
				},
			},
			wantError: "Up before",
		},
		{
			name: "no transaction is rejected",
			source: fstest.MapFS{
				"migrations/000001_create.sql": {
					Data: append([]byte("-- +goose NO TRANSACTION\n"), valid...),
				},
			},
			wantError: "cannot disable transactions",
		},
		{
			name: "environment substitution is rejected",
			source: fstest.MapFS{
				"migrations/000001_create.sql": {
					Data: append([]byte("-- +goose ENVSUB ON\n"), valid...),
				},
			},
			wantError: "environment substitution",
		},
		{
			name: "case variant transaction escape is rejected",
			source: fstest.MapFS{
				"migrations/000001_create.sql": {
					Data: append([]byte("-- +goose no transaction\n"), valid...),
				},
			},
			wantError: "non-canonical",
		},
		{
			name: "leading annotation whitespace is rejected",
			source: fstest.MapFS{
				"migrations/000001_create.sql": {
					Data: []byte(" -- +goose Up\nSELECT 1;\n-- +goose Down\nSELECT 1;\n"),
				},
			},
			wantError: "non-canonical",
		},
		{
			name: "unexpected file is rejected",
			source: fstest.MapFS{
				"migrations/README.md": {Data: []byte("not a migration")},
			},
			wantError: "not canonical",
		},
		{
			name: "symbolic link is rejected",
			source: fstest.MapFS{
				"migrations/000001_create.sql": {Mode: fs.ModeSymlink},
			},
			wantError: "symbolic link",
		},
		{
			name: "non-regular file is rejected",
			source: fstest.MapFS{
				"migrations/000001_create.sql": {Mode: fs.ModeNamedPipe},
			},
			wantError: "non-regular",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := admitSource(tc.source, "migrations")
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("admitSource() error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("admitSource() error = %v", err)
			}
			gotFiles := make([]string, 0, len(got.Migrations))
			for _, migration := range got.Migrations {
				gotFiles = append(gotFiles, migration.Filename)
			}
			if !slices.Equal(gotFiles, tc.wantFiles) {
				t.Fatalf("admitSource() files = %v, want %v", gotFiles, tc.wantFiles)
			}
		})
	}
}

func TestMigrateUpRejectsSourceAndConfigBeforeConnecting(t *testing.T) {
	t.Parallel()

	_, err := admitSource(fstest.MapFS{}, "../migrations")
	if err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("admitSource() traversal error = %v, want filesystem escape rejection", err)
	}

	_, err = MigrateUp(context.Background(), MigrationOptions{
		DSN:        "postgres://user:secret@127.0.0.1:1/app?sslmode=disable",
		SourceFS:   fstest.MapFS{},
		SourcePath: "missing",
	})
	if FailureStageOf(err) != FailureSource {
		t.Fatalf("MigrateUp() stage = %q, want %q; error = %v", FailureStageOf(err), FailureSource, err)
	}

	_, err = MigrateUp(context.Background(), MigrationOptions{
		DSN:        "postgres://user:secret@127.0.0.1:1/app?sslmode=disable",
		SourceFS:   fstest.MapFS{},
		SourcePath: ".",
	})
	if FailureStageOf(err) != FailureSource {
		t.Fatalf("MigrateUp() empty path stage = %q, want %q; error = %v", FailureStageOf(err), FailureSource, err)
	}

	_, err = MigrateUp(context.Background(), MigrationOptions{
		DSN:        "postgres://user:secret@127.0.0.1:1/app?sslmode=disable",
		SourceFS:   fstest.MapFS{"migrations/.keep": {Data: nil}},
		SourcePath: "migrations",
	})
	if FailureStageOf(err) != FailureSource {
		t.Fatalf("MigrateUp() noncanonical source stage = %q, want %q; error = %v", FailureStageOf(err), FailureSource, err)
	}

	_, err = MigrateUp(context.Background(), MigrationOptions{
		DSN:        "postgres://user:secret@127.0.0.1:1/app?sslmode=disable",
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
		DSN:              "postgres://user:secret@127.0.0.1:1/app?sslmode=disable",
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

func TestValidateMigrationState(t *testing.T) {
	t.Parallel()

	source := []sourceMigration{
		{Version: 1, Filename: "000001_create.sql"},
		{Version: 2, Filename: "000002_add.sql"},
	}
	tests := []struct {
		name        string
		tableExists bool
		rows        []*database.ListMigrationsResult
		source      []sourceMigration
		want        migrationState
		wantError   string
	}{
		{
			name:   "absent table is empty state",
			source: source,
			want:   migrationState{Target: 2},
		},
		{
			name:        "bootstrap only",
			tableExists: true,
			rows:        historyRow(0),
			source:      source,
			want:        migrationState{Target: 2},
		},
		{
			name:        "valid full prefix",
			tableExists: true,
			rows:        historyRow(2, 1, 0),
			source:      source,
			want:        migrationState{Applied: []int64{1, 2}, Current: 2, Target: 2},
		},
		{
			name:        "missing bootstrap",
			tableExists: true,
			rows:        historyRow(1),
			source:      source,
			wantError:   "bootstrap",
		},
		{
			name:        "duplicate version",
			tableExists: true,
			rows:        historyRow(1, 1, 0),
			source:      source,
			wantError:   "duplicated",
		},
		{
			name:        "out of order application",
			tableExists: true,
			rows:        historyRow(1, 2, 0),
			source:      source,
			wantError:   "not ascending",
		},
		{
			name:        "not a source prefix",
			tableExists: true,
			rows:        historyRow(2, 0),
			source:      source,
			wantError:   "not a source prefix",
		},
		{
			name:        "database ahead",
			tableExists: true,
			rows:        historyRow(2, 1, 0),
			source:      source[:1],
			wantError:   "ahead",
		},
		{
			name:        "unapplied row",
			tableExists: true,
			rows: []*database.ListMigrationsResult{
				{Version: 1, IsApplied: false},
				{Version: 0, IsApplied: true},
			},
			source:    source,
			wantError: "not applied",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := validateMigrationState(tc.rows, tc.tableExists, tc.source)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("validateMigrationState() error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateMigrationState() error = %v", err)
			}
			if got.Current != tc.want.Current || got.Target != tc.want.Target ||
				!slices.Equal(got.Applied, tc.want.Applied) {
				t.Fatalf("validateMigrationState() = %+v, want %+v", got, tc.want)
			}
		})
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

func TestSetPartialAfter(t *testing.T) {
	t.Parallel()

	source := []sourceMigration{{Version: 1}, {Version: 2}, {Version: 3}}
	up := RunResult{
		Before:     1,
		After:      1,
		Migrations: []MigrationResult{{Version: 2}},
	}
	setPartialAfter(&up, directionUp, source)
	if up.After != 2 {
		t.Fatalf("up.After = %d, want 2", up.After)
	}

	down := RunResult{
		Before:     3,
		After:      3,
		Migrations: []MigrationResult{{Version: 3}, {Version: 2}},
	}
	setPartialAfter(&down, directionDown, source)
	if down.After != 1 {
		t.Fatalf("down.After = %d, want 1", down.After)
	}
}

func TestMigrationResultFromGooseUsesSafeMetadata(t *testing.T) {
	t.Parallel()

	got := migrationResultFromGoose(&goose.MigrationResult{
		Source: &goose.Source{
			Type:    goose.TypeSQL,
			Path:    "nested/000001_create.sql",
			Version: 1,
		},
		Duration:  time.Second,
		Direction: "up",
		Empty:     true,
	})
	if got.Filename != "000001_create.sql" || got.Version != 1 || got.Direction != "up" || !got.Empty {
		t.Fatalf("migrationResultFromGoose() = %+v", got)
	}
}

func migrationSQL(up, down string) []byte {
	return []byte("-- +goose Up\n" + up + "\n-- +goose Down\n" + down + "\n")
}

func historyRow(versions ...int64) []*database.ListMigrationsResult {
	rows := make([]*database.ListMigrationsResult, 0, len(versions))
	for _, version := range versions {
		rows = append(rows, &database.ListMigrationsResult{Version: version, IsApplied: true})
	}
	return rows
}

package postgresmigrate

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	migrate "github.com/golang-migrate/migrate/v4"
)

type closeFunc struct {
	err error
}

func (f closeFunc) Close() error {
	return f.err
}

type migrationExecutorStub struct {
	upErrs           []error
	upCalls          int
	downErr          error
	sourceCloseErr   error
	databaseCloseErr error
	calls            []string
	onUp             func()
}

func (s *migrationExecutorStub) Up() error {
	s.calls = append(s.calls, "up")
	if s.onUp != nil {
		s.onUp()
	}
	var err error
	if s.upCalls < len(s.upErrs) {
		err = s.upErrs[s.upCalls]
	}
	s.upCalls++
	return err
}

func (s *migrationExecutorStub) Down() error {
	s.calls = append(s.calls, "down")
	return s.downErr
}

func (s *migrationExecutorStub) Close() (error, error) {
	s.calls = append(s.calls, "close")
	return s.sourceCloseErr, s.databaseCloseErr
}

func TestMigrateUpRejectsInvalidInputsBeforeConnecting(t *testing.T) {
	t.Parallel()

	validDSN := "postgres://user:pass@127.0.0.1:5432/app?sslmode=disable"

	testCases := []struct {
		name        string
		opts        MigrationOptions
		wantErr     string
		wantWrapped error
	}{
		{
			name: "empty source path",
			opts: MigrationOptions{
				DSN:              validDSN,
				SourcePath:       " \t\n",
				StatementTimeout: time.Minute,
				LockTimeout:      time.Second,
			},
			wantErr: "migration source path is empty",
		},
		{
			name: "empty dsn",
			opts: MigrationOptions{
				DSN:              " \t\n",
				SourcePath:       "migrations",
				StatementTimeout: time.Minute,
				LockTimeout:      time.Second,
			},
			wantErr:     "postgres dsn is empty",
			wantWrapped: postgres.ErrConfig,
		},
		{
			name: "missing source",
			opts: MigrationOptions{
				DSN:              validDSN,
				SourceFS:         fstest.MapFS{},
				SourcePath:       "migrations",
				StatementTimeout: time.Minute,
				LockTimeout:      time.Second,
			},
			wantErr: "open migration source",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := MigrateUp(context.Background(), tc.opts)
			if err == nil {
				t.Fatal("MigrateUp() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("MigrateUp() error = %q, want to contain %q", err.Error(), tc.wantErr)
			}
			if tc.wantWrapped != nil && !errors.Is(err, tc.wantWrapped) {
				t.Fatalf("MigrateUp() error = %v, want wrapped %v", err, tc.wantWrapped)
			}
		})
	}
}

func TestMigrateUpReportsRunFailureWithReadableSource(t *testing.T) {
	t.Parallel()

	sourceFS := fstest.MapFS{
		"migrations/000001_init.up.sql": {
			Data: []byte("CREATE TABLE migrate_smoke (id integer);"),
		},
		"migrations/000001_init.down.sql": {
			Data: []byte("DROP TABLE migrate_smoke;"),
		},
	}

	_, err := MigrateUp(context.Background(), MigrationOptions{
		DSN:              "postgres://user:pass@127.0.0.1:1/app?sslmode=disable",
		SourceFS:         sourceFS,
		SourcePath:       "migrations",
		StatementTimeout: time.Minute,
		LockTimeout:      time.Second,
	})
	if err == nil {
		t.Fatal("MigrateUp() error = nil, want run failure")
	}
	if !strings.Contains(err.Error(), "open postgres migration driver") {
		t.Fatalf("MigrateUp() error = %q, want postgres driver context", err.Error())
	}
}

func TestMigrateDownRejectsInvalidInputBeforeConnecting(t *testing.T) {
	t.Parallel()

	err := MigrateDown(context.Background(), MigrationOptions{})
	if err == nil {
		t.Fatal("MigrateDown() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "migration source path is empty") {
		t.Fatalf("MigrateDown() error = %q, want source path validation", err.Error())
	}
}

func TestApplyMigrations(t *testing.T) {
	t.Parallel()

	upErr := errors.New("up failed")

	testCases := []struct {
		name        string
		upErrs      []error
		wantChanged bool
		wantErr     error
		wantErrText string
	}{
		{
			name:        "applies pending migrations",
			wantChanged: true,
		},
		{
			// An already-current schema is the ordinary outcome of every deploy
			// after the one that introduced a migration, not a failure.
			name:   "up-to-date schema reports no change",
			upErrs: []error{migrate.ErrNoChange},
		},
		{
			name:        "apply failure",
			upErrs:      []error{upErr},
			wantErr:     upErr,
			wantErrText: "run postgres migrations",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runner := &migrationExecutorStub{upErrs: tc.upErrs}
			changed, err := applyMigrations(context.Background(), runner)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("applyMigrations() error = %v, want wrapped %v", err, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErrText) {
					t.Fatalf("applyMigrations() error = %q, want to contain %q", err.Error(), tc.wantErrText)
				}
			} else if err != nil {
				t.Fatalf("applyMigrations() error = %v, want nil", err)
			}
			if changed != tc.wantChanged {
				t.Fatalf("applyMigrations() changed = %t, want %t", changed, tc.wantChanged)
			}
			if !slices.Equal(runner.calls, []string{"up"}) {
				t.Fatalf("applyMigrations() calls = %v, want [up]", runner.calls)
			}
		})
	}
}

func TestRunMigrationOperationJoinsOperationAndCloseFailures(t *testing.T) {
	t.Parallel()

	upErr := errors.New("up failed")
	sourceCloseErr := errors.New("source close failed")
	databaseCloseErr := errors.New("database close failed")
	runner := &migrationExecutorStub{
		upErrs:           []error{upErr},
		sourceCloseErr:   sourceCloseErr,
		databaseCloseErr: databaseCloseErr,
	}

	err := runMigrationOperation(context.Background(), runner, func(executor migrationExecutor) error {
		_, applyErr := applyMigrations(context.Background(), executor)
		return applyErr
	})
	for _, want := range []error{upErr, sourceCloseErr, databaseCloseErr} {
		if !errors.Is(err, want) {
			t.Fatalf("runMigrationOperation() error = %v, want wrapped %v", err, want)
		}
	}
	if !strings.Contains(err.Error(), "close migration runner") {
		t.Fatalf("runMigrationOperation() error = %q, want close context", err.Error())
	}
	if !slices.Equal(runner.calls, []string{"up", "close"}) {
		t.Fatalf("runMigrationOperation() calls = %v, want [up close]", runner.calls)
	}
}

func TestRunMigrationOperationReportsCancellationAndClosesRunner(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	runner := &migrationExecutorStub{onUp: cancel}

	changed := false
	err := runMigrationOperation(ctx, runner, func(executor migrationExecutor) error {
		applied, applyErr := applyMigrations(ctx, executor)
		changed = applied
		return applyErr
	})
	if changed {
		t.Fatal("runMigrationOperation() reported a schema change, want none after cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runMigrationOperation() error = %v, want context.Canceled", err)
	}
	if !slices.Equal(runner.calls, []string{"up", "close"}) {
		t.Fatalf("runMigrationOperation() calls = %v, want [up close]", runner.calls)
	}
}

func TestMigrationOperationErrorReportsDirtyVersionAndManualRecovery(t *testing.T) {
	t.Parallel()

	err := migrationOperationError("run postgres migrations", migrate.ErrDirty{Version: 42})
	for _, want := range []string{
		"dirty at version 42",
		"automatic force is disabled",
		"docs/railway-deployment-profile.md",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("migrationOperationError() = %q, want %q", err, want)
		}
	}
	var dirty migrate.ErrDirty
	if !errors.As(err, &dirty) || dirty.Version != 42 {
		t.Fatalf("migrationOperationError() error = %v, want wrapped dirty version 42", err)
	}
}

func TestValidateMigrationTimeouts(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		opts    MigrationOptions
		wantErr string
	}{
		{
			name:    "missing statement timeout",
			opts:    MigrationOptions{LockTimeout: time.Second},
			wantErr: "statement timeout",
		},
		{
			name:    "missing lock timeout",
			opts:    MigrationOptions{StatementTimeout: time.Second},
			wantErr: "lock timeout",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateMigrationTimeouts(tc.opts)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("validateMigrationTimeouts() error = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestNormalizeMigrationSourcePath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		raw     string
		want    string
		wantErr string
	}{
		{name: "relative clean path", raw: "migrations", want: "migrations"},
		{name: "absolute path converted to fs path", raw: "/migrations", want: "migrations"},
		{name: "trimmed and cleaned path", raw: " ./migrations// ", want: "migrations"},
		{name: "empty", raw: " \t\n", wantErr: "migration source path is empty"},
		{name: "root", raw: "/", wantErr: "migration source path is empty"},
		{name: "dot", raw: ".", wantErr: "migration source path is empty"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeMigrationSourcePath(tc.raw)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("normalizeMigrationSourcePath() error = nil, want non-nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("normalizeMigrationSourcePath() error = %q, want to contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeMigrationSourcePath() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Fatalf("normalizeMigrationSourcePath() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCloseMigrationResources(t *testing.T) {
	t.Parallel()

	sourceErr := errors.New("source close failed")
	databaseErr := errors.New("database close failed")

	err := closeMigrationResources(closeFunc{err: sourceErr}, closeFunc{err: databaseErr})
	if !errors.Is(err, sourceErr) {
		t.Fatalf("closeMigrationResources() error = %v, want source error", err)
	}
	if !errors.Is(err, databaseErr) {
		t.Fatalf("closeMigrationResources() error = %v, want database error", err)
	}

	if err := closeMigrationResources(nil, nil); err != nil {
		t.Fatalf("closeMigrationResources(nil, nil) error = %v, want nil", err)
	}
}

package postgres

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

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
}

func (s *migrationExecutorStub) Up() error {
	s.calls = append(s.calls, "up")
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
				DSN:        validDSN,
				SourcePath: " \t\n",
			},
			wantErr: "migration source path is empty",
		},
		{
			name: "empty dsn",
			opts: MigrationOptions{
				DSN:        " \t\n",
				SourcePath: "migrations",
			},
			wantErr:     "postgres dsn is empty",
			wantWrapped: ErrConfig,
		},
		{
			name: "missing source",
			opts: MigrationOptions{
				DSN:        validDSN,
				SourceFS:   fstest.MapFS{},
				SourcePath: "migrations",
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
		DSN:        "postgres://user:pass@127.0.0.1:1/app?sslmode=disable",
		SourceFS:   sourceFS,
		SourcePath: "migrations",
	})
	if err == nil {
		t.Fatal("MigrateUp() error = nil, want run failure")
	}
	if !strings.Contains(err.Error(), "open postgres migration driver") {
		t.Fatalf("MigrateUp() error = %q, want postgres driver context", err.Error())
	}
}

func TestValidateMigrationsRejectsInvalidInputBeforeConnecting(t *testing.T) {
	t.Parallel()

	err := ValidateMigrations(context.Background(), MigrationOptions{})
	if err == nil {
		t.Fatal("ValidateMigrations() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "migration source path is empty") {
		t.Fatalf("ValidateMigrations() error = %q, want source path validation", err.Error())
	}
}

func TestExecuteMigrations(t *testing.T) {
	t.Parallel()

	upErr := errors.New("up failed")
	downErr := errors.New("down failed")
	reapplyErr := errors.New("reapply failed")

	testCases := []struct {
		name        string
		rehearse    bool
		upErrs      []error
		downErr     error
		wantChanged bool
		wantCalls   []string
		wantErr     error
		wantErrText string
	}{
		{
			name:        "apply only",
			wantChanged: true,
			wantCalls:   []string{"up", "close"},
		},
		{
			name:        "rehearse full migration chain",
			rehearse:    true,
			wantChanged: true,
			wantCalls:   []string{"up", "down", "up", "close"},
		},
		{
			name:      "rehearse up-to-date schema",
			rehearse:  true,
			upErrs:    []error{migrate.ErrNoChange},
			wantCalls: []string{"up", "down", "up", "close"},
		},
		{
			name:        "apply failure",
			upErrs:      []error{upErr},
			wantCalls:   []string{"up", "close"},
			wantErr:     upErr,
			wantErrText: "run postgres migrations",
		},
		{
			name:        "rollback failure",
			rehearse:    true,
			downErr:     downErr,
			wantCalls:   []string{"up", "down", "close"},
			wantErr:     downErr,
			wantErrText: "roll back all postgres migrations",
		},
		{
			name:        "reapply failure",
			rehearse:    true,
			upErrs:      []error{nil, reapplyErr},
			wantCalls:   []string{"up", "down", "up", "close"},
			wantErr:     reapplyErr,
			wantErrText: "reapply all postgres migrations",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runner := &migrationExecutorStub{
				upErrs:  tc.upErrs,
				downErr: tc.downErr,
			}
			changed, err := executeMigrations(runner, tc.rehearse)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("executeMigrations() error = %v, want wrapped %v", err, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErrText) {
					t.Fatalf("executeMigrations() error = %q, want to contain %q", err.Error(), tc.wantErrText)
				}
			} else if err != nil {
				t.Fatalf("executeMigrations() error = %v, want nil", err)
			}
			if changed != tc.wantChanged {
				t.Fatalf("executeMigrations() changed = %t, want %t", changed, tc.wantChanged)
			}
			if !slices.Equal(runner.calls, tc.wantCalls) {
				t.Fatalf("executeMigrations() calls = %v, want %v", runner.calls, tc.wantCalls)
			}
		})
	}
}

func TestExecuteMigrationsJoinsOperationAndCloseFailures(t *testing.T) {
	t.Parallel()

	upErr := errors.New("up failed")
	sourceCloseErr := errors.New("source close failed")
	databaseCloseErr := errors.New("database close failed")
	runner := &migrationExecutorStub{
		upErrs:           []error{upErr},
		sourceCloseErr:   sourceCloseErr,
		databaseCloseErr: databaseCloseErr,
	}

	_, err := executeMigrations(runner, false)
	for _, want := range []error{upErr, sourceCloseErr, databaseCloseErr} {
		if !errors.Is(err, want) {
			t.Fatalf("executeMigrations() error = %v, want wrapped %v", err, want)
		}
	}
	if !strings.Contains(err.Error(), "close migration runner") {
		t.Fatalf("executeMigrations() error = %q, want close context", err.Error())
	}
	if !slices.Equal(runner.calls, []string{"up", "close"}) {
		t.Fatalf("executeMigrations() calls = %v, want [up close]", runner.calls)
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

func TestPoolNilAndZeroValueSafety(t *testing.T) {
	t.Parallel()

	var nilPool *Pool
	nilPool.Close()
	if err := nilPool.Check(context.Background()); err == nil {
		t.Fatal("nil Pool Check() error = nil, want non-nil")
	} else if !errors.Is(err, ErrHealthcheck) {
		t.Fatalf("nil Pool Check() error = %v, want ErrHealthcheck", err)
	}

	zeroPool := &Pool{}
	zeroPool.Close()
	if err := zeroPool.Check(context.Background()); err == nil {
		t.Fatal("zero Pool Check() error = nil, want non-nil")
	} else if !errors.Is(err, ErrHealthcheck) {
		t.Fatalf("zero Pool Check() error = %v, want ErrHealthcheck", err)
	}
}

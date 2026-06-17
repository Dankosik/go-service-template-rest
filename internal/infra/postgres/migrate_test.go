package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/fstest"
)

type closeFunc struct {
	err error
}

func (f closeFunc) Close() error {
	return f.err
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

func TestNormalizeMigrationSourcePath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		raw     string
		want    string
		wantErr string
	}{
		{name: "relative clean path", raw: "env/migrations", want: "env/migrations"},
		{name: "absolute path converted to fs path", raw: "/env/migrations", want: "env/migrations"},
		{name: "trimmed and cleaned path", raw: " ./env//migrations/ ", want: "env/migrations"},
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

func TestCloseMigrationRunnerAllowsNil(t *testing.T) {
	t.Parallel()

	if err := closeMigrationRunner(nil); err != nil {
		t.Fatalf("closeMigrationRunner(nil) error = %v, want nil", err)
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

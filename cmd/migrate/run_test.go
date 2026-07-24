package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRejectsDisabledPostgresProfile(t *testing.T) { //nolint:paralleltest // t.Chdir cannot run in parallel.
	clearAppEnvForTest(t)

	t.Chdir(t.TempDir())
	t.Setenv("APP__POSTGRES__ENABLED", "false")

	var stdout bytes.Buffer

	err := run(nil, &stdout)
	if err == nil {
		t.Fatal("run() error = nil, want required-profile rejection")
	}
	if !strings.Contains(err.Error(), "required by the DATABASE=postgres profile") {
		t.Fatalf("run() error = %q, want profile context", err)
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("run() stdout = %q, want empty", got)
	}
}

func TestRunReturnsConfigLoadError(t *testing.T) {
	clearAppEnvForTest(t)

	t.Chdir(t.TempDir())
	t.Setenv("APP__HTTP__ADDR", "")

	var stdout bytes.Buffer

	err := run(nil, &stdout)
	if err == nil {
		t.Fatal("run() error = nil, want config load error")
	}
	if !strings.Contains(err.Error(), "load config") {
		t.Fatalf("run() error = %q, want load config context", err.Error())
	}
}

func TestRunSkipsWhenNoMigrationFilesExist(t *testing.T) {
	clearAppEnvForTest(t)

	t.Chdir(t.TempDir())
	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "postgres://app:app@localhost:5432/app?sslmode=disable")

	var stdout bytes.Buffer

	if err := run(nil, &stdout); err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}
	if got := stdout.String(); got != "no migration files found; skipping migrations\n" {
		t.Fatalf("run() stdout = %q, want no-migrations message", got)
	}
}

func TestRunReturnsMigrationApplyError(t *testing.T) {
	clearAppEnvForTest(t)

	t.Chdir(t.TempDir())
	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "not-a-postgres-dsn")
	t.Setenv("MIGRATION_PATH", "missing-migrations")

	var stdout bytes.Buffer

	err := run(nil, &stdout)
	if err == nil {
		t.Fatal("run() error = nil, want migration apply error")
	}
	if !strings.Contains(err.Error(), "apply postgres migrations") {
		t.Fatalf("run() error = %q, want migration apply context", err.Error())
	}
}

func TestRunRejectsUnexpectedArguments(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := run([]string{"unexpected"}, &stdout)
	if err == nil {
		t.Fatal("run() error = nil, want usage error")
	}
	if !strings.Contains(err.Error(), "usage: migrate [validate]") {
		t.Fatalf("run() error = %q, want usage", err.Error())
	}
}

func TestResolveMigrationSourceUsesConfiguredPath(t *testing.T) { //nolint:paralleltest // t.Chdir cannot run in parallel.
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("custom/migrations", 0o755); err != nil {
		t.Fatalf("create configured migrations dir: %v", err)
	}
	t.Setenv("MIGRATION_PATH", " custom/migrations ")

	sourceFS, sourcePath, found, err := resolveMigrationSource()
	if err != nil {
		t.Fatalf("resolveMigrationSource() error = %v, want nil", err)
	}
	if !found {
		t.Fatal("resolveMigrationSource() found = false, want true for configured path")
	}
	if sourcePath != "custom/migrations" {
		t.Fatalf("resolveMigrationSource() path = %q, want %q", sourcePath, "custom/migrations")
	}
	if sourceFS == nil {
		t.Fatal("resolveMigrationSource() fs = nil, want configured local fs")
	}
	if _, err := fs.Stat(sourceFS, sourcePath); err != nil {
		t.Fatalf("resolved configured source is not readable: %v", err)
	}
}

func TestResolveMigrationSourceUsesLocalMigrationsWhenPresent(t *testing.T) { //nolint:paralleltest // t.Chdir cannot run in parallel.
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("migrations", 0o755); err != nil {
		t.Fatalf("create local migrations dir: %v", err)
	}
	if err := os.WriteFile("migrations/000001_test.up.sql", []byte("select 1;"), 0o600); err != nil {
		t.Fatalf("create local migration: %v", err)
	}

	imagePath := filepath.Join(t.TempDir(), "image-migrations")
	sourceFS, sourcePath, found, err := resolveMigrationSourceFrom(imagePath)
	if err != nil {
		t.Fatalf("resolveMigrationSourceFrom() error = %v, want nil", err)
	}
	if !found {
		t.Fatal("resolveMigrationSourceFrom() found = false, want true")
	}
	if sourcePath != localMigrationSourcePath {
		t.Fatalf("resolveMigrationSourceFrom() path = %q, want %q", sourcePath, localMigrationSourcePath)
	}
	if sourceFS == nil {
		t.Fatal("resolveMigrationSourceFrom() fs = nil, want local fs")
	}
	if _, err := fs.Stat(sourceFS, sourcePath); err != nil {
		t.Fatalf("resolved local source is not readable: %v", err)
	}
}

func TestResolveMigrationSourceFallsBackToImageMigrations(t *testing.T) { //nolint:paralleltest // t.Chdir cannot run in parallel.
	t.Chdir(t.TempDir())

	imagePath := filepath.Join(t.TempDir(), "image-migrations")
	if err := os.MkdirAll(imagePath, 0o755); err != nil {
		t.Fatalf("create image migrations dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imagePath, "000001_test.up.sql"), []byte("select 1;"), 0o600); err != nil {
		t.Fatalf("create image migration: %v", err)
	}
	sourceFS, sourcePath, found, err := resolveMigrationSourceFrom(imagePath)
	if err != nil {
		t.Fatalf("resolveMigrationSourceFrom() error = %v, want nil", err)
	}
	if !found {
		t.Fatal("resolveMigrationSourceFrom() found = false, want true")
	}
	if sourcePath != imagePath {
		t.Fatalf("resolveMigrationSourceFrom() path = %q, want %q", sourcePath, imagePath)
	}
	if sourceFS != nil {
		t.Fatalf("resolveMigrationSourceFrom() fs = %T, want nil image fs", sourceFS)
	}
}

func TestResolveMigrationSourceReportsNoMigrations(t *testing.T) {
	t.Chdir(t.TempDir())

	sourceFS, sourcePath, found, err := resolveMigrationSourceFrom("image-migrations")
	if err != nil {
		t.Fatalf("resolveMigrationSourceFrom() error = %v, want nil", err)
	}
	if found {
		t.Fatal("resolveMigrationSourceFrom() found = true, want false")
	}
	if sourceFS != nil || sourcePath != "" {
		t.Fatalf("resolveMigrationSourceFrom() = (%T, %q), want (nil, empty)", sourceFS, sourcePath)
	}
}

func TestResolveMigrationSourcePropagatesDirectoryReadFailure(t *testing.T) {
	t.Chdir(t.TempDir())

	_, _, _, err := resolveMigrationSourceFrom("\x00")
	if err == nil {
		t.Fatal("resolveMigrationSourceFrom() error = nil, want directory read failure")
	}
	if !strings.Contains(err.Error(), "inspect image migration source") {
		t.Fatalf("resolveMigrationSourceFrom() error = %q, want source context", err.Error())
	}
}

//nolint:usetesting // This helper must unset variables; t.Setenv only sets values.
func clearAppEnvForTest(t *testing.T) {
	t.Helper()

	const prefix = "APP__"

	type envState struct {
		name  string
		value string
		set   bool
	}

	var states []envState
	for _, entry := range os.Environ() {
		name, value, ok := strings.Cut(entry, "=")
		if !ok || !strings.HasPrefix(name, prefix) {
			continue
		}
		states = append(states, envState{name: name, value: value, set: true})
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset %s: %v", name, err)
		}
	}

	t.Cleanup(func() {
		for _, entry := range os.Environ() {
			name, _, ok := strings.Cut(entry, "=")
			if ok && strings.HasPrefix(name, prefix) {
				_ = os.Unsetenv(name)
			}
		}
		for _, state := range states {
			if state.set {
				_ = os.Setenv(state.name, state.value)
				continue
			}
			_ = os.Unsetenv(state.name)
		}
	})
}

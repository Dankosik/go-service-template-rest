package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSkipsMigrationsWhenPostgresDisabled(t *testing.T) { //nolint:paralleltest // t.Chdir cannot run in parallel.
	clearPrefixedEnvForTest(t, "APP__")

	t.Chdir(t.TempDir())

	var stdout bytes.Buffer

	if err := run(nil, &stdout); err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}

	if got := stdout.String(); got != "postgres is disabled; skipping migrations\n" {
		t.Fatalf("run() stdout = %q, want disabled migration message", got)
	}
}

func TestRunReturnsConfigLoadError(t *testing.T) {
	clearPrefixedEnvForTest(t, "APP__")

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

func TestRunReturnsMigrationApplyError(t *testing.T) {
	clearPrefixedEnvForTest(t, "APP__")

	t.Chdir(t.TempDir())
	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "not-a-postgres-dsn")

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

	sourceFS, sourcePath := resolveMigrationSource()
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
	if err := os.MkdirAll("env/migrations", 0o755); err != nil {
		t.Fatalf("create local migrations dir: %v", err)
	}

	imagePath := filepath.Join(t.TempDir(), "image-migrations")
	sourceFS, sourcePath := resolveMigrationSourceFrom(imagePath, localMigrationSourcePath)
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
	sourceFS, sourcePath := resolveMigrationSourceFrom(imagePath, localMigrationSourcePath)
	if sourcePath != imagePath {
		t.Fatalf("resolveMigrationSourceFrom() path = %q, want %q", sourcePath, imagePath)
	}
	if sourceFS != nil {
		t.Fatalf("resolveMigrationSourceFrom() fs = %T, want nil image fs", sourceFS)
	}
}

//nolint:usetesting // This helper must unset variables; t.Setenv only sets values.
func clearPrefixedEnvForTest(t *testing.T, prefix string) {
	t.Helper()

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

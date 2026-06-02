package main

import (
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"
)

//nolint:paralleltest // Uses process-wide cwd and APP__ environment cleanup.
func TestRunSkipsMigrationsWhenPostgresDisabled(t *testing.T) {
	clearPrefixedEnvForTest(t, "APP__")

	t.Chdir(t.TempDir())

	stdout, err := os.CreateTemp(t.TempDir(), "migrate-stdout-*")
	if err != nil {
		t.Fatalf("create stdout temp file: %v", err)
	}
	t.Cleanup(func() {
		if err := stdout.Close(); err != nil {
			t.Errorf("close stdout temp file: %v", err)
		}
	})

	if err := run(stdout); err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}

	if _, err := stdout.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("seek stdout: %v", err)
	}
	output, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if got := string(output); got != "postgres is disabled; skipping migrations\n" {
		t.Fatalf("run() stdout = %q, want disabled migration message", got)
	}
}

func TestRunReturnsConfigLoadError(t *testing.T) {
	clearPrefixedEnvForTest(t, "APP__")

	t.Chdir(t.TempDir())
	t.Setenv("APP__HTTP__ADDR", "")

	stdout, err := os.CreateTemp(t.TempDir(), "migrate-stdout-*")
	if err != nil {
		t.Fatalf("create stdout temp file: %v", err)
	}
	t.Cleanup(func() {
		if err := stdout.Close(); err != nil {
			t.Errorf("close stdout temp file: %v", err)
		}
	})

	err = run(stdout)
	if err == nil {
		t.Fatal("run() error = nil, want config load error")
	}
	if !strings.Contains(err.Error(), "load config") {
		t.Fatalf("run() error = %q, want load config context", err.Error())
	}
}

//nolint:paralleltest // Uses process-wide cwd through t.Chdir.
func TestResolveMigrationSourceUsesLocalMigrationsWhenPresent(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("env/migrations", 0o755); err != nil {
		t.Fatalf("create local migrations dir: %v", err)
	}

	sourceFS, sourcePath := resolveMigrationSource()
	if sourcePath != "env/migrations" {
		t.Fatalf("resolveMigrationSource() path = %q, want env/migrations", sourcePath)
	}
	if sourceFS == nil {
		t.Fatal("resolveMigrationSource() fs = nil, want local fs")
	}
	if _, err := fs.Stat(sourceFS, sourcePath); err != nil {
		t.Fatalf("resolved local source is not readable: %v", err)
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

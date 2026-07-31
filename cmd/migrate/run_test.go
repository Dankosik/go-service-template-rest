package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgresmigrate"
)

func TestRunRejectsArgumentsWithSafeTerminalRecord(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := run([]string{"down", "secret-canary"}, &output)
	if err == nil || !strings.Contains(err.Error(), "no arguments") {
		t.Fatalf("run() error = %v, want usage error", err)
	}
	record := decodeLastJSONRecord(t, output.Bytes())
	if record["msg"] != "migration_run_finished" ||
		record["outcome"] != "error" ||
		record["failure.stage"] != "config" {
		t.Fatalf("terminal record = %#v", record)
	}
	if strings.Contains(output.String(), "secret-canary") {
		t.Fatalf("terminal output disclosed argument: %s", output.String())
	}
}

func TestResolveMigrationSourceUsesConfiguredAbsoluteDirectory(t *testing.T) {
	directory := t.TempDir()
	t.Setenv(migrationPathEnv, directory)

	source, sourcePath, err := resolveMigrationSource()
	if err != nil {
		t.Fatalf("resolveMigrationSource() error = %v", err)
	}
	if filepath.IsAbs(sourcePath) {
		t.Fatalf("sourcePath = %q, want fs-relative path", sourcePath)
	}
	if _, err := fs.ReadDir(source, sourcePath); err != nil {
		t.Fatalf("resolved source cannot read directory: %v", err)
	}
}

func TestResolveMigrationSourceUsesConfiguredRelativeDirectory(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)
	if err := os.Mkdir("owned_migrations", 0o750); err != nil {
		t.Fatalf("mkdir owned_migrations: %v", err)
	}
	t.Setenv(migrationPathEnv, "owned_migrations")

	source, sourcePath, err := resolveMigrationSource()
	if err != nil {
		t.Fatalf("resolveMigrationSource() error = %v", err)
	}
	if sourcePath != "owned_migrations" {
		t.Fatalf("sourcePath = %q, want owned_migrations", sourcePath)
	}
	if _, err := fs.ReadDir(source, sourcePath); err != nil {
		t.Fatalf("resolved source cannot read directory: %v", err)
	}
}

func TestResolveMigrationSourceRejectsMissingConfiguredDirectory(t *testing.T) {
	t.Setenv(migrationPathEnv, filepath.Join(t.TempDir(), "missing"))
	_, _, err := resolveMigrationSource()
	if !errors.Is(err, errMigrationSourceMissing) {
		t.Fatalf("resolveMigrationSource() error = %v, want errMigrationSourceMissing", err)
	}
}

func TestResolveMigrationSourceRejectsConfiguredFile(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "migration.sql")
	if err := os.WriteFile(filename, []byte("SELECT 1"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	t.Setenv(migrationPathEnv, filename)
	_, _, err := resolveMigrationSource()
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("resolveMigrationSource() error = %v, want directory error", err)
	}
}

func TestLogMigrationTerminalDoesNotDiscloseWrappedError(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := newJSONLogger(&output)
	sensitive := errors.New("postgres://user:secret-canary@db/app SELECT secret_canary")
	err := &postgresmigrate.RunError{
		Stage: postgresmigrate.FailureExecute,
		Err:   sensitive,
	}
	logMigrationTerminal(
		logger,
		postgresmigrate.RunResult{
			Before:   1,
			Target:   2,
			After:    1,
			Duration: time.Second,
			Failed:   &postgresmigrate.MigrationResult{Version: 2, Filename: "000002_add.sql"},
		},
		err,
		postgresmigrate.FailureStageOf(err),
	)

	if strings.Contains(output.String(), "secret-canary") || strings.Contains(output.String(), "SELECT") {
		t.Fatalf("terminal output disclosed wrapped cause: %s", output.String())
	}
	record := decodeLastJSONRecord(t, output.Bytes())
	if record["failure.stage"] != "execute" ||
		record["recovery.document"] != recoveryDocument ||
		record["outcome"] != "error" {
		t.Fatalf("terminal record = %#v", record)
	}
}

func TestContextFailureClass(t *testing.T) {
	t.Parallel()

	if got := contextFailureClass(context.Canceled); got != "canceled" {
		t.Fatalf("contextFailureClass(canceled) = %q", got)
	}
	if got := contextFailureClass(context.DeadlineExceeded); got != "deadline_exceeded" {
		t.Fatalf("contextFailureClass(deadline) = %q", got)
	}
	if got := contextFailureClass(errors.New("other")); got != "none" {
		t.Fatalf("contextFailureClass(other) = %q", got)
	}
}

func newJSONLogger(output *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(output, nil))
}

func decodeLastJSONRecord(t *testing.T, data []byte) map[string]any {
	t.Helper()

	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(lines) == 0 {
		t.Fatal("no JSON log records")
	}
	var record map[string]any
	if err := json.Unmarshal(lines[len(lines)-1], &record); err != nil {
		t.Fatalf("decode terminal JSON: %v\n%s", err, data)
	}
	return record
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
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

//nolint:paralleltest // This test mutates process-global environment or working directory.
func TestLogMigrationTerminalDoesNotDiscloseWrappedError(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
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

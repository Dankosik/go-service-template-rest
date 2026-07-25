package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
)

func TestRecordDependencyProbeRejectionLogsRootCause(t *testing.T) {
	t.Parallel()

	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))
	rootCause := errors.New("postgres probe connection refused")
	ctx, span := otel.Tracer("test").Start(context.Background(), "dependency-probe-log")
	runtime := postgresStartupRuntime{
		tracer:        otel.Tracer("test"),
		bootstrapSpan: span,
		log:           logger,
	}

	recordDependencyProbeRejection(ctx, runtime, rootCause)
	span.End()

	logLine := logBuffer.String()
	if !strings.Contains(logLine, `"msg":"startup_blocked"`) {
		t.Fatalf("dependency probe rejection log = %q, want startup_blocked message", logLine)
	}
	if !strings.Contains(logLine, `"dependency":"postgres"`) {
		t.Fatalf("dependency probe rejection log = %q, want postgres dependency", logLine)
	}
	if !strings.Contains(logLine, `"operation":"postgres_probe"`) {
		t.Fatalf("dependency probe rejection log = %q, want probe operation", logLine)
	}
	if !strings.Contains(logLine, `"err":"postgres probe connection refused"`) {
		t.Fatalf("dependency probe rejection log = %q, want root cause err", logLine)
	}
}

func TestRejectPostgresStartupForDependencyInitLogsRootCause(t *testing.T) {
	t.Parallel()

	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))
	rootCause := errors.New("postgres dsn is empty")

	ctx, span := otel.Tracer("test").Start(context.Background(), "dependency-init-log")
	err := rejectPostgresStartupForDependencyInit(ctx, span, logger, rootCause)
	span.End()

	if err == nil {
		t.Fatal("rejectPostgresStartupForDependencyInit() error = nil, want non-nil")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("rejectPostgresStartupForDependencyInit() error = %v, want wrapped dependency init sentinel", err)
	}
	if !strings.Contains(err.Error(), "postgres dsn is empty") {
		t.Fatalf("rejectPostgresStartupForDependencyInit() error = %v, want root cause detail", err)
	}

	logLine := logBuffer.String()
	if !strings.Contains(logLine, `"msg":"startup_blocked"`) {
		t.Fatalf("dependency init rejection log = %q, want startup_blocked message", logLine)
	}
	if !strings.Contains(logLine, `"dependency":"postgres"`) {
		t.Fatalf("dependency init rejection log = %q, want postgres dependency", logLine)
	}
	if !strings.Contains(logLine, "postgres dsn is empty") {
		t.Fatalf("dependency init rejection log = %q, want root cause err", logLine)
	}
}

func TestRejectPostgresStartupForDependencyInitDoesNotDuplicateSentinel(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	cause := fmt.Errorf("%w: postgres init failed: %w", errDependencyInit, errors.New("dsn rejected"))

	ctx, span := otel.Tracer("test").Start(context.Background(), "dependency-init-idempotent")
	err := rejectPostgresStartupForDependencyInit(ctx, span, logger, cause)
	span.End()

	if err == nil {
		t.Fatal("rejectPostgresStartupForDependencyInit() error = nil, want non-nil")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("rejectPostgresStartupForDependencyInit() error = %v, want wrapped dependency init sentinel", err)
	}
	if count := strings.Count(err.Error(), errDependencyInit.Error()); count != 1 {
		t.Fatalf("rejectPostgresStartupForDependencyInit() error = %v, dependency init count = %d, want 1", err, count)
	}
	if !strings.Contains(err.Error(), "dsn rejected") {
		t.Fatalf("rejectPostgresStartupForDependencyInit() error = %v, want original cause detail", err)
	}
}

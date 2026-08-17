package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
	"go.opentelemetry.io/otel"
)

// TestProcessLoggerCorrelatesRecords proves the wiring rather than the decorator,
// which internal/observability/logctx already covers on its own. What can regress
// here is logctx.NewProcessLogger being rebuilt from a bare JSONHandler: the records look
// identical minus the two keys that join a startup failure to its span, and no
// other test would notice.
//
//nolint:paralleltest // Installs a process-wide tracer provider for span capture.
func TestProcessLoggerCorrelatesRecords(t *testing.T) {
	t.Parallel()
	telemetrytest.InstallSpanRecorder(t)

	var out bytes.Buffer
	log := logctx.NewProcessLogger(&out, slog.LevelInfo)

	ctx, span := otel.Tracer("test").Start(context.Background(), "startup-log-test")
	log.InfoContext(ctx, "startup_stage", startupLogArgs("c", "o", "ok", "k", "v")...)
	span.End()

	var record map[string]any
	if err := json.Unmarshal(out.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal record %q: %v", out.String(), err)
	}

	for _, key := range []string{"trace_id", "span_id"} {
		value, ok := record[key].(string)
		if !ok || value == "" {
			t.Fatalf("record %v is missing %s", record, key)
		}
	}
	for key, want := range map[string]any{"component": "c", "operation": "o", "outcome": "ok", "k": "v"} {
		if got := record[key]; got != want {
			t.Fatalf("record %s = %v, want %v", key, got, want)
		}
	}
}

// TestStartupLogArgsCarriesNoCorrelation pins the deduplication: the helper builds
// stage attributes only, and the logger owns correlation. Adding trace keys back
// here would emit each one twice on every startup record.
func TestStartupLogArgsCarriesNoCorrelation(t *testing.T) {
	t.Parallel()

	args := startupLogArgs("c", "o", "ok", "k", "v")
	for i := 0; i < len(args)-1; i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		switch key {
		case "trace_id", "span_id", "request_id":
			t.Fatalf("startupLogArgs published %q, which the logger already adds", key)
		}
	}
}

func TestReadinessTransitionLogCarriesStateAndCause(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	log := logctx.NewProcessLogger(&out, slog.LevelInfo)
	logReadinessTransition(context.Background(), log, errors.New("db probe failed"))
	logReadinessTransition(context.Background(), log, nil)

	decoder := json.NewDecoder(&out)
	for _, want := range []struct {
		level   string
		outcome string
		err     string
	}{
		{level: "WARN", outcome: "unhealthy", err: "db probe failed"},
		{level: "INFO", outcome: "healthy"},
	} {
		var record map[string]any
		if err := decoder.Decode(&record); err != nil {
			t.Fatalf("decode readiness record: %v", err)
		}
		for key, value := range map[string]any{
			"msg":       "readiness_transition",
			"level":     want.level,
			"component": "readiness",
			"operation": "refresh",
			"outcome":   want.outcome,
		} {
			if got := record[key]; got != value {
				t.Fatalf("record %s = %v, want %v", key, got, value)
			}
		}
		if got, _ := record["err"].(string); got != want.err {
			t.Fatalf("record err = %q, want %q", got, want.err)
		}
	}
}

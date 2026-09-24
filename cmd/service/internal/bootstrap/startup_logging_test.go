package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/example/go-service-template-rest/internal/observability/logctx"
)

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

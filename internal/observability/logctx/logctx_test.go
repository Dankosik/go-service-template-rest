package logctx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/example/go-service-template-rest/internal/observability/logctx"
	"github.com/example/go-service-template-rest/internal/reqctx"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func newTestLogger(t *testing.T) (*slog.Logger, *bytes.Buffer) {
	t.Helper()

	var out bytes.Buffer
	return logctx.NewProcessLogger(&out, slog.LevelInfo), &out
}

func decodeRecord(t *testing.T, out *bytes.Buffer) map[string]any {
	t.Helper()

	var record map[string]any
	if err := json.Unmarshal(out.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal record %q: %v", out.String(), err)
	}
	return record
}

func acceptedRequestIDContext(requestID string) context.Context {
	ctx, _ := reqctx.ContextWithAcceptedRequestID(context.Background(), requestID)
	return ctx
}

func TestHandlerAddsRequestID(t *testing.T) {
	t.Parallel()

	log, out := newTestLogger(t)
	ctx := acceptedRequestIDContext("req-1")

	log.InfoContext(ctx, "handled")

	if got := decodeRecord(t, out)["request_id"]; got != "req-1" {
		t.Fatalf("request_id = %v, want %q", got, "req-1")
	}
}

func TestHandlerAddsTraceIdentifiers(t *testing.T) {
	t.Parallel()

	provider := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown test tracer provider: %v", err)
		}
	})

	log, out := newTestLogger(t)
	ctx, span := provider.Tracer("logctx-test").Start(context.Background(), "operation")
	defer span.End()

	log.ErrorContext(ctx, "failed")

	record := decodeRecord(t, out)
	wantTraceID := span.SpanContext().TraceID().String()
	wantSpanID := span.SpanContext().SpanID().String()
	if got := record["trace_id"]; got != wantTraceID {
		t.Fatalf("trace_id = %v, want %q", got, wantTraceID)
	}
	if got := record["span_id"]; got != wantSpanID {
		t.Fatalf("span_id = %v, want %q", got, wantSpanID)
	}
}

// TestHandlerSurvivesWith is the first of the two regressions that matter.
// WithAttrs is promoted from an embedded handler, so a decorator that does not
// override it loses correlation on every logger a service derives with its own
// component fields — silently, and only on the loggers actually used in handlers.
func TestHandlerSurvivesWith(t *testing.T) {
	t.Parallel()

	log, out := newTestLogger(t)
	ctx := acceptedRequestIDContext("req-2")

	log.With("component", "orders").InfoContext(ctx, "queried")

	record := decodeRecord(t, out)
	if got := record["request_id"]; got != "req-2" {
		t.Fatalf("request_id after With = %v, want %q", got, "req-2")
	}
	if got := record["component"]; got != "orders" {
		t.Fatalf("component = %v, want %q", got, "orders")
	}
}

// TestHandlerKeepsCorrelationOutsideGroups is the second, and it is the one a
// naive decorator fails. slog nests every attribute added after WithGroup inside
// that group, so a record from a logger derived with WithGroup("db") carried
// `db.request_id` while every other record carried `request_id` — and the log
// query that found one found none of the other.
func TestHandlerKeepsCorrelationOutsideGroups(t *testing.T) {
	t.Parallel()

	log, out := newTestLogger(t)
	ctx := acceptedRequestIDContext("req-3")

	log.With("component", "orders").WithGroup("db").InfoContext(ctx, "queried", "table", "orders")

	record := decodeRecord(t, out)
	if got := record["request_id"]; got != "req-3" {
		t.Fatalf("top-level request_id = %v, want %q", got, "req-3")
	}
	if got := record["component"]; got != "orders" {
		t.Fatalf("component = %v, want %q", got, "orders")
	}
	group, ok := record["db"].(map[string]any)
	if !ok {
		t.Fatalf("record is missing the db group: %v", record)
	}
	if got := group["table"]; got != "orders" {
		t.Fatalf("db.table = %v, want %q", got, "orders")
	}
	if _, nested := group["request_id"]; nested {
		t.Fatalf("correlation was nested inside the db group: %v", group)
	}
}

// TestHandlerNestsGroupsInOrder keeps the replay honest: rebuilding the caller's
// derivation on top of the correlation must not reorder or drop the groups and
// attributes it was built from.
func TestHandlerNestsGroupsInOrder(t *testing.T) {
	t.Parallel()

	log, out := newTestLogger(t)
	ctx := acceptedRequestIDContext("req-4")

	log.WithGroup("outer").With("a", 1).WithGroup("inner").InfoContext(ctx, "nested", "b", 2)

	record := decodeRecord(t, out)
	if got := record["request_id"]; got != "req-4" {
		t.Fatalf("top-level request_id = %v, want %q", got, "req-4")
	}
	outer, ok := record["outer"].(map[string]any)
	if !ok {
		t.Fatalf("record is missing the outer group: %v", record)
	}
	if got := outer["a"]; got != float64(1) {
		t.Fatalf("outer.a = %v, want 1", got)
	}
	inner, ok := outer["inner"].(map[string]any)
	if !ok {
		t.Fatalf("outer group is missing the inner group: %v", outer)
	}
	if got := inner["b"]; got != float64(2) {
		t.Fatalf("outer.inner.b = %v, want 2", got)
	}
}

// TestHandlerRespectsLevel keeps Enabled delegated. A decorator that answered on
// its own would either log below the configured level or silence records the
// handler would have written.
func TestHandlerRespectsLevel(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	log := logctx.NewProcessLogger(&out, slog.LevelWarn)

	log.InfoContext(acceptedRequestIDContext("req-5"), "below level")

	if out.Len() != 0 {
		t.Fatalf("record below the configured level was written: %q", out.String())
	}
}

// TestHandlerOmitsAbsentCorrelation keeps startup and background records clean:
// a log line outside a request must not carry empty correlation fields that look
// like a lost identifier.
func TestHandlerOmitsAbsentCorrelation(t *testing.T) {
	t.Parallel()

	log, out := newTestLogger(t)

	log.InfoContext(context.Background(), "started")

	record := decodeRecord(t, out)
	for _, key := range []string{"request_id", "trace_id", "span_id"} {
		if value, present := record[key]; present {
			t.Fatalf("record carries %s = %v without a request or span", key, value)
		}
	}
}

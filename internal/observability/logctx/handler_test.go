package logctx

import (
	"context"
	"log/slog"
	"slices"
	"testing"

	"github.com/example/go-service-template-rest/internal/reqctx"
)

type consumingHandler struct {
	attrs   []slog.Attr
	records *[][]slog.Attr
}

func (consumingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h consumingHandler) Handle(context.Context, slog.Record) error {
	*h.records = append(*h.records, slices.Clone(h.attrs))
	return nil
}

func (h consumingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.attrs = append(slices.Clone(h.attrs), attrs...)
	clear(attrs)
	return h
}

func (h consumingHandler) WithGroup(string) slog.Handler { return h }

func TestHandlerKeepsOwnershipOfReplayedAttrs(t *testing.T) {
	t.Parallel()

	var records [][]slog.Attr
	next := consumingHandler{records: &records}
	log := slog.New(handler{base: next, derived: next})
	ctx, _ := reqctx.ContextWithAcceptedRequestID(t.Context(), "req-ownership")
	grouped := log.With("component", "orders").WithGroup("db")

	grouped.InfoContext(ctx, "first")
	grouped.InfoContext(ctx, "second")

	if len(records) != 2 {
		t.Fatalf("handled records = %d, want 2", len(records))
	}
	for index, attrs := range records {
		if !slices.ContainsFunc(attrs, func(attr slog.Attr) bool {
			return attr.Key == "component" && attr.Value.String() == "orders"
		}) {
			t.Fatalf("record %d attrs = %v, want component=orders", index, attrs)
		}
	}
}

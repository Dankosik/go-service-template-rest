// Package logctx attaches request correlation to every log record.
//
// It is a leaf for the same reason internal/reqctx and
// internal/observability/otelconfig are: the composition root installs it, and
// every layer above and below logs through it.
//
// Without it, correlation is something each call site remembers to add, and a
// record missing the ids cannot be joined to the trace that explains it except by
// timestamp — a guess that is wrong under any concurrency.
//
// The one thing a caller must do is pass a context: slog's Info/Error take none
// and hand the handler context.Background(), which carries no request. The
// repository's sloglint `context: scope` setting enforces that.
package logctx

import (
	"context"
	"io"
	"log/slog"
	"slices"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/otel/trace"
)

// NewProcessLogger builds the JSON logger every binary in this repository writes
// through: slog's JSON handler wrapped in this package's correlating handler.
//
// Correlation is the whole reason it is one constructor. A logger built without
// New here still logs, and silently drops request_id, trace_id, and span_id from
// every record for the life of the process — a loss no test of the surrounding
// code would notice.
//
// Service identity is deliberately not attached here. The bootstrap logger a
// binary uses before its configuration loads must state that it does not know the
// version yet, so the With call belongs at each composition root.
func NewProcessLogger(out io.Writer, level slog.Level) *slog.Logger {
	return slog.New(New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: level})))
}

// Attribute keys published on every record that has them available. They match
// the keys the access log already used, so existing log queries keep working.
const (
	requestIDKey = "request_id"
	traceIDKey   = "trace_id"
	spanIDKey    = "span_id"
)

// correlationAttrCount is how many attributes a fully correlated record gains.
const correlationAttrCount = 3

// operation is one WithAttrs or WithGroup call a caller made on the logger. Both
// are recorded rather than applied eagerly, because a group changes where later
// attributes land and the correlation must not land inside one.
type operation struct {
	attrs []slog.Attr
	group string
}

type handler struct {
	// base carries none of the caller's operations, so correlation applied to it
	// is published at the top level of the record.
	base slog.Handler
	// derived carries all of them, and formats the record when the correlation
	// needs no special placement.
	derived slog.Handler
	ops     []operation
	// grouped reports whether any operation opened a group, which is the only
	// case that cannot use derived directly.
	grouped bool
}

// New wraps next so every record it handles carries the request identifier and
// trace identifiers of the context it was logged with.
//
// A nil next returns nil, so a caller cannot accidentally build a logger around
// nothing.
func New(next slog.Handler) slog.Handler {
	if next == nil {
		return nil
	}
	return handler{base: next, derived: next}
}

func (h handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.derived.Enabled(ctx, level)
}

func (h handler) Handle(ctx context.Context, record slog.Record) error {
	attrs := correlationAttrs(ctx)

	switch {
	case len(attrs) == 0:
		// Startup, shutdown, and background records have no request and no span.
		// They must not carry empty correlation keys that read as a lost
		// identifier.
		//nolint:wrapcheck // A handler decorator passes the underlying handler's error through unchanged.
		return h.derived.Handle(ctx, record)

	case !h.grouped:
		// No group is open, so nothing can nest these: adding them to the record
		// is equivalent to applying them to the handler and costs no rebuild.
		// This is the path every logger in this repository takes.
		record.AddAttrs(attrs...)
		//nolint:wrapcheck // As above.
		return h.derived.Handle(ctx, record)

	default:
		// A group is open, so added attributes would nest inside it —
		// `db.request_id` for a logger derived with WithGroup("db"), which no log
		// query matching `request_id` would find. Applying the correlation to the
		// ungrouped base and replaying the caller's operations on top keeps the
		// keys where every query expects them.
		//nolint:wrapcheck // As above.
		return h.replay(attrs).Handle(ctx, record)
	}
}

// WithAttrs and WithGroup have to be overridden, and this is the part that is
// easy to get wrong. Both would otherwise be promoted from an embedded handler,
// so a logger derived through slog.Logger.With would return the inner handler and
// the correlation would silently stop — on exactly the loggers a service builds
// when it adds its own component fields.
func (h handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return handler{
		base:    h.base,
		derived: h.derived.WithAttrs(attrs),
		ops:     append(slices.Clone(h.ops), operation{attrs: slices.Clone(attrs)}),
		grouped: h.grouped,
	}
}

func (h handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return handler{
		base:    h.base,
		derived: h.derived.WithGroup(name),
		ops:     append(slices.Clone(h.ops), operation{group: name}),
		grouped: true,
	}
}

// replay rebuilds the caller's derivation on top of the correlation attributes.
func (h handler) replay(attrs []slog.Attr) slog.Handler {
	target := h.base.WithAttrs(attrs)
	for _, op := range h.ops {
		if op.group != "" {
			target = target.WithGroup(op.group)
			continue
		}
		target = target.WithAttrs(op.attrs)
	}
	return target
}

func correlationAttrs(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}

	attrs := make([]slog.Attr, 0, correlationAttrCount)
	if requestID := reqctx.RequestID(ctx); requestID != "" {
		attrs = append(attrs, slog.String(requestIDKey, requestID))
	}
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		attrs = append(attrs,
			slog.String(traceIDKey, spanContext.TraceID().String()),
			slog.String(spanIDKey, spanContext.SpanID().String()),
		)
	}
	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

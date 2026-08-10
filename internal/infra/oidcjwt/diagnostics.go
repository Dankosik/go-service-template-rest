package oidcjwt

import (
	"context"
	"log/slog"
	"runtime/debug"

	"github.com/example/go-service-template-rest/internal/observability/logctx"
)

// logRecoveredPanic reports a panic this package converted into a sanitized
// failure, which the metric alone cannot locate: it says a panic occurred, and
// this says where the service defect is.
//
// It is the only record of where such a panic came from, because both
// converters answer before any transport recovery could see it. The shared
// panic-attribute policy withholds the panic's value while
// publishing its type, class, and stack, so providerError's redaction rule
// reaches logs as much as errors.
func logRecoveredPanic(ctx context.Context, log *slog.Logger, operation string, recovered any) {
	if log == nil {
		log = slog.Default()
	}
	// debug.Stack is taken here, inside the caller's deferred recovery, because
	// that is the only point the panicking frames still exist.
	log.ErrorContext(
		ctx,
		"authn_panic_recovered",
		append(
			[]any{"component", "authn", "authn.operation", operation},
			logctx.PanicAttrs(recovered, debug.Stack())...,
		)...,
	)
}

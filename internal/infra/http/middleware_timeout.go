package httpx

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/example/go-service-template-rest/internal/problem"
)

// RequestTimeout bounds how long one request may occupy a handler.
//
// The net/http server timeouts do not cover this. ReadTimeout and WriteTimeout
// are connection deadlines, and Go documents WriteTimeout as one that "does not
// let Handlers make decisions on a per-request basis" — neither cancels
// r.Context(). Without this budget a handler blocked on a slow dependency keeps
// its goroutine, its pooled database connection, and its memory long after the
// client is gone, and a drain then waits http.shutdown_timeout for work nobody
// is receiving.
//
// The cancellation is the protection. Every adapter in this repository takes a
// context, so an expired budget unwinds the outbound call, the query, and the
// handler with it. A handler that ignores its context cannot be stopped from
// here — running it on another goroutine, as http.TimeoutHandler does, would
// return early while leaking that goroutine anyway, and would buffer every
// response to do it.
//
// The response here is a fallback: it is written only when the handler returned
// without committing one. A generated strict-server operation that returns its
// expired context commits a response of its own first, so that path is mapped to
// the same problem in generatedStrictServerOptions.
//
// Expiry maps to 504 rather than the 503 that http.TimeoutHandler uses, because
// this service already answers 503 for "not ready" — keeping the two apart lets
// an operator tell a draining or unhealthy instance from one whose dependencies
// went slow.
func RequestTimeout(timeout time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if timeout <= 0 {
			next.ServeHTTP(w, r)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		// The deadline is installed on the caller's request rather than on a
		// copy, because chi assigns the matched route to r.Pattern in place
		// (chi/v5 mux.go:481) and otelhttp reads that field back off the request
		// it handed down (otelhttp handler.go:187). A middleware here that
		// forwarded r.WithContext(ctx) would give routing a different struct to
		// write to, and every span and metric would silently lose http.route —
		// which is exactly what collapses 404s and 405s into per-method buckets.
		// RequestBodyLimit already mutates the same request for the same reason.
		*r = *r.WithContext(ctx)

		trackedWriter, committed := trackResponseCommit(w)
		next.ServeHTTP(trackedWriter, r)

		if committed() || !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return
		}
		writeProblem(w, r, problemResponse{
			code:   problem.CodeGatewayTimeout,
			detail: "request exceeded its time budget",
		})
	})
}

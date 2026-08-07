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
// The net/http server timeouts do not cover this: ReadTimeout and WriteTimeout
// are connection deadlines and neither cancels r.Context(), so a handler blocked
// on a slow dependency keeps its goroutine, its pooled connection, and its memory
// long after the client is gone.
//
// The cancellation is the protection, not the response. A handler that ignores
// its context cannot be stopped from here — running it on another goroutine, as
// http.TimeoutHandler does, would return early while leaking that goroutine
// anyway. The response is a fallback, written only when the handler returned
// without committing one; a generated strict-server operation commits its own
// first, and generatedStrictServerOptions maps that path to the same problem.
//
// Expiry maps to 504 rather than http.TimeoutHandler's 503, because this service
// already answers 503 for "not ready" — keeping the two apart lets an operator
// tell a draining instance from one whose dependencies went slow.
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

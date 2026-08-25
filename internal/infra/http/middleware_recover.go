package httpx

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/example/go-service-template-rest/internal/failure"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
	"github.com/example/go-service-template-rest/internal/problem"
)

func Recover(log *slog.Logger, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trackedWriter, committed := trackResponseCommit(w)
		defer func(ctx context.Context, request *http.Request) {
			rec := recover()
			if rec == nil {
				return
			}
			// http.ErrAbortHandler is the standard way to abandon a response on
			// purpose: net/http's own recovery suppresses it, and
			// httputil.ReverseProxy raises it. Recovering it here instead would turn
			// every deliberate abort into an ERROR log with a full stack trace and a
			// spurious 500 attempt, at traffic volume. net/http suppresses its own panic
			// log only for the exact sentinel, so canonicalize wrappers here.
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(http.ErrAbortHandler)
			}

			route := cmp.Or(joinMethodAndPattern(request.Method, routePathTemplateForRequest(request)), "<unmatched>")
			// debug.Stack is taken here, inside the deferred recovery, because
			// that is the only point the panicking frames still exist.
			log.ErrorContext(
				ctx,
				"http_panic_recovered",
				append(
					[]any{"method", request.Method, "route", route},
					logctx.PanicAttrs(rec, debug.Stack())...,
				)...,
			)
			if committed() {
				return
			}
			writeProblem(w, r, problemResponse{code: problem.CodeInternalError, detail: failure.SanitizedDetail})
		}(r.Context(), r)
		next.ServeHTTP(trackedWriter, r)
	})
}

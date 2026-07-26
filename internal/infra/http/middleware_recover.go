package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"runtime/debug"

	"github.com/example/go-service-template-rest/internal/problem"
)

func Recover(log *slog.Logger, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trackedWriter, committed := trackResponseCommit(w)
		defer func(ctx context.Context, method, path string) {
			rec := recover()
			if rec == nil {
				return
			}
			// http.ErrAbortHandler is the standard way to abandon a response on
			// purpose: net/http's own recovery suppresses it, and
			// httputil.ReverseProxy raises it. Re-panicking hands it back to the
			// server unchanged. Recovering it here instead would turn every
			// deliberate abort into an ERROR log with a full stack trace and a
			// spurious 500 attempt, at traffic volume.
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}

			log.ErrorContext(
				ctx,
				"panic recovered",
				"panic_class", panicClass(rec),
				"panic_type", fmt.Sprintf("%T", rec),
				"method", method,
				"path", path,
				"stack", string(debug.Stack()),
			)
			if committed() {
				return
			}
			writeProblem(w, r, problemResponse{code: problem.CodeInternalError, detail: "request failed"})
		}(r.Context(), r.Method, r.URL.Path)
		next.ServeHTTP(trackedWriter, r)
	})
}

func panicClass(rec any) string {
	switch rec.(type) {
	case runtime.Error:
		return "runtime_error"
	case error:
		return "error"
	case string:
		return "string"
	default:
		return "value"
	}
}

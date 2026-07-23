package httpx

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime"

	"github.com/felixge/httpsnoop"
)

func Recover(log *slog.Logger, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		committed := false
		trackedWriter := httpsnoop.Wrap(w, httpsnoop.Hooks{
			WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
				return func(code int) {
					committed = true
					next(code)
				}
			},
			Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
				return func(b []byte) (int, error) {
					committed = true
					return next(b)
				}
			},
			Flush: func(next httpsnoop.FlushFunc) httpsnoop.FlushFunc {
				return func() {
					committed = true
					next()
				}
			},
			ReadFrom: func(next httpsnoop.ReadFromFunc) httpsnoop.ReadFromFunc {
				return func(src io.Reader) (int64, error) {
					committed = true
					return next(src)
				}
			},
		})
		defer func(ctx context.Context, method, path string) {
			if rec := recover(); rec != nil {
				traceID, spanID := traceIDsFromContext(ctx)
				log.Error(
					"panic recovered",
					"panic_class", panicClass(rec),
					"panic_type", fmt.Sprintf("%T", rec),
					"method", method,
					"path", path,
					"request_id", requestIDFromContext(ctx),
					"trace_id", traceID,
					"span_id", spanID,
				)
				if committed {
					return
				}
				writeProblem(w, r, problemResponse{code: problemCodeInternalError, detail: "request failed"})
			}
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

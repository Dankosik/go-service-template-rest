package httpx

import (
	"io"
	"net/http"

	"github.com/felixge/httpsnoop"
)

// trackResponseCommit wraps w and reports whether the response has been
// committed — a status write, a body write, a flush, or a ReadFrom.
//
// Two middlewares need this and neither may replace a response the handler has
// already started sending: Recover cannot turn a half-written 200 into a 500,
// and RequestTimeout cannot turn a streamed response into a 504. The returned
// predicate is read on the same goroutine that served the request.
func trackResponseCommit(w http.ResponseWriter) (http.ResponseWriter, func() bool) {
	committed := false

	wrapped := httpsnoop.Wrap(w, httpsnoop.Hooks{
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

	return wrapped, func() bool { return committed }
}

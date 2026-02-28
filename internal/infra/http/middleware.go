package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func Recover(log *slog.Logger, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic recovered", "panic", rec, "method", r.Method, "path", r.URL.Path)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func AccessLog(log *slog.Logger, metrics *telemetry.Metrics, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}

		next.ServeHTTP(sw, r)

		if sw.status == 0 {
			sw.status = http.StatusOK
		}

		route := r.Pattern
		if route == "" {
			route = "<unmatched>"
		}

		log.Info(
			"request",
			"method", r.Method,
			"path", r.URL.Path,
			"route", route,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)

		if metrics != nil {
			metrics.ObserveHTTPRequest(r.Method, route, sw.status)
		}
	})
}

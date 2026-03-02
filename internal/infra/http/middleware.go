package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const requestIDHeader = "X-Request-ID"
const contentTypeOptionsHeader = "X-Content-Type-Options"

type requestIDContextKey struct{}

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

func RequestCorrelation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get(requestIDHeader))
		if requestID == "" {
			requestID = newRequestID()
		}

		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		if !trace.SpanContextFromContext(ctx).IsValid() {
			ctx = propagation.TraceContext{}.Extract(ctx, propagation.HeaderCarrier(r.Header))
		}
		r = r.WithContext(ctx)

		w.Header().Set(requestIDHeader, requestID)
		next.ServeHTTP(w, r)
	})
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeOptionsHeader, "nosniff")
		next.ServeHTTP(w, r)
	})
}

func RequestFramingGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hasTransferEncoding := len(r.TransferEncoding) > 0 || strings.TrimSpace(r.Header.Get("Transfer-Encoding")) != ""
		hasContentLength := strings.TrimSpace(r.Header.Get("Content-Length")) != ""
		if hasTransferEncoding && hasContentLength {
			w.Header().Set("Connection", "close")
			writeProblem(w, r, http.StatusBadRequest, "bad request", "invalid request framing")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RequestBodyLimit(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if maxBytes <= 0 {
			next.ServeHTTP(w, r)
			return
		}
		if r.ContentLength > maxBytes {
			writeProblem(w, r, http.StatusRequestEntityTooLarge, "request entity too large", "request body exceeds limit")
			return
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		}
		next.ServeHTTP(w, r)
	})
}

func Recover(log *slog.Logger, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func(ctx context.Context, method, path string) {
			if rec := recover(); rec != nil {
				traceID, spanID := traceIDsFromContext(ctx)
				log.Error(
					"panic recovered",
					"panic", rec,
					"method", method,
					"path", path,
					"request_id", requestIDFromContext(ctx),
					"trace_id", traceID,
					"span_id", spanID,
				)
				writeProblem(w, r, http.StatusInternalServerError, "internal server error", "request failed")
			}
		}(r.Context(), r.Method, r.URL.Path)
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
		duration := time.Since(start)

		if sw.status == 0 {
			sw.status = http.StatusOK
		}

		route := r.Pattern
		if route == "" {
			route = "<unmatched>"
		}

		traceID, spanID := traceIDsFromContext(r.Context())
		log.Info(
			"request",
			"method", r.Method,
			"path", r.URL.Path,
			"route", route,
			"status", sw.status,
			"duration_ms", duration.Milliseconds(),
			"request_id", requestIDFromContext(r.Context()),
			"trace_id", traceID,
			"span_id", spanID,
		)

		if metrics != nil {
			metrics.ObserveHTTPRequest(r.Method, route, sw.status)
			metrics.ObserveHTTPRequestDuration(r.Method, route, sw.status, duration)
		}
	})
}

func requestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDContextKey{}).(string); ok {
		return requestID
	}
	return ""
}

func traceIDsFromContext(ctx context.Context) (string, string) {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return "", ""
	}
	return spanContext.TraceID().String(), spanContext.SpanID().String()
}

func newRequestID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf[:])
}

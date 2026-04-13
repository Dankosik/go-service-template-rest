package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/felixge/httpsnoop"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	requestIDHeader          = "X-Request-ID"
	contentTypeOptionsHeader = "X-Content-Type-Options"
	maxRequestIDLength       = 128
)

type requestIDContextKey struct{}
type routeLabelContextKey struct{}

type routeLabelHolder struct {
	value string
}

func RequestCorrelation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := requestIDFromHeader(r.Header.Get(requestIDHeader))
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
				writeProblem(w, r, http.StatusInternalServerError, "internal server error", "request failed")
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

func AccessLog(log *slog.Logger, metrics *telemetry.Metrics, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routeHolder := &routeLabelHolder{}
		ctxWithRouteHolder := context.WithValue(r.Context(), routeLabelContextKey{}, routeHolder)

		captured := httpsnoop.CaptureMetricsFn(w, func(capturedWriter http.ResponseWriter) {
			next.ServeHTTP(capturedWriter, r.WithContext(ctxWithRouteHolder))
		})

		route := routeHolder.value
		if route == "" {
			route = routeLabelForRequest(r)
		}
		if route == "" {
			route = "<unmatched>"
		}

		traceID, spanID := traceIDsFromContext(r.Context())
		log.Info(
			"request",
			"method", r.Method,
			"path", r.URL.Path,
			"route", route,
			"status", captured.Code,
			"duration_ms", captured.Duration.Milliseconds(),
			"request_id", requestIDFromContext(r.Context()),
			"trace_id", traceID,
			"span_id", spanID,
		)

		methodLabel := requestMethodLabel(r)
		if metrics != nil {
			metrics.ObserveHTTPRequest(methodLabel, route, captured.Code)
			metrics.ObserveHTTPRequestDuration(methodLabel, route, captured.Code, captured.Duration)
		}
	})
}

func captureRouteLabelMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer captureRouteMetadata(r)
		next.ServeHTTP(w, r)
	})
}

func captureRouteMetadata(r *http.Request) {
	if r == nil {
		return
	}

	routePathTemplate := routePathTemplateForRequest(r)
	routeLabel := joinMethodAndPattern(requestMethodLabel(r), routePathTemplate)

	if routePathTemplate != "" {
		routeAttr := semconv.HTTPRoute(routePathTemplate)
		if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() && routeLabel != "" {
			span.SetName(routeLabel)
			span.SetAttributes(routeAttr)
		}
		if labeler, ok := otelhttp.LabelerFromContext(r.Context()); ok {
			labeler.Add(routeAttr)
		}
	}

	holder, _ := r.Context().Value(routeLabelContextKey{}).(*routeLabelHolder)
	if holder == nil || holder.value != "" {
		return
	}
	holder.value = routeLabel
}

func routeLabelForRequest(r *http.Request) string {
	return joinMethodAndPattern(requestMethodLabel(r), routePathTemplateForRequest(r))
}

func routePathTemplateForRequest(r *http.Request) string {
	if r == nil {
		return ""
	}

	if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
		if pattern := normalizeRoutePathTemplate(r.Method, routeContext.RoutePattern()); pattern != "" {
			return pattern
		}
	}

	return normalizeRoutePathTemplate(r.Method, r.Pattern)
}

func normalizeRoutePathTemplate(method, pattern string) string {
	pattern = strings.TrimSpace(pattern)
	method = strings.TrimSpace(method)
	if method != "" && strings.HasPrefix(pattern, method+" ") {
		pattern = strings.TrimSpace(strings.TrimPrefix(pattern, method+" "))
	}
	if pattern == "/" {
		return ""
	}
	return pattern
}

func requestMethodLabel(r *http.Request) string {
	if r == nil {
		return otherHTTPMethodLabel
	}
	return normalizeHTTPMethodLabel(r.Method)
}

func joinMethodAndPattern(method, pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return ""
	}

	method = strings.TrimSpace(method)
	if method == "" {
		return pattern
	}
	return method + " " + pattern
}

func requestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDContextKey{}).(string); ok {
		return requestID
	}
	return ""
}

func requestIDFromHeader(value string) string {
	requestID := strings.TrimSpace(value)
	if !validRequestID(requestID) {
		return ""
	}
	return requestID
}

func validRequestID(value string) bool {
	if len(value) == 0 || len(value) > maxRequestIDLength {
		return false
	}
	for i := 0; i < len(value); i++ {
		b := value[i]
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') {
			continue
		}
		switch b {
		case '.', '_', '~', '-':
			continue
		default:
			return false
		}
	}
	return true
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

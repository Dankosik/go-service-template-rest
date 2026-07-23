package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/felixge/httpsnoop"
	"github.com/go-chi/chi/v5"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

func AccessLog(log *slog.Logger, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured := httpsnoop.CaptureMetricsFn(w, func(capturedWriter http.ResponseWriter) {
			next.ServeHTTP(capturedWriter, r)
		})

		routePathTemplate := routePathTemplateForRequest(r)
		if routePathTemplate != "" {
			trace.SpanFromContext(r.Context()).SetAttributes(semconv.HTTPRoute(routePathTemplate))
		}
		route := joinMethodAndPattern(requestMethodLabel(r), routePathTemplate)
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
	})
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

func traceIDsFromContext(ctx context.Context) (string, string) {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return "", ""
	}
	return spanContext.TraceID().String(), spanContext.SpanID().String()
}

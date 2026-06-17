package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/felixge/httpsnoop"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

type routeLabelContextKey struct{}

type routeLabelHolder struct {
	value string
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

func traceIDsFromContext(ctx context.Context) (string, string) {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return "", ""
	}
	return spanContext.TraceID().String(), spanContext.SpanID().String()
}

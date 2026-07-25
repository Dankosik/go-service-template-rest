package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/felixge/httpsnoop"
	"github.com/go-chi/chi/v5"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// healthProbeRoutePaths are the platform probe routes owned by this template's
// OpenAPI contract. Orchestrators poll them every few seconds for the life of
// the pod, so logging them produces steady no-signal volume that is billed by
// every log backend.
var healthProbeRoutePaths = []string{"/health/live", "/health/ready"}

// AccessLog records one structured line per request. When logHealthProbes is
// false, matched health probe routes are served without a log line; span route
// attribution is unaffected either way.
func AccessLog(log *slog.Logger, logHealthProbes bool, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !log.Enabled(r.Context(), slog.LevelInfo) {
			next.ServeHTTP(w, r)
			if routePathTemplate := routePathTemplateForRequest(r); routePathTemplate != "" {
				trace.SpanFromContext(r.Context()).SetAttributes(semconv.HTTPRoute(routePathTemplate))
			}
			return
		}

		captured := httpsnoop.CaptureMetricsFn(w, func(capturedWriter http.ResponseWriter) {
			next.ServeHTTP(capturedWriter, r)
		})

		routePathTemplate := routePathTemplateForRequest(r)
		if routePathTemplate != "" {
			trace.SpanFromContext(r.Context()).SetAttributes(semconv.HTTPRoute(routePathTemplate))
		}
		// Route identity exists only after routing completes, so the probe
		// decision belongs here and not on the level-disabled fast path.
		if skipHealthProbeLog(r, routePathTemplate, logHealthProbes) {
			return
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

// skipHealthProbeLog matches on the routed template rather than the raw path,
// so an unmatched request that merely looks like a probe is still recorded.
func skipHealthProbeLog(r *http.Request, routePathTemplate string, logHealthProbes bool) bool {
	if logHealthProbes || r == nil || r.Method != http.MethodGet {
		return false
	}
	return slices.Contains(healthProbeRoutePaths, routePathTemplate)
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
	// "/" and "/*" are the root mount's own patterns, reported when nothing
	// matched or the path matched under a different method. Neither is a route
	// template: OpenTelemetry expects http.route to carry the matched route or
	// be absent, and collapsing 404s and 405s into per-method wildcards hides
	// which path was actually asked for.
	if pattern == "/" || pattern == "/*" {
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

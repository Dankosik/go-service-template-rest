package httpx

import (
	"log/slog"
	"net/http"
	"slices"

	"github.com/felixge/httpsnoop"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

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
		// The method is used verbatim. Normalizing it to a bounded label was
		// unreachable: joinMethodAndPattern discards the method whenever the
		// route template is empty, and a non-empty template means chi matched a
		// route, which only exists for the methods the contract declares. The
		// bounded label that observability does need is otelhttp's, which maps
		// anything outside the RFC methods to _OTHER on its own spans and metrics.
		route := joinMethodAndPattern(r.Method, routePathTemplate)
		if route == "" {
			route = "<unmatched>"
		}

		// Correlation is not listed here. The process logger publishes
		// request_id, trace_id, and span_id from the context every record is
		// logged with; see internal/observability/logctx. A logger built without
		// that decorator loses them, which is what the wiring test in
		// cmd/service/internal/bootstrap exists to catch.
		//
		// problem_code separates the failures that share a status: a 503 is load
		// shedding, a saturated connection pool, or a draining instance, and during
		// an incident that distinction is the whole question. Its cardinality is
		// bounded by the problem catalog.
		attrs := []any{
			"method", r.Method,
			"route", route,
			"status", captured.Code,
			"duration_ms", captured.Duration.Milliseconds(),
		}
		if code := problemCodeForRequest(r); code != "" {
			attrs = append(attrs, "problem_code", code)
		}
		log.InfoContext(r.Context(), "http_request", attrs...)
	})
}

// problemCodeForRequest returns the problem code this request was answered with,
// or the empty string when it was not answered with one.
func problemCodeForRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	record, ok := r.Context().Value(problemRecordContextKey{}).(*problemRecord)
	if !ok {
		return ""
	}
	return string(record.code)
}

// skipHealthProbeLog matches on the routed template rather than the raw path,
// so an unmatched request that merely looks like a probe is still recorded.
func skipHealthProbeLog(r *http.Request, routePathTemplate string, logHealthProbes bool) bool {
	if logHealthProbes || r == nil || r.Method != http.MethodGet {
		return false
	}
	return slices.Contains(healthProbeRoutePaths, routePathTemplate)
}

// Reading trace identifiers off the context is deliberately absent from this
// package. internal/observability/logctx publishes them on every record from the
// context it was logged with, so a helper here would only let one of the two
// copies drift.

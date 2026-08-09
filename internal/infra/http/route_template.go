package httpx

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Route naming is owned here rather than by any one of its callers: AccessLog,
// Recover, and the generated router's rejection callbacks all report which route
// answered a request, and a template that differed between them would split one
// request across two route identities in logs, traces, and metrics.

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

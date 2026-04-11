# Route Template Observability

## Behavior Change Thesis
When loaded for symptom `metrics, traces, logs, span names, route labels, RoutePattern, Match/Find, raw-path labels, or fallback route identity`, this file makes the model choose bounded route-template labels after route resolution instead of likely mistake `use raw URL paths or incomplete pre-handler route patterns`.

## When To Load
Load when the routing design touches logs, metrics, traces, span names, `http.route`, route labels, path cardinality, `RoutePattern()`, chi route context, `Find`/`Match`, or fallback labels for unmatched routes.

## Decision Rubric
- Metrics and traces use route templates or operation IDs. For OpenTelemetry `http.route`, set it only when a matched route template is available; fixed fallback labels belong in repo-owned metrics/log fields, not `http.route`. Never use raw paths, IDs, slugs, query strings, wildcard captures, or request IDs as labels.
- For mounted APIs, prove whether the emitted route template includes the application root or mount prefix. OpenTelemetry semantic conventions say `http.route` should include the application root when one exists, so do not silently label `/widgets/{id}` if the externally meaningful route is `/api/widgets/{id}`.
- For post-handler telemetry, call `next.ServeHTTP` first, then read `chi.RouteContext(r.Context()).RoutePattern()` when chi resolved the route.
- For pre-handler route identity, use `Find` or `Match` only when the installed chi version and subrouter behavior are verified. Use fresh route contexts for probes.
- Define fallback labels deliberately, such as `<unmatched>`, `not_found`, or `method_not_allowed`. Do not fall back to the raw path when the template is empty.
- Logs may include raw path as a field if the repo accepts it, but metrics and span names must stay bounded. When no route template is available, omit `http.route` rather than substituting a raw path.
- If the design uses `r.Pattern`, require repository or installed-version proof that the active router sets it for the relevant requests.

## Imitate
```go
func observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		pattern := "<unmatched>"
		if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
			pattern = rctx.RoutePattern()
		}
		record(pattern)
	})
}
```

Copy the timing shape: final route extraction happens after downstream routing and uses a bounded fallback.

```go
// Pre-handler design sketch:
// if router.Find(chi.NewRouteContext(), r.Method, r.URL.Path) == "" { label = "<unmatched>" }
// Gate this on the installed chi version and prove subrouter behavior with tests.
```

Copy the proof gate: pre-handler lookup is not a default; it must be version-checked and subrouter-tested.

```go
assertSameRouteLabel(t, "/users/123/orders/456", "/users/789/orders/000", "/users/{userID}/orders/{orderID}")
assertSameRouteLabel(t, "/missing/a", "/missing/b", "<unmatched>")
```

Copy the assertion shape: concrete path values collapse to one template or one fallback label.

## Reject
```go
metrics.Record("route", r.URL.Path)
```

Reject because raw paths create high-cardinality labels.

```go
route := chi.RouteContext(r.Context()).RoutePattern()
next.ServeHTTP(w, r)
metrics.Record("route", route)
```

Reject because the route template was read before final route resolution.

```go
if route == "" {
	route = r.URL.Path
}
```

Reject because unmatched routes become unbounded labels.

## Agent Traps
- Do not stop at "use `RoutePattern`"; the timing matters.
- Do not allow metrics to be bounded while span names or `http.route` still use raw paths for the same surface.
- Do not invent string parsing of chi path templates when route context or version-supported lookup can answer the question.
- Do not treat `404` and `405` as normal matched route labels unless the repo has an explicit policy for them.

## Validation Shape
- Two concrete parameter values produce the same route-template or operation-ID label.
- Two unmatched paths produce the same bounded fallback label.
- A method-disallowed request records the chosen `405` fallback label while preserving `Allow` behavior.
- Subrouter and wildcard routes have representative tests because they are where route-template assumptions often fail.

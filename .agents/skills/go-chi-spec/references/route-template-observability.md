# Route Template Observability

## When To Load
Load this when logs, metrics, traces, route labels, path cardinality, `RoutePattern()`, chi route context, or pre-handler route matching are in scope.

## Recommended Design Options
- Use route templates, operation IDs, or a fixed fallback label for metrics and traces. Never use raw paths with IDs or user-controlled path fragments as labels.
- For post-handler telemetry, call `next.ServeHTTP` first, then read `chi.RouteContext(r.Context()).RoutePattern()` if the route matched.
- For pre-handler route identification, prefer a version-supported chi lookup such as `Find` or `Match` only when the installed chi version provides the needed behavior and tests cover misses and subrouters.
- Define fallback labels deliberately, for example `unmatched`, `method_not_allowed`, or `unknown`, so misses do not explode label cardinality.
- Keep logs richer than metrics if needed, but avoid promoting raw path fields into metric labels.

## Rejected Alternatives
- Metrics labeled with `r.URL.Path`, request IDs, user IDs, slugs, or database identifiers.
- Reading `RoutePattern()` before `next.ServeHTTP` in middleware and assuming the final route has already been resolved.
- Treating `404` and `405` as normal route templates when no final handler was reached. Use an explicit fallback label.
- Building custom route matching logic by string parsing when chi already has route context or version-supported route lookup APIs.

## Example Sketches
```go
func observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		pattern := "unmatched"
		if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
			pattern = rctx.RoutePattern()
		}
		record(pattern)
	})
}
```

```go
// Pre-handler design sketch:
// if router.Find(chi.NewRouteContext(), r.Method, r.URL.Path) == "" { label = "unmatched" }
// Gate this on the installed chi version and prove subrouter behavior with tests.
```

## Testable Acceptance Boundaries
- A path like `/users/123/orders/456` records `/users/{userID}/orders/{orderID}` or an operation ID, never the raw path.
- A missing route records the chosen fallback label and bounded status label.
- A method-disallowed request records the chosen `405` fallback label without dropping `Allow` behavior.
- Subrouter and wildcard routes produce stable templates in tests.
- Middleware-order tests prove route-label extraction happens at the intended point in the handler chain.

## Source Links Gathered Through Exa
- chi context source for `RoutePattern()` timing and nil fallback: https://github.com/go-chi/chi/blob/master/context.go
- chi mux source for `Find`, `Match`, `r.Pattern`, and fallback routing flow: https://github.com/go-chi/chi/blob/master/mux.go
- chi package docs for route context, route patterns, and router traversal: https://pkg.go.dev/github.com/go-chi/chi/v5
- chi route-pattern discussion with maintainer guidance on post-handler extraction: https://github.com/go-chi/chi/issues/692
- chi README for named params and wildcards: https://github.com/go-chi/chi/blob/master/README.md

# Route Context And Match Probing

## Behavior Change Thesis
When loaded for symptom `RouteContext, RoutePattern, Match, Find, custom Allow, OPTIONS, or alternate-method probing appears in the diff`, this file makes the model choose post-routing route-pattern reads and isolated probe contexts instead of likely mistake `reuse the live request route context or trust an incomplete route template`.

## When To Load
Load when a diff reads `chi.RouteContext`, calls `RoutePattern`, calls `Match` or `Find`, computes `Allow` or `OPTIONS`, probes alternate methods, or derives route labels from route state.

## Decision Rubric
- `RoutePattern()` changes during routing. Middleware should read it after `next.ServeHTTP` when it needs the final template.
- `Match` and `Find` mutate the `*chi.Context` they receive. Use `chi.NewRouteContext()` for each independent method/path probe.
- Reusing one probe context across a loop is reviewable risk unless the code clearly resets it before every probe.
- Do not call `router.Match(chi.RouteContext(r.Context()), ...)` for custom `405`, `Allow`, `OPTIONS`, or telemetry discovery.
- Do not infer `HEAD` support from a `GET` match unless the router has explicit `Head(...)` routes or `middleware.GetHead` installed before routes. Load `http-fallback-head-options-cors.md` when method policy is the primary issue.
- If the only defect is raw-path metric cardinality, load `route-observability-labels.md` as the primary reference.

## Imitate

```go
func allowed(router chi.Routes, method, path string) bool {
	rctx := chi.NewRouteContext()
	return router.Match(rctx, method, path)
}
```

Copy the probe shape: fresh route context per independent check, no mutation of the live request.

```go
func instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		recordRoute(route)
	})
}
```

Copy the timing shape: read the final route template after downstream routing and bound the unmatched fallback.

```go
for _, method := range []string{http.MethodGet, http.MethodPost} {
	if router.Match(chi.NewRouteContext(), method, path) {
		allowed = append(allowed, method)
	}
}
```

Copy the loop shape: each method probe receives a new context unless the helper has an obvious reset.

## Reject

```go
func allowed(router chi.Routes, r *http.Request, method string) bool {
	return router.Match(chi.RouteContext(r.Context()), method, r.URL.Path)
}
```

Reject because the helper mutates the live request route context while computing alternate method state.

```go
rctx := chi.NewRouteContext()
for _, method := range methods {
	if router.Match(rctx, method, path) {
		allowed = append(allowed, method)
	}
}
```

Reject unless the context is reset per iteration. The subtle failure is stale route state from the previous method probe.

```go
route := chi.RouteContext(r.Context()).RoutePattern()
next.ServeHTTP(w, r)
recordRoute(route)
```

Reject because the route template is read before final route resolution.

## Agent Traps
- Do not call live `RouteContext` mutation a cleanup concern. It can corrupt request-local routing state.
- Do not say "use `Match`" without also naming the required fresh context.
- Do not advertise `HEAD` support just because `GET` matched.
- Do not hide an incomplete route-template bug under a general observability complaint; call out timing if that is the defect.

## Validation Shape
- Probe helpers: call the helper for two methods on the same request and verify the original request still serves correctly.
- Custom `Allow`: test an allowed method, a disallowed method on an existing path, and an unknown path.
- Route template timing: send two concrete parameter values and verify the same template is recorded after routing.
- Suggested command: `go test ./... -run 'Test.*Allow|Test.*RouteContext|Test.*RoutePattern|Test.*Probe|Test.*Head'`.

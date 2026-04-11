# Route Observability Labels

## Behavior Change Thesis
When loaded for symptom `metrics, traces, logs, span names, http.route, route label extraction, or unmatched-route labels changed`, this file makes the model choose bounded route-template telemetry shared across signals instead of likely mistake `use raw URL paths or inconsistent route identities`.

## When To Load
Load when a diff touches metrics, traces, logs, span names, `http.route`, route-template extraction, unmatched-route fallback labels, or middleware that records route identity.

## Decision Rubric
- Route labels should come from chi route templates when available, not `r.URL.Path`, query strings, captures, IDs, slugs, or wildcard values.
- Read final chi route templates after downstream routing. If the main defect is context timing rather than telemetry cardinality, load `route-context-and-match-probing.md`.
- On Go versions where chi sets `http.Request.Pattern`, treat `r.Pattern` as a route-template source, not a raw path, but verify it is the full intended template for the router topology. Under mounted or nested chi routers, prefer `chi.RouteContext(r.Context()).RoutePattern()` after downstream routing or require tests proving `r.Pattern` keeps the parent prefix.
- Custom metrics and logs for unmatched requests need one bounded fallback label, such as `unmatched`, `not_found`, or another project-approved constant. Do not fall back to user-controlled paths.
- Metrics, traces, span names, and structured logs should use the same route-template source unless the diff records a deliberate policy difference.
- For OpenTelemetry, set `http.route` only when a real route template is available. For server span naming, prefer method plus route template when the template exists; otherwise use method-only naming instead of substituting a raw path or synthetic fallback as `http.route`.
- For `otelhttp.WithSpanNameFormatter`, account for the formatter running before and after middleware when `r.Pattern` is set: the first name must stay bounded, and the final name should use a verified full route template, not a leaf-only mounted-router pattern.
- If standards wording matters, verify against the current OpenTelemetry HTTP semantic conventions, but do not turn the review reference into a link dump.

## Imitate

```go
next.ServeHTTP(w, r)
route := chi.RouteContext(r.Context()).RoutePattern()
if route == "" {
	route = "unmatched"
}
metrics.Record("route", route)
```

Copy the label shape: post-route template extraction plus bounded fallback.

```go
if route := chi.RouteContext(r.Context()).RoutePattern(); route != "" {
	span.SetName(r.Method + " " + route)
	span.SetAttributes(attribute.String("http.route", route))
} else {
	span.SetName(r.Method)
}
```

Copy the trace shape: use method plus route template and set `http.route` from the same template; if no template exists, keep the span name bounded and leave `http.route` unset.

```go
assertSameRouteLabel(t, "/users/1", "/users/2", "/users/{id}")
assertSameRouteLabel(t, "/missing/a", "/missing/b", "unmatched")
```

Copy the proof shape: parameterized routes collapse together, and unknown routes collapse together.

## Reject

```go
metrics.Record("route", r.URL.Path)
```

Reject because raw paths create high-cardinality labels.

```go
span.SetName(r.Method + " " + r.URL.Path)
span.SetAttributes(attribute.String("http.route", r.URL.Path))
```

Reject because trace names and `http.route` now include user-controlled path data.

```go
if route == "" {
	route = r.URL.Path
}
```

Reject because the fallback converts unmatched routes into unbounded labels.

## Agent Traps
- Do not stop at "use `RoutePattern`" if the code still reads it before routing finishes.
- Do not accept a bounded metric label while traces or logs still use raw paths for the same route identity.
- Do not write `http.route="unmatched"`; use a separate custom label if project policy needs an unmatched fallback.
- Do not substitute wildcard captures, IDs, or query strings when a template is unavailable.
- Do not approve `r.Pattern` as a replacement for `RoutePattern()` across chi mounts unless tests prove it preserves the full mounted prefix.
- Do not make this a performance-only finding; the merge risk is telemetry cardinality and debuggability drift.

## Validation Shape
- Send two concrete parameter values, such as `/users/1` and `/users/2`, and assert one route-template label.
- If code uses `r.Pattern` under mounted or nested routers, include a mounted-prefix case and assert the full expected template, not only the leaf route.
- Send two unknown paths and assert both collapse to the same bounded fallback label.
- Assert metrics, span name, `http.route`, and structured logs use the same route-template source when the diff touches more than one signal; for unmatched paths, assert any custom fallback is bounded and `http.route` is omitted.
- Suggested command: `go test ./... -run 'Test.*RouteLabel|Test.*Observability|Test.*Telemetry|Test.*Trace|Test.*Metrics|Test.*Log'`.

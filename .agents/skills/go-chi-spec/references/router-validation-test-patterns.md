# Router Validation Test Patterns

## When To Load
Load this when designing proof obligations for chi routing work, including route collision checks, middleware scope/order tests, fallback tests, OpenAPI route coverage, CORS preflight tests, or observability-label assertions.

## Recommended Design Options
- Use `httptest` for client-visible behavior: status, headers, body shape, middleware side effects, and fallback behavior.
- Use route-table inspection for topology expectations: `chi.Walk`, `Routes`, `Middlewares`, `Match`, or `Find` depending on the installed chi version and the exact question.
- Add focused probes for middleware order by appending markers to request context or headers in test-only middleware.
- Validate CORS through real preflight request shapes: `OPTIONS` plus `Origin` and `Access-Control-Request-Method`.
- For OpenAPI-generated routes, compare generated operation paths against registered route inventory or representative generated handler requests.

## Rejected Alternatives
- Only snapshotting route docs without behavioral `httptest` coverage. It can miss fallback, middleware, and header behavior.
- Testing just happy-path `GET` routes when the design changes `405`, `OPTIONS`, or CORS.
- Using broad string contains checks on generated code as the only route proof.
- Accepting raw-path telemetry by eyeballing logs instead of asserting label values in a test hook.

## Example Sketches
```go
req := httptest.NewRequest(http.MethodOptions, "/api/widgets", nil)
req.Header.Set("Origin", "https://app.example")
req.Header.Set("Access-Control-Request-Method", "POST")
rr := httptest.NewRecorder()
handler.ServeHTTP(rr, req)
```

```go
seen := map[string]bool{}
_ = chi.Walk(r, func(method, route string, h http.Handler, mws ...func(http.Handler) http.Handler) error {
	seen[method+" "+route] = true
	return nil
})
```

```go
rr := httptest.NewRecorder()
handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/widgets/123", nil))
// Assert status, Allow header if applicable, and emitted route label.
```

## Testable Acceptance Boundaries
- Route inventory contains all expected method/path pairs and no disallowed prefix collisions.
- `404`, `405`, `Allow`, and preflight headers are asserted at the HTTP boundary.
- Middleware scope tests show admin-only middleware does not affect public health or debug routes unless designed to.
- Observability tests assert low-cardinality route labels and explicit fallback labels.
- Generated and manual route coexistence is covered by a collision test or route inventory comparison.

## Source Links Gathered Through Exa
- chi README for router traversal, doc generation, examples, params, and stdlib testing compatibility: https://github.com/go-chi/chi/blob/master/README.md
- chi mux source for `Match`, `Find`, `Routes`, fallback flow, and `r.Pattern`: https://github.com/go-chi/chi/blob/master/mux.go
- chi tree source for `Walk` and route traversal: https://github.com/go-chi/chi/blob/master/tree.go
- go-chi/cors tests for preflight and header assertion examples: https://github.com/go-chi/cors/blob/master/cors_test.go
- Go `net/http` docs for handlers and `httptest`-compatible handler behavior: https://pkg.go.dev/net/http

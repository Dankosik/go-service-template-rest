# Router Validation Test Patterns

## Behavior Change Thesis
When loaded for symptom `proof obligations for router topology, fallback, middleware order, CORS preflight, OpenAPI route coverage, or observability labels`, this file makes the model choose a small test/proof matrix tied to the routing risk instead of likely mistake `write generic happy-path route tests or prose-only validation`.

## When To Load
Load when the design already has the routing choice and needs validation obligations for route collision checks, middleware scope/order, fallback policy, CORS preflight, generated route coverage, route-label assertions, or route inventory.

## Decision Rubric
- Use `httptest` for client-visible behavior: status, headers, body shape, middleware side effects, fallback behavior, and CORS.
- Use route inventory for topology: `chi.Walk`, `Routes`, `Middlewares`, `Match`, or `Find`, depending on installed-version support and the exact question.
- Use targeted middleware probes for order or scope, such as headers, context markers, or a test-only recorder. Status-only tests are usually too weak.
- Validate CORS with real preflight shape: `OPTIONS` plus `Origin` and `Access-Control-Request-Method`.
- For OpenAPI-generated routes, compare expected operation paths against registered route inventory or representative generated handler requests.
- Keep the proof matrix small. One proof per independent routing risk is better than a broad "test all routes" demand.

## Imitate
```go
req := httptest.NewRequest(http.MethodOptions, "/api/widgets", nil)
req.Header.Set("Origin", "https://app.example")
req.Header.Set("Access-Control-Request-Method", "POST")
rr := httptest.NewRecorder()
handler.ServeHTTP(rr, req)
```

Copy the CORS proof shape: it is not a bare `OPTIONS` request.

```go
seen := map[string]bool{}
_ = chi.Walk(r, func(method, route string, h http.Handler, mws ...func(http.Handler) http.Handler) error {
	seen[method+" "+route] = true
	return nil
})
```

Copy the route inventory shape: topology proof should see actual registered method-path pairs.

```go
rr := httptest.NewRecorder()
handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/widgets/123", nil))
// Assert status, Allow header if applicable, and emitted route label.
```

Copy the bundled proof shape: one request can prove fallback status, headers, and route label when all three changed together.

## Reject
```go
// Test only GET /api/widgets returns 200.
```

Reject when the design changed `405`, `OPTIONS`, CORS, fallback, route ownership, or middleware scope.

```go
if !strings.Contains(string(generatedFile), "/api/widgets") {
	t.Fatal("missing route")
}
```

Reject as the only proof because generated text does not prove mounted prefix, middleware, or fallback behavior.

```go
// Manually checked logs for /users/123 looked fine.
```

Reject because raw-path telemetry bugs need asserted labels for multiple concrete values.

## Agent Traps
- Do not turn validation into a generic checklist. Tie each proof to the decision that could regress.
- Do not use route inventory alone for client-visible behavior. It misses status, headers, body, and middleware side effects.
- Do not use `httptest` happy paths alone for topology. It can miss duplicate or shadowed routes.
- Do not forget negative cases: unmatched path, wrong method, sibling route outside middleware scope, and concrete parameter values for route labels.

## Validation Shape
- Topology: route inventory contains expected method-path pairs and no disallowed prefix collisions.
- Fallback: HTTP-boundary tests assert `404`, `405`, `Allow`, and any JSON body contract in separate missing-path and wrong-method cases.
- CORS: preflight includes `Origin` and `Access-Control-Request-Method` and asserts headers.
- Middleware: in-scope and out-of-scope paths prove coverage; order-sensitive stacks use a recorder.
- Observability: two concrete parameter values collapse to one route label; unmatched routes collapse to one fallback label.
- Generated/manual coexistence: route inventory or representative requests prove there is no collision and policy is consistent.

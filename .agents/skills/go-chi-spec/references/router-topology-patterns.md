# Router Topology Patterns

## When To Load
Load this when designing chi root router shape, `Route`/`Mount`/`Group` use, generated/manual route ownership, public vs internal path boundaries, route conflict controls, or route-table validation before coding.

## Recommended Design Options
- Single root composition point: create one exported `http.Handler`/router factory that owns global transport middleware, fallback policy, and top-level mounts.
- Prefix-owned subrouters: give each top-level path set one owner, for example `/api`, `/admin`, `/internal`, and `/debug`. Use chi subrouters to keep path-local middleware and handlers near the owner.
- Generated API surface under one prefix: mount `oapi-codegen` chi handlers at a deterministic API prefix and keep manual routes outside that prefix unless explicitly reserved in the OpenAPI spec.
- Manual operational routes as separate branches: keep `/internal/health`, `/ready`, `/debug/pprof`, or similar operational handlers separate from public API contract paths.
- Route inventory as a validation artifact: require `chi.Walk`, `Routes()`, `Match`, `Find`, or `httptest` coverage to prove the registered routes match the design.

## Rejected Alternatives
- One large root router where every module registers routes directly. This hides path ownership and makes middleware scope drift easy.
- Generated and manual routes interleaved under the same prefix without an ownership rule. This makes collision and fallback behavior ambiguous.
- Depending on duplicate route registration behavior as a feature. Treat duplicates, shadowing, and wildcard overlap as defects to prove away with tests.
- Mounting a broad wildcard route before proving it cannot swallow specific routes or custom fallback handlers.

## Example Sketches
```go
func NewHTTPHandler(api http.Handler, admin http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(requestID, recoverer)

	r.Mount("/api", api)       // generated OpenAPI-owned surface
	r.Mount("/admin", admin)   // manual admin surface
	r.Get("/internal/health", health)
	return r
}
```

```go
func adminRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(adminOnly)
	r.Get("/", adminIndex)
	return r
}
```

## Testable Acceptance Boundaries
- Every top-level prefix has one named owner and one registration point.
- Generated routes and manual routes do not register the same method/path.
- Route-table tests fail on duplicate method/path, unexpected wildcard routes, or missing expected prefixes.
- `httptest` proves a representative route from each prefix reaches the intended handler and middleware.
- If a catch-all route exists, tests prove it does not hide API `404` or operational-route behavior.

## Source Links Gathered Through Exa
- go-chi README, router interface, stdlib compatibility, `Route`, `Mount`, examples: https://github.com/go-chi/chi/blob/master/README.md
- chi package docs, `Routes`, `Middlewares`, `Match`, route patterns: https://pkg.go.dev/github.com/go-chi/chi/v5
- chi REST example with resource routes and mounted admin router: https://github.com/go-chi/chi/blob/master/_examples/rest/main.go
- chi mux source for `Mount`, `Route`, `Group`, route matching, `Find`, and fallback flow: https://github.com/go-chi/chi/blob/master/mux.go
- chi tree source for route traversal and `Walk`: https://github.com/go-chi/chi/blob/master/tree.go

# NotFound, MethodNotAllowed, OPTIONS, And CORS

## When To Load
Load this when client-visible `404`, `405`, `OPTIONS`, `Allow`, or CORS preflight behavior is in scope, when CORS placement is disputed, or when custom fallback JSON/error responses must be designed before coding.

## Recommended Design Options
- Pin API fallback behavior explicitly with `NotFound` and `MethodNotAllowed` on the router that owns the API surface.
- Preserve `Allow` header behavior for `405`. If writing a custom `MethodNotAllowed`, make header behavior an acceptance boundary.
- Treat CORS as transport policy, not handler business logic. Prefer top-level or API-prefix middleware when preflight must apply across the mounted API surface.
- Use scoped CORS only when every expected preflight path has matching `OPTIONS` behavior or the CORS middleware is configured to pass through intentionally.
- Keep preflight responses small and deterministic; avoid duplicating custom `OPTIONS` handlers and CORS middleware for the same path unless the interaction is tested.

## Rejected Alternatives
- Leaving public API `404`/`405` response shape to framework defaults when clients require JSON, stable headers, or documented status behavior.
- Custom `405` handlers that omit `Allow` coverage.
- Route-local CORS with no `OPTIONS` route plan. The go-chi/cors README warns that `Group`/`With` placement needs matching `OPTIONS` routes.
- Duplicating preflight policy in both CORS middleware and hand-written `OPTIONS` handlers without tests for precedence and headers.

## Example Sketches
```go
api := chi.NewRouter()
api.Use(cors.Handler(cors.Options{AllowedMethods: []string{"GET", "POST", "OPTIONS"}}))
api.NotFound(jsonNotFound)
api.MethodNotAllowed(jsonMethodNotAllowedWithAllow)
```

```go
// Acceptance sketch:
// OPTIONS /api/widgets with Origin and Access-Control-Request-Method returns the expected CORS headers.
// POST /api/widgets when only GET exists returns 405 plus Allow.
// GET /api/missing returns the API JSON 404 shape.
```

## Testable Acceptance Boundaries
- `GET` for a missing API path returns the chosen JSON `404` contract.
- Unsupported method on an existing path returns `405` and expected `Allow` values.
- Preflight requests include `Origin` and `Access-Control-Request-Method` in tests and prove the expected CORS headers.
- Non-preflight `OPTIONS` behavior is intentionally accepted, passed through, or rejected with documented status.
- Scoped CORS tests cover one matching route and one unmatched route so fallback behavior is not accidental.

## Source Links Gathered Through Exa
- chi mux source for default `NotFound`, `MethodNotAllowed`, and `Allow` header behavior: https://github.com/go-chi/chi/blob/master/mux.go
- chi package docs for `NotFound`, `MethodNotAllowed`, and `Options`: https://pkg.go.dev/github.com/go-chi/chi/v5
- go-chi/cors README for top-level middleware recommendation and scoped `OPTIONS` caveat: https://github.com/go-chi/cors/blob/master/README.md
- go-chi/cors tests for preflight, `Vary`, allowed methods, allowed headers, and `OptionsPassthrough`: https://github.com/go-chi/cors/blob/master/cors_test.go
- WHATWG Fetch Standard for CORS preflight semantics: https://fetch.spec.whatwg.org/
- Go `net/http` docs for handler and server behavior: https://pkg.go.dev/net/http

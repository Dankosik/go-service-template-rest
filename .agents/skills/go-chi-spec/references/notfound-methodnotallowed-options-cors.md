# NotFound, MethodNotAllowed, OPTIONS, And CORS

## Behavior Change Thesis
When loaded for symptom `NotFound, MethodNotAllowed, Allow, HEAD, OPTIONS, CORS preflight, custom fallback JSON, or scoped CORS placement`, this file makes the model choose explicit fallback and preflight policy with header proof instead of likely mistake `trust framework defaults or duplicate CORS and hand-written OPTIONS behavior`.

## When To Load
Load when client-visible `404`, `405`, `Allow`, `HEAD`, `OPTIONS`, CORS preflight, fallback JSON/error shape, or CORS placement must be designed before coding.

## Decision Rubric
- Put `NotFound` and `MethodNotAllowed` on the router that owns the client-visible surface, not inside unrelated business handlers.
- chi's default `405` sets `Allow`, but a custom `MethodNotAllowed` handler replaces that default and cannot read chi's unexported allowed-method set through a public handler API. A custom JSON `405` design must name a tested `Allow` source, such as route inventory or method probing with fresh route contexts, or explicitly accept that dynamic `Allow` is not preserved.
- If the API advertises `HEAD`, make support explicit with `Head(...)`, a documented `middleware.GetHead` decision, or a handoff to the API contract owner. When `Allow` semantics matter, prove that `HEAD` appears where intended instead of inferring it from `GET`.
- Treat CORS as transport policy. Prefer top-level or API-prefix middleware when preflight must apply across a mounted API surface.
- Use scoped CORS only when expected preflight paths have matching `OPTIONS` behavior or the pass-through behavior is deliberately accepted.
- If hand-written `OPTIONS` must run after `cors.Handler`, make `OptionsPassthrough` an explicit decision and test the resulting status and headers. This only affects CORS preflight shape, not every bare `OPTIONS` request.
- Do not assume preflight success status from memory; `github.com/go-chi/cors` and `github.com/rs/cors` versions differ, so design or test the status explicitly when clients observe it.
- Do not duplicate CORS middleware and hand-written `OPTIONS` handlers for the same path unless the precedence, status, and headers are explicitly tested.
- For generated and manual siblings under one prefix, require the same fallback and CORS policy unless the contract says they differ.

## Imitate
```go
api := chi.NewRouter()
api.Use(cors.Handler(cors.Options{AllowedMethods: []string{"GET", "POST"}}))
api.NotFound(jsonNotFound)
api.MethodNotAllowed(jsonMethodNotAllowed) // only after the design names the exact Allow behavior
```

Copy the surface shape: fallback and CORS policy are owned by the API router.

```go
// Acceptance sketch:
// OPTIONS /api/widgets with Origin and Access-Control-Request-Method returns the expected CORS headers.
// POST /api/widgets when only GET exists returns 405 plus Allow.
// GET /api/missing returns the API JSON 404 shape.
```

Copy the proof shape: include real preflight headers and separate missing-path from wrong-method behavior.

## Reject
```go
api.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusMethodNotAllowed, problem)
})
```

Reject unless the design explains how `Allow` is preserved or why the API contract does not require it; chi will not add the default dynamic `Allow` header after a custom handler replaces the default.

```go
api.With(cors.Handler(opts)).Post("/widgets", createWidget)
// no matching OPTIONS route or broader CORS scope
```

Reject because route-local CORS does not by itself prove preflight behavior for the path.

```go
api.Use(cors.Handler(opts))
api.Options("/widgets", customPreflight)
```

Reject unless the design names which layer wins and tests status, `Vary`, allow-method, and allow-header behavior.

## Agent Traps
- Do not say "custom JSON fallback" and forget `Allow`.
- Do not assume `middleware.GetHead` makes `HEAD` appear in `Allow`; prove it or register `Head(...)` where the API contract requires it.
- Do not treat `Allow` header presence as exact when overlapping wildcard or parameterized routes are involved; assert the method set when `405` behavior is client-visible.
- Do not assume bare `OPTIONS` tests prove CORS. Preflight includes `Origin` and `Access-Control-Request-Method`, and some middleware only takes its preflight branch when that shape is present.
- Do not let API fallback policy accidentally cover `/debug`, `/metrics`, or internal health routes unless those routes are API-owned.
- Do not settle `HEAD` support in the chi reference when it is actually an API-contract decision; record the handoff when needed.

## Validation Shape
- Unknown API path returns the chosen JSON `404` shape.
- Existing route with unsupported method returns `405` and expected `Allow`.
- Preflight request includes `Origin` and `Access-Control-Request-Method` and proves the intended CORS headers.
- Scoped CORS proof covers one matching route and one unmatched route.
- Generated/manual sibling proof covers the same fallback scenario on both sides if they share a prefix.

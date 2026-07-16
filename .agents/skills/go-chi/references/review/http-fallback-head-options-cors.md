# HTTP Fallback, HEAD, OPTIONS, And CORS

## Behavior Change Thesis
When loaded for symptom `NotFound, MethodNotAllowed, Allow, HEAD, OPTIONS, CORS, or fallback wrappers changed`, this file makes the model choose actual router capability and fallback-contract proof instead of likely mistake `infer method support from GET routes or hardcode method lists`.

## When To Load
Load when a diff changes fallback handlers, custom method discovery, `Allow` headers, `Head` routes, `middleware.GetHead`, `Options` routes, preflight handling, CORS middleware, or generated handler fallback wrappers.

## Decision Rubric
- Unknown path should exercise the intended `404`; known path with wrong method should exercise the intended `405`.
- Custom `MethodNotAllowed` can change body format, but it must preserve route-accurate method disclosure when the contract depends on `Allow`.
- `HEAD` is not automatic for every `GET` route. Require `Head(...)` or `middleware.GetHead` installed before route registration when `HEAD` support is advertised.
- If `middleware.GetHead` is the only HEAD mechanism, validate `Allow` independently. It can serve an undefined `HEAD` request through the `GET` handler without making the router's 405 method disclosure include `HEAD`.
- If `middleware.GetHead` is added after routes on the same mux, primary finding is the late-`Use` startup hazard; load `chi-router-registration-hazards.md`.
- CORS preflight needs matching `OPTIONS` handling at the path/scope receiving the preflight. Do not assume a grouped or inline middleware covers unmatched `OPTIONS` routes.
- Generated and manual routes under the same prefix should expose consistent fallback, `Allow`, `HEAD`, `OPTIONS`, and CORS behavior. If ownership drift is primary, load `generated-and-manual-route-drift.md`.
- If custom discovery uses `Match` or `Find`, load `route-context-and-match-probing.md` for the probing mechanics.

## Imitate

```go
r.Use(middleware.GetHead)
r.Get("/reports/{id}", getReport)
```

Copy the capability shape: `HEAD` support is deliberate because the middleware is installed before routes; `Allow` still needs a separate assertion if the contract requires it.

```go
r.Use(cors.Handler(cors.Options{
	AllowedMethods: []string{"GET", "POST", "OPTIONS"},
}))
r.Route("/api", func(r chi.Router) {
	r.Get("/users", listUsers)
})
```

Copy the scope shape when using chi CORS middleware: top-level CORS can see preflight routing instead of being hidden inside a route-local stack.

```go
req := httptest.NewRequest(http.MethodOptions, "/api/users", nil)
req.Header.Set("Origin", "https://example.test")
req.Header.Set("Access-Control-Request-Method", http.MethodPost)
```

Copy the proof shape: preflight tests must include `Origin` and `Access-Control-Request-Method`, not just a bare `OPTIONS` request.

## Reject

```go
r.Get("/reports/{id}", getReport)
// custom Allow: GET, HEAD
```

Reject unless the router can actually serve `HEAD` through `Head(...)` or `middleware.GetHead`.

```go
r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "GET, POST, HEAD, OPTIONS")
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
})
```

Reject because hardcoded `Allow` can drift from the route tree and may overclaim `HEAD` or `OPTIONS`.

```go
r.With(cors.Handler(opts)).Get("/users", listUsers)
```

Reject when the code expects preflight to work for `/users` without a matching `OPTIONS` route or broader CORS scope.

## Agent Traps
- Do not assume chi maps `HEAD` to `GET` by default.
- Do not report only the missing `Allow` header when the same helper also mutates live route context; split both first-class defects or load the probing reference.
- Do not accept a fallback wrapper just because status codes look right. Check headers and method-specific behavior too.
- Do not let generated/manual siblings expose different CORS or fallback behavior without naming the contract risk.

## Validation Shape
- Table-test unknown path (`404`), known path with wrong method (`405`), `Allow`, `HEAD`, and `OPTIONS`.
- For CORS, include `Origin`, `Access-Control-Request-Method`, and the requested method.
- For generated/manual prefixes, test one generated route and one manual sibling route with the same fallback scenario.
- Suggested command: `go test ./... -run 'Test.*NotFound|Test.*MethodNotAllowed|Test.*Head|Test.*Options|Test.*Cors|Test.*CORS|Test.*Allow'`.

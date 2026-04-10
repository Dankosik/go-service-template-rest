# HTTP Fallback, HEAD, OPTIONS, And CORS

## When To Read
Read this when a diff touches `NotFound`, `MethodNotAllowed`, `Allow`, `HEAD`, `OPTIONS`, CORS middleware, generated handler fallback wrappers, or custom method discovery.

## Review Smell Patterns
- A custom `MethodNotAllowed` response omits or hardcodes `Allow` and can drift from actual registered methods.
- A route adds `Get(...)` and assumes `HEAD` will work without `Head(...)` or `middleware.GetHead`.
- `middleware.GetHead` is added after routes on the same mux, which is both a HEAD behavior change and a registration-order startup hazard.
- CORS middleware is installed with `With` or inside `Group` while preflight requests have no explicit matching `OPTIONS` route.
- `OPTIONS` and preflight behavior differs between a mounted generated router and adjacent manual routes.
- Related resources return inconsistent `404`, `405`, or preflight statuses after a routing refactor.

## Minimal Examples

```diff
 r := chi.NewRouter()
+r.Use(middleware.GetHead)
 r.Get("/reports/{id}", getReport)
```

Review finding shape: if `HEAD` is part of the contract, make it explicit. The smallest safe fix is either `middleware.GetHead` before route registration or a dedicated `Head` handler when the response headers differ from `GET`.

```diff
-r.Route("/api", func(r chi.Router) {
-  r.Use(cors.Handler(cors.Options{AllowedMethods: []string{"GET", "POST", "OPTIONS"}}))
-  r.Get("/users", listUsers)
-})
+r.Use(cors.Handler(cors.Options{AllowedMethods: []string{"GET", "POST", "OPTIONS"}}))
+r.Route("/api", func(r chi.Router) {
+  r.Get("/users", listUsers)
+})
```

Review finding shape: go-chi/cors expects top-level middleware unless explicit `OPTIONS` routes exist. The smallest safe fix is top-level CORS scope or concrete `Options` routes for the group.

```diff
-r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
-  http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
-})
+// Prefer chi's default MethodNotAllowed unless the custom responder preserves
+// accurate Allow behavior for the matched route.
```

Review finding shape: a custom body format is fine, but it must preserve route-accurate method disclosure. If that requires probing, use fresh route contexts.

## Validation
- Use `httptest` tables for unknown path (`404`), known path with wrong method (`405`), `Allow` header, `HEAD`, and `OPTIONS` preflight.
- For CORS, include `Origin`, `Access-Control-Request-Method`, and the relevant request method in the preflight request.
- For `HEAD`, assert both status and body behavior expected by the project. Do not infer contract support from a successful `GET` test.
- Suggested validation command: `go test ./... -run 'Test.*NotFound|Test.*MethodNotAllowed|Test.*Head|Test.*Options|Test.*Cors|Test.*CORS'`.

## Sources Gathered With Exa
- [chi package docs on pkg.go.dev for NotFound, MethodNotAllowed, Head, and Options](https://pkg.go.dev/github.com/go-chi/chi/v5)
- [go-chi README middleware list including GetHead](https://github.com/go-chi/chi/blob/master/README.md)
- [go-chi mux.go source for default 404 and 405 behavior](https://raw.githubusercontent.com/go-chi/chi/master/mux.go)
- [go-chi/cors docs for top-level middleware guidance](https://pkg.go.dev/github.com/go-chi/cors)
- [Go net/http docs for HTTP method and status constants](https://pkg.go.dev/net/http)


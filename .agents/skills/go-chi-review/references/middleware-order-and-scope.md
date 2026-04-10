# Middleware Order And Scope

## When To Read
Read this when a diff touches `Use`, `With`, `Group`, `Route`, `Mount`, middleware ordering, route-local middleware, logging, tracing, auth, panic recovery, body limits, or request context setup. The review question is whether runtime scope changed, not whether the code looks tidier.

## Review Smell Patterns
- Request ID, real IP, logger, recoverer, auth, tracing, body limit, or timeout middleware is reordered without a reason tied to runtime behavior.
- Middleware that depends on the final chi route pattern reads `RoutePattern()` before `next.ServeHTTP`.
- `Use(...)` is moved into or out of a `Group`/`Route`, silently widening or narrowing coverage.
- `With(...)` is called without immediately registering an endpoint on the returned inline router.
- A mounted child router is expected to inherit child-only middleware from a sibling `Group` or `Route`.
- CORS middleware is applied with `With` or inside a group without matching `OPTIONS` routes. Use the HTTP fallback reference for the CORS-specific finding.

## Minimal Examples

```diff
 func routeLabel(next http.Handler) http.Handler {
   return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
-    label := chi.RouteContext(r.Context()).RoutePattern()
-    record(label)
     next.ServeHTTP(w, r)
+    label := chi.RouteContext(r.Context()).RoutePattern()
+    record(label)
   })
 }
```

Review finding shape: route identity is incomplete before route resolution. The smallest safe fix is to record after `next` while the route context still belongs to the request.

```diff
-r.Use(adminOnly)
-r.Route("/admin", func(r chi.Router) {
-  r.Get("/users", listUsers)
-})
+r.Route("/admin", func(r chi.Router) {
+  r.Use(adminOnly)
+  r.Get("/users", listUsers)
+})
 r.Get("/status", status)
```

Review finding shape: this narrows auth coverage to `/admin` instead of every later route on the mux. If the old scope was intentional, keep it global and preserve registration order; if the new scope is intentional, require a test proving `/status` remains unauthenticated.

```diff
-r.With(auth)
-r.Get("/me", me)
+r.With(auth).Get("/me", me)
```

Review finding shape: `With` returns an inline router. The smallest safe fix is to register the endpoint on that returned router.

## Validation
- Use an `httptest` table that proves coverage, not just status codes. Record middleware execution in a slice or inject a test header from the middleware and assert where it appears.
- For order-sensitive stacks, build tiny middleware that appends names before and after `next`, then assert the sequence for a representative route.
- For scope changes, include at least one in-scope path and one sibling/out-of-scope path.
- Suggested validation command: `go test ./... -run 'Test.*Middleware|Test.*Scope|Test.*Auth'`.

## Sources Gathered With Exa
- [chi package docs on pkg.go.dev](https://pkg.go.dev/github.com/go-chi/chi/v5)
- [go-chi README middleware, Route, Group, With, and Mount examples](https://github.com/go-chi/chi/blob/master/README.md)
- [go-chi mux.go source for Use, With, Group, Route, and Mount behavior](https://raw.githubusercontent.com/go-chi/chi/master/mux.go)
- [go-chi context.go source for RoutePattern timing](https://raw.githubusercontent.com/go-chi/chi/master/context.go)
- [go-chi/cors docs for top-level CORS middleware scope](https://pkg.go.dev/github.com/go-chi/cors)


# Route Context And Match Probing

## When To Read
Read this when a diff reads `chi.RouteContext`, calls `RoutePattern`, uses `Match` or `Find`, computes custom `Allow` or `OPTIONS` responses, probes alternate methods, or builds observability labels from route state.

## Review Smell Patterns
- Middleware reads `RoutePattern()` before `next.ServeHTTP` and treats it as the final route template.
- Custom `405`, `Allow`, or `OPTIONS` code calls `router.Match(chi.RouteContext(r.Context()), method, path)`.
- A helper reuses one probe context across multiple paths or methods without resetting it.
- A probe mutates the live request route context and then the request continues to downstream handlers.
- A fallback label uses `r.URL.Path` after route-template extraction fails.

## Minimal Examples

```diff
 func allowed(router chi.Routes, r *http.Request, method string) bool {
-  rctx := chi.RouteContext(r.Context())
+  rctx := chi.NewRouteContext()
   return router.Match(rctx, method, r.URL.Path)
 }
```

Review finding shape: `Match` and `Find` update the `*chi.Context` they receive. The smallest safe fix is a fresh probe context per check, or an explicit reset before reuse when a loop needs to avoid allocations.

```diff
 func instrument(next http.Handler) http.Handler {
   return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
-    route := chi.RouteContext(r.Context()).RoutePattern()
     next.ServeHTTP(w, r)
+    route := chi.RouteContext(r.Context()).RoutePattern()
+    if route == "" {
+      route = "unmatched"
+    }
     recordRoute(route)
   })
 }
```

Review finding shape: read route templates after downstream routing, and keep unmatched labels bounded.

## Validation
- For probe helpers, call the helper for two methods on the same request and assert the live request still serves the original route correctly.
- For `Allow` logic, test at least one allowed method, one disallowed method on an existing path, and one method on an unknown path.
- For route label middleware, send two concrete parameter values, such as `/users/1` and `/users/2`, and assert they collapse to the same route template.
- Suggested validation command: `go test ./... -run 'Test.*Allow|Test.*RouteContext|Test.*RoutePattern|Test.*Probe'`.

## Sources Gathered With Exa
- [chi package docs on pkg.go.dev for Routes.Match](https://pkg.go.dev/github.com/go-chi/chi/v5)
- [go-chi mux.go source for Match and Find context mutation notes](https://raw.githubusercontent.com/go-chi/chi/master/mux.go)
- [go-chi context.go source for RouteContext, NewRouteContext, and RoutePattern timing](https://raw.githubusercontent.com/go-chi/chi/master/context.go)


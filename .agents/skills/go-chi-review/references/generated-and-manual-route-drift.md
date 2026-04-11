# Generated And Manual Route Drift

## When To Read
Read this when a diff touches OpenAPI-generated handlers, generated route registration, manual chi routes around generated handlers, fallback wrappers around generated handlers, or route documentation generated from chi routes.

## Review Smell Patterns
- A manual route claims the same method/path as a generated handler.
- A manual route is added inside the generated API subtree without passing through the same auth, body limit, fallback, CORS, or observability policy.
- A generated handler is mounted under a path prefix while manual routes use absolute paths that look equivalent but bypass the generated subtree.
- `NotFound`, `MethodNotAllowed`, or CORS policy is applied only to the generated handler or only to manual siblings.
- Route documentation or contract tests are updated for generated routes but not for manual chi additions.
- A no-touch generated file is edited instead of adapting the handwritten wiring around it.

## Minimal Examples

```diff
 r := chi.NewRouter()
 r.Mount("/api", generated.Handler(api))
-r.Get("/api/users/{id}", getUserManual)
+r.Get("/internal/users/{id}", getUserManual)
```

Review finding shape: this avoids split ownership of the generated `/api/users/{id}` resource. The smallest safe fix is to keep manual routes outside generated API ownership unless the approved contract says the manual handler replaces generated behavior.

```diff
-r.Mount("/v1", generated.Handler(api))
-r.Route("/v1", func(r chi.Router) {
-  r.Get("/debug/routes", debugRoutes)
-})
+v1 := chi.NewRouter()
+generated.RegisterHandlers(v1, api) // use the local generator's chi hook
+v1.Get("/debug/routes", debugRoutes)
+r.Mount("/v1", v1)
```

Review finding shape: consolidate one owner for `/v1` and prove fallback/middleware behavior for both generated and manual children.

```diff
-// edit generated server file to add one emergency route
+// leave generated files unchanged; add manual wiring in the router constructor
```

Review finding shape: generated artifacts are not a stable ownership surface for handwritten fixes.

## Validation
- Add `httptest` cases for every generated path touched by the diff plus each manual sibling path that shares the prefix.
- Assert generated and manual routes see the same required middleware side effects, such as auth failure, request ID, CORS headers, or route label format.
- When the project exposes chi route traversal, use `Routes()` or `chi.Walk` in a constructor test to detect duplicate method/path ownership. If traversal is not exposed, rely on explicit method/path request tests.
- Suggested validation command: `go test ./... -run 'Test.*Generated|Test.*OpenAPI|Test.*Routes|Test.*Router'`.

## Sources Gathered With Exa
- [chi package docs on pkg.go.dev for Routes and route traversal](https://pkg.go.dev/github.com/go-chi/chi/v5)
- [go-chi README on generated route docs and composable subrouters](https://github.com/go-chi/chi/blob/master/README.md)
- [go-chi mux.go source for Mount, NotFound, and MethodNotAllowed propagation](https://raw.githubusercontent.com/go-chi/chi/master/mux.go)

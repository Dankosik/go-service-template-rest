# Router Topology Patterns

## Behavior Change Thesis
When loaded for symptom `root router shape, Route/Mount/Group choice, top-level prefix ownership, generated/manual coexistence, or wildcard conflict`, this file makes the model choose one path owner with route-inventory proof instead of likely mistake `let modules, registration order, or broad wildcards decide ownership`.

## When To Load
Load when designing or revising the root router, top-level prefix owners, generated/manual route coexistence, `Route` vs `Mount` vs `Group`, broad catch-all routes, or route conflict controls before coding.

## Decision Rubric
- Start with path ownership, not package ownership. Name one owner for each top-level path family such as `/api`, `/admin`, `/internal`, `/debug`, or `/metrics`.
- Use one root composition point to own global transport middleware, fallback policy, and top-level mounts. Subrouters own path-local policy below that point.
- Use `Mount` when attaching a prebuilt `http.Handler` or generated handler subtree. Treat the mounted prefix as subtree ownership, not an exact-path shortcut.
- Use `Route` when building a chi subtree inline under a prefix. Middleware added inside the route belongs to that subtree only.
- Use `Group` when sharing middleware across sibling routes without implying a new path prefix.
- Keep generated OpenAPI paths and manual operational paths out of the same ownership zone unless the spec explicitly reserves the manual route.
- Treat catch-all and wildcard routes as fallback owners. Require proof that they do not hide API fallback, operational routes, or generated handlers.

## Imitate
```go
func NewHTTPHandler(api http.Handler, admin http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(requestID, recoverer)

	r.Mount("/api", api)     // generated API owner
	r.Mount("/admin", admin) // manual admin owner
	r.Get("/internal/health", health)
	return r
}
```

Copy the ownership shape: one root composition point, one owner per top-level prefix, and operational routes outside the generated API owner.

```go
func adminRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(adminOnly)
	r.Get("/", adminIndex)
	return r
}
```

Copy the subrouter shape: local middleware sits next to the path owner instead of drifting into the root.

```go
api := chi.NewRouter()
generated.RegisterHandlers(api, impl)
api.Get("/debug/routes", debugRoutes)
root.Mount("/api", api)
```

Copy only when the manual route is intentionally inside the API owner and must share the same API middleware, fallback, and observability policy.

## Reject
```go
root.Mount("/api", generated.Handler(impl))
root.Get("/api/debug/routes", debugRoutes)
```

Reject because generated and manual routes now share a prefix without one visible owner or collision rule.

```go
root.Get("/*", spaFallback)
root.Mount("/api", api)
```

Reject unless route inventory and HTTP tests prove the catch-all cannot hide API fallback behavior.

```go
func RegisterRoutes(r chi.Router) {
	users.Register(r)
	admin.Register(r)
	debug.Register(r)
}
```

Reject as the primary design shape when it hides which package owns each path family and which middleware/fallback policy applies.

## Agent Traps
- Do not describe `Route`, `Mount`, and `Group` as stylistic equivalents. They encode different ownership and middleware scope choices.
- Do not let "module owns its own routes" replace path ownership. The design needs both the module owner and the URL subtree owner.
- Do not place manual routes under a generated prefix just because it is convenient; decide whether they are part of the API contract first.
- Do not claim duplicate or shadowed routes are impossible without a route inventory or representative `httptest` proof.

## Validation Shape
- Route inventory contains every expected top-level prefix and no disallowed generated/manual duplicate method-path pairs.
- `httptest` proves one representative route from each prefix reaches the intended owner and middleware.
- Catch-all or wildcard designs include cases for a generated route, an API miss, an operational route, and an unrelated miss.

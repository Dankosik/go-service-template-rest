# Middleware Order And Scope

## Behavior Change Thesis
When loaded for symptom `middleware order or scope changed around Use, With, Group, Route, or Mount`, this file makes the model choose exact runtime coverage and order proof instead of likely mistake `treat nested middleware refactors as harmless cleanup`.

## When To Load
Load when a diff moves middleware between global, group, route, inline, or mounted scopes; reorders request ID, recovery, auth, tracing, logging, body-limit, or response-shaping middleware; or uses `With`/`Group` in a way that may not affect the intended endpoint.

## Decision Rubric
- `Use(...)` on a mux changes that mux's middleware stack for later routing. If it appears after route registration on the same mux, switch to `chi-router-registration-hazards.md`.
- `With(...)` returns an inline router with extra middleware. The middleware only applies to routes registered on the returned router.
- `Group(...)` copies the current stack and allows local additions. Middleware added inside the group does not cover siblings outside the group.
- `Route(...)` creates a child router and mounts it at the pattern. Middleware inside the route covers that subtree, not the parent or sibling routes.
- `Mount(...)` runs under parent middleware but does not inherit middleware from sibling groups. Do not assume a group around one subtree covers another mount.
- If the primary issue is route-template timing or `RoutePattern()` extraction, prefer `route-context-and-match-probing.md` or `route-observability-labels.md`.

## Imitate

```go
r.Route("/admin", func(r chi.Router) {
	r.Use(adminOnly)
	r.Get("/users", listUsers)
})
r.Get("/status", status)
```

Copy the reasoning: this intentionally narrows `adminOnly` to `/admin`; review asks whether that scope change is approved and tested.

```go
r.With(authRequired, audit).Get("/me", me)
```

Copy the shape: `With` is immediately used to register the endpoint on the inline router it returns.

```go
var hits []string
mw := func(name string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits = append(hits, name+":before")
			next.ServeHTTP(w, r)
			hits = append(hits, name+":after")
		})
	}
}
```

Copy the proof shape: order-sensitive middleware reviews should suggest tests that observe execution order, not only final status codes.

## Reject

```go
r.With(authRequired)
r.Get("/me", me)
```

Reject because the route is registered on the parent, not the inline router returned by `With`.

```go
r.Group(func(r chi.Router) {
	r.Use(adminOnly)
})
r.Get("/admin/users", listUsers)
```

Reject because the group-local middleware is not attached to the sibling route registered after the group.

```go
r.Route("/api", func(r chi.Router) {
	r.Use(apiOnly)
})
r.Mount("/api", generated.Handler(api))
```

Reject because the mounted handler is a sibling of the route block, not a child covered by its middleware.

## Agent Traps
- Do not assume a prettier nesting shape preserves middleware coverage.
- Do not approve a `With(...)` call unless the endpoint is registered on the returned router.
- Do not treat auth or body-limit movement as style. Ask which paths gained or lost coverage.
- Do not duplicate the route-label timing finding here when the review is really about route context or telemetry labels; load the narrower reference.

## Validation Shape
- Scope change: test one in-scope path and one sibling/out-of-scope path.
- Order change: use test middleware that records before/after sequencing.
- Coverage change: assert an observable side effect, such as a header, auth rejection, request ID, or body-limit response.
- Suggested command: `go test ./... -run 'Test.*Middleware|Test.*Scope|Test.*Auth|Test.*BodyLimit|Test.*Trace'`.

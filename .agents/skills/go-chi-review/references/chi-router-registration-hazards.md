# Chi Router Registration Hazards

## Behavior Change Thesis
When loaded for symptom `router construction order, Mount, catch-all, or subtree ownership changed`, this file makes the model choose a startup-safety or subtree-ownership finding instead of likely mistake `treat the diff as style, generic duplicate routing, or harmless registration order`.

## When To Load
Load when a review diff changes `Use`, `Route`, `Mount`, catch-all/wildcard paths, duplicate subtree owners, or nil handlers passed into router composition.

## Decision Rubric
- On one mux, `Use(...)` belongs before the first route-like registration that builds the handler stack. A later `Use` is startup panic risk, not a style preference.
- `Route(pattern, fn)` is a mount shorthand. A nil `fn` is a constructor panic, and the child router still owns the mounted subtree.
- `Mount("/x", h)` owns `/x`, `/x/`, and `/x/*`. Two owners for that subtree should become one owner that delegates below `/x`.
- Do not rely on the duplicate-mount panic as the only conflict signal. An exact route such as `Get("/x", ...)` plus a later `Mount("/x", ...)` is still an ownership defect even when it presents as overwrite or shadowing rather than the documented duplicate-mount panic.
- Wildcard handlers such as `/files/*` and a later mount at `/files` are ownership conflicts until the diff proves one is intentionally removed or nested.
- If the conflict is generated/manual ownership without a constructor panic, prefer `generated-and-manual-route-drift.md` as the primary reference.
- If the code probes routes with `Match` or `Find`, prefer `route-context-and-match-probing.md` for the context-mutation part.

## Imitate

```go
r.Use(requestID)
r.Get("/healthz", healthz)
```

Copy the review posture: middleware order is safe because the global stack is finalized before route registration.

```go
v1 := chi.NewRouter()
v1.Mount("/public", publicRouter())
v1.Mount("/admin", adminRouter())
r.Mount("/v1", v1)
```

Copy the ownership shape: one parent owns `/v1`, and children divide responsibility beneath it.

```go
func TestNewRouterDoesNotPanic(t *testing.T) {
	defer func() {
		if got := recover(); got != nil {
			t.Fatalf("NewRouter panicked: %v", got)
		}
	}()
	_ = NewRouter()
}
```

Copy the proof shape: constructor tests catch registration panics before request-path tests can run.

## Reject

```go
r.Get("/healthz", healthz)
r.Use(requestID)
```

Reject because chi panics when middleware is added after routes on the same mux.

```go
r.Get("/files/*", legacyFiles)
r.Mount("/files", filesRouter())
```

Reject because both registrations claim the same mounted subtree. Do not present this as a benign "more specific route wins" pattern unless the repo proves that exact behavior and the change avoids chi's mount conflict.

```go
r.Mount("/api", generated.Handler(api))
r.Mount("/api", manualAPI())
```

Reject because one path prefix has two mounted owners. The safe review fix is a single `/api` owner that delegates internally.

## Agent Traps
- Do not downgrade constructor panics to style nits because no request has been served yet.
- Do not describe `Mount("/x", h)` as an exact-path registration.
- Do not suggest "move the route below the middleware" when the safer fix is route-local `With`/`Group` middleware for only one endpoint.
- Do not use source-link quotes as the finding. Explain the subtree ownership or startup failure in runtime terms, then suggest the smallest local repair.

## Validation Shape
- Constructor panic risk: add a router-construction test with `recover`.
- Ownership ambiguity: add `httptest` cases for the mount root, trailing slash, and at least one child path.
- Suggested command: `go test ./... -run 'TestNewRouter|TestRoutes|Test.*Mount|Test.*Router'`.

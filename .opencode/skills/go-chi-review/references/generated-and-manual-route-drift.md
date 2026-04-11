# Generated And Manual Route Drift

## Behavior Change Thesis
When loaded for symptom `OpenAPI/generated handlers and manual chi routes overlap or share wrappers`, this file makes the model choose one visible route owner with policy parity instead of likely mistake `patch generated files or add manual routes that shadow the generated contract`.

## When To Load
Load when a diff touches OpenAPI-generated handlers, generated route registration, manual chi routes under a generated prefix, fallback wrappers around generated handlers, route docs generated from chi traversal, or no-touch generated files.

## Decision Rubric
- Generated files are not the handwritten ownership surface. Prefer adapting the router constructor or generator hook over editing generated output.
- One prefix should have one obvious owner. If generated and manual routes both claim `/api`, compose them under a single parent router or move the manual route outside the generated contract.
- Manual exceptions inside a generated subtree must pass through the same auth, body-limit, CORS, fallback, and observability policy unless the approved contract says otherwise.
- Same method/path as a generated handler is a replacement decision, not a local cleanup. Escalate or require an approved contract reference.
- If the overlap is a chi `Mount` panic, load `chi-router-registration-hazards.md` as the primary reference.
- If the issue is only fallback/`Allow`/CORS behavior, load `http-fallback-head-options-cors.md`.

## Imitate

```go
v1 := chi.NewRouter()
generated.RegisterHandlers(v1, api)
v1.Get("/debug/routes", debugRoutes)
r.Mount("/v1", v1)
```

Copy the composition shape: one `/v1` owner, with generated and manual children sharing the same parent policy.

```go
r.Mount("/api", generated.Handler(api))
r.Get("/internal/users/{id}", getUserManual)
```

Copy the ownership shape when the manual route is not part of the generated API contract: put it outside the generated prefix.

```go
if strings.Contains(path, "/internal/") {
	t.Fatalf("generated API traversal unexpectedly includes internal route %s", path)
}
```

Copy the proof shape when route traversal exists: detect contract pollution directly instead of relying only on happy-path requests.

## Reject

```go
r.Mount("/api", generated.Handler(api))
r.Get("/api/users/{id}", getUserManual)
```

Reject because manual and generated surfaces both appear to own the same resource path.

```go
// edit generated server file to add emergency route
```

Reject because generated files are not a stable handwritten patch surface.

```go
r.Mount("/v1", generated.Handler(api))
r.Route("/v1", func(r chi.Router) {
	r.Get("/debug/routes", debugRoutes)
})
```

Reject because the same prefix now has separate owners with potentially different fallback, middleware, and traversal behavior.

## Agent Traps
- Do not call generated/manual overlap a style problem; it can change `404`, `405`, `Allow`, `OPTIONS`, middleware, docs, and traversal behavior.
- Do not suggest editing generated files as the smallest safe fix unless the project explicitly treats them as handwritten source.
- Do not miss manual routes that look equivalent only because of a mount prefix.
- Do not report fallback drift without also checking whether route ownership should be unified first.

## Validation Shape
- Add `httptest` cases for each touched generated path plus each manual sibling sharing the prefix.
- Assert generated and manual routes observe the same required middleware side effects when they share a contract surface.
- If route traversal is available, use `Routes()` or `chi.Walk` to detect duplicate or polluted method/path ownership.
- Suggested command: `go test ./... -run 'Test.*Generated|Test.*OpenAPI|Test.*Routes|Test.*Router|Test.*Contract'`.

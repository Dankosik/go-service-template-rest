# OpenAPI And oapi-codegen Integration

## Behavior Change Thesis
When loaded for symptom `OpenAPI-generated chi handlers, oapi-codegen strict server wiring, BaseURL/mount prefix choice, generated/manual route boundaries, or generated-code ownership`, this file makes the model choose one generated API owner and config/wrapper changes instead of likely mistake `edit generated files, double-prefix routes, or add manual handlers inside the generated contract surface`.

## When To Load
Load when chi routing is generated from OpenAPI, `oapi-codegen` chi or strict server wiring is in scope, `BaseURL` and mount prefix placement are disputed, generated and manual routes coexist, or OpenAPI path matching affects router topology.

## Decision Rubric
- Keep OpenAPI as the source of truth for public API paths, methods, parameters, and response shape. Chi wiring serves that contract.
- Choose one generated router target for the surface. Combine `chi-server` with `strict-server` only when typed request/response wrappers are desired.
- Do not treat `strict-server` or generated chi wrappers as full request or security validation. If contract validation is required, choose the validation middleware explicitly and place it once in the chi stack.
- Choose exactly one prefix source: the OpenAPI path itself, `ChiServerOptions.BaseURL`, or the parent chi mount/route prefix. Do not accidentally apply the same prefix twice.
- Keep manual routes outside the generated contract surface unless they are intentionally represented in the OpenAPI spec or share one parent router with the same policy.
- Treat generated files as generated artifacts. Change generator config, templates, wrappers, or hand-written implementations instead of editing generated code manually.
- Record compatibility settings that affect routing or middleware semantics, especially generated chi middleware execution order.

## Imitate
```yaml
package: api
generate:
  chi-server: true
  strict-server: true
  models: true
compatibility:
  apply-chi-middleware-first-to-last: true
```

Copy the config shape when middleware order affects behavior: the setting is visible and reviewable.

```go
apiRouter := chi.NewRouter()
api.HandlerWithOptions(impl, api.ChiServerOptions{
	BaseRouter: apiRouter,
	BaseURL:    "",
})
root.Mount("/api", apiRouter)
```

Copy only when generated paths are relative to `/api`; the parent mount is the prefix source.

```go
api.HandlerWithOptions(impl, api.ChiServerOptions{
	BaseRouter: root,
	BaseURL:    "/api",
})
```

Copy only when `BaseURL` is the prefix source and the handler is not also registered under an `/api` subrouter.

## Reject
```go
root.Route("/api", func(r chi.Router) {
	api.HandlerWithOptions(impl, api.ChiServerOptions{
		BaseRouter: r,
		BaseURL:    "/api",
	})
})
```

Reject unless the intended route really is double-prefixed; both the parent route and `BaseURL` are claiming the same prefix.

```go
// edit openapi.gen.go to add one missing route
```

Reject because generated files are not the stable design surface.

```go
root.Mount("/v1", generated.Handler(impl))
root.Get("/v1/admin/metrics", manualMetrics)
```

Reject when the manual route is not in the OpenAPI contract or does not share the generated surface's policy.

## Agent Traps
- Do not say "OpenAPI owns the contract" while hand-writing a sibling route under the generated prefix.
- Do not hide a `BaseURL` or mount-prefix decision in example code; state which layer owns the prefix.
- Do not patch generated files as the design answer. Prefer config, wrapper, or handwritten implementation changes.
- Do not assume generated middleware order; require config or generated-output proof when order matters.

## Validation Shape
- Generated config names the router target and strict-server choice.
- Route inventory or representative `httptest` proves each expected OpenAPI operation appears at the intended full path.
- Prefix proof includes at least one path that would expose accidental double-prefixing.
- Generated/manual coexistence proof detects duplicate method-path ownership.
- Generated files retain their generated-code marker and are not hand-edited.

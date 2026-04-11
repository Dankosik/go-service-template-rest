# OpenAPI And oapi-codegen Integration

## When To Load
Load this when chi routing is generated from OpenAPI, when `oapi-codegen` strict server or chi server wiring is in scope, when generated and manual routes coexist, or when OpenAPI path matching affects router topology.

## Recommended Design Options
- Keep OpenAPI as the source of truth for public API paths, methods, parameters, and response shape. Let chi integration serve that contract.
- Choose one `oapi-codegen` server mode for the generated surface, commonly `chi-server` plus `strict-server` when typed request/response objects are desired.
- Mount or register generated handlers under one explicit prefix and keep manual routes out of that ownership zone unless they are intentionally represented in the spec.
- Treat generated files as generated artifacts. Change config, templates, wrappers, or hand-written implementations instead of editing generated code manually.
- Record codegen compatibility settings that affect routing or middleware semantics, especially chi middleware execution order.

## Rejected Alternatives
- Splitting public API path ownership between OpenAPI and hand-written chi routes without a collision rule.
- Editing generated code to fix routing policy. That creates regeneration drift.
- Using OpenAPI for payload contracts while letting chi route templates diverge in path names, prefixes, or method ownership.
- Hiding generated middleware inside a wrapper without documenting whether it runs before or after chi route-level middleware.

## Example Sketches
```yaml
package: api
generate:
  chi-server: true
  strict-server: true
  models: true
compatibility:
  apply-chi-middleware-first-to-last: true
```

```go
r := chi.NewRouter()
r.Route("/api", func(r chi.Router) {
	api.HandlerWithOptions(impl, api.ChiServerOptions{
		BaseRouter: r,
		BaseURL:    "/api",
	})
})
```

## Testable Acceptance Boundaries
- Generated config names the server mode and whether strict server is enabled.
- Route inventory proves every OpenAPI operation is registered at the expected prefix and no manual route collides with it.
- Middleware-order tests or generated-output inspection prove wrapper middleware order when behavior depends on it.
- Generated files include a generated-code marker and are not hand-edited.
- If OpenAPI templated paths differ from chi path templates, the design explains and tests the mapping.

## Source Links Gathered Through Exa
- oapi-codegen README for chi server and strict server generation: https://github.com/oapi-codegen/oapi-codegen/blob/09919e79/README.md
- oapi-codegen configuration source for `chi-server`, `strict-server`, and compatibility options: https://github.com/oapi-codegen/oapi-codegen/blob/09919e79/pkg/codegen/configuration.go
- oapi-codegen generated chi strict-server example: https://github.com/oapi-codegen/oapi-codegen/blob/09919e79/internal/test/strict-server/chi/server.gen.go
- OpenAPI Specification 3.1.2 for path templating and path matching rules: https://spec.openapis.org/oas/v3.1.2.html
- chi README for route patterns and `net/http` compatible routing: https://github.com/go-chi/chi/blob/master/README.md

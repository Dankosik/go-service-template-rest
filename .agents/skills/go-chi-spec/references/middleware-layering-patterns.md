# Middleware Layering Patterns

## When To Load
Load this when deciding middleware order, global vs scoped placement, route-local `With`/`Group` usage, generated middleware wrappers, request context mutation, logging, panic recovery, or middleware-order validation.

## Recommended Design Options
- Separate global transport middleware from path-local policy. Put request ID, panic recovery, common framing, and common logging at the root when they truly apply to every route.
- Put authorization, tenant policy, admin-only checks, and API-only CORS on the smallest router branch that owns the path set.
- Preserve chi's registration rule: `Use` middleware before route registration; use `Group`, `Route`, or `With` for additional path-local or endpoint-local middleware.
- For generated `oapi-codegen` chi wrappers, define whether generated handler middleware or chi router middleware owns the concern. Do not duplicate the same concern in both layers.
- For order-sensitive work, write down the exact stack as it should execute from outermost to innermost.

## Rejected Alternatives
- Global auth by default. It can accidentally cover health, debug, or preflight routes and blur trust-boundary policy.
- Logging or metrics that assume the final chi route pattern exists before the downstream handler has run. Use post-handler extraction or a version-supported lookup path instead.
- Reordering middleware without naming behavior impact on headers, context values, body reads, panic recovery, or status capture.
- Passing generated `oapi-codegen` middleware lists without checking the active compatibility setting for chi middleware order.

## Example Sketches
```go
r := chi.NewRouter()
r.Use(requestID)
r.Use(recoverer)
r.Use(accessLogAfterNext)

r.Route("/api", func(r chi.Router) {
	r.Use(apiCORS)
	api.RegisterHandlers(r, impl)
})

r.Route("/admin", func(r chi.Router) {
	r.Use(adminAuth)
	r.Get("/", adminIndex)
})
```

```go
// Order statement, not implementation prescription:
// request id -> recovery -> CORS preflight -> auth -> generated validation -> handler -> post-handler route label.
```

## Testable Acceptance Boundaries
- Tests or design notes identify which middleware runs globally, per prefix, and per endpoint.
- A route-local policy change proves unaffected paths are still reachable with expected status and headers.
- Panic/recovery tests cover at least one API route when recovery order changes.
- Logging/metrics tests prove status, route label, and request ID are available at the point where telemetry is emitted.
- If `oapi-codegen` generated middleware is used, generated output or configuration proves the expected first-to-last or legacy order.

## Source Links Gathered Through Exa
- go-chi README, middleware examples and stdlib `net/http` middleware model: https://github.com/go-chi/chi/blob/master/README.md
- chi mux source for `Use`, `With`, `Group`, route handler construction, and middleware registration timing: https://github.com/go-chi/chi/blob/master/mux.go
- chi context source for route-pattern timing after `next.ServeHTTP`: https://github.com/go-chi/chi/blob/master/context.go
- oapi-codegen configuration source for `apply-chi-middleware-first-to-last`: https://github.com/oapi-codegen/oapi-codegen/blob/09919e79/pkg/codegen/configuration.go
- oapi-codegen issue on chi middleware order compatibility: https://github.com/deepmap/oapi-codegen/issues/786

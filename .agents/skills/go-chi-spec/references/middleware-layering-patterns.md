# Middleware Layering Patterns

## Behavior Change Thesis
When loaded for symptom `global vs scoped middleware, exact execution order, request context mutation, panic recovery, body limits, logging, or generated middleware order`, this file makes the model choose explicit scope and outer-to-inner stack semantics instead of likely mistake `make auth/CORS/logging global or assume route identity is available before routing finishes`.

## When To Load
Load when the design must decide middleware placement, stack order, global vs route-local scope, `With`/`Group`/`Route` behavior, generated wrapper middleware, context mutation, body limits, recovery, logging, tracing, or route-label timing.

## Decision Rubric
- Separate global transport concerns from path-local policy. Request correlation, panic recovery, framing guards, and common access logging may be global only when they truly apply to every path.
- Put authorization, tenant checks, admin-only policy, API-only CORS, and generated request validation on the smallest router branch that owns the path set.
- Write the stack in execution order from outermost to innermost whenever order changes status capture, panic handling, body reads, context values, tracing, or route labels.
- For telemetry that needs the final route template, design the middleware so route-label capture happens after downstream routing.
- For generated `oapi-codegen` middleware, define whether the generated wrapper or chi router middleware owns the concern. Do not implement auth, CORS, validation, or telemetry twice.
- If generated middleware order matters, require the active config or generated output to prove the order rather than assuming the current `oapi-codegen` default.

## Imitate
```go
r := chi.NewRouter()
r.Use(requestID)
r.Use(recoverer)
r.Use(accessLogAfterNext) // reads final route label after next.ServeHTTP

r.Route("/api", func(r chi.Router) {
	r.Use(apiCORS)
	r.Use(apiAuth)
	api.RegisterHandlers(r, impl)
})

r.Route("/admin", func(r chi.Router) {
	r.Use(adminAuth)
	r.Get("/", adminIndex)
})
```

Copy the scope shape: API and admin policy sit on their owning subtrees, while root middleware stays transport-wide.

```go
// Design statement:
// request correlation -> recovery -> body limit -> access log wrapper -> API CORS -> API auth -> generated validation -> handler -> post-handler route label capture.
```

Copy the order shape: it names behavior, not just middleware function names.

## Reject
```go
r.Use(globalAuth)
r.Use(corsForPublicAPI)
r.Get("/internal/health", health)
r.Mount("/debug", pprofHandler)
r.Mount("/api", api)
```

Reject when health or debug routes should not receive public API auth/CORS behavior.

```go
r.Use(func(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record(chi.RouteContext(r.Context()).RoutePattern())
		next.ServeHTTP(w, r)
	})
})
```

Reject because final route identity is read before the downstream handler has resolved the route.

```go
api.HandlerWithOptions(server, api.ChiServerOptions{
	Middlewares: []api.MiddlewareFunc{auth, cors, auth},
})
```

Reject duplicate concern ownership unless the design explains why the repeated middleware is intentional and harmless.

## Agent Traps
- Do not make auth or CORS global because it is easier to describe. Scope it to the path owner unless global behavior is part of the contract.
- Do not write "middleware order: standard stack" as a decision. Name the order and the behavior each sensitive position protects.
- Do not assume a generated middleware list runs in the intuitive order; require generated output or config proof.
- Do not solve route-label timing in this file if the task is mainly telemetry cardinality. Load `route-template-observability.md`.

## Validation Shape
- Scope proof hits one route that should receive the middleware and one sibling route that should not.
- Order proof uses test middleware or observable headers/status to show before/after sequencing.
- Recovery/order changes include one panic path and one normal path.
- Telemetry middleware proof shows status, request ID, and bounded route label are available where logging or metrics emit.

# Route Tree And Middleware

## Load When
Load when a change adds or moves middleware, adds a path, wires a generated handler, or introduces a router beside the one `internal/infra/http/router.go` builds.

## What Already Owns This
- `Harden` owns the transport chain, and its doc comment states the order outermost-first. It takes any `http.Handler`, so a service with its own generated contract inherits correlation, tracing, access logging, body limit, request budget, shedding, rate limiting, and recovery by passing its API handler in.
- `newRootRouter` mounts that handler at `/`. There is no prefix and no `ChiServerOptions.BaseURL`, so a path question here is about what the OpenAPI contract declares, not about chi prefixes.
- `TestOpenAPIRuntimeContractRootRouteTreeContainsOnlyGeneratedRoutes` walks the root tree and fails on any route the embedded spec does not declare.

## Placing New Middleware
Transport-wide behavior belongs in the `Harden` chain, in the position its consequence requires — the existing entries record why each sits where it does. Behavior that must run inside routing but outside the operation belongs in `ChiServerOptions.Middlewares`, where the request validator already sits.

That slice is applied `handler = mw(handler)` in index order by oapi-codegen v2.8.0's default chi template, so **its last entry is outermost**. Both routers here pass exactly one middleware, which leaves the ordering latent until a second arrives. `compatibility.apply-chi-middleware-first-to-last` reverses the loop and therefore reverses every service already generated without it.

`chi.Mux.Use` panics once the middleware stack is built, and `Group`, `With`, and `HandlerWithOptions` with a `BaseRouter` all build it. Middleware for a router the generated handler shares is registered before that call.

## Reject
- A hand-built second chain instead of passing the handler to `Harden`: the service loses every protection above while its router reads complete.
- A manual chi route beside the mounted API handler: it is served without the contract validator, and on this template's own root it fails the route-tree walk above.

## Proof
Construct the router in a test — registration order fails at construction, not at request time. Prove topology with `chi.Walk` over the built tree, and prove order with middleware that records its own before/after sequence; status codes alone cannot distinguish two orders.

# Fallbacks And Method Policy

## Load When
Load when a change touches fallback responses, method discovery, the `Allow` header, `HEAD` or `OPTIONS` behavior, or cross-origin preflight.

## What Already Owns This
`applyHTTPPolicy` in `internal/infra/http/router.go` owns 404, 405, `Allow`, and bare `OPTIONS` for every router built through `NewRouter` or `Harden`. A service inherits that policy; it does not re-register its own.

## Why Allow Is Rebuilt By Hand
chi's built-in 405 handler writes `Allow` from the methods it matched. Setting a custom `MethodNotAllowed` handler replaces that handler outright — chi still computes the allowed set but has nowhere to hand it, so the header silently disappears. Any JSON 405 pays this cost.

`allowedMethodsForPath` is the repair: it probes `boundedHTTPMethods` against the root router, one **fresh** `chi.NewRouteContext()` per method. Each probe needs its own context because `Match` and `Find` write into the context they are given. Handing them the live request's context rewrites `RoutePattern()` and the `*` URL param, which is what the access log and `http.route` read — the request ends up labelled with a route it never matched.

`HEAD` is matched exactly, never implied by `GET`, so it appears in `Allow` only when the contract declares it. `boundedHTTPMethods` leaves CONNECT and TRACE unprobed; the reason is recorded beside it.

## Cross-Origin Requests
Preflight is fail-closed: an `OPTIONS` carrying `Origin` and `Access-Control-Request-Method` gets 405 with `Allow`. Serving it means adding a CORS dependency and deciding which origins, methods, headers, and credentials are trusted — an API-contract and security decision that arrives here already made.

## Reject
- A `MethodNotAllowed` handler writing a literal `Allow` list: it drifts from the route tree and can advertise a method the router answers with 404.
- `Match` or `Find` called with `chi.RouteContext(r.Context())`: it overwrites live routing state mid-request.

## Proof
Keep the missing-path and wrong-method cases separate, and assert the exact `Allow` set rather than its presence. Preflight assertions carry both request headers, since a bare `OPTIONS` takes a different branch. `internal/infra/http/router_contract_test.go` holds the current cases.

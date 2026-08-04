# Abuse And Cost Bounds

## When To Load

Load this when a caller can scale work, repeat an attempt, or drive cost —
pagination, batching, fan-out, retries, or a paid provider call.

## Behavior Change Thesis

Without this file, "the service has rate limiting" is read off the middleware
chain and the endpoint is treated as covered. It is not. `RateLimit` is a seam:
`RouterConfig.RateLimit` is nil in the shipped default, so the middleware is
left out of the chain entirely, and there is deliberately no default key
function — the repository refuses to guess an identity, because behind an edge
`RemoteAddr` is the edge and `X-Forwarded-For` is caller-controlled.

## Decision Rubric

- Name the scarce resource and the identity a budget is charged to before
  choosing a control. Charging is the decision; the limiter is mechanism.
- `MaxInFlight` sheds when concurrent handlers exceed capacity and
  `RequestTimeout` bounds a single request. Neither provides per-caller
  fairness: one client holding every slot gets everyone else shed for the
  capacity it took.
- Wiring `RateLimit` requires supplying `RateLimitKey`; the router refuses one
  without the other at construction. `HeaderRateLimitKey` hashes the header
  value, because the header identifying a caller before authentication is
  usually the credential itself, and a credential used as a map key is one heap
  dump from disclosure.
- The chain order is
  `RequestTimeout -> MaxInFlight -> RateLimit -> Recover -> apiSubrouter`, so
  the limiter runs ahead of the OpenAPI validator and therefore ahead of the
  resolved principal. Charging an authenticated caller means installing a second
  `RateLimit` inside the service's own chain and keying on
  `reqctx.PrincipalFromContext`.
- Clamp caller-supplied dimensions — page size, batch length, fan-out depth — to
  a server-owned maximum at the boundary, ahead of the work they scale. A
  default page size is not a maximum.
- Put the cheap check before the expensive effect. A budget consulted after the
  provider call, the enqueue, or the row scan has already paid for the abuse.

## Reject

- A global limiter as the answer to per-tenant abuse: it throttles unrelated
  callers while the abusive one stays comfortably inside the shared budget.
- Degrading open on a security dependency under load: shedding may drop a
  request, never its authorization or its tenant scope.

## Validation Shape

Assert the denial and the absence of the effect — no provider call, no enqueue,
no mutation — because a `429` alone does not show the work was skipped.
`middleware_ratelimit_test.go` and `middleware_inflight_test.go` hold the
transport-level cases; run `make test-race` when fan-out or worker bounds
change. Detailed worker-pool lifetime and race questions belong to
`go-concurrency`.

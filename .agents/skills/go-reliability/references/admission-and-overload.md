# Admission And Overload

## Behavior Change Thesis
When loaded for symptom `work that can arrive faster than it completes, or a new limiter, queue, pool, or breaker`, this file makes the model bound the work in the admission layer that already exists instead of likely mistake `add a semaphore, bounded queue, or circuit breaker beside the ones the service already runs`.

## When To Load
Load when a change adds concurrency, fan-out, queued work, a new pooled dependency, or a rejection path — or proposes a limiter, bulkhead, or circuit breaker.

## Three Admission Layers Already Exist
Outermost first, and each covers what the next cannot see:

- `boundedAPIListener` in `cmd/service/internal/bootstrap/startup_server.go` caps accepted connections at `http.max_connections` (4096). Excess callers wait in the kernel accept queue, which costs this process nothing. A connection rejected inside a handler has already bought a goroutine, a read buffer, a write buffer, and a header parse.
- `MaxInFlight` in `internal/infra/http/middleware_inflight.go` caps concurrent handler execution at `http.max_in_flight` (256) and sheds the rest with `TryAcquire` — never `Acquire`, because blocking rebuilds the queue the middleware exists to prevent, one layer further out. A shed answers 503 with `Retry-After: 1`, and `telemetry.ServerLoad` records `Shed` and `Admitted` so the limit is observable rather than a number set once from a guess.
- `RateLimit` in `internal/infra/http/middleware_ratelimit.go` answers 429 for a caller over its own budget. It is nil by default.

Chain order is semantics, not arrangement: `RequestCorrelation → OTel → SecurityHeaders → AccessLog → RequestBodyLimit → RequestTimeout → MaxInFlight → RateLimit → Recover → apiSubrouter`. Shedding sits inside `RequestTimeout` so it is timed and access-logged, and outside `Recover` because it never runs a handler.

The three statuses are already allocated: 503 is at-capacity or not-ready, 429 is over-budget-for-this-caller, 504 is this request's own budget expired. Collapsing any two removes a distinction an operator uses during an incident.

Health probe routes are exempt from shedding. Shedding a readiness probe evicts the instance for being busy, which is the opposite of what shedding is for.

## What A Service Still Has To Decide
`RateLimit` ships with no default key function on purpose. Behind a proxy `RemoteAddr` is the proxy, so limiting by it throttles the whole fleet as one caller; `X-Forwarded-For` is attacker-controlled unless the edge topology is known, which a template cannot know. `HeaderRateLimitKey` hashes the header because the header identifying a caller before authentication is usually the credential itself.

Per-dependency isolation is already the pattern for the database: `postgres.max_open_conns` (25) plus the 1s acquire budget returning `ErrSaturated`, so a slow database sheds its own callers rather than holding in-flight slots that requests touching no database need. A new pooled dependency earns isolation the same way.

There is no circuit breaker in this tree and no breaker dependency in `go.mod`. Adding one is a new dependency and a new state machine on a path that already fails fast; its probe traffic also creates effects, which `external-api-integration` owns. `httpclient.RetryPolicy` is disabled by default for the same reason: how many attempts a dependency can absorb is a per-dependency decision, not a template default.

Non-HTTP work belongs to `background.Supervisor`, which owns cancellation, panic containment, and the join — not schedules, retries, or locking. Work that must outlive the process goes to `durable-background-jobs`.

## Reject
- An unbounded channel or slice as a spike absorber: it converts overload into memory pressure and a later, worse failure.
- A second limiter beside `MaxInFlight` for the same resource. Two limits on one resource means neither number describes the service.

## Proof
`go test ./internal/infra/http/... -run 'InFlight|RateLimit|Shed'`. Fill the limiter and assert the rejection returns promptly with its documented status, rather than that the limiter exists.

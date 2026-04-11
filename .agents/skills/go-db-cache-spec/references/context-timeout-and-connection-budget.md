# Context Timeout And Connection Budget

## Behavior Change Thesis
When loaded for slow cache/origin behavior, request cancellation, pool saturation, or deadline ambiguity, this file makes the model choose an explicit cache-origin-pool budget hierarchy instead of likely mistake `trust the handler timeout or raise the pool limit without fallback math`.

## When To Load
Load this when the spec must define request context propagation, DB/cache deadlines, transaction cancellation, connection pool capacity, dedicated connection use, or timeout hierarchy.

Keep the focus on runtime contracts and proof obligations. Do not turn this into low-level implementation tuning unless the spec needs an explicit acceptance boundary.

## Decision Rubric
- Use the inbound request context as the parent for ordinary request work so caller cancellation can stop DB/cache operations.
- Use a bounded detached context only for an approved side effect that must outlive the caller; define deadline, retry/observation path, and reconciliation.
- Make cache budget shorter than origin budget for read acceleration, leaving time for fallback and response mapping.
- Count pool wait against the caller deadline; pool sizing must account for service replicas, transaction duration, DB limits, and other app instances.
- Reserve a dedicated `sql.Conn` only when one connection is required and a transaction is not the better model; state release obligations.
- Treat a larger pool as capacity policy, not a latency fix, unless the acceptance proof includes pool wait and DB headroom.

## Imitate
- Redis catalog lookup: small Redis budget, bounded PostgreSQL budget, total request deadline, Redis timeout treated as miss, and fallback concurrency cap. Copy the habit of reserving time for the origin.
- Write transaction: `BeginTx(ctx, ...)` uses the use-case deadline; cancellation before commit rolls back and outward response follows the commit-outcome policy. Copy the habit of tying cancellation to outcome semantics.
- Background audit write: bounded background context because it must survive client disconnect, plus an observation or retry path. Copy only when the side effect is explicitly allowed to outlive the request.

## Reject
- DB/cache calls without deadlines because the handler has a timeout. Reject because implicit outer timeouts are easy to bypass at dependency boundaries.
- Large `MaxOpenConns` without replica count, transaction duration, and DB-capacity math. Reject because it can move the bottleneck to the database.
- Cache retry loop that consumes the whole request budget and leaves no time for origin fallback. Reject because fail-open becomes fail-slow or fail-closed.

## Agent Traps
- Do not propose detached contexts to make cancellation bugs disappear; they require a business reason and reconciliation.
- Do not describe pool tuning without validation signals such as wait count, wait duration, in-use connections, and DB headroom.
- Do not let a stale cache hit hide an origin timeout unless the freshness contract explicitly permits stale serve.

## Validation Shape
- Cache timeout normally behaves like a miss for read acceleration paths.
- DB timeout is an origin failure and must map to the API or workflow error contract; it should not be hidden as a stale cache hit unless stale serving is explicitly allowed.
- Transaction context cancellation rolls back the transaction; commit errors require the spec to define unknown-outcome handling.
- Pool waits count against the caller's deadline. If pool wait is material, the acceptance checks should include pool wait or saturation telemetry.
- The spec states parent context, DB deadline, cache deadline, and total request deadline relationship.
- Cache deadline is shorter than origin deadline unless an explicit exception is justified.
- Fallback concurrency is bounded so cache outage does not overload the origin.
- Connection pool assumptions include max open connections, expected in-flight work, replicas, and database limit headroom.
- Dedicated connections are released and justified by a single-connection requirement.
- Resource-return obligations are named for implementations: close rows, check row iteration errors, and end every transaction with commit or rollback.

# Context Timeout And Connection Budget

## When To Load
Load this when the spec must define request context propagation, DB/cache deadlines, transaction cancellation, connection pool capacity, dedicated connection use, or timeout hierarchy. Use it before coding any DB/cache path where slow cache, slow origin, or pool exhaustion can change behavior.

Keep the focus on runtime contracts and proof obligations. Do not turn this into low-level implementation tuning unless the spec needs an explicit acceptance boundary.

## Viable Options
- Request-derived context: use the inbound request context as the parent so client cancellation can stop DB/cache work.
- Bounded detached context: use only when an approved side effect must outlive the client request; still set a short deadline and define outcome reconciliation.
- Cache-shorter-than-origin budget: give cache a tight deadline and leave enough time for origin fallback.
- Dedicated DB connection: reserve `sql.Conn` only when a sequence must run on one connection and is not better modeled as a transaction.
- Explicit pool cap: set max-open and max-idle assumptions from service concurrency, DB limits, and other app instances; validate with `DB.Stats` or equivalent telemetry.

## Selected And Rejected Examples
Selected example: a read path may assign a small Redis lookup budget, then fall through to a larger but still bounded PostgreSQL query budget. The spec states the total request deadline, what happens when Redis times out, and how fallback concurrency is limited.

Selected example: a write path uses `BeginTx(ctx, ...)` with the use-case deadline. If the context is canceled before commit, the transaction is rolled back by the SQL package and the outward response must follow the commit-outcome policy.

Selected example: a background audit write that must survive client disconnect uses a bounded background context, not the request context, and records how the caller can observe completion or retry safely.

Rejected example: DB and cache calls without deadlines because the handler already has a timeout elsewhere. The spec should require explicit propagation to DB/cache calls.

Rejected example: setting max open connections to a large number without accounting for service replicas, transaction duration, and database capacity.

Rejected example: cache retry loops that consume the whole request budget and leave no time for origin fallback.

## Staleness And Failure Semantics
- Cache timeout normally behaves like a miss for read acceleration paths.
- DB timeout is an origin failure and must map to the API or workflow error contract; it should not be hidden as a stale cache hit unless stale serving is explicitly allowed.
- Transaction context cancellation rolls back the transaction; commit errors require the spec to define unknown-outcome handling.
- Pool waits count against the caller's deadline. If pool wait is material, the acceptance checks should include pool wait or saturation telemetry.

## Acceptance Checks
- The spec states parent context, DB deadline, cache deadline, and total request deadline relationship.
- Cache deadline is shorter than origin deadline unless an explicit exception is justified.
- Fallback concurrency is bounded so cache outage does not overload the origin.
- Connection pool assumptions include max open connections, expected in-flight work, replicas, and database limit headroom.
- Dedicated connections are released and justified by a single-connection requirement.
- Resource-return obligations are named for implementations: close rows, check row iteration errors, and end every transaction with commit or rollback.

## Exa Source Links
- [Go: Canceling in-progress operations](https://go.dev/doc/database/cancel-operations)
- [Go: Managing connections](https://go.dev/doc/database/manage-connections)
- [Go: Executing transactions](https://go.dev/doc/database/execute-transactions)
- [Go `database/sql` package](https://pkg.go.dev/database/sql)

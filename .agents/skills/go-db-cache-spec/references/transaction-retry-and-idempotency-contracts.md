# Transaction Retry And Idempotency Contracts

## When To Load
Load this when the spec must define write transaction ownership, isolation, retry rules, idempotency, `ON CONFLICT`, or the way cache invalidation/update publication is linked to a write. Use it before coding write paths that will be retried or that must keep DB state and cache freshness aligned.

Stay in the runtime contract. If the work requires designing the primary schema, ledger/history model, or distributed saga, record the handoff instead of owning it here.

## Viable Options
- No retry: use when the write is not idempotent, the error class is persistent, or the application cannot safely replay all decision logic.
- Whole-transaction retry: use for PostgreSQL serialization failures and carefully selected transient classes, with bounded attempts, jitter, and replay of all logic that chose SQL and values.
- Idempotent write contract: use request idempotency keys, `INSERT ... ON CONFLICT`, compare-and-set versions, or equivalent uniqueness guarantees before allowing retries of externally triggered writes.
- Same-transaction invalidation publication: write state and record the invalidation/update event in one DB transaction, then publish asynchronously from the durable record.
- Synchronous cache update after commit: acceptable only when best-effort loss is harmless because TTL/bypass/reconciliation semantics are explicit.

## Selected And Rejected Examples
Selected example: a permissions update commits the permission row and a cache-invalidation event in the same DB transaction. Admin permission reads bypass cache or require a fresh version, while public profile reads may use a 30-second eventual cache. If the publisher is delayed, the spec says public data may be stale within the window and admin data must not be.

Selected example: invoice creation can retry on PostgreSQL `40001` only by rerunning the whole use-case transaction with an idempotency key or `ON CONFLICT` outcome. The spec also states max attempts, backoff, and the result returned when retries exhaust.

Rejected example: retrying only the failed SQL statement inside a transaction after a serialization failure. The decision logic and all SQL choices must be replayed together.

Rejected example: retrying every `23505` unique violation. Some unique violations are permanent input conflicts; retry only when the spec explains why the violation is a transient serialization-equivalent race.

Rejected example: updating the database and then publishing a Redis invalidation directly as an untracked dual write when stale data would violate the contract. Use durable publication, TTL fallback, or downgrade the freshness guarantee.

## Staleness And Failure Semantics
- Data read inside an aborted transaction is not valid for user-visible decisions; retry or return a failure according to the use-case contract.
- If `Commit` returns an error and the application cannot prove the outcome, the spec must define the reconciliation or idempotent status-check behavior before retrying outward side effects.
- Cache invalidation failure after a successful DB commit must have an explicit fallback: bounded TTL, durable replay, origin bypass for strong paths, or a documented degraded mode.
- Write retries must not duplicate side effects, emit duplicate durable events without idempotent consumers, or extend transaction scope across network calls.

## Acceptance Checks
- Transaction owner and boundary are named at the use-case level.
- Isolation level is stated when default isolation is not enough; otherwise default reliance is explicit.
- Retryable classes are enumerated, including max attempts and backoff.
- Retried writes have an idempotency mechanism and duplicate-result semantics.
- Cache invalidation/update publication is linked to the write path or explicitly accepted as best-effort with TTL and validation boundaries.
- The spec forbids cross-service I/O inside an open DB transaction unless a separately approved architecture decision exists.

## Exa Source Links
- [Go: Executing transactions](https://go.dev/doc/database/execute-transactions)
- [Go `database/sql` package](https://pkg.go.dev/database/sql)
- [PostgreSQL: Serialization failure handling](https://www.postgresql.org/docs/current/mvcc-serialization-failure-handling.html)
- [PostgreSQL: Transaction isolation](https://www.postgresql.org/docs/current/transaction-iso.html)
- [PostgreSQL: `INSERT ... ON CONFLICT`](https://www.postgresql.org/docs/current/sql-insert.html)

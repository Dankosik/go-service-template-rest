# Transaction Retry And Idempotency Contracts

## Behavior Change Thesis
When loaded for write retries, ambiguous commit outcomes, or cache invalidation tied to writes, this file makes the model choose use-case-level transaction retry with idempotency and durable invalidation linkage or an explicit harmless-loss fallback instead of likely mistake `retry the failed statement or dual-write DB plus Redis best effort`.

## When To Load
Load this when the spec must define write transaction ownership, isolation, retry rules, idempotency, `ON CONFLICT`, commit-outcome handling, or the way cache invalidation/update publication is linked to a write.

Stay in the runtime contract. If the work requires designing the primary schema, ledger/history model, or distributed saga, record the handoff instead of owning it here.

## Decision Rubric
- Prefer no retry when the write is not idempotent, the error is persistent, or the application cannot safely replay all decision logic.
- For serialization failures and selected transient classes, retry the whole use-case transaction, not one SQL statement. State max attempts, backoff/jitter, and exhausted-retry result.
- Allow outward write retries only with an idempotency mechanism such as request keys, `INSERT ... ON CONFLICT`, compare-and-set versions, or equivalent uniqueness guarantees.
- Treat unique violations as retryable only when the spec explains why the conflict is a transient race; most `23505` errors are permanent input conflicts.
- Link cache invalidation or update publication to the write via the same DB transaction when stale data would violate the contract.
- Allow best-effort post-commit cache update only when TTL, bypass, reconciliation, or degraded freshness semantics make loss harmless.

## Imitate
- Permissions update: commit the permission row and an invalidation event in one transaction. Admin permission reads bypass cache or require a fresh version; public profile reads may tolerate a 30-second cache. Copy the habit of splitting strong and bounded-stale readers.
- Invoice creation retry: retry PostgreSQL `40001` by rerunning the whole use-case transaction with an idempotency key or `ON CONFLICT` result. Copy the habit of naming max attempts, backoff, and exhausted behavior.

## Reject
- Retrying only the failed SQL statement inside a transaction after serialization failure. Reject because the decisions that chose SQL and values must be replayed together.
- Retrying every unique violation. Reject because many uniqueness failures are durable business conflicts, not transient concurrency.
- Updating DB state and then directly publishing Redis invalidation as an untracked dual write when stale data would break the contract. Reject in favor of durable publication, TTL fallback, bypass, or weaker freshness.

## Agent Traps
- Do not hold a DB transaction open across cross-service I/O to "make invalidation atomic"; record the architecture/distributed-design handoff.
- Do not treat `Commit` errors as safe to blindly retry outward effects; unknown outcomes need idempotent status check or reconciliation.
- Do not emit duplicate durable events unless the consumer contract is idempotent.

## Validation Shape
- Data read inside an aborted transaction is not valid for user-visible decisions; retry or return a failure according to the use-case contract.
- If `Commit` returns an error and the application cannot prove the outcome, the spec must define the reconciliation or idempotent status-check behavior before retrying outward side effects.
- Cache invalidation failure after a successful DB commit must have an explicit fallback: bounded TTL, durable replay, origin bypass for strong paths, or a documented degraded mode.
- Write retries must not duplicate side effects, emit duplicate durable events without idempotent consumers, or extend transaction scope across network calls.
- Transaction owner and boundary are named at the use-case level.
- Isolation level is stated when default isolation is not enough; otherwise default reliance is explicit.
- Retryable classes are enumerated, including max attempts and backoff.
- Retried writes have an idempotency mechanism and duplicate-result semantics.
- Cache invalidation/update publication is linked to the write path or explicitly accepted as best-effort with TTL and validation boundaries.
- The spec forbids cross-service I/O inside an open DB transaction unless a separately approved architecture decision exists.

# Database-backed queue mechanics

Read this reference only for a PostgreSQL or other database-backed job queue. Verify the installed engine version and schema before relying on implementation details.

## Claim and lease contract

PostgreSQL row locks are transaction-scoped: `FOR UPDATE` prevents conflicting lockers or writers until the transaction ends, and a process/connection failure causes its transaction to end. `SKIP LOCKED` skips rows that cannot be locked immediately and deliberately returns an inconsistent view; PostgreSQL documents queue-like multi-consumer access as its suitable use.

Map those semantics to the common lease with a short claim transaction, not a row lock held around execution:

1. select eligible jobs with a deterministic order and `FOR UPDATE SKIP LOCKED`;
2. atomically write `leased/running`, owner, attempt ID, monotonically increasing generation, `leased_at`, and `lease_expires_at`;
3. commit before executing work;
4. condition renewal, checkpoint, completion, and failure on job ID, owner, generation, and current state;
5. let a sweeper expose expired attempts to the disposition defined by the [common crash matrix](../SKILL.md#define-the-lease-and-crash-matrix).

A durable `locked_at` column without periodic renewal is a stale-lock timeout, not proof that the worker remains alive. A PostgreSQL row lock has no wall-clock expiry. Keep these mechanisms distinct.

## Transactional enqueue

PostgreSQL can own the business row and job row in one transaction; Graphile Worker's SQL `add_job` is an example of a database-side enqueue API that can join the producer transaction. When the queue cannot join it, apply the [durable-acceptance contract](../SKILL.md#protect-durable-acceptance).

Graphile Worker's `job_key` can replace or preserve an unlocked job, while a locked job can cause a second job to be scheduled. Completed jobs are deleted, so `job_key` is producer deduplication rather than a permanent business-effect ledger.

## Retry, scheduling, and recovery

Graphile Worker is an execution-model example, not a portable contract:

- it documents at-least-once execution, exponential-backoff retries, and a configurable attempt limit;
- an unhandled instantaneous worker exit leaves jobs locked for at least four hours by default, after which workers sweep stale locks; graceful shutdown stops new jobs and waits, while forceful shutdown fails running jobs for retry;
- recurring schedules use UTC, can optionally backfill a bounded period, and enqueue ordinary retryable jobs;
- its completion batching can leave a completed effect behind a still-locked job, allowing re-execution after catastrophic failure.

Treat those as Graphile defaults, not portable guarantees. A custom queue still needs explicit lease and retry bounds. Civil-time recurrence needs an IANA zone and occurrence identity outside an engine that schedules only UTC.

## Database-specific hazards

- A stable `ORDER BY` plus a unique tiebreaker makes selection explainable; `LIMIT` without it produces an unpredictable subset.
- `SKIP LOCKED` avoids waiting but can starve old rows. Apply the [fairness proof](operations.md#capacity-fairness-and-backpressure) when selection policy is in scope.
- Keep the claim transaction short. Network or business work inside it holds row locks and snapshots, increases contention, and makes crash recovery depend on connection cleanup.
- Lease acquisition, renewal, and sweeps need indexes that match eligibility and expiry predicates. Hand concrete DDL and migration safety to `postgres-schema-design`, and plan/load/index proof to `postgres-performance`.
- Use database/server time consistently for lease decisions. Reject a stale completion with a conditional row-count check.

## Executable proof

Use a real PostgreSQL instance and at least two worker processes for the [core proof](../SKILL.md#prove-the-contract). Include stale-generation rejection, recovery after lease expiry, producer-transaction rollback, and reconciliation of an accepted business row with no visible runnable job. Add selection/fairness tests only when that branch is in scope.

## Primary sources

- [PostgreSQL row-level locking](https://www.postgresql.org/docs/current/explicit-locking.html#LOCKING-ROWS)
- [PostgreSQL `SELECT`, `NOWAIT`, and `SKIP LOCKED`](https://www.postgresql.org/docs/current/sql-select.html#SQL-FOR-UPDATE-SHARE)
- [Graphile Worker reliability and execution model](https://worker.graphile.org/docs)
- [Graphile Worker error and crash recovery](https://worker.graphile.org/docs/error-handling)
- [Graphile Worker recurring schedules](https://worker.graphile.org/docs/cron)
- [Graphile Worker job-key behavior](https://worker.graphile.org/docs/job-key)

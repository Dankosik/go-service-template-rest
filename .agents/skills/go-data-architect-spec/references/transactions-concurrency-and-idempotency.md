# Transactions, Concurrency, And Idempotency

## When To Load
Load this when the task involves transaction boundaries, isolation level, optimistic concurrency, row locks, advisory locks, work claiming, duplicate callbacks, retries, idempotency keys, holds, leases, scarce capacity, or outbox/inbox semantics.

Use it to choose the smallest concurrency mechanism that preserves the invariant. Keep this at the data-spec level: name the transaction scope, constraint, retry class, idempotency record, and recovery consequence without drifting into low-level query tuning.

## Decision Examples

### Example 1: Scarce inventory reservation
Context: Buyers can reserve a scarce item, reservations expire, and duplicate client retries must not consume capacity twice.

Selected option: Keep reservation creation and capacity update in one local transaction. Use a row lock, constraint, or ledger-style allocation table that matches the invariant. Store a tenant-scoped idempotency key with request fingerprint and resulting reservation ID.

Rejected options:
- Check available capacity, commit, then insert reservation in a later transaction.
- Use cache counters as the source of truth for capacity.
- Rely on a blanket higher isolation level without retry and contention evidence.

Migration and rollback consequences:
- Add idempotency storage before enabling automatic client retries.
- If a new constraint prevents overbooking, clean existing conflicts before validation.
- Rollback after confirmed reservations are issued usually requires forward release or reconciliation, not deleting evidence rows.

### Example 2: Queue-like work claiming
Context: Multiple workers claim pending backfill chunks or outbox rows.

Selected option: Use claim rows or lease semantics with `FOR UPDATE SKIP LOCKED` only for queue-like work where an inconsistent snapshot is acceptable. Record lease owner, lease deadline, retry count, and stuck-work recovery.

Rejected options:
- Use `SKIP LOCKED` for general user-facing list queries.
- Hold a transaction open while doing network calls or long CPU work.
- Use advisory locks without a stable scope, timeout, and deadlock story.

Migration and rollback consequences:
- Introduce lease columns additively and allow old workers to drain before requiring them.
- Rollback must release or ignore leases created by the new worker version.
- If chunk state was advanced incorrectly, recovery should be idempotent and restart from checkpoints.

### Example 3: Provider callback deduplication
Context: A payment provider can deliver the same callback multiple times and in a different order than local processing time.

Selected option: Store provider event identity and idempotency/dedup state in SQL with tenant/provider scope. Preserve event time and processed time separately. Apply state transitions in one transaction with checks that reject stale or duplicate state movement.

Rejected options:
- Deduplicate only in memory or cache.
- Treat arrival time as provider event time.
- Let provider status strings directly become canonical local lifecycle states.

Migration and rollback consequences:
- Backfill dedup keys from raw payload history if available; otherwise mark the gap as an accepted risk.
- New dedup constraints can change callback behavior, so roll out behind monitoring and replay tests.
- Once duplicate callbacks are suppressed by persisted state, rollback must not reprocess old duplicates without a replay plan.

## Source Links Gathered Through Exa
- PostgreSQL, "Transaction Isolation": https://www.postgresql.org/docs/current/transaction-iso.html
- PostgreSQL, "Explicit Locking": https://www.postgresql.org/docs/current/explicit-locking.html
- PostgreSQL, "SELECT": https://www.postgresql.org/docs/current/sql-select.html
- Go, "Executing transactions": https://go.dev/doc/database/execute-transactions
- Go, "Accessing relational databases": https://go.dev/doc/database
- Stripe, "Idempotent requests": https://docs.stripe.com/api/idempotent_requests


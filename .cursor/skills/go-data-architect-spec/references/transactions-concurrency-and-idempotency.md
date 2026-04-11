# Transactions, Concurrency, And Idempotency

## Behavior Change Thesis
When loaded for a task where scarce capacity, worker claiming, retries, duplicate callbacks, or competing writers might be solved with vague "transactions" or cache state, this file makes the model choose the smallest invariant-preserving transaction, constraint, lock, lease, and idempotency record instead of likely mistake "raise isolation, add retries, or trust Redis."

## When To Load
Load this for transaction boundaries, isolation, locks, optimistic concurrency, work claiming, retries, idempotency keys, holds, leases, callbacks, or duplicate delivery.

## Decision Rubric
- State the invariant first, then choose the concurrency mechanism. Mechanism follows invariant class, not taste.
- Keep invariant-preserving writes in one local transaction where possible. Reject cross-service global ACID assumptions.
- Use uniqueness or partial uniqueness for duplicate prevention; version checks for lost updates; exclusion constraints or explicit lease tables for overlap and scarce allocation; `FOR UPDATE SKIP LOCKED` for queue-like claiming only.
- Store idempotency records durably with scope, request fingerprint, status/result, expiry or retention, and conflict behavior.
- Use stronger isolation only for a named anomaly, with retry semantics and tests for serialization/deadlock behavior.
- Use advisory locks only with stable lock scope, acquisition order, timeout behavior, and deadlock recovery.

## Imitate

### Scarce inventory reservation
Context: Buyers can reserve a scarce item, reservations expire, and duplicate retries must not consume capacity twice.

Keep reservation creation and capacity update in one local transaction. Use a row lock, constraint, exclusion rule, or ledger-style allocation table that matches the invariant. Store a tenant-scoped idempotency key with request fingerprint and resulting reservation ID.

Copy this because it makes the idempotent operation and capacity invariant durable together.

### Queue-like work claiming
Context: Multiple workers claim pending backfill chunks or outbox rows.

Use claim rows or lease semantics with `FOR UPDATE SKIP LOCKED` only where an inconsistent snapshot is acceptable. Record lease owner, lease deadline, retry count, and stuck-work recovery.

Copy this because queue claiming is a special workload; it is not a generic list-query pattern.

### Provider callback deduplication
Context: A payment provider can deliver the same callback multiple times and out of event-time order.

Store provider event identity and dedupe state in SQL with tenant/provider scope. Preserve event time and processed time separately. Apply local state transitions in one transaction with checks that reject stale or duplicate movement.

Copy this because duplicate delivery and late delivery are separate failure modes.

## Reject
- "Check available capacity, commit, then insert reservation later." The gap permits overbooking.
- "Redis counters are the source of truth for seats or money." Cache loss or split brain becomes a correctness bug.
- "Use serializable everywhere." Stronger isolation without anomaly, retry, and contention evidence is a blunt instrument.
- "Use `SKIP LOCKED` for user-facing pages." Skipped rows create misleading lists outside queue-like work.
- "Deduplicate callbacks in memory." Restarts and parallel instances reprocess duplicates.

## Agent Traps
- Do not put network calls or long CPU work inside the invariant transaction.
- Do not reuse correlation IDs as idempotency keys without a semantic operation fingerprint.
- Do not describe retries without classifying deadlock, serialization failure, timeout, duplicate request, and provider replay separately.
- Do not propose advisory locks without a stable key and release story.

## Validation Shape
- Concurrency proof names the invariant, transaction scope, lock or constraint, expected conflict error, retry class, and deadlock or serialization test.
- Idempotency proof covers duplicate request same-fingerprint, duplicate request different-fingerprint, replay after success, replay after failure, and retention/expiry behavior.
- Worker proof covers lease expiry, stuck-work recovery, idempotent chunk processing, and rollback behavior for claims created by the new version.

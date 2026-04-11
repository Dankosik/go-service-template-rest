# Retry, Duplicate, And Reorder Domain Risks

## Behavior Change Thesis
When loaded for symptom "retry, replay, duplicate, stale, or out-of-order input can repeat or reorder business effects", this file makes the model tie idempotency and ordering to a concrete domain consequence instead of likely mistake "say add dedupe or global ordering generically."

## When To Load
Load this when a review touches message consumers, retry loops, idempotency keys, deduplication, replay, out-of-order events, version checks, stale updates, backfills, command reprocessing, refund/capture/reservation duplication, or optimistic concurrency around business state.

## Decision Rubric
- Review the business entity key, current-state guard, processed-message key, event version, and side-effect idempotency together.
- Report a finding when replay or stale input can repeat an irreversible effect, skip a legal transition check, overwrite newer business state, or make stale facts look current.
- Prefer an existing local processed-event key, idempotency key, current-state guard, or version check before proposing new storage.
- Escalate when safe dedupe/order handling requires new storage, event partitioning, transaction design, retry policy, public idempotency semantics, or reconciliation behavior.

## Imitate
```text
[high] [go-domain-invariant-review] internal/orders/consumer.go:23
Issue:
Approved behavior allows `paid -> cancelled` once and says duplicate cancellation events must not trigger a second refund, but `HandleOrderCancelled` saves `StatusCancelled` and calls `refunds.Create` without checking the current state or processed event ID.
Impact:
A replayed or redelivered cancellation can issue another refund for the same payment, creating duplicate money movement while the order still appears simply `cancelled`.
Suggested fix:
Guard the transition so only the first valid `paid -> cancelled` path can create the refund, using the local processed-event key, refund idempotency key, or current-state check already established in this service.
Reference:
Local cancellation event contract or duplicate-cancellation test.
```

Copy the shape: duplicate/stale path, concrete repeated or overwritten business effect, existing local guard if present.

## Reject
```text
[medium] internal/orders/consumer.go:23
This consumer is not idempotent. Add dedupe.
```

Failure: this does not tie duplicate handling to a domain transition or repeated business effect.

## Agent Traps
- Do not require global ordering when the local rule only needs per-aggregate ordering or a stale-version guard.
- Do not flag every retry loop; flag retries that can change business outcome or repeat an effect.
- Do not require a new inbox table when a local idempotency key or transition guard already provides the approved guarantee.
- Do not assume "last write wins" is acceptable for business state unless a local contract says stale overwrites are allowed.

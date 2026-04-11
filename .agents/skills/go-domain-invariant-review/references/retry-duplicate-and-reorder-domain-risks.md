# Retry, Duplicate, And Reorder Domain Risk Examples

## When To Load
Load this when a review touches message consumers, retry loops, idempotency keys, deduplication, replay, out-of-order events, version checks, stale updates, backfills, command reprocessing, refund/capture/reservation duplication, or optimistic concurrency around business state.

Use local event contracts, specs, tests, and task artifacts as authority for what must be idempotent, monotonic, ordered, or rejected. External sources calibrate common failure modes: at-least-once delivery, outbox replay, duplicate messages, and per-aggregate ordering.

## Review Lens
Retries and duplicates are domain risks when they repeat irreversible effects, skip legal transition checks, overwrite newer business state, or make stale facts look current. Review the business entity key, current-state guard, processed-message key, event version, and side-effect idempotency together.

## Bad Finding Example
```text
[medium] internal/orders/consumer.go:23
This consumer is not idempotent. Add dedupe.
```

Why it fails: it does not tie duplicate handling to a domain transition or specific repeated business effect.

## Good Finding Example
```text
[high] [go-domain-invariant-review] internal/orders/consumer.go:23
Issue:
Approved behavior allows `paid -> cancelled` once and says duplicate cancellation events must not trigger a second refund, but `HandleOrderCancelled` saves `StatusCancelled` and calls `refunds.Create` without checking the current state or processed event ID.
Impact:
A replayed or redelivered cancellation can issue another refund for the same payment, creating duplicate money movement while the order still appears simply `cancelled`.
Suggested fix:
Guard the transition so only the first valid `paid -> cancelled` path can create the refund, using the local processed-event key, refund idempotency key, or current-state check already established in this service.
Reference:
Local cancellation event contract or tests; idempotent-consumer guidance is calibration only.
```

## Non-Findings To Avoid
- Do not require global ordering when the local rule only needs per-aggregate ordering or a stale-version guard.
- Do not flag every retry loop; flag retries that can change business outcome or repeat an effect.
- Do not require a new inbox table when a local idempotency key or transition guard already provides the approved guarantee.
- Do not assume "last write wins" is acceptable for business state unless a local contract says stale overwrites are allowed.

## Smallest Safe Correction
Prefer a targeted guard:
- check current state before applying a transition or side effect;
- record and check the processed message or command key inside the existing transaction when available;
- pass a stable idempotency key to external side-effect APIs;
- reject or ignore stale event versions according to local contract;
- keep duplicate handling close to the consumer path that performs the irreversible action.

## Escalation Cases
Escalate when:
- the local contract does not define duplicate, replay, or stale-event behavior;
- safe dedupe requires new storage, transaction, or outbox/inbox design;
- event partitioning or ordering keys conflict with the aggregate boundary;
- the fix changes retry policy, public idempotency semantics, or reconciliation behavior;
- several producers can authoritatively update the same business entity without a single sequencing rule.

## Source Links From Exa
- [Microservices.io: Idempotent Consumer](https://microservices.io/patterns/communication-style/idempotent-consumer.html)
- [Microservices.io: Transactional outbox](https://microservices.io/patterns/data/transactional-outbox.html)
- [NILUS: Event Ordering Tradeoffs in Event Streaming](https://www.nilus.be/blog/event_ordering_tradeoffs_in_event_streaming/)
- [Milan Jovanovic: Solving Message Ordering from First Principles](https://www.milanjovanovic.tech/blog/solving-message-ordering-from-first-principles)

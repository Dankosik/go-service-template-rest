# Outbox, Inbox, And Idempotency

## Behavior Change Thesis
When loaded for dual-write, ACK-order, dedup, or idempotency-key symptoms, this file makes the model require durable outbox/inbox and business idempotency boundaries instead of trusting broker "exactly once," direct publish-after-commit, or in-memory duplicate tracking.

## When To Load
Load when a spec must avoid dual writes, define outbox relay behavior, choose consumer dedup or inbox semantics, set ACK or offset-commit order, or define idempotency keys and retention.

## Decision Rubric
- Reject dual writes as the correctness mechanism; database commit and broker publish can diverge.
- Use a transactional outbox when producer state and message intent must be atomically linked, then publish asynchronously.
- Use Dapr outbox only when Dapr state transactions and pub/sub are already part of the service; still model transaction markers, verification, and at-least-once publish.
- Use inline idempotent consumers when message handling and the processed-message marker can commit in the same local transaction.
- Use an inbox when receiving, batching, retrying, or independently scaling message processing needs durable separation from broker ACK.
- Accept natural idempotency only when repeat execution is domain-equivalent and protected by durable uniqueness or version checks.

## Imitate
- Producer transaction writes `order.status=PENDING` and `outbox(OrderCreated, message_id, aggregate_id, version)`. Relay publishes later and may publish twice. Copy the atomic state-plus-intent boundary.
- Consumer transaction inserts `(subscriber_id, message_id)` into a dedup table, applies the business side effect, writes outgoing outbox messages, commits, then ACKs. Copy the ACK-after-durable-effect order.
- Inbox receiver stores `message_id`, `type`, `payload`, and `received_at` with conflict-ignore semantics; a processor claims rows and records terminal outcomes. Copy the durable receive/process split.
- For a refund retry bug where envelope IDs change, dedup by stable refund operation key plus step name, not only the Kafka envelope ID. Copy the business-idempotency distinction.

## Reject
- Publish event to broker, then commit database state.
- Commit state, then publish directly to broker without a durable outbox or equivalent.
- ACK the broker before durable side effects and dedup markers are committed.
- Keep processed message IDs only in process memory.
- Treat an outbox as eliminating the need for consumer idempotency.

## Agent Traps
- Letting "exactly once" marketing language erase application-level side-effect idempotency.
- Choosing a message envelope ID as the only dedup key when retries can create new envelopes for the same business operation.
- Forgetting dedup retention and replay/redrive windows.
- Putting the outbox on the producer but leaving side-effecting consumers non-idempotent.

## Validation Shape
- Relay publishes then crashes before marking delivered: consumers see a duplicate and must dedup or handle idempotently.
- Consumer commits side effect but crashes before ACK: broker redelivers; dedup marker prevents repeat side effects.
- Dedup retention expires too early: redrive or replay can reapply an old side effect; retention must cover plausible replay windows.
- Inbox table grows forever: the spec defines retention, partitioning, or cleanup after the redrive risk window.
- Framework claims "exactly once": interpret it as a scoped infrastructure behavior and still document broker redelivery, handler idempotency, and storage boundaries.

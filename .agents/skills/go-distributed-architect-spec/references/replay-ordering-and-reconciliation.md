# Replay, Ordering, And Reconciliation

## Behavior Change Thesis
When loaded for replay, ordering, projection drift, or distributed-lock symptoms, this file makes the model choose per-key ordering/version checks and owner-driven reconciliation instead of assuming global broker order, direct projection repair, or lock-only correctness.

## When To Load
Load when replay, redrive, broker ordering, partitioning, stale projections, duplicate delivery, distributed locks, or repair jobs affect correctness.

## Decision Rubric
- Prefer no ordering dependency: consumers should tolerate out-of-order delivery with version checks, idempotency, and owner queries.
- Use per-aggregate ordering only when needed, via broker partition key, FIFO message group, stream partition, or one-active-consumer rule tied to the business key.
- Avoid global ordering unless a single serialized lane is acceptable for throughput and availability.
- Use replay from log or stream only with deterministic handlers, versioned event interpretation, and dedup windows.
- Use queue redrive with dedup, poison-message policy, and backpressure guardrails.
- Use reconciliation when eventual consistency can drift; repairs must be owner-driven, resumable, and auditable.
- Treat distributed locks as a technical optimization, not the correctness boundary, unless the spec includes fencing-token and owner-state analysis.

## Imitate
- Events for one order use `order_id` as the partition or FIFO group key; consumers also check aggregate version before applying. Copy the broker-plus-domain guard.
- A consumer stores last-seen version per aggregate, ignores duplicates or older events, and routes gaps to reconciliation. Copy the version-aware gap policy.
- Replay job reads from a bounded offset or watermark, applies dedup keys, limits throughput, and emits repair commands to owners. Copy the replay safety envelope.
- A write path using a projection checks projection lag; if lag exceeds budget, it queries the owner or fails/accepts by contract. Copy the stale-projection fallback.

## Reject
- Assume a Kafka topic, RabbitMQ queue, or SQS standard queue gives global business ordering.
- Process redriven messages with new code without specifying old event version handling.
- Requeue poison messages indefinitely and create a retry storm.
- Run a reconciliation job that writes directly into another service's database.
- Use distributed locks as the only protection against concurrent flows on the same aggregate.

## Agent Traps
- Confusing partition order with global business order.
- Fixing duplicate or concurrent flow bugs by adding a lock but no owner-side uniqueness, version check, or fencing token.
- Treating projection rebuild as source-of-truth repair.
- Redriving a DLQ without backpressure, poison-message classification, or version compatibility.

## Validation Shape
- RabbitMQ redelivery can alter observed order when multiple consumers or requeueing are involved; use streams or single active consumer if order is required.
- SQS standard queues can deliver duplicates and best-effort order only; FIFO queues preserve order within a message group but still require application-level side-effect idempotency.
- Kafka gives ordered offsets within a partition, not across partitions; key selection becomes part of the correctness contract.
- Replay after dedup retention expiry can reapply old events; keep retention aligned with replay windows or use aggregate version checks.
- Reconciliation finds owner state and projection state disagree: emit a repair command or rebuild the projection from authoritative history.

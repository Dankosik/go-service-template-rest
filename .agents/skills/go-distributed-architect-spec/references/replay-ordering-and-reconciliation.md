# Replay, Ordering, And Reconciliation

## When To Load
Load this when replay, redrive, broker ordering, partitioning, stale projections, duplicate delivery, or repair jobs affect correctness.

## Option Comparisons
- No ordering dependency: prefer when possible. Design consumers to accept out-of-order delivery through version checks, idempotency, and owner queries.
- Per-aggregate ordering: use a broker partition key, FIFO message group, stream partition, or one-active-consumer rule tied to the business key.
- Global ordering: avoid unless a single serialized lane is truly acceptable for throughput and availability.
- Replay from log or stream: choose when historical reprocessing is needed; require deterministic handlers and versioned event interpretation.
- Queue redrive: choose for failed work items; require dedup, poison-message policy, and backpressure guardrails.
- Reconciliation job: use when eventual consistency can drift and repair must be owner-driven, resumable, and auditable.

## Good Flow Examples
- Events for one order use `order_id` as the partition or FIFO group key; consumers also check aggregate version before applying.
- A consumer stores last-seen version per aggregate and ignores duplicates or older events while routing gaps to reconciliation.
- Replay job reads from a bounded offset or watermark, applies dedup keys, limits throughput, and emits repair commands to owners.
- A write path using a projection checks projection lag; if lag exceeds budget, it queries the owner or fails/accepts by contract.

## Bad Flow Examples
- Assume a Kafka topic, RabbitMQ queue, or SQS standard queue gives global business ordering.
- Process redriven messages with new code without specifying old event version handling.
- Requeue poison messages indefinitely and create a retry storm.
- Run a reconciliation job that writes directly into another service's database.
- Use distributed locks as the only protection against concurrent flows on the same aggregate.

## Failure-Mode Examples
- RabbitMQ redelivery can alter observed order when multiple consumers or requeueing are involved; use streams or single active consumer if order is required.
- SQS standard queues can deliver duplicates and best-effort order only; FIFO queues preserve order within a message group but still require application-level side-effect idempotency.
- Kafka gives ordered offsets within a partition, not across partitions; key selection becomes part of the correctness contract.
- Replay after dedup retention expiry can reapply old events; keep retention aligned with replay windows or use aggregate version checks.
- Reconciliation finds owner state and projection state disagree: emit a repair command or rebuild the projection from authoritative history.

## Exa Source Links
- [RabbitMQ consumer acknowledgements and publisher confirms](https://www.rabbitmq.com/docs/confirms)
- [RabbitMQ queues and message ordering](https://www.rabbitmq.com/docs/queues)
- [RabbitMQ streams](https://www.rabbitmq.com/docs/streams)
- [Amazon SQS queue types](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-queue-types.html)
- [Kafka design docs](https://kafka.apache.org/25/design/design/)
- [Confluent Kafka delivery semantics docs](https://docs.confluent.io/kafka/design/delivery-semantics.html)
- [Dapr pub/sub docs](https://docs.dapr.io/developing-applications/building-blocks/pubsub/_print/)

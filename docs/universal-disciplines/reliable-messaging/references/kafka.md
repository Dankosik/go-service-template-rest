# Apache Kafka delivery decisions

Use this reference only for Kafka-specific choices. Verify defaults and feature availability against the deployed broker and client versions.

## Publish acceptance and ambiguity

- Treat a successful produce response as acceptance by the selected acknowledgement policy, not as completion of any consumer effect.
- For durable application messages, combine `acks=all` with an intentional replication factor and `min.insync.replicas`. `acks=1` can acknowledge before followers replicate; `acks=0` supplies no broker result.
- Enable producer idempotence with compatible retry and in-flight settings. It suppresses duplicates created by the producer protocol's own retries and preserves per-partition order under its documented constraints. It does not deduplicate a logical operation re-created by an application after an arbitrary delay or identity loss.
- A timeout or broken connection can occur after the record was appended but before the response arrived. Retry the same outbox/message identity and make consumers idempotent.
- A `transactional.id` adds producer fencing and transactions across Kafka partitions and sessions. It does not join a PostgreSQL or other external database transaction.

## Consume, acknowledge, and effect boundary

- A consumer's current position advances during polling; its committed offset is the recovery position. Disable automatic offset commits when processing owns a durable external effect.
- Commit only the next offset after every preceding effect in that partition has durably completed. Parallel processing needs a per-partition completion frontier so a fast later record cannot commit past slow earlier work.
- A crash after an external effect commits but before the offset commits causes redelivery. Protect the effect with an inbox/business key or atomically store the consumed offset with the external effect and recover from that store.
- Kafka transactions can atomically write output records and consumed offsets to Kafka. Downstream consumers use `isolation.level=read_committed`. This guarantee begins at committed Kafka input and ends at committed Kafka output; an external business effect remains outside it.
- During rebalance, stop new work for revoked partitions and commit only completed frontiers while ownership is still valid. A failed or timed-out commit is an ambiguous recovery event, not permission to skip records.

## Ordering, retention, and recovery

- Ordering is per topic-partition. Use the stable business ordering key so related records land in one partition; there is no total order across partitions.
- A consumer group assigns each partition to one group member at a time. Partitions set the useful parallelism ceiling for ordered consumption; additional worker threads need their own ordered completion accounting.
- Topic retention is independent of consumption. A slow or offline consumer can lose replayability when its needed offsets fall before the retained start.
- Log compaction retains the latest record per key eventually; it is not a complete immutable history. Use deletion retention when every historical record must remain replayable.
- Replay with a new group or a reviewed offset reset after confirming the requested range is still retained. Preserve original message identity and rate-limit replay against live traffic.

## Signals and security that change the design

- Observe outbox age, produce error/timeout rate, acknowledgement latency, under-replicated or unavailable partitions, consumer lag by partition, oldest retained offset/time, rebalance rate, poll deadline violations, and inbox conflicts.
- Use distinct producer, consumer-group, transaction, and operator principals. Restrict topic, group, and transactional-ID permissions; use TLS and an authenticated SASL mechanism appropriate to the deployment.

## Primary sources

- [Apache Kafka producer configuration](https://kafka.apache.org/42/configuration/producer-configs/)
- [Apache Kafka consumer API and offset semantics](https://kafka.apache.org/42/javadoc/org/apache/kafka/clients/consumer/KafkaConsumer.html)
- [Apache Kafka transactional consumption](https://kafka.apache.org/42/javadoc/org/apache/kafka/clients/consumer/KafkaConsumer.html#reading-transactional-messages)
- [Apache Kafka topic retention configuration](https://kafka.apache.org/42/configuration/topic-configs/)
- [Apache Kafka concepts and partition ordering](https://kafka.apache.org/documentation/)

Sources checked 2026-08-02. Recheck the target version before relying on defaults.

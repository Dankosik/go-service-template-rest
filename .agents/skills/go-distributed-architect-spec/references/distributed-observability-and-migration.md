# Distributed Observability And Migration

## When To Load
Load this when a distributed-flow spec changes event contracts, compatibility windows, rollout sequence, DLQ handling, replay tooling, saga dashboards, or operator-facing recovery.

## Option Comparisons
- Correlation-only logs: useful but insufficient for distributed recovery. Add metrics and trace links for saga state, message IDs, retries, and reconciliation.
- Saga dashboard or operation resource: choose when humans or clients need outcome visibility for long-running flows.
- Additive event evolution: prefer when consumers may run mixed versions; keep old fields valid until consumers migrate.
- Breaking event change: require versioned topics/types or a dual-publish/dual-read window with replay and rollback rules.
- Big-bang migration: avoid unless all producers, consumers, brokers, and stored messages can be upgraded atomically, which is rare.

## Good Flow Examples
- Every command/event includes `correlation_id`, `causation_id`, `message_id`, `producer`, `schema_version`, and the business key used for dedup or ordering.
- Metrics distinguish produced, relayed, consumed, deduped, retried, DLQ, compensated, reconciled, and manually repaired outcomes.
- Migration adds `PaymentAuthorized.v2` while consumers continue accepting v1. Replay tooling knows how to handle both.
- A `202 Accepted` API returns an operation ID; the saga updates operation state from durable flow transitions.

## Bad Flow Examples
- Logs include only transport offsets and not business keys or saga IDs.
- DLQ messages are treated as broker operations with no domain owner or repair path.
- Event schema is narrowed while old consumers or old stored messages still exist.
- Reconciliation publishes repair events without correlation to the original failed command or message.
- Observability labels include unbounded message IDs as metric dimensions.

## Failure-Mode Examples
- Consumer fails after deploy because it cannot parse old messages: rollback or enable compatibility reader, then replay from a safe watermark.
- Outbox relay backlog grows: alert on age and count, not only process liveness; expose oldest message age by low-cardinality topic or producer.
- DLQ spike after a contract change: freeze redrive, identify schema/version split, deploy compatible consumer, then redrive with throughput limits.
- Reconciliation fixes a projection but not the source owner: mark as invalid repair and route through owner command/event.
- Mixed-version saga instances exist: keep old transition handling until all durable instances have reached terminal states or have been migrated.

## Exa Source Links
- [Dapr pub/sub docs](https://docs.dapr.io/developing-applications/building-blocks/pubsub/_print/)
- [Dapr Workflow features and concepts](https://docs.dapr.io/developing-applications/building-blocks/workflow/workflow-features-concepts)
- [MassTransit transactional outbox configuration](https://masstransit.io/advanced/transactional-outbox.html)
- [NServiceBus Outbox docs](https://docs.particular.net/nservicebus/outbox/?version=core_10)
- [RabbitMQ consumer acknowledgements and publisher confirms](https://www.rabbitmq.com/docs/confirms)

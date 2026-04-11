# Distributed Observability And Migration

## Behavior Change Thesis
When loaded for event-contract rollout, mixed-version, DLQ, or operator-visibility symptoms, this file makes the model require compatibility windows and recovery-grade telemetry instead of big-bang event changes or offset-only logs that cannot drive repair.

## When To Load
Load when a distributed-flow spec changes event contracts, compatibility windows, rollout sequence, DLQ handling, replay tooling, saga dashboards, or operator-facing recovery.

## Decision Rubric
- Treat correlation-only logs as insufficient for recovery; add metrics and traces for saga state, message IDs, retries, DLQ, and reconciliation.
- Use a saga dashboard or operation resource when humans or clients need outcome visibility for long-running flows.
- Prefer additive event evolution while consumers may run mixed versions; keep old fields valid until stored messages and consumers migrate.
- For breaking event changes, require versioned topics/types or a dual-publish/dual-read window with replay and rollback rules.
- Avoid big-bang migrations unless all producers, consumers, brokers, and stored messages can be upgraded atomically.
- Keep telemetry labels bounded; IDs belong in logs/traces, not metric dimensions.

## Imitate
- Recovery-relevant commands or events include `correlation_id`, `causation_id`, `message_id`, `producer`, `schema_version`, and the business key used for dedup or ordering. Copy the recovery identifiers.
- Metrics distinguish produced, relayed, consumed, deduped, retried, DLQ, compensated, reconciled, and manually repaired outcomes. Copy the outcome taxonomy, not unbounded labels.
- Migration adds `PaymentAuthorized.v2` while consumers continue accepting v1; replay tooling handles both. Copy the mixed-version window.
- A `202 Accepted` API returns an operation ID; the saga updates operation state from durable flow transitions. Copy the client-visible recovery surface.

## Reject
- Logs include only transport offsets and not business keys or saga IDs.
- DLQ messages are treated as broker operations with no domain owner or repair path.
- Event schema is narrowed while old consumers or old stored messages still exist.
- Reconciliation publishes repair events without correlation to the original failed command or message.
- Observability labels include unbounded message IDs as metric dimensions.

## Agent Traps
- Treating DLQ as an infrastructure queue instead of a domain-owned recovery queue.
- Forgetting old stored messages when proposing a consumer-breaking schema change.
- Adding traces but no alertable backlog, oldest-age, or stuck-state metric.
- Making operation status client-visible without tying it to durable saga transitions.

## Validation Shape
- Consumer fails after deploy because it cannot parse old messages: rollback or enable compatibility reader, then replay from a safe watermark.
- Outbox relay backlog grows: alert on age and count, not only process liveness; expose oldest message age by low-cardinality topic or producer.
- DLQ spike after a contract change: freeze redrive, identify schema/version split, deploy compatible consumer, then redrive with throughput limits.
- Reconciliation fixes a projection but not the source owner: mark as invalid repair and route through owner command/event.
- Mixed-version saga instances exist: keep old transition handling until all durable instances have reached terminal states or have been migrated.

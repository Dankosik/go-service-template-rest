---
name: go-distributed
description: "Recovery: Use for cross-service consistency, sagas, replay, ordering, compensation, redrive, or reconciliation. Own recovery; Skip local concurrency."
---

# Go Distributed

A cross-service flow is defined by its **recovery**.

`flow contract -> durable steps -> partial failure -> ordering and duplicates -> compensation -> redrive -> reconciliation -> proof`

Load the [shared specialist contract](../specialist-contract.md). Reconstruct
every durable step from accepted flows, producers, consumers, persistence or
broker boundaries, recovery paths, and success semantics. Replay duplicate
delivery, reordering, partial completion, process loss, redrive, and mixed
versions. Each step is idempotent under redelivery or has a named compensation;
ordering exists only where a key serializes it; reconciliation distinguishes
applied, unapplied, and unknown.

Read the mechanism present in this checkout before changing policy:
[PostgreSQL Transactional Outbox](../../../docs/postgres-transactional-outbox.md)
owns its transaction, fencing, ordering, retry, redrive, retention, and outcome
classification; [NATS JetStream](../../../docs/durable-messaging.md) owns its
publication identity, acknowledgements, and dead-letter transfer. An absent pack
is a selected profile, not a gap. Reuse the existing pack and reject a second
publication path without a distinct invariant.

For a **Decision**, first prove cross-service coordination is necessary and load
[compensation guidance](references/decision/pivot-compensation-and-forward-recovery.md)
only when another owner already performed an effect. For **Review**, replay each
affected step against its pack contract. Complete when every step has an
invariant, recovery or compensation, reconciliation action, and focused proof.

Hand business identity to `go-domain-invariant`, local access to `go-db-cache`,
broker guarantees to [reliable messaging](../../../docs/universal-disciplines/reliable-messaging/SKILL.md),
and multi-mechanism topology to [distributed system design](../../../docs/universal-disciplines/distributed-system-design/SKILL.md).

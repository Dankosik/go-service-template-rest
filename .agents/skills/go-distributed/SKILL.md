---
name: go-distributed
description: "Durable recovery. Use when cross-service consistency, replay, ordering, compensation, redrive, or reconciliation must survive process or owner boundaries."
metadata:
  invocation: model
  kind: method
---

# Go Distributed

A cross-service flow is defined by its **recovery**.

`flow contract -> durable steps -> partial failure -> ordering and duplicates -> compensation -> redrive -> reconciliation -> proof`

For a delegated Decision or Review, or when the active artifact requires its
result interface, load the
[shared specialist contract](../../contracts/specialist-contract.md).
Trace every accepted producer effect through terminal reconciliation. For
interacting effects, exhaustive recovery-path coverage, or a decision/review
handoff, record
`DurableStep{identity, authority, commit, duplicate, reorder, unknown,
compensate, redrive, reconcile, proof}` per step. A single local path can keep
its judgment and evidence in the code or existing task artifact. Replay duplicate
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

Load [reliable messaging](../../../docs/universal-disciplines/reliable-messaging/SKILL.md)
for broker guarantees and [distributed system
design](../../../docs/universal-disciplines/distributed-system-design/SKILL.md)
when multiple coordination mechanisms change topology.

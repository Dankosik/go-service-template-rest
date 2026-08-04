---
name: go-distributed
description: "Distributed flow: Use for cross-service consistency, sagas, durable replay, ordering, compensation, redrive, or reconciliation. Own recovery; Skip local transactions, synchronization, lifecycle, or topology."
---

# Go Distributed

A cross-service flow is defined by its **recovery**: what happens when it halts halfway is the design, and the happy path is the special case.

`flow contract -> durable steps -> partial failure -> ordering and duplicates -> compensation -> redrive and replay -> reconciliation -> proof`

Every durable step is idempotent under redelivery or compensated by a named inverse; ordering holds only where a key serializes it; and a step that survives process loss must decide what a duplicate, a reorder, and a mixed-version replay each do to its invariant. Reconciliation is the flow's audit: applied, unapplied, and unknown are three different states with three different actions.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct every durable step from accepted flows, producers, consumers, broker/persistence boundaries, recovery paths, and success semantics; replay duplicate delivery, reordering, partial completion, process loss, redrive, and mixed versions while preserving process ownership and success boundaries.

## Read The Mechanism This Checkout Owns

Read whichever pack the checkout ships before designing a durable step. Initialization removes the packages a service did not select, so an absent pack is a decision to state, not a gap to fill silently.

- `internal/infra/postgresoutbox` — [PostgreSQL Transactional Outbox](../../../docs/postgres-transactional-outbox.md) owns appending inside the caller's own transaction, lease fencing, per-key claim order, retry and poison and operator redrive, retention, and the relay's applied/unapplied/unknown dispositions.
- `internal/infra/natsjs` — [Durable messaging with NATS JetStream](../../../docs/durable-messaging.md) owns publish-acknowledgement classification, the logical-versus-publication identity split, worker acknowledgement, and dead-letter transfer.

Reuse the pack a durable step already has, and look first for a second publication path opened beside it. Both packs are at-least-once: a consumer effect that is not naturally idempotent still needs a durable guard on business identity, and a pack's own delivery polarity — the outbox relay retries an unclassified failure indefinitely rather than risk loss — belongs to the adapter to invert when the destination cannot absorb a duplicate.

## Choose The Branch

- **Decision** — select when durable-flow policy is absent or changing. First prove cross-service coordination is needed. Load [pivot, compensation, and forward recovery](references/decision/pivot-compensation-and-forward-recovery.md) when the flow must undo work another owner already performed. Complete when shared Decision dispositions cover every durable step and replay shows its invariant, compensation, or reconciliation with focused proof.
- **Review** — select when changed durable behavior must conform to accepted policy. Replay every affected durable step against the pack contract above into the shared finding envelope.

Hand business identity to `go-domain-invariant` and local DB/cache mechanics to `go-db-cache`. Load [`reliable-messaging`](../../../docs/universal-disciplines/reliable-messaging/SKILL.md) when the broker boundary itself carries the guarantee: it forces each guarantee to name its durable commit, its accepted loss or duplicate window, and its redrive path, instead of inheriting a promise from broker documentation. Load [`distributed-system-design`](../../../docs/universal-disciplines/distributed-system-design/SKILL.md) when several mechanisms must compose into one system contract: it forces each component to be earned by a named force instead of assembled from a pattern catalog.

---
name: go-distributed-spec
description: "Use when a cross-service durable flow needs consistency, saga, orchestration or choreography, outbox/inbox, idempotency, compensation, redrive, or reconciliation decisions before coding; Own distributed recovery policy and invariant handoffs; Skip when the primary decision is local transactions, service resilience, system topology, or implementation."
---

# Go Distributed Spec

Load the [shared specialist contract](../specialist-contract.md), then apply this durable-boundary policy.

## Outcome And Boundary

Turn accepted business behavior into an explicit consistency and recovery contract across durable process or service boundaries. Own process/saga authority, consistency, orchestration/choreography, durable flow state, outbox/inbox and delivery idempotency, replay and ordering, pivot/compensation/forward recovery, redrive/reconciliation, and distributed migration observability.

Consume business acceptance and business idempotency identity from `go-domain-invariant-spec` and authoritative models/local transaction capabilities from `go-data-architecture-spec`. Do not redesign local SQL/cache access, endpoint payloads, service topology, or low-level resilience tuning; record only consequences forced by the distributed decision.

## Distributed Core

1. **Start with invariant and process ownership.** Name the stable business key, authoritative owner per fact, invariant owner, durable flow boundary, command/event intent, allowed staleness, and failure outcome. Reject ownerless process invariants and writes authorized by stale read models.
2. **Keep commit-time invariants local when possible.** Stay inside one local transaction when compensation is unacceptable, intermediate states are intolerable, or a commit-time check is required. Use a saga only when local ownership cannot satisfy the accepted outcome and explicit convergence is allowed.
3. **Choose one legible coordination model.** Prefer orchestration for business-critical multi-step flows needing central retry, timeout, compensation, DLQ, or reconciliation authority; use choreography for independent reactions with no central process outcome and no event cycle. Name the boundary of any bounded hybrid.
4. **Persist resumable process state.** Require one active flow per business key or an explicit concurrency rule; use monotonic states, version checks, timeout classes, stuck-flow handling, and operator-visible terminal states. For every step, define trigger, local transaction scope, stable idempotency source, dedup boundary, retry class, success transition, and compensation or forward-recovery rule.
5. **Assume at-least-once delivery.** Default cross-boundary consistency to explicit eventual convergence with outbox-equivalent atomic linkage of local state and message intent, durable consumer inbox/dedup for side effects, and business effect plus dedup persistence before ACK or offset commit. Do not claim global ordering or end-to-end exactly-once without an explicit partition, FIFO group, stream, or single-active-consumer contract.
6. **Make irreversible progress and repair explicit.** Identify the pivot in every nontrivial saga; keep pre-pivot steps compensable and post-pivot steps idempotent, retryable, and forward-recoverable. Define replay/redrive, per-key ordering or version checks, poison/DLQ handling, idempotent resumable watermark-based reconciliation, manual intervention, and repair commands/events rather than direct cross-service table writes.
7. **Keep migration and recovery observable.** Define event compatibility windows, stored-message/version behavior, dedup retention, mixed-version replay, and migration sequence when live. Correlate producer, consumer, retries, DLQ, redrive, and reconciliation with bounded-cardinality telemetry sufficient to detect stuck or divergent flows. When the flow forces client-visible asynchrony, record the acceptance/status, operation-resource, idempotency, and retry consequences for `go-api-contract-spec` without choosing its representation.

## Symptom-Driven References

| Symptom | Load | Behavior Change |
| --- | --- | --- |
| Source-of-truth ownership, hard vs eventual consistency, projection freshness, or owner-unavailable behavior is unclear | [invariant-ownership-and-consistency-contracts.md](references/invariant-ownership-and-consistency-contracts.md) | Makes the spec route hard decisions to the invariant owner or a durable pending process instead of approving writes from stale projections or ownerless "eventual consistency." |
| The prompt says "just publish events," asks orchestration vs choreography, risks event cycles, or considers a workflow engine | [orchestration-vs-choreography.md](references/orchestration-vs-choreography.md) | Makes the spec choose an owned durable process or bounded terminal-event handoff instead of an unowned event chain with no timeout or compensation authority. |
| The flow needs saga identity, one-active-flow rules, step contracts, stuck-flow handling, or durable workflow execution | [saga-state-model-and-step-contracts.md](references/saga-state-model-and-step-contracts.md) | Makes the spec define a resumable monotonic state machine instead of keeping "current step" in memory or retrying forever without terminal states. |
| State-plus-message atomicity, outbox relay behavior, consumer dedup, inbox, ACK timing, or idempotency keys affect correctness | [outbox-inbox-and-idempotency.md](references/outbox-inbox-and-idempotency.md) | Makes the spec require durable outbox/inbox and business idempotency boundaries instead of relying on dual writes, broker "exactly once," or in-memory dedup. |
| The flow needs a pivot transaction, compensation policy, cancellation behavior, non-compensable step handling, or operator recovery | [pivot-compensation-and-forward-recovery.md](references/pivot-compensation-and-forward-recovery.md) | Makes the spec distinguish semantic compensation from post-pivot forward recovery instead of promising generic rollback or assuming timeouts mean no side effect. |
| Replay, redrive, broker ordering, per-aggregate serialization, stale projections, distributed locks, or repair jobs affect correctness | [replay-ordering-and-reconciliation.md](references/replay-ordering-and-reconciliation.md) | Makes the spec choose per-key ordering/version checks and owner-driven repair instead of assuming global order, direct projection writes, or lock-only correctness. |
| Event contract rollout, mixed versions, stored-message compatibility, DLQ/reconciliation observability, or migration sequencing changes the design | [distributed-observability-and-migration.md](references/distributed-observability-and-migration.md) | Makes the spec include compatibility windows and recovery telemetry instead of big-bang event changes or logs that cannot drive repair. |

## Return And Stop

Return flow and invariant ownership; selected coordination and consistency contract; durable state and step contracts; outbox/inbox, idempotency, dedup, ACK/commit, replay, and ordering rules; pivot, compensation, forward recovery, redrive, reconciliation, and operator repair; migration/compatibility/observability where live; assumptions, accepted risks, forced neighbor consequences, proof, and reopen conditions.

Stop when domain acceptance or business identity, authoritative ownership, allowed staleness, pivot meaning, compensation eligibility, or recovery ownership is unset; name the decision owner rather than inventing it. Reject database/broker dual writes, distributed locks as business correctness (and require fencing analysis for an unavoidable technical lock), global-order or exactly-once assumptions, stale-projection write authority, ownerless DLQ/replay/reconciliation, and specs reduced to payloads, SQL scripts, or retry knobs.

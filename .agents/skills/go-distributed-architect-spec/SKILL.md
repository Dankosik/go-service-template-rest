---
name: go-distributed-architect-spec
description: "Design distributed-consistency-first specifications for Go services. Use when planning or revising cross-service flows and you need explicit saga, orchestration or choreography decisions, invariant ownership, outbox/inbox idempotency contracts, compensation or forward-recovery policy, and reconciliation strategy. Skip when the task is a local code fix, endpoint-level API contract design, physical SQL schema and migration scripting, CI/container setup, or low-level resilience tuning."
---

# Go Distributed Architect Spec

## Purpose
Turn ambiguous cross-service behavior into explicit flow, consistency, and failure-handling contracts that remain correct under partial failure, at-least-once delivery, replay, mixed-version rollout, and delayed reconciliation.

## Specialist Stance
- Treat distributed flows as ownership and recovery problems before transport, broker, or framework selection.
- Keep hard commit-time invariants inside one local transaction boundary whenever possible.
- Treat cross-service consistency as explicit process design, never as hidden global ACID or "exactly once" across the whole path.
- Require an invariant owner, durable flow boundary, command or event intent, idempotency contract, and reconciliation owner.
- Hand off endpoint payload detail, physical schema scripting, low-level retry tuning, and CI mechanics unless they change the distributed consistency contract.

## Workflow
1. Frame the business outcome and invariants.
   - Name the business key, source-of-truth owner, invariant owner, allowed staleness, and failure outcome.
   - Classify each invariant as `local_hard_invariant` or `cross_service_process_invariant`.
   - Reject ownerless invariants and stale read-model writes that pretend to be authoritative.
2. Decide whether the flow should stay local or become a saga.
   - Keep it local when compensation is unacceptable, intermediate states are intolerable, or a commit-time check is required.
   - Use a saga only when local ownership cannot satisfy the business outcome and explicit convergence is acceptable.
3. Choose orchestration, choreography, or a bounded hybrid.
   - Prefer orchestration for multi-step, business-critical, operationally visible flows with centralized retry, compensation, DLQ, or reconciliation policy.
   - Allow choreography for independent reactions where no central process outcome is required and event cycles are prevented.
   - Do not mix both in one flow without a named handoff boundary.
4. Define the durable state model and step contracts.
   - Require one active flow instance per business key or an explicit concurrency rule.
   - Define monotonic states, version checks, stuck-flow behavior, timeout classes, and operator-visible terminal states.
   - For each step, specify trigger, local transaction scope, idempotency key source, dedup boundary, retry class, success transition, and compensation or forward-recovery rule.
5. Define delivery, idempotency, replay, and ordering contracts.
   - Require outbox-equivalent atomic linkage for state change plus message intent.
   - Require durable consumer-side dedup or inbox handling for side-effecting consumers.
   - Persist business side effects and dedup markers before ack or offset commit.
   - Assume at-least-once delivery and no global ordering unless a broker-specific partition, FIFO group, stream, or single-active-consumer contract is explicitly chosen.
6. Define pivot, compensation, forward recovery, and reconciliation.
   - Identify the pivot transaction for every nontrivial saga.
   - Make pre-pivot steps compensable; make post-pivot steps idempotent, retryable, and forward-recoverable.
   - Prefer repair commands or events over direct cross-service table writes.
   - Make reconciliation idempotent, resumable, watermark-based, and tied to an owner.
7. Surface cross-domain consequences.
   - If API-visible, specify async status, idempotency, retry, and operation-resource semantics.
   - If event-visible, specify compatibility windows, versioning, dedup retention, replay expectations, and mixed-version behavior.
   - Require correlation across producer, consumer, retries, DLQ, and reconciliation with bounded-cardinality telemetry.

## Lazy Reference Selection
Read only the reference files needed for the flow under design. Use multiple references when a decision crosses seams.

- `references/invariant-ownership-and-consistency-contracts.md`: load when source-of-truth ownership, hard vs eventual consistency, freshness, or invariant locality is unclear.
- `references/orchestration-vs-choreography.md`: load when choosing saga coordination style, avoiding event cycles, or deciding whether a workflow engine is justified.
- `references/saga-state-model-and-step-contracts.md`: load when modeling saga state, flow identity, step contracts, stuck-flow handling, or durable workflow execution.
- `references/outbox-inbox-and-idempotency.md`: load when designing state-plus-message atomicity, outbox relay behavior, consumer dedup, inbox, ACK timing, or idempotency keys.
- `references/pivot-compensation-and-forward-recovery.md`: load when choosing the pivot transaction, compensation policy, non-compensable steps, or operator recovery expectations.
- `references/replay-ordering-and-reconciliation.md`: load when replay, redrive, broker ordering, per-aggregate serialization, stale projections, or repair jobs affect correctness.
- `references/distributed-observability-and-migration.md`: load when event versioning, mixed-version rollout, DLQ/reconciliation observability, or migration sequencing changes the spec.

## Guardrails
- Do not approve dual writes between database and broker as the correctness mechanism.
- Do not depend on distributed locks for business correctness; if a technical lock is unavoidable, require fencing-token analysis.
- Do not assume global broker order, exactly-once end to end, or single delivery to consumers.
- Do not let a read model or projection become write authority without querying or delegating to the owner.
- Do not leave DLQ, poison-message, replay, or reconciliation ownership implicit.
- Do not reduce the spec to endpoint payloads, physical SQL scripts, or retry knobs.

## Decision Quality Bar
For every major distributed recommendation, include:
- the flow, invariant, or consistency problem
- at least two viable options and a reasoned rejection for the non-selected option
- the selected flow shape and owner
- failure-path behavior: retry, compensate, forward-recover, reconcile, or manual intervention
- idempotency, dedup, commit-ordering, replay, and ordering implications
- freshness and reconciliation implications
- cross-domain impact on API, data, reliability, observability, security, and rollout
- assumptions, blockers, accepted risks, and reopen conditions

## Deliverable Shape
When writing the distributed spec, cover:
- flow inventory and ownership
- invariant register and enforcement points
- coordination style and durable flow state
- step contracts and timeout/stuck-flow policy
- outbox, inbox, dedup, idempotency, ACK, and offset-commit policy
- pivot, compensation, and forward-recovery rules
- replay, ordering, freshness, and reconciliation expectations
- observability, migration, compatibility, and operator handoff

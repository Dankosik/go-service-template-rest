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
   - Record only the downstream effects that force a new decision, handoff, or proof obligation before the current flow can be considered usable.
   - If API-visible, specify async status, idempotency, retry, and operation-resource semantics only when the flow decision changes client behavior now.
   - If event-visible, specify compatibility windows, versioning, dedup retention, replay expectations, and mixed-version behavior only when the event contract is part of the current decision frontier.
   - Require correlation across producer, consumer, retries, DLQ, and reconciliation with bounded-cardinality telemetry when the flow depends on it.

## Lazy Reference Selection
References are compact rubrics and example banks, not exhaustive checklists or domain primers. Load at most one reference by default; load multiple only when the task clearly spans independent decision pressures, such as both pivot recovery and event-version rollout.

| Symptom | Load | Behavior Change |
| --- | --- | --- |
| Source-of-truth ownership, hard vs eventual consistency, projection freshness, or owner-unavailable behavior is unclear | `references/invariant-ownership-and-consistency-contracts.md` | Makes the spec route hard decisions to the invariant owner or a durable pending process instead of approving writes from stale projections or ownerless "eventual consistency." |
| The prompt says "just publish events," asks orchestration vs choreography, risks event cycles, or considers a workflow engine | `references/orchestration-vs-choreography.md` | Makes the spec choose an owned durable process or bounded terminal-event handoff instead of an unowned event chain with no timeout or compensation authority. |
| The flow needs saga identity, one-active-flow rules, step contracts, stuck-flow handling, or durable workflow execution | `references/saga-state-model-and-step-contracts.md` | Makes the spec define a resumable monotonic state machine instead of keeping "current step" in memory or retrying forever without terminal states. |
| State-plus-message atomicity, outbox relay behavior, consumer dedup, inbox, ACK timing, or idempotency keys affect correctness | `references/outbox-inbox-and-idempotency.md` | Makes the spec require durable outbox/inbox and business idempotency boundaries instead of relying on dual writes, broker "exactly once," or in-memory dedup. |
| The flow needs a pivot transaction, compensation policy, cancellation behavior, non-compensable step handling, or operator recovery | `references/pivot-compensation-and-forward-recovery.md` | Makes the spec distinguish semantic compensation from post-pivot forward recovery instead of promising generic rollback or assuming timeouts mean no side effect. |
| Replay, redrive, broker ordering, per-aggregate serialization, stale projections, distributed locks, or repair jobs affect correctness | `references/replay-ordering-and-reconciliation.md` | Makes the spec choose per-key ordering/version checks and owner-driven repair instead of assuming global order, direct projection writes, or lock-only correctness. |
| Event contract rollout, mixed versions, stored-message compatibility, DLQ/reconciliation observability, or migration sequencing changes the design | `references/distributed-observability-and-migration.md` | Makes the spec include compatibility windows and recovery telemetry instead of big-bang event changes or logs that cannot drive repair. |

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
- whether a real `live fork` exists
- when a `live fork` exists, the viable options and a reasoned rejection for the non-selected option
- the selected flow shape and owner
- failure-path behavior: retry, compensate, forward-recover, reconcile, or manual intervention
- idempotency, dedup, commit-ordering, replay, and ordering implications
- freshness and reconciliation implications
- only the downstream API, data, reliability, observability, security, and rollout effects that force a new decision, handoff, or proof obligation now
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
- observability, migration, compatibility, and operator handoff only when those domains must act now; otherwise use `no new decision required in <domain>`

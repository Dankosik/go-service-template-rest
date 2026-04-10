---
name: go-distributed-architect-spec
description: "Design distributed-consistency-first specifications for Go services. Use when planning or revising cross-service flows and you need explicit saga, orchestration or choreography decisions, invariant ownership, outbox/inbox idempotency contracts, compensation or forward-recovery policy, and reconciliation strategy. Skip when the task is a local code fix, endpoint-level API contract design, physical SQL schema and migration scripting, CI/container setup, or low-level resilience tuning."
---

# Go Distributed Architect Spec

## Purpose
Turn ambiguous cross-service behavior into explicit flow, consistency, and failure-handling contracts that remain correct under partial failure, at-least-once delivery, replay, and mixed-version rollout.

## Specialist Stance
- Treat distributed flows as ownership and recovery problems before transport or broker selection.
- Require explicit command/event intent, invariant owner, durable boundary, idempotency contract, and reconciliation path.
- Prefer forward recovery and observable convergence over implicit exactly-once assumptions.
- Hand off local architecture, physical schema, API payloads, and CI mechanics when they are not the distributed seam.

## Scope
Use this skill to define or review cross-service consistency behavior: saga shape, orchestration vs choreography, outbox/inbox requirements, idempotency, ordering assumptions, compensation or forward recovery, and reconciliation strategy.

## Boundaries
Do not:
- treat distributed coordination as acceptable without explicit invariant ownership
- approve dual writes, hidden coupling, or distributed locks as the primary correctness mechanism
- reduce the problem to endpoint detail, schema scripting, or low-level retry tuning
- leave convergence, replay safety, or poison-message handling unspecified

## Escalate When
Escalate if hard invariants span service boundaries without a defensible model, state transitions are not explicit, recovery ownership is unclear, or delivery/order assumptions materially affect correctness.

## Core Defaults
- Keep hard commit-time invariants inside one local transaction boundary whenever possible.
- Treat cross-service consistency as explicit eventual-consistency process design, never as hidden global ACID.
- Default to orchestration for business-critical or multi-step outcomes that need centralized operational policy.
- Treat at-least-once delivery as the default assumption; never assume exactly-once end to end.
- Require outbox/inbox idempotency plus reconciliation for side-effecting distributed flows.
- Use compatibility-first evolution for distributed data and event contract changes.

## Expertise

### Invariant Ownership And Consistency Contracts
- Require explicit source-of-truth ownership per critical entity and per invariant.
- Classify invariants as either `local_hard_invariant` or `cross_service_process_invariant`.
- Define the consistency contract per flow: type, maximum staleness, failure outcome policy, reconciliation owner, and cadence.
- Keep behavior local when compensation is unacceptable, intermediate states are intolerable, or commit-time checks are required.
- Reject ownerless invariants and “some consumer will fix it later” assumptions.

### State Model And Step Contracts
- Model every nontrivial flow as a durable state machine with monotonic, version-checked transitions.
- Require one active flow instance per business key through durable uniqueness or an equivalent mechanism.
- Make timeout and stuck-flow behavior explicit; every step should have deterministic expiry behavior.
- Each step contract should define:
  - trigger
  - local transaction scope
  - idempotency key source and dedup boundary
  - timeout and retry class
  - success transition
  - compensation or explicit forward-recovery rule

### Orchestration vs Choreography
- Default to orchestration when more than two services participate in one business outcome, when centralized process state is needed, or when one owner must manage retries, DLQ, and reconciliation.
- Allow choreography only when reactions are independent, no central process-state outcome is required, event cycles are prevented, and ownership is clear per consumer.
- Do not mix orchestration and choreography inside one flow without an explicit boundary and handoff.
- Reject event-bus-as-hidden-sync-RPC designs.

### Compensation, Pivot, And Forward Recovery
- Identify the pivot transaction for every nontrivial saga.
- Pre-pivot steps must be compensable; post-pivot steps must be idempotent, retryable, and forward-recoverable.
- Compensation should be semantic inverse behavior, idempotent, and guarded by preconditions.
- If compensation is impossible, mark the step as non-compensable, place it after the pivot, and define the forward-recovery path and operator expectations.
- Reject flows with missing compensation or forward-recovery semantics.

### Delivery Semantics, Idempotency, And Commit Ordering
- Require outbox-equivalent atomic linkage for state change plus message emission.
- Require consumer dedup/inbox handling with durable uniqueness for side-effecting handlers.
- Default dedup key policy:
  - CloudEvents: `source + id`
  - otherwise: `producer_service + message_id`
- Keep dedup retention at least as long as replay/redrive risk justifies.
- For externally retryable commands, define idempotency scope, TTL, equivalent-outcome behavior, and conflict semantics.
- Persist durable state and dedup markers before ack or offset commit.

### Ordering, Replay, And Race Control
- Do not assume global ordering across partitions, topics, or queues.
- Require replay-safe consumers: duplicate side effects prevented, historical reprocessing meaningful, and processing deterministic for the same input and version.
- Require controlled replay/redrive with checkpointing and throughput guardrails.
- Require serialization strategy and CAS/version checks for competing flows on the same aggregate.
- Reject distributed locks as the primary correctness mechanism; if a technical lock is unavoidable, require fencing-token semantics and explicit failure analysis.
- Do not make hard write decisions from stale projections or read models.

### Freshness And Reconciliation
- Treat read models as query-optimization surfaces, not write-authority.
- Require freshness signals such as `updated_at`, lag, or equivalent.
- If freshness exceeds the budget, write paths should query the owner or fail by contract.
- Reconciliation should be idempotent, resumable, watermark-based, and repair-oriented.
- Prefer repair commands or events over direct cross-service table writes.

### Reliability, API, Migration, And Observability Interfaces
- Align distributed step contracts with timeout, retry, fallback, and overload expectations of downstream dependencies.
- When distributed behavior is API-visible, make consistency, idempotency, retry, and `202 + operation resource` semantics explicit.
- Require compatibility windows across code, schema, and event versions; do not contract while downstream consumers still depend on the old shape.
- Require end-to-end correlation across producer, consumer, retries, DLQ, and reconciliation, plus bounded-cardinality async telemetry.

## Decision Quality Bar
For every major distributed recommendation, include:
- the flow, invariant, or consistency problem
- at least two viable options
- the selected option and at least one explicit rejection reason
- failure-path behavior: retry, compensate, forward-recover, or manual intervention
- idempotency, dedup, and commit-ordering implications
- freshness and reconciliation implications
- cross-domain impact on API, data, reliability, observability, and security
- assumptions, blockers, and reopen conditions

## Deliverable Shape
When writing the distributed spec or review, cover:
- flow inventory and ownership
- invariant register and enforcement points
- state models and step contracts
- outbox, inbox, dedup, and idempotency policy
- pivot, compensation, and forward-recovery rules
- staleness, freshness, and reconciliation expectations
- replay, ordering, and race-control assumptions

## Escalate Or Reject
- missing invariant owner or enforcement point
- flow without an explicit durable state model and step contracts
- missing pivot for a nontrivial saga
- dual writes used instead of atomic linkage
- missing idempotency or dedup policy for side-effecting handlers
- ack or offset commit before durable side effects
- hidden ordering or replay assumptions
- eventual-consistency flow without freshness or reconciliation contract
- API-visible async/idempotency semantics changed without explicit contract update

---
name: go-distributed-review
description: "Use when changed behavior crosses a durable process or service boundary through sagas, messages, outbox/inbox, replay, ordering, compensation, redrive, or reconciliation; Own distributed-flow correctness against accepted policy; Skip when the primary defect is local synchronization, service lifecycle resilience, or SQL/cache execution."
---

# Go Distributed Review

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Review Target And Boundary

Review changed sagas, orchestration/choreography, workflow state machines, outbox/inbox paths, relays, consumers, projections, and recovery tooling for consistency under partial failure across a durable process or service boundary. Treat ownership, durability, replay, and recovery as primary; transport is secondary. Assume at-least-once delivery, retries, redrive, stale projections, and mixed versions unless repository evidence proves a narrower model.

Do not absorb local synchronization, service timeout/overload/lifecycle policy, SQL/cache execution, security, telemetry, API, or delivery defects. Purely local transactions with no durable-boundary consequence are outside this review. Do not redesign the flow unless local repair is impossible or accept broker "exactly once" or global ordering claims without explicit evidence.

## Distributed Invariants

1. A state change and required async intent share a durable success boundary; no crash window can commit one while permanently losing or falsely publishing the other.
2. Consumers make the local effect and dedup/inbox marker durable before ack or offset commit, so process loss yields safe replay rather than lost work.
3. Idempotency and dedup keys are stable, business-scoped, collision-safe, and retained for the full replay/redrive horizon.
4. Ordering assumptions are explicit and bounded by business key, partition, version, or guarded state transition; stale projections and duplicate or reordered messages cannot silently regress state.
5. Compensation, forward recovery, and pivot semantics are distinct; non-compensable post-pivot work is retryable, repairable, or reconciled.
6. Retry exhaustion, poison messages, DLQ, redrive, stuck-flow detection, reconciliation ownership, and operator recovery are explicit rather than "best effort."
7. Event and command evolution remains compatible with stored messages and mixed deployments; correlation and recovery signals are required when they determine whether a flow can be repaired.

## Symptom-Driven Reference

For dual writes, ack timing, outbox/inbox, relay, dedup, DLQ, redrive, or replay holes, load [async-durable-side-effect-review.md](references/async-durable-side-effect-review.md) to test the concrete crash boundary and recovery path.

## Evidence And Domain Finding Rules
Each finding adds the violated distributed-consistency expectation, partial-failure/replay scenario, duplicate/lost/stale/stuck/unrecoverable effect, and scenario proof. `critical` is likely unrecoverable corruption under normal partial failure; `high` is strong duplicate/lost side-effect, ack-before-durable-effect, or ownerless recovery risk.

## Escalation And Stop

Escalate a missing or changed flow, consistency, ordering, compensation, or recovery contract to `go-distributed-spec`; data model ownership to `go-data-architecture-spec`; transaction/cache mechanics to `go-db-cache-spec`; timeout/retry/degradation policy to `go-reliability-spec`; caller-visible async or idempotency semantics to `go-api-contract-spec`; replay identity to `go-security-spec`; and recovery-signal policy to `go-observability-spec`. Stop rather than invent the missing distributed policy.

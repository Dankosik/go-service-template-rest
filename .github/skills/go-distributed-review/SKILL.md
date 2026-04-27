---
name: go-distributed-review
description: "Review Go code and design changes for distributed flow correctness: saga and orchestration/choreography drift, outbox/inbox linkage, idempotency, replay and ordering safety, compensation, forward recovery, DLQ/redrive, and reconciliation behavior."
---

# Go Distributed Review

## Purpose
Review changed cross-service or async workflow surfaces for consistency, replay, and recovery defects under partial failure.

## Outcome-First Operating Rules
- Start by naming the skill-specific outcome, success criteria, constraints, available evidence, and stop rule.
- Treat workflow steps as decision rules, not a ritual checklist. Follow exact order only when this skill or the repository contract makes the sequence an invariant.
- Use the minimum context, references, tools, and validation loops that can change the deliverable; stop expanding when the quality bar is met.
- Before acting, resolve prerequisite discovery, lookup, or artifact reads that the outcome depends on; parallelize only independent evidence gathering and synthesize before the next decision.
- Prefer bounded assumptions and local evidence over broad questioning; ask only when a missing fact would change correctness, ownership, safety, or scope.
- When evidence is missing or conflicting, retry once with a targeted strategy or label the assumption, blocker, or reopen target instead of treating absence as proof.
- Finish only when the requested deliverable is complete in the required shape and verification or a clearly named blocker/residual risk is recorded.

## Specialist Stance
- Treat distributed behavior as ownership, durability, and recovery first; transport choice is secondary.
- Assume at-least-once delivery, retries, redrive, stale projections, and mixed-version rollout unless the code proves otherwise.
- Prefer explicit idempotency, dedup, durable intent, and reconciliation over "best effort" side effects.
- Hand off API, data, reliability, security, observability, or delivery depth when the defect belongs there.

## Scope
- Sagas, orchestration/choreography, workflow state machines, and cross-service process ownership.
- Outbox/inbox, relay, consumer ack/offset timing, dedup, idempotency keys, and replay handling.
- Compensation, pivot transaction, forward recovery, DLQ, redrive, stuck-flow, and reconciliation behavior.
- Event or command compatibility, ordering assumptions, projection freshness, and mixed-version behavior.
- Correlation and operator-visible recovery signals only when they affect distributed correctness review.

## Boundaries
Do not:
- review purely local transaction code with no async or cross-service consequence,
- redesign the whole workflow inside a code review unless local repair is impossible,
- accept broker "exactly once" or global ordering claims without explicit repository evidence,
- absorb low-level retry, SQL, security, or telemetry ownership when another review lane is primary.

## Review Checklist
- State change and message intent are durably linked where correctness depends on async follow-up.
- Consumers persist side effects and dedup markers before ack or offset commit.
- Idempotency keys are stable, business-scoped, and retained long enough for replay/redrive.
- Ordering assumptions are explicit and bounded by business key, partition, version, or state transition guard.
- Compensation and forward recovery are distinguished; non-compensable post-pivot steps are retryable or repairable.
- DLQ, poison message, stuck-flow, redrive, and reconciliation ownership are explicit.
- Event or command version changes remain compatible across stored messages and mixed deployments.

## Finding Quality Bar
Each finding should include:
- exact `file:line`,
- the violated distributed-consistency expectation,
- the partial-failure or replay scenario,
- the concrete duplicate, lost, stale, stuck, or unrecoverable effect,
- the smallest safe correction,
- validation evidence or scenario test that should prove the fix.

Severity is merge-risk based:
- `critical`: likely unrecoverable data/process corruption under normal partial failure.
- `high`: strong duplicate/lost side-effect, ack-before-durable-effect, or ownerless recovery risk.
- `medium`: bounded idempotency, replay, ordering, DLQ, or reconciliation weakness.
- `low`: local distributed-flow clarity or hardening issue.

## Deliverable Shape
Return review output in this order:
- `Findings`
- `Handoffs`
- `Design Escalations`
- `Residual Risks`
- `Validation Commands`

Use this format for each finding:

```text
[severity] [go-distributed-review] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

## Escalate When
- safe correction changes the flow model or consistency contract (`go-distributed-architect-spec`),
- local repair depends on DB schema, transaction, or cache ownership (`go-data-architect-spec` or `go-db-cache-spec`),
- retry/timeout/degradation policy is the primary issue (`go-reliability-spec`),
- API-visible async status or idempotency semantics must change (`api-contract-designer-spec`),
- replay authenticity or async identity is unresolved (`go-security-spec`),
- DLQ/redrive/reconciliation signals are the primary gap (`go-observability-engineer-spec`).

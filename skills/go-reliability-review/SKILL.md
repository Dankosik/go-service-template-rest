---
name: go-reliability-review
description: "Review Go code changes for timeout and deadline propagation, retry policy, backpressure, degradation, startup and shutdown behavior, and rollout safety."
---

# Go Reliability Review

## Purpose
Protect changed failure paths from outage, cascading-failure, retry-amplification, overload, shutdown, and degraded-mode defects.

## Scope
- review timeouts, deadlines, and cancellation behavior in changed critical paths
- review retry eligibility, retry budget, idempotency expectations, and jitter or backoff behavior
- review backpressure, bounded queues or concurrency, overload handling, and isolation
- review startup, readiness, liveness, and shutdown behavior
- review degradation, fallback, and mode-transition behavior
- review rollout, rollback, and progressive-delivery safety signals
- review async and distributed reliability touchpoints when they appear in changed code
- review validation signals for fail-path behavior

## Boundaries
Do not:
- turn reliability review into architecture redesign or broad code cleanup
- take primary ownership of DB correctness, security policy, or test-strategy depth when reliability is only the symptom surface
- accept implicit infinite waits, retries, or queues
- hide missing failure-path reasoning behind happy-path success

## Core Defaults
- Review failure paths before happy paths.
- Implicit or unbounded behavior is unsafe until proven bounded.
- Timeout, retry, and overload behavior should be explicit and observable.
- Prefer rollback-safe, bounded fixes over heroics or hidden fallback.
- Prefer the smallest safe correction that restores predictable failure behavior.

## Expertise

### Timeout And Deadline Semantics
- Require explicit deadlines on outbound calls in critical paths.
- Preserve inbound deadline and cancellation propagation.
- Flag request-path replacement of request context.
- Require blocking work to have bounded wait behavior.

### Retry Budget, Eligibility, And Idempotency
- Default retry posture is no retry unless the operation and failure class are explicitly retry-safe.
- Flag retries on validation, auth, conflict, not-found, or caller-cancel classes.
- Require bounded backoff and jitter on approved retries.
- Require idempotency protection when retried operations can duplicate effects.
- For async work, require deterministic retry classes and bounded retry loops.

### Backpressure, Overload, And Isolation
- Flag unbounded queues, unbounded fan-out, and shared global pools that can amplify failure.
- Prefer fast rejection or bounded waiting over indefinite accumulation.
- Require per-dependency or per-lane isolation where overload matters.
- Verify overload behavior remains explicit enough for callers and operators to reason about.

### Startup, Readiness, Liveness, And Shutdown
- Verify readiness and startup behavior do not admit traffic before the component is actually ready.
- Keep liveness independent from optional external dependencies when appropriate.
- Flag shutdown sequences that can lose requests, data, or goroutine cleanup.
- Require bounded drain and explicit transition to not-ready before teardown.

### Degradation And Fallback
- Require fallback and degraded behavior to stay explicit and bounded.
- Flag hidden fallback that changes correctness or user-visible semantics without a contract.
- Verify activation and recovery conditions remain observable.
- Keep fail-open vs fail-closed behavior aligned with dependency criticality.

### Rollout And Rollback Safety
- Review risky changes for progressive-delivery and rollback-friendliness.
- Flag changes that require operator heroics to undo safely.
- Treat irreversible or poorly bounded rollout behavior as a reliability risk even if happy-path code is small.

### Async And Distributed Reliability
- Require durable publish or side-effect linkage where state changes depend on async follow-up.
- Require ack or commit only after durable local side effects when applicable.
- Require dedup, DLQ, and reconciliation behavior to stay explicit when async flows are touched.
- Hand off broader saga or workflow design when the fix is no longer local.

### Data, Cache, And Observability Interaction
- Flag retries across long transactions, cache outage paths that storm the origin, and hidden stale-data fallbacks on correctness-sensitive paths.
- Require reliability state changes to stay observable with bounded-cardinality signals.
- Keep these checks focused on failure behavior; hand off primary DB/cache and telemetry ownership when needed.

### Cross-Domain Handoffs
- Hand off DB/cache contract defects to `go-db-cache-review`.
- Hand off race, goroutine, and shutdown coordination depth to `go-concurrency-review`.
- Hand off exploit or fail-open security depth to `go-security-review`.
- Hand off benchmark and saturation proof to `go-performance-review`.
- Hand off coverage completeness to `go-qa-review`.
- Hand off broader structural drift to `go-design-review`.

## Finding Quality Bar
Each finding should include:
- exact `file:line`
- the violated reliability expectation
- the concrete failure mode and blast radius
- the smallest safe correction
- a validation command when useful
- whether the issue is local code drift or needs design escalation

Severity is merge-risk based:
- `critical`: outage or cascading-failure risk that makes merge unsafe
- `high`: strong evidence of significant reliability contract mismatch
- `medium`: bounded but meaningful reliability weakness
- `low`: local hardening improvement

## Deliverable Shape
Return review output in this order:
- `Findings`
- `Handoffs`
- `Design Escalations`
- `Residual Risks`
- `Validation Commands`

Use this format for each finding:

```text
[severity] [go-reliability-review] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

## Escalate When
Escalate when:
- safe correction changes timeout, retry, degradation, or startup/shutdown policy (`go-reliability-spec`)
- the right fix needs a new async workflow, outbox, compensation, or reconciliation model (`go-distributed-architect-spec`)
- local repair depends on new DB/cache fallback or consistency design (`go-db-cache-spec`)
- API-visible retry, async, or overload semantics must change (`api-contract-designer-spec`)
- the reliability issue is really a broader architecture or rollout-design problem (`go-design-spec` or `go-devops-spec`)

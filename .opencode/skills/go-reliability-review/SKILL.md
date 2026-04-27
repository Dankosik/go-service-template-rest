---
name: go-reliability-review
description: "Review Go code changes for timeout and deadline propagation, retry policy, backpressure, degradation, startup and shutdown behavior, and rollout safety."
---

# Go Reliability Review

## Purpose
Protect changed failure paths from outage, cascading-failure, retry-amplification, overload, shutdown, and degraded-mode defects.

## Outcome-First Operating Rules
- Start by naming the skill-specific outcome, success criteria, constraints, available evidence, and stop rule.
- Treat workflow steps as decision rules, not a ritual checklist. Follow exact order only when this skill or the repository contract makes the sequence an invariant.
- Use the minimum context, references, tools, and validation loops that can change the deliverable; stop expanding when the quality bar is met.
- Before acting, resolve prerequisite discovery, lookup, or artifact reads that the outcome depends on; parallelize only independent evidence gathering and synthesize before the next decision.
- Prefer bounded assumptions and local evidence over broad questioning; ask only when a missing fact would change correctness, ownership, safety, or scope.
- When evidence is missing or conflicting, retry once with a targeted strategy or label the assumption, blocker, or reopen target instead of treating absence as proof.
- Finish only when the requested deliverable is complete in the required shape and verification or a clearly named blocker/residual risk is recorded.

## Specialist Stance
- Review failure behavior before happy-path success.
- Prioritize unbounded waits, retries, queues, goroutines, degraded-mode correctness, and rollout/rollback traps.
- Treat timeout, retry, overload, readiness, and shutdown semantics as caller- and operator-visible behavior.
- Hand off concurrency, DB/cache, security, performance, or distributed design when reliability is not the primary owner of the fix.

## Scope
- review timeouts, deadlines, and cancellation behavior in changed critical paths
- review retry eligibility, retry budget, idempotency expectations, and jitter or backoff behavior
- review backpressure, bounded queues or concurrency, overload handling, and isolation
- review startup, readiness, liveness, and shutdown behavior
- review degradation, fallback, and mode-transition behavior
- review rollout, rollback, and progressive-delivery safety signals
- review async and distributed reliability touchpoints when they appear in changed code
- review validation signals for fail-path behavior

## Reference Loading
Load references lazily as compact rubrics and example banks, not as exhaustive checklists or documentation dumps. Load at most one reference by default. Load multiple references only when the diff clearly spans independent decision pressures, such as retry idempotency plus rollback compatibility.

Pick the narrowest matching reference by symptom:

| Reference | Load For Symptom | Behavior Change |
| --- | --- | --- |
| [references/timeout-deadline-and-cancellation-review.md](references/timeout-deadline-and-cancellation-review.md) | request context propagation, derived deadlines, DB/HTTP cancellation, sleeps, polling, or detached request work | makes the model report dropped caller cancellation and unbounded waits instead of accepting `context.Background`, arbitrary local timeouts, or context-unaware APIs |
| [references/retry-budget-and-idempotency-review.md](references/retry-budget-and-idempotency-review.md) | retry loops, backoff, jitter, retry classification, redrive, idempotency keys, duplicate suppression, or replay behavior | makes the model require retry eligibility, a bounded budget, cancellation, and duplicate-effect protection instead of treating retries as harmless resilience |
| [references/backpressure-overload-and-bulkheads.md](references/backpressure-overload-and-bulkheads.md) | fan-out, worker pools, semaphores, queues, buffered channels, rate limits, circuit breakers, or shared dependency pools | makes the model ask how work is bounded, rejected, or isolated instead of accepting more goroutines, larger queues, or hidden wait-until-timeout behavior |
| [references/startup-readiness-liveness-shutdown.md](references/startup-readiness-liveness-shutdown.md) | service bootstrap, health endpoints, Kubernetes probes, readiness gates, liveness checks, signal handling, HTTP drain, or shutdown sequencing | makes the model distinguish process liveness, traffic readiness, and drain completion instead of approving one health endpoint or fire-and-forget shutdown |
| [references/degradation-fallback-and-fail-open-closed.md](references/degradation-fallback-and-fail-open-closed.md) | fallback behavior, stale data, optional dependency handling, degraded responses, feature-disable paths, or fail-open/fail-closed decisions | makes the model check whether degradation is bounded, observable, and contract-safe instead of accepting silent defaults, origin storms, or fail-open access behavior |
| [references/async-durable-side-effect-review.md](references/async-durable-side-effect-review.md) | async side effects, event publishing, message acking, outbox/inbox logic, webhook enqueueing, relays, DLQs, dedup, or redrive | makes the model find dual-write, ack-before-durable-effect, and replay holes instead of assuming eventual retry makes async side effects reliable |
| [references/rollout-rollback-safety-review.md](references/rollout-rollback-safety-review.md) | feature flags, config rollout, mixed-version compatibility, schema compatibility, canary signals, rollback behavior, or capacity-sensitive release paths | makes the model preserve safe partial rollout and rollback instead of assuming all instances, config, data, and capacity change together |

If a narrower positive reference matches, prefer it over broad smell triage. If a finding crosses references, name the primary reliability behavior in the finding and use the second reference only to sharpen validation. Keep findings failure-path-first and local to the changed code. Hand off broader distributed, DB/cache, security, concurrency, performance, API, or delivery ownership when the smallest safe fix no longer fits this review lane.

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
- Require outbound calls in critical paths to be covered by a caller-derived deadline or timeout budget.
- Preserve inbound deadline and cancellation propagation.
- Flag request-path replacement of request context.
- Require blocking work to have bounded wait behavior.

### Retry Budget, Eligibility, And Idempotency
- Default retry posture is no retry unless the operation and failure class are explicitly retry-safe.
- Flag retries on validation, auth, caller-cancel, or unqualified conflict/not-found classes.
- Require capped backoff and jitter when repeated or correlated retries can align.
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
- Require bounded drain and a traffic-removal signal before teardown, either app-owned not-ready state or a proven platform-owned drain path.

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

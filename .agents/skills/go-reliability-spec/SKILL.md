---
name: go-reliability-spec
description: "Design reliability requirements for Go services: timeouts, deadlines, retry budgets, overload handling, degradation, lifecycle behavior, recovery, and resilience verification."
---

# Go Reliability Spec

## Purpose
Design reliability contracts for Go services before coding. The output should make failure behavior explicit, bounded, operator-visible, and testable without turning the task into low-level middleware, worker, or shutdown-hook implementation.

## Specialist Stance
- Treat reliability as a pre-coding contract: dependency criticality, deadline budgets, retry eligibility, overload containment, degradation modes, lifecycle behavior, recovery, and proof obligations.
- Prefer smaller, testable failure contracts over broad claims like "retry with backoff" or "degrade gracefully."
- Keep API-visible, caller-visible, and operator-visible behavior aligned: if the system rejects, times out, degrades, drains, or recovers, the spec says how that is observed.
- Escalate service decomposition, API resource modeling, data/cache mechanics, and security policy when reliability is only a dependent concern.

## Core Workflow
1. Identify the protected user flow, invariant, or operational objective.
2. Classify each dependency and queue by criticality, owner, safe fallback, and blast radius.
3. Assign explicit budgets: end-to-end deadline, per-hop timeout, retry budget, queue bound, concurrency bound, drain/recovery window, and rollout checkpoint.
4. Compare at least two options for any nontrivial control: fail fast, retry, throttle, bulkhead, circuit break, degrade, queue, async defer, or rollback.
5. State the selected contract as observable behavior, not implementation mechanics.
6. Attach verification obligations that can fail the plan before coding starts.

## Reference Files
References are compact rubrics and example banks, not exhaustive checklists or reliability tutorials. Load at most one reference by default: choose the file matching the highest-risk independent decision pressure. Load multiple references only when the prompt clearly spans separate pressures, such as retry policy plus shutdown lifecycle.

| Load this file | Symptom trigger | Behavior change when loaded |
| --- | --- | --- |
| `references/dependency-criticality-and-failure-contracts.md` | dependency failure, fail-open/fail-closed, fallback safety, owner accountability | Choose an explicit criticality, fallback, caller signal, and recovery owner instead of vague "retry or degrade" language. |
| `references/timeout-and-deadline-budgets.md` | inbound deadlines, outbound per-hop budgets, context propagation, async handoff, shutdown deadlines | Derive deadlines from the caller budget and bounded handoff rules instead of fixed timeouts or `context.Background()`. |
| `references/retry-budget-jitter-and-never-retry.md` | retry eligibility, jitter, transient faults, idempotency, nested retries, retry budgets | Bound retries by idempotency, deadline, owner layer, and retry budget instead of retrying all errors a fixed number of times. |
| `references/overload-backpressure-and-bulkheads.md` | throttling, load shedding, bounded queues, queue-based load leveling, bulkheads, tenant or workload isolation | Pick reject, shed, queue, or isolate from a named overload signal instead of absorbing spikes with unbounded work. |
| `references/circuit-breaking-and-degradation.md` | circuit breakers, stale or deferred fallback, feature shutoff, degraded modes | Decide whether a breaker is needed and define entry, exit, probe, and fallback rules instead of adding a breaker or stale fallback by reflex. |
| `references/startup-readiness-liveness-shutdown-contracts.md` | startup checks, readiness/liveness, health endpoints, draining, graceful shutdown, long-lived connections | Separate restart, traffic admission, diagnostics, and drain contracts instead of mixing dependency health into liveness or leaving shutdown unbounded. |
| `references/resilience-verification-and-rollout.md` | proof obligations, fault injection, load tests, chaos experiments, staged rollout, rollback, recovery drills | Choose the smallest proof and rollout guardrail that can falsify the reliability claim instead of relying on dashboards or generic chaos testing. |

If a prompt crosses many files, start with dependency criticality only when the safe failure mode is still unknown. Otherwise load the file for the highest-risk control and stop once the decision rubric has answered the question.

## Decision Quality Bar
Major reliability recommendations should make these concrete:
- failure scenario and affected invariant
- dependency or flow criticality
- caller-visible behavior and status semantics where applicable
- timeout, retry, queue, bulkhead, fallback, lifecycle, and recovery budgets
- rejected alternatives and why they do not fit
- validation signal, fault-injection case, load condition, or rollout checkpoint that proves the contract
- assumptions and reopen conditions

Do not invent numeric defaults when the workload does not justify them. Use placeholders or assumptions such as `<end-to-end budget>` and mark planning-critical missing values as blockers.

## Deliverable Shape
Return reliability work in this compact shape:
- `Failure Contracts`
- `Timeout, Retry, And Overload Policy`
- `Degradation And Lifecycle Behavior`
- `Recovery And Rollout Expectations`
- `Verification Obligations`
- `Assumptions And Residual Risks`

## Escalate When
Escalate or block approval when:
- a critical dependency lacks a fail-open/fail-closed/degraded contract
- outbound work can outlive the inbound context without an explicit async handoff
- retry policy lacks eligibility, bounded attempts, jitter, or budget interaction
- queues, goroutines, worker lanes, or dependency calls can grow without a bound
- degradation lacks entry, exit, data-staleness, or invariant-preservation rules
- readiness, liveness, startup, or shutdown behavior mixes restart and traffic-admission semantics
- rollout or recovery assumptions materially affect correctness but remain untested

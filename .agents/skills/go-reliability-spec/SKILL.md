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
Load only the files needed for the current reliability question.

| Load this file | When the task asks about |
| --- | --- |
| `references/dependency-criticality-and-failure-contracts.md` | dependency classes, fail-open/fail-closed behavior, fallback safety, owner accountability |
| `references/timeout-and-deadline-budgets.md` | context propagation, request deadlines, per-hop budgets, fail-fast thresholds, shutdown deadlines |
| `references/retry-budget-jitter-and-never-retry.md` | retry eligibility, retry budgets, jitter, transient faults, idempotency, never-retry cases |
| `references/overload-backpressure-and-bulkheads.md` | throttling, load shedding, bounded queues, queue-based load leveling, concurrency isolation |
| `references/circuit-breaking-and-degradation.md` | circuit breaker state, soft breakers, fallback modes, stale/deferred responses, graceful degradation |
| `references/startup-readiness-liveness-shutdown-contracts.md` | startup/readiness/liveness probes, health endpoints, draining, graceful shutdown, hijacked or long-lived connections |
| `references/resilience-verification-and-rollout.md` | failure testing, load tests, fault injection, chaos experiments, staged rollout, rollback and recovery evidence |

If a prompt crosses many files, load the dependency-criticality file first, then the file for the highest-risk control. Avoid loading all references by default.

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

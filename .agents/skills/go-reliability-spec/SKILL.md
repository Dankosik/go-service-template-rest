---
name: go-reliability-spec
description: "Use when timeout and deadline budgets, retries, overload handling, degradation, readiness, startup, drain, shutdown, recovery, or rollout behavior must be decided before coding; Own end-to-end service resilience policy and proof; Skip when the primary decision is concrete synchronization, durable distributed recovery, or implementation."
---

# Go Reliability Spec

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Outcome And Boundary

Define bounded, caller/operator-visible, falsifiable behavior for dependency failure, cancellation, overload, degradation, process lifecycle, recovery, and rollout. Own end-to-end deadline/cancellation policy, retry budgets, admission/backpressure, fail-open/closed behavior, startup/readiness/liveness, drain/shutdown, recovery, and rollout proof.

Do not choose concrete synchronization, durable saga recovery, security, topology, data/cache, API-resource, or placement policy. Record only forced consequences and stop on an unset upstream policy. Breaking an end-to-end deadline is reliability; a goroutine that cannot cancel and join is concurrency; replay after durable process loss is distributed.

## Owned Contract

- Anchor policy to a protected flow, invariant, or objective. Classify dependencies and queues by criticality, owner, blast radius, caller signal, safe fallback, and recovery owner.
- Derive per-hop timeouts from the end-to-end budget; preserve cancellation across blocking work; require bounded async handoff before work outlives a request; include drain and recovery windows.
- Default retries off until operation and error class are safe. Name one owner layer, eligible/never-retry classes, idempotency/dedup, attempts, jitter, combined deadline/load budget, exhaustion signal, and async terminal behavior.
- Drive admission from a named overload signal. Bound queue depth/age, concurrency, and worker lanes; define priority, tenant/workload isolation, reject/shed/throttle/bulkhead/defer semantics, and recovery horizon without fabricated acceptance.
- Choose fail fast, retry, breaker, stale/deferred fallback, feature-off, queue, async defer, or rollback only when correctness permits. Define degradation entry/exit, probes, staleness/expiry, lost capability, invariant safety, and visibility; critical uncertainty fails closed.
- Separate startup, local-progress liveness, traffic-admission readiness, diagnostics, and drain. Before hard kill, stop admission and bound drain/close/flush/exclusion for ordinary, streaming, hijacked, background, and telemetry work.
- Define recovery objectives, restore/failover evidence, mixed-version behavior, staged rollout, guardrails, observation windows, rollback triggers, and operator ownership. Prefer deterministic failure proof before chaos.
- Use the smallest falsifying component, fault-injection, overload/load, lifecycle, recovery-drill, or canary proof. Do not invent numbers; mark unsupported targets as assumptions and planning-critical gaps as blockers.

## Symptom-Driven References

References are compact rubrics and example banks. Load the file matching the highest-risk reliability pressure.

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

## Return And Stop

Return failure contracts; budgets and policies; caller/operator signals; measurable proof; forced consequences; assumptions, risks, and reopen conditions. Stay at observable policy level.

Block on a missing critical failure mode/owner, escaped cancellation, unsafe or unbudgeted retry, unbounded work, incomplete degradation, conflated lifecycle signals, unauthorized targets, or unproved recovery/rollout assumptions that affect correctness.

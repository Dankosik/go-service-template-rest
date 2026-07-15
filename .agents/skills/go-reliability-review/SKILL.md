---
name: go-reliability-review
description: "Use when changed Go may violate accepted end-to-end timeout, deadline, retry, overload, degradation, readiness, startup, drain, shutdown, recovery, or rollout behavior; Own service resilience-policy conformance; Skip when the primary defect is concrete synchronization, durable replay, or Go context API semantics."
---

# Go Reliability Review

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Review Target And Boundary

Review changed failure paths against accepted end-to-end resilience behavior before judging happy-path success. Timeouts, retries, overload, degradation, readiness, startup, drain, shutdown, recovery, and rollout are caller- and operator-visible contracts; implicit infinite waits, retries, or queues are unsafe unless the accepted contract says otherwise.

Stay on policy conformance. Concrete happens-before, channel, goroutine, timer, cancellation-unblock, and join defects belong to `go-concurrency-review`; replay, ordering, compensation, redrive, or reconciliation across a durable boundary belong to `go-distributed-review`. Hand off DB/cache, security, performance, API, delivery, test, observability, or placement depth when that axis is decisive.

## Reliability Invariants

1. Critical blocking dependencies preserve caller cancellation and fit the accepted end-to-end deadline budget; replacing the caller context or using a context-unaware operation must not let work outlive that contract.
2. Retries have explicit eligible failures and operations, one bounded total budget, cancellation-aware capped backoff with jitter when correlated retries can align, and duplicate-effect protection for replayable mutations; layered retries do not multiply the hidden attempt count.
3. Accepted overload behavior bounds active and queued work, wait time, and shared-resource blast radius, and exposes a deliberate reject, shed, degrade, or isolate outcome instead of accumulating until timeout.
4. Startup and readiness admit traffic only when required state is ready; liveness measures process progress; shutdown removes traffic before bounded drain and dependency teardown, including a proven platform-owned drain path when the app does not own readiness.
5. Fallback and degraded modes are explicit, bounded, observable, contract-safe, and recoverable; stale/default output, origin fallback, optional-dependency failure, and fail-open/fail-closed behavior match dependency criticality and accepted API/security/domain semantics.
6. Rollout tolerates mixed versions, config, data, and capacity; defaults are safe, canary signals identify the risky path, rollback does not require operator heroics, and irreversible behavior is explicitly accepted.
7. Reliability interactions stay local: do not retry across unsafe transaction scope, storm an origin during cache failure, hide correctness-sensitive stale data, or make resilience state invisible or unbounded-cardinality. Hand off the primary DB/cache or telemetry defect.

## Symptom-Driven References

| Symptom | Load | Review effect |
| --- | --- | --- |
| Caller context, derived deadline, blocking DB/HTTP work, sleeps, polling, or detached request work. | [timeout-deadline-and-cancellation-review.md](references/timeout-deadline-and-cancellation-review.md) | Test the accepted end-to-end budget and cancellation contract, not generic context style. |
| Operation-local retry, backoff, jitter, retry class, idempotency, or duplicate effects. | [retry-budget-and-idempotency-review.md](references/retry-budget-and-idempotency-review.md) | Require one eligible, bounded, cancellation-aware retry budget; route durable replay/redrive to distributed review. |
| Accepted overload, queue, fan-out, limiter, circuit-breaker, or bulkhead behavior. | [backpressure-overload-and-bulkheads.md](references/backpressure-overload-and-bulkheads.md) | Judge reject/degrade/isolation behavior; route concrete worker/channel/semaphore mechanics to concurrency review. |
| Bootstrap, probes, readiness, liveness, signal handling, HTTP drain, or shutdown order. | [startup-readiness-liveness-shutdown.md](references/startup-readiness-liveness-shutdown.md) | Distinguish service lifecycle policy from the local join/unblock mechanism. |
| Fallback, stale data, optional dependency, degraded response, feature disablement, or fail-open/closed behavior. | [degradation-fallback-and-fail-open-closed.md](references/degradation-fallback-and-fail-open-closed.md) | Require bounded, observable, contract-safe degradation without inventing API, security, or domain policy. |
| Feature/config rollout, mixed-version/schema compatibility, canary evidence, rollback, or capacity-sensitive release. | [rollout-rollback-safety-review.md](references/rollout-rollback-safety-review.md) | Preserve safe partial rollout and rollback rather than assuming an atomic fleet change. |

Use a second reference only for an independent reliability pressure. Durable async side effects are not a reliability-reference branch; hand them to distributed review.

## Evidence And Domain Finding Rules
Each finding adds the violated reliability expectation, concrete failure mode/blast radius, and failure-path validation. `critical` is merge-unsafe outage or cascading-failure risk; `high` is strong evidence of a significant reliability-contract mismatch.

## Escalation And Stop

Escalate missing or changed timeout, retry, overload, degradation, lifecycle, recovery, or rollout policy to `go-reliability-spec`; durable-flow policy to `go-distributed-spec`; DB/cache fallback or consistency to `go-db-cache-spec`; API-visible retry, async, or overload semantics to `go-api-contract-spec`; deployment policy to `go-delivery-platform-spec`; and placement to `go-implementation-ownership-spec`. Stop rather than invent the missing contract.

# Reference Selector

References are compact rubrics and example banks. Load the file matching the highest-risk reliability pressure.

| Load this file | Symptom trigger | Behavior change when loaded |
| --- | --- | --- |
| [dependency-criticality-and-failure-contracts.md](dependency-criticality-and-failure-contracts.md) | dependency failure, fail-open/fail-closed, fallback safety, owner accountability | Choose an explicit criticality, fallback, caller signal, and recovery owner instead of vague "retry or degrade" language. |
| [timeout-and-deadline-budgets.md](timeout-and-deadline-budgets.md) | inbound deadlines, outbound per-hop budgets, context propagation, async handoff, shutdown deadlines | Derive deadlines from the caller budget and bounded handoff rules instead of fixed timeouts or `context.Background()`. |
| [retry-budget-jitter-and-never-retry.md](retry-budget-jitter-and-never-retry.md) | retry eligibility, jitter, transient faults, idempotency, nested retries, retry budgets | Bound retries by idempotency, deadline, owner layer, and retry budget instead of retrying all errors a fixed number of times. |
| [overload-backpressure-and-bulkheads.md](overload-backpressure-and-bulkheads.md) | throttling, load shedding, bounded queues, queue-based load leveling, bulkheads, tenant or workload isolation | Pick reject, shed, queue, or isolate from a named overload signal instead of absorbing spikes with unbounded work. |
| [circuit-breaking-and-degradation.md](circuit-breaking-and-degradation.md) | circuit breakers, stale or deferred fallback, feature shutoff, degraded modes | Decide whether a breaker is needed and define entry, exit, probe, and fallback rules instead of adding a breaker or stale fallback by reflex. |
| [startup-readiness-liveness-shutdown-contracts.md](startup-readiness-liveness-shutdown-contracts.md) | startup checks, readiness/liveness, health endpoints, draining, graceful shutdown, long-lived connections | Separate restart, traffic admission, diagnostics, and drain contracts instead of mixing dependency health into liveness or leaving shutdown unbounded. |
| [resilience-verification-and-rollout.md](resilience-verification-and-rollout.md) | proof obligations, fault injection, load tests, chaos experiments, staged rollout, rollback, recovery drills | Choose the smallest proof and rollout guardrail that can falsify the reliability claim instead of relying on dashboards or generic chaos testing. |

If a prompt crosses many files, start with dependency criticality only when the safe failure mode is still unknown. Otherwise load the file for the highest-risk control and stop once the decision rubric has answered the question.

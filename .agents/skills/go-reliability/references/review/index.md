# Reference Selector

| Symptom | Load | Review effect |
| --- | --- | --- |
| Caller context, derived deadline, blocking DB/HTTP work, sleeps, polling, or detached request work. | [timeout-deadline-and-cancellation-review.md](timeout-deadline-and-cancellation-review.md) | Test the accepted end-to-end budget and cancellation contract, not generic context style. |
| Operation-local retry, backoff, jitter, retry class, idempotency, or duplicate effects. | [retry-budget-and-idempotency-review.md](retry-budget-and-idempotency-review.md) | Require one eligible, bounded, cancellation-aware retry budget; route durable replay/redrive to distributed review. |
| Accepted overload, queue, fan-out, limiter, circuit-breaker, or bulkhead behavior. | [backpressure-overload-and-bulkheads.md](backpressure-overload-and-bulkheads.md) | Judge reject/degrade/isolation behavior; route concrete worker/channel/semaphore mechanics to concurrency review. |
| Bootstrap, probes, readiness, liveness, signal handling, HTTP drain, or shutdown order. | [startup-readiness-liveness-shutdown.md](startup-readiness-liveness-shutdown.md) | Distinguish service lifecycle policy from the local join/unblock mechanism. |
| Fallback, stale data, optional dependency, degraded response, feature disablement, or fail-open/closed behavior. | [degradation-fallback-and-fail-open-closed.md](degradation-fallback-and-fail-open-closed.md) | Require bounded, observable, contract-safe degradation without inventing API, security, or domain policy. |
| Feature/config rollout, mixed-version/schema compatibility, canary evidence, rollback, or capacity-sensitive release. | [rollout-rollback-safety-review.md](rollout-rollback-safety-review.md) | Preserve safe partial rollout and rollback rather than assuming an atomic fleet change. |

Use a second reference only for an independent reliability pressure. Durable async side effects are not a reliability-reference branch; hand them to distributed review.

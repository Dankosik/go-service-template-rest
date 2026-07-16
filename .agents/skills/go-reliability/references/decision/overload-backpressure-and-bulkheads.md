# Overload, Backpressure, And Bulkheads

## Behavior Change Thesis
When loaded for overload control, this file makes the model choose a named overload signal plus bounded reject, shed, queue, or bulkhead behavior instead of likely mistake "absorb the spike" with unbounded channels or a global worker pool.

## When To Load
Load when the spec needs throttling, load shedding, bounded queues, queue-based load leveling, dependency isolation, tenant isolation, concurrency limits, or overload-visible status semantics.

## Decision Rubric
- Pick the authoritative overload signal: concurrency, queue depth, queue age, dependency latency, CPU, memory, quota, or SLO burn.
- Decide which work is rejected, delayed, shed, or degraded first; correctness-critical work does not get shed to preserve optional work.
- Use fast rejection when work cannot be safely queued or the queue is full: `429` for policy/quota throttling, `503` for capacity/dependency exhaustion, and `Retry-After` only when the recovery horizon is meaningful.
- Use queue-based load leveling only when intake can be decoupled from processing and the spec names queue bound, max age, worker rate, reply/tracking model, and backlog-over-limit behavior.
- Use a bulkhead when a dependency, tenant, class of work, or consumer can exhaust shared resources; isolate with a separate pool, queue, worker lane, connection pool, or quota plus independent metrics.
- Use client-side throttling only when clients can be updated and the server exposes reliable throttling signals.

## Imitate
- "When optional image enrichment queue age exceeds `<max age>`, stop accepting enrichment work, keep checkout running, and emit `degraded_optional_off`."
- "Tenant import jobs use a separate worker pool and queue bound from checkout-critical inventory calls so import backlog cannot consume checkout capacity."
- "If the service is at capacity and cannot predict recovery, return `503`; if a tenant exceeds policy quota, return `429` with the documented retry window."

## Reject
- "Use an unbounded channel to absorb spikes." This turns overload into memory pressure and delayed failure.
- "Queue every request and scale workers later." This fails when callers need low latency or the queue can stay permanently behind.
- "One global worker pool handles all dependencies." A slow optional dependency can starve critical work.
- "Return success when a bounded queue rejected the work." Callers lose the real acceptance contract.

## Agent Traps
- Do not specify a queue without a max age. Queue length alone can hide stale work.
- Do not use `429` and `503` interchangeably; quota or policy and service capacity are different caller contracts.
- Do not add a bulkhead without naming what it protects from what.

## Validation Shape
- Given queue depth or age exceeds the entry threshold, optional work is shed or rejected before core work.
- Given a tenant or dependency saturates its bulkhead, other bulkheads continue serving within their budgets.
- Given overload rejection, response status and `Retry-After` behavior match the documented policy.
- Given load-leveling backlog exceeds max age or bound, the system enters the named fail-fast or degradation mode instead of accepting unbounded work.

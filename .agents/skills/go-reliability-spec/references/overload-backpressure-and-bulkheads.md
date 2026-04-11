# Overload, Backpressure, And Bulkheads

## When To Load This
Load this file when the spec needs throttling, load shedding, bounded queues, queue-based load leveling, dependency isolation, tenant isolation, concurrency limits, or overload-visible status semantics.

## Contract Questions
- What overload signal is authoritative: concurrency, queue depth, queue age, dependency latency, CPU, memory, quota, or SLO burn?
- Which work is rejected, delayed, shed, or degraded first?
- Which dependency, tenant, or workload gets its own bulkhead?
- What status, header, or async acknowledgment tells callers what happened?

## Option Comparisons
| Option | Use when | Contract shape | Reject when |
| --- | --- | --- | --- |
| Fast rejection | Work cannot be safely queued or the queue is full. | Return `429` for policy/quota throttling or `503` for capacity/dependency exhaustion; include `Retry-After` when the recovery horizon is meaningful. | The caller contract requires durable acceptance. |
| Load shedding | The service approaches overload but can prioritize core work. | Drop optional or lower-priority work based on named signals and thresholds. | Shedding would remove work needed for correctness. |
| Queue-based load leveling | Intake can be decoupled from processing. | Queue bound, max age, worker rate, reply/tracking model, and overload behavior when backlog exceeds limit. | The caller requires a low-latency synchronous response. |
| Bulkhead | A dependency, tenant, class of work, or consumer can exhaust shared resources. | Separate capacity pool, queue, worker lane, connection pool, or quota with independent metrics. | The isolation cost exceeds the blast-radius reduction for a low-risk flow. |
| Client-side throttling | Rejected requests themselves can overload the backend. | Clients self-limit after a rejection ratio or quota signal and fail locally above cap. | Clients cannot be updated or the server cannot expose reliable throttling signals. |

## Accepted Examples
- "When optional image enrichment queue age exceeds `<max age>`, stop accepting enrichment work, keep checkout running, and emit `degraded_optional_off`."
- "Tenant import jobs use a separate worker pool and queue bound from checkout-critical inventory calls so import backlog cannot consume checkout capacity."
- "If the service is at capacity and cannot predict recovery, return `503`; if a tenant exceeds its policy quota, return `429` with the documented retry window."

## Rejected Examples
- "Use an unbounded channel to absorb spikes." Rejected because it turns overload into memory pressure and delayed failure.
- "Queue every request and scale workers later." Rejected if callers need low latency or the queue can stay permanently behind.
- "One global worker pool handles all dependencies." Rejected when a slow optional dependency can starve critical work.
- "Return success when a bounded queue rejected the work." Rejected because callers lose the actual acceptance contract.

## Testable Failure Contracts
- Given queue depth or age exceeds the entry threshold, optional work is shed or rejected before core work.
- Given a tenant or dependency saturates its bulkhead, other bulkheads continue serving within their budgets.
- Given overload rejection, the response status and `Retry-After` behavior match the documented policy.
- Given a load-leveling queue backlog exceeds max age or bound, the system enters the named fail-fast or degradation mode instead of accepting unbounded work.

## Exa Source Links
- Google SRE, Handling Overload: https://sre.google/sre-book/handling-overload/
- Google SRE, Addressing Cascading Failures: https://sre.google/sre-book/addressing-cascading-failures/
- Microsoft Azure, Bulkhead pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/bulkhead
- Microsoft Azure, Throttling pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/throttling
- Microsoft Azure, Queue-Based Load Leveling pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/queue-based-load-leveling
- Microsoft Azure, Rate Limiting pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/rate-limiting-pattern

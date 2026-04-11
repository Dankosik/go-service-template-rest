# Circuit Breaking And Degradation

## When To Load This
Load this file when the spec must choose between soft breakers, state-machine circuit breakers, stale or deferred fallback, feature shutoff, or graceful degradation.

## Contract Questions
- Is the dependency failure likely transient, persistent, capacity-related, or correctness-threatening?
- Can simpler controls, such as deadline, retry budget, bulkhead, and shedding, contain the failure?
- What does each degradation mode allow, forbid, and expose?
- What evidence closes the breaker or exits degradation?

## Option Comparisons
| Option | Use when | Contract shape | Reject when |
| --- | --- | --- | --- |
| Soft breaker | Retry budget, timeout, bulkhead, and shedding can contain the failure without explicit breaker state. | Stop retrying, shed/degrade optional work, and fail fast when budgets are exhausted. | Call attempts themselves keep harming the dependency or caller latency. |
| State-machine circuit breaker | Persistent remote faults need a fail-fast proxy and controlled probes. | Closed/open/half-open states, trip threshold, open window, probe concurrency, success criteria, and fallback behavior. | The threshold cannot be measured or the flow has no safe fallback/fail-fast response. |
| Stale fallback | A read path can safely serve older data. | Max staleness, source of stale data, visible stale marker or signal, and recovery rule. | Data controls authorization, money movement, or other hard invariants. |
| Deferred fallback | Work can complete later with durable state. | Accepted/deferred response, tracking reference, retry/reconciliation owner, expiry, and terminal states. | The caller observes success as if work already completed. |
| Feature-off degradation | Optional work is expensive or failing. | Entry/exit thresholds, affected features, priority order, and user/operator signal. | The feature is part of the core invariant or contract. |

## Accepted Examples
- "Use a soft breaker for search suggestions: when retry budget is exhausted, skip suggestions for `<cooldown>` and emit a degradation signal."
- "Use a state-machine breaker for a quota-limited payment-risk API: open after the defined failure threshold, allow `<probe concurrency>` half-open probes, and fail checkout closed while open."
- "Serve product metadata from cache up to `<max staleness>` during catalog outage, but never serve stale price authorization."

## Rejected Examples
- "Add a circuit breaker to every dependency." Rejected because breakers add state and policy; deadlines, retry budgets, and bulkheads may be sufficient.
- "Fallback to stale data for authorization." Rejected unless a specific emergency access policy is approved and audited.
- "Half-open sends normal traffic." Rejected because recovery probes should be limited to avoid flooding a recovering dependency.
- "Degraded mode has no exit condition." Rejected because operators cannot know when normal behavior is restored.

## Testable Failure Contracts
- Given breaker-open state, calls fail fast or use the documented fallback without reaching the dependency.
- Given half-open state, only the allowed number of probe requests reaches the dependency.
- Given fallback activation, the response or telemetry records the degradation mode and any staleness/deferred-work marker.
- Given recovery criteria are met, the system exits degradation and records the transition.
- Given fallback would violate a hard invariant, the spec selects fail-closed instead.

## Exa Source Links
- Microsoft Azure, Circuit Breaker pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker
- Microsoft Azure, Design for self-healing: https://learn.microsoft.com/en-us/azure/architecture/guide/design-principles/self-healing
- Google SRE, Handling Overload: https://sre.google/sre-book/handling-overload/
- Google SRE, Addressing Cascading Failures: https://sre.google/sre-book/addressing-cascading-failures/

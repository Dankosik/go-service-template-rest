# Circuit Breaking And Degradation

## Behavior Change Thesis
When loaded for degraded-mode design, this file makes the model choose between soft containment, state-machine breakers, stale fallback, deferred fallback, and feature shutoff instead of likely mistake "add a circuit breaker" or "serve stale data" by reflex.

## When To Load
Load when the spec must choose between soft breakers, state-machine circuit breakers, stale fallback, deferred fallback, feature shutoff, or graceful degradation.

## Decision Rubric
- First ask whether deadline, retry budget, bulkhead, and shedding already contain the failure. A breaker is not the default.
- Use a soft breaker when retry budget, timeout, bulkhead, and shedding can stop harm without explicit breaker state.
- Use a state-machine breaker when persistent remote faults need fail-fast behavior and controlled probes; define closed/open/half-open states, trip threshold, open window, probe concurrency, success criteria, and fallback.
- Use stale fallback only for read paths where older data cannot violate hard invariants; state max staleness, source, visible marker or signal, and recovery rule.
- Use deferred fallback only when the caller sees accepted/deferred semantics with tracking, retry/reconciliation owner, expiry, and terminal states.
- Use feature-off degradation only for optional work; define entry/exit thresholds, affected features, priority order, and user/operator signal.

## Imitate
- "Use a soft breaker for search suggestions: when retry budget is exhausted, skip suggestions for `<cooldown>` and emit a degradation signal."
- "Use a state-machine breaker for a quota-limited payment-risk API: open after the defined failure threshold, allow `<probe concurrency>` half-open probes, and fail checkout closed while open."
- "Serve product metadata from cache up to `<max staleness>` during catalog outage, but never serve stale price authorization."

## Reject
- "Add a circuit breaker to every dependency." Breakers add state and policy; deadlines, retry budgets, and bulkheads may be sufficient.
- "Fallback to stale data for authorization." This is unsafe unless a specific emergency access policy is approved and audited.
- "Half-open sends normal traffic." Recovery probes must be limited to avoid flooding a recovering dependency.
- "Degraded mode has no exit condition." Operators cannot know when normal behavior is restored.

## Agent Traps
- Do not let the breaker hide a missing criticality decision. If fallback would violate a hard invariant, choose fail-closed.
- Do not make half-open a percentage rollout without a probe concurrency cap.
- Do not define degraded entry without degraded exit, or operators inherit a sticky degraded mode.

## Validation Shape
- Given breaker-open state, calls fail fast or use the documented fallback without reaching the dependency.
- Given half-open state, only the allowed number of probe requests reaches the dependency.
- Given fallback activation, response or telemetry records degradation mode and any staleness/deferred-work marker.
- Given recovery criteria are met, the system exits degradation and records the transition.
- Given fallback would violate a hard invariant, the spec selects fail-closed instead.

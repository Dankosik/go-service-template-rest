# Resilience Verification And Rollout

## Behavior Change Thesis
When loaded for reliability proof planning, this file makes the model choose a falsifiable verification and rollout shape matched to the risk instead of likely mistake "we have retries/backups" or "roll out and watch dashboards."

## When To Load
Load when the spec needs proof obligations, failure testing, load tests, fault injection, chaos experiments, staged rollout, rollback triggers, recovery drills, or post-deploy evidence.

## Decision Rubric
- Use unit or component failure tests when a local contract can be isolated deterministically: timeout, cancellation, dependency error, queue-full, or breaker state.
- Use integration fault injection when cross-component behavior must be observed: slow/unavailable dependency, throttling, retry exhaustion, or shutdown while asserting the external contract.
- Use load or overload tests when backpressure, queue bounds, autoscaling, or shedding must be calibrated; drive to named thresholds and prove rejection/degradation before collapse.
- Use chaos experiments only after deterministic contracts, blast radius, abort conditions, steady-state signal, and rollback are defined.
- Use staged rollout when runtime behavior may differ under real traffic; define canary scope, metrics, error-budget guard, rollback trigger, and observation window.
- Use recovery drills when backup, restore, failover, replay, or reconciliation is part of the claim; prove recovery time, data-loss window, idempotency, and operator runbook.

## Imitate
- "Fault-inject the catalog dependency to return slow responses and 503s; prove checkout either uses bounded stale fallback or fails closed within `<budget>`."
- "Run an overload test until queue age reaches `<threshold>`; prove optional work sheds before core API latency exceeds `<SLO threshold>`."
- "Canary the retry-policy change to `<scope>`; rollback if retry attempts exceed `<retry budget>` or dependency 503s increase beyond `<guardrail>` for `<window>`."
- "Restore from backup in a test environment and record restore time plus accepted data-loss window before claiming recoverability."

## Reject
- "We have retries, so resilience is covered." Retry behavior must be tested against transient, persistent, and overload failures.
- "We have backups." Restore behavior and recovery objectives must be tested.
- "Chaos test first." Deterministic failure contracts, abort conditions, and rollback need to exist first.
- "Roll out globally and watch dashboards." This is weak when the change can cause cascading retries, queue buildup, breaker flapping, readiness churn, or sticky degradation.

## Agent Traps
- Do not select chaos testing to sound rigorous when a component failure test would better falsify the contract.
- Do not let rollout guardrails be pure observability nouns; name the rollback trigger and observation window.
- Do not claim recoverability from backup existence; require restore evidence.

## Validation Shape
- Given dependency timeout/error injection, the selected fail-closed/degraded/fail-open behavior occurs within the documented deadline.
- Given overload load test reaches the entry threshold, shedding/throttling/bulkhead behavior activates before resource exhaustion.
- Given retry storms are simulated, retry budgets cap extra load and expose exhaustion signals.
- Given shutdown or rollout restart, readiness/drain contracts hold under in-flight traffic.
- Given rollback trigger fires during canary, rollout stops and recovery path is executed or explicitly handed to an operator.

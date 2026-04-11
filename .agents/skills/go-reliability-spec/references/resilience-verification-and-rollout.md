# Resilience Verification And Rollout

## When To Load This
Load this file when the spec needs proof obligations, failure testing, load tests, fault injection, chaos experiments, staged rollout, rollback triggers, recovery drills, or post-deploy evidence.

## Contract Questions
- Which failure modes must be proved before coding is considered ready?
- Which tests prove overload behavior without harming shared environments?
- Which rollout checkpoint catches retry storms, queue buildup, breaker flapping, readiness churn, or degraded-mode stickiness?
- Which recovery claim depends on a tested restore, failover, replay, reconciliation, or rollback path?

## Option Comparisons
| Option | Use when | Contract shape | Reject when |
| --- | --- | --- | --- |
| Unit or component failure test | A local contract can be isolated deterministically. | Inject timeout, cancellation, dependency error, queue-full, or breaker state and assert behavior. | The risk depends on multiple deployed services or load interactions. |
| Integration fault injection | Cross-component behavior must be observed. | Simulate slow/unavailable dependency, throttling, retry exhaustion, or shutdown while asserting external contract. | It requires unsafe production disruption without controls. |
| Load or overload test | Backpressure, queue bounds, autoscaling, or shedding must be calibrated. | Drive load to named thresholds and prove rejection/degradation before collapse. | The environment cannot represent capacity or the test would damage shared dependencies. |
| Chaos experiment | Recovery automation or self-healing needs confidence under controlled production-like faults. | Define blast radius, abort conditions, steady-state signal, and rollback. | The system lacks basic deterministic tests or rollback controls. |
| Staged rollout | Runtime behavior may differ under real traffic. | Canary scope, metrics, error budget guard, rollback trigger, and observation window. | The change is tiny and has no runtime reliability surface. |
| Recovery drill | Backup, restore, failover, replay, or reconciliation is part of the contract. | Prove recovery time, data loss window, idempotency, and operator runbook. | The spec only claims backup exists without restore evidence. |

## Accepted Examples
- "Fault-inject the catalog dependency to return slow responses and 503s; prove checkout either uses the bounded stale fallback or fails closed within `<budget>`."
- "Run an overload test until queue age reaches `<threshold>`; prove optional work sheds before core API latency exceeds `<SLO threshold>`."
- "Canary the retry-policy change to `<scope>`; rollback if retry attempts exceed `<retry budget>` or dependency 503s increase beyond `<guardrail>` for `<window>`."
- "Restore from backup in a test environment and record restore time plus accepted data-loss window before claiming recoverability."

## Rejected Examples
- "We have retries, so resilience is covered." Rejected because retry behavior must be tested against transient, persistent, and overload failures.
- "We have backups." Rejected unless restore behavior and recovery objectives are tested.
- "Chaos test first." Rejected when deterministic failure contracts, abort conditions, and rollback are not in place.
- "Roll out globally and watch dashboards." Rejected when the change can cause cascading retries, queue buildup, or breaker flapping.

## Testable Failure Contracts
- Given dependency timeout/error injection, the selected fail-closed/degraded/fail-open behavior occurs within the documented deadline.
- Given overload load test reaches the entry threshold, shedding/throttling/bulkhead behavior activates before resource exhaustion.
- Given retry storms are simulated, retry budgets cap extra load and expose exhaustion signals.
- Given shutdown or rollout restart, readiness/drain contracts hold under in-flight traffic.
- Given rollback trigger fires during canary, rollout stops and recovery path is executed or explicitly handed to an operator.

## Exa Source Links
- Google SRE, Addressing Cascading Failures: https://sre.google/sre-book/addressing-cascading-failures/
- Google SRE, Production Services Best Practices: https://sre.google/sre-book/service-best-practices/
- Google SRE, Managing Load: https://sre.google/workbook/managing-load/
- Microsoft Azure, Design for self-healing: https://learn.microsoft.com/en-us/azure/architecture/guide/design-principles/self-healing
- Microsoft Azure, Retry pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/retry
- Microsoft Azure, Circuit Breaker pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker
- Microsoft Azure, Queue-Based Load Leveling pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/queue-based-load-leveling

# Dependency Criticality And Failure Contracts

## When To Load This
Load this file when the spec must classify dependencies, decide fail-open versus fail-closed behavior, choose fallback eligibility, or expose dependency failure behavior to callers and operators.

## Contract Questions
- Which user flow or invariant does this dependency protect?
- Is the dependency required for correctness, only for quality, or only for optional enrichment?
- What must happen when the dependency is slow, unavailable, overloaded, returning invalid data, or partially degraded?
- Who owns recovery and how does an operator know the contract is active?

## Option Comparisons
| Option | Use when | Contract shape | Reject when |
| --- | --- | --- | --- |
| `critical_fail_closed` | Continuing could violate money, authorization, privacy, inventory, idempotency, or other hard invariants. | Fail the operation, preserve state, expose a typed failure, and avoid fabricated success. | The flow can safely produce a reduced result without weakening the invariant. |
| `critical_fail_degraded` | Core flow may continue with bounded stale data, deferred work, or a reduced capability. | State max staleness, lost capability, recovery path, and user/operator signal. | The fallback can make an irreversible wrong decision. |
| `optional_fail_open` | The dependency enriches quality but is not required for the core flow. | Omit/disable the optional feature and continue the core path. | The optional result is later treated as authoritative. |
| Async defer | The caller does not need synchronous completion and the system can track/reconcile later. | Return an explicit accepted/deferred result with durable tracking and retry/recovery policy. | The caller expects completed side effects or immediate consistency. |

## Accepted Examples
- Payment authorization: `critical_fail_closed`; on timeout or breaker-open, do not record a paid order, return a retryable service failure, and emit a dependency-unavailable signal owned by payments.
- Recommendation lookup: `optional_fail_open`; omit recommendations, keep product detail rendering, record the degradation mode, and exit that mode after the dependency meets the recovery check.
- Fraud score for manual-review-capable order: `critical_fail_degraded`; accept the order into `pending_review`, do not ship, and require reconciliation before fulfillment.

## Rejected Examples
- "If auth is down, use the last successful authorization decision." Rejected because stale authorization can violate access-control invariants unless a bounded, audited emergency policy already exists.
- "Retry the dependency and continue if it still fails." Rejected because it lacks a criticality class, fallback result, caller contract, and recovery owner.
- "Return success and enqueue the work" for a synchronous state-changing operation. Rejected unless the API contract already promises an async accepted/deferred result with tracking.

## Testable Failure Contracts
- Given a `critical_fail_closed` dependency times out, the operation returns the specified failure response and no externally visible state mutation occurs.
- Given an `optional_fail_open` dependency fails, the core response completes without optional data and emits the named degradation signal.
- Given a degraded fallback is used, the result includes or records its fallback mode, max staleness/defer window, and recovery owner.
- Given dependency ownership is absent, the spec remains blocked rather than accepting an ownerless failure mode.

## Exa Source Links
- Google SRE, Handling Overload: https://sre.google/sre-book/handling-overload/
- Google SRE, Addressing Cascading Failures: https://sre.google/sre-book/addressing-cascading-failures/
- Microsoft Azure, Design for self-healing: https://learn.microsoft.com/en-us/azure/architecture/guide/design-principles/self-healing
- Microsoft Azure, Bulkhead pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/bulkhead
- Microsoft Azure, Circuit Breaker pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker

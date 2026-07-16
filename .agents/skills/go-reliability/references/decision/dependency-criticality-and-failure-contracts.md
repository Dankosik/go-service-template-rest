# Dependency Criticality And Failure Contracts

## Behavior Change Thesis
When loaded for dependency failure classification, this file makes the model choose an explicit criticality, fallback, caller signal, and recovery owner instead of likely mistake "retry the dependency or degrade gracefully" without saying what correctness allows.

## When To Load
Load when a reliability spec mentions dependency failure, fallback, fail-open/fail-closed behavior, or an ownerless recovery path.

## Decision Rubric
- First name the protected flow or invariant. If that cannot be named, dependency policy is under-specified.
- Choose `critical_fail_closed` when continuing could violate money, authorization, privacy, inventory, idempotency, or another hard invariant.
- Choose `critical_fail_degraded` only when the degraded state prevents irreversible harm and names max staleness, lost capability, reconciliation, and user/operator signal.
- Choose `optional_fail_open` only when omitted optional data is never later treated as authoritative.
- Choose async defer only when the caller contract says accepted/deferred, durable tracking exists, and retry/recovery ownership is named.
- Treat missing owner, missing signal, or fabricated success as a spec blocker, not as an implementation detail.

## Imitate
- Payment authorization: `critical_fail_closed`; on timeout or breaker-open, do not record a paid order, return the selected retryable failure, and emit a dependency-unavailable signal owned by payments.
- Recommendation lookup: `optional_fail_open`; omit recommendations, keep product detail rendering, record the degradation mode, and exit only after the dependency meets the recovery check.
- Fraud score for a manual-review-capable order: `critical_fail_degraded`; accept into `pending_review`, do not ship, and require reconciliation before fulfillment.

## Reject
- "If auth is down, use the last successful authorization decision." This makes stale authorization authoritative unless a bounded, audited emergency policy already exists.
- "Retry the dependency and continue if it still fails." This lacks a criticality class, fallback result, caller contract, and recovery owner.
- "Return success and enqueue the work" for a synchronous state-changing operation. This is fake success unless the API already promises accepted/deferred completion with tracking.

## Agent Traps
- Do not call a fallback "safe" until the affected invariant is named.
- Do not let "optional" data become a hidden input to a later authoritative decision.
- Do not assign recovery to "ops" or "the system"; name the owning component, team, or runbook surface the spec can depend on.

## Validation Shape
- Given `critical_fail_closed` timeout, the operation returns the specified failure and no externally visible state mutation occurs.
- Given `optional_fail_open` failure, the core response completes without optional data and emits the named degradation signal.
- Given degraded fallback activation, the result records fallback mode, max staleness or defer window, and recovery owner.
- Given no dependency owner exists, the spec remains blocked instead of accepting an ownerless failure mode.

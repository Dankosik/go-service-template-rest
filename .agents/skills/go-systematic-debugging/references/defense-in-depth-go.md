# Defense-In-Depth After Root-Cause Fix

## Behavior Change Thesis
When loaded after a root-cause fix, this file makes the model add only the owning-layer guardrail justified by the failure mode instead of broad hardening, retries, metrics, or redesign.

## When To Load
Load after the root cause is proven and the local fix is clear, but recurrence prevention, diagnostics, or an additional guardrail is still being considered.

## Decision Rubric
- Fix the earliest valid boundary first; guardrails come after root cause, not instead of it.
- Add a guardrail only when it blocks the same defect class or materially improves future triage.
- Put the guardrail at the layer that owns the invariant: transport, application/domain, infrastructure adapter, or diagnostics.
- Reject plausible guardrails explicitly when they add complexity without blocking recurrence.
- Escalate if the guardrail changes API, timeout, retry, durability, security, data, or rollout semantics.

## Imitate

```go
// Transport boundary: reject impossible request shape.
if req.AccountID == "" {
	return problem.BadRequest("account_id is required")
}

// Domain/application invariant: reject invalid state transition.
if amount <= 0 {
	return ErrInvalidAmount
}

// Infrastructure safety: preserve caller context and bound adapter work.
ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
defer cancel()
if err := repo.Save(ctx, entity); err != nil {
	return fmt.Errorf("save payment: %w", err)
}
```

Copy the ownership: each guardrail lives at the boundary that can enforce the specific invariant.

```text
Rejected guardrail: package-wide nil checks in handlers.
Reason: the proven defect was repository missing-row mapping; handler checks would hide the contract violation and add no new detection.
```

Copy the explicit rejection when hardening sounds plausible but does not block the recurrence.

## Reject

```text
Also added retries, extra validation, metrics, and a cache fallback while fixing the nil panic.
```

This expands a local root-cause fix into unrelated policy and observability changes.

```go
if result == nil {
	return nil
}
```

This may hide the bad state instead of adding a guardrail at the owner of the invariant.

## Agent Traps
- Treating "defense in depth" as permission for unrelated refactors.
- Adding diagnostics with secrets, PII, or unbounded-cardinality fields.
- Keeping a guardrail only because it was useful during debugging.
- Forgetting to prove the original failure separately from the new guardrail path.
- Changing retries or timeouts as a guardrail without reliability/design approval.

## Validation Shape
Record the proven root cause, first broken invariant, guardrail layer, why the guardrail blocks recurrence, RED/GREEN proof for the original failure, and a separate command for any new guardrail behavior.

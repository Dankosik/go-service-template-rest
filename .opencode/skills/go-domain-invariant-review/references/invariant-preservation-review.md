# Invariant Preservation Review

## Behavior Change Thesis
When loaded for symptom "mutation, construction, or save path may bypass a required business rule", this file makes the model prove an accepted invalid state instead of likely mistake "ask for generic aggregate/value-object reshaping without a local invariant."

## When To Load
Load this when a review touches constructors, factories, mutators, aggregate methods, service guards, repository saves, direct field mutation, value-object creation, totals, limits, quotas, eligibility, ownership, or another rule that must remain true after every accepted operation.

## Decision Rubric
- Report a finding only when the diff can accept, persist, or expose state the local contract forbids.
- Name the invariant in repo terms before naming the implementation pattern.
- Prefer a local guard, constructor use, or mutation-order fix when ownership is already clear.
- Escalate when the invariant is absent, contradictory, split across owners, or needs a new consistency boundary.

## Imitate
```text
[critical] [go-domain-invariant-review] internal/orders/service.go:72
Issue:
Approved behavior says submitted orders are price-locked, but `AddLine` appends a line and recalculates `Total` before checking `order.Status`. A caller can persist a changed total on an already submitted order.
Impact:
Customer-visible and billing totals can diverge after checkout commitment, creating undercharge, overcharge, or fulfillment disputes for an order that should be immutable.
Suggested fix:
Check `order.Status == StatusDraft` before mutating `Lines` or `Total`, or route the update through the existing domain method that already enforces the draft-only rule.
Reference:
Local order lifecycle spec or tests for the price-lock invariant.
```

Copy the shape: local invariant, bypass path, accepted bad state, smallest owner-preserving fix.

## Reject
```text
[medium] internal/orders/service.go:72
This looks like an anemic domain model. Move the logic into the aggregate.
```

Failure: this redesigns shape without proving a bad state or citing local business authority.

## Agent Traps
- Do not flag public fields, setters, or service-layer checks by themselves when no invariant bypass is shown.
- Do not invent rules such as "orders can never change after creation" from general domain modeling taste.
- Do not call a style concern an invariant finding when the bad state cannot be accepted, persisted, or observed.
- Do not accept "repair later" for a hard invariant unless a local process contract explicitly defines the repair path.

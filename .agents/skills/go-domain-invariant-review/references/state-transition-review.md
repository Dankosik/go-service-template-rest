# State Transition Review

## Behavior Change Thesis
When loaded for symptom "lifecycle movement or terminal-state handling changed", this file makes the model check approved legal transitions instead of likely mistake "demand a formal state machine or transition table without showing an illegal move."

## When To Load
Load this when a review touches status enums, transition tables, lifecycle methods, state guards, terminal states, event consumers, process managers, handler-level status updates, or any code that changes what state can follow another state.

## Decision Rubric
- Treat state names as business language when local specs, tests, fixtures, or callers give them product meaning.
- Report a finding when the diff permits a forbidden transition, blocks an approved one, weakens terminal-state behavior, or hides a state change behind a side effect or retry path.
- Preserve approved no-op, rejection, and idempotent-success semantics exactly; do not normalize them from taste.
- Escalate when the fix needs a new lifecycle state, public transition contract, process manager, saga, or reconciliation model.

## Imitate
```text
[high] [go-domain-invariant-review] internal/subscription/service.go:118
Issue:
Approved behavior only allows `cancelled -> active` reactivation after billing has resumed, but `Reactivate` writes `StatusActive` before checking `CancelledAt` or confirming the billing resume. Non-cancelled or billing-failed subscriptions can become active.
Impact:
Customers can receive active entitlement without a valid reactivation path or resumed billing, and support will see an `active` state that contradicts the subscription lifecycle.
Suggested fix:
Validate the current state and resume billing before saving `StatusActive`, or use the existing pending reactivation state if the local lifecycle already defines one.
Reference:
Local subscription lifecycle tests or approved state diagram.
```

Copy the shape: current state, attempted transition, violated transition rule, observable wrong lifecycle outcome.

## Reject
```text
[medium] internal/subscription/service.go:118
The transition logic is not a real state machine.
```

Failure: this asks for design form rather than proving a forbidden or missing transition.

## Agent Traps
- Do not require a transition table when a small local guard already makes legal moves clear.
- Do not flag "too many states" or "not enough states" without a broken local transition rule.
- Do not assume every repeated event is an error; duplicates may be no-ops when the local contract says so.
- Do not conflate technical states such as `retrying` or `published` with business lifecycle states unless product semantics depend on them.

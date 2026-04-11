# State Transition Review Examples

## When To Load
Load this when a review touches status enums, transition tables, lifecycle methods, state guards, terminal states, event consumers, process managers, handler-level status updates, or any code that changes what state can follow another state.

Use local specs, state diagrams, tests, fixtures, and accepted task decisions as the transition authority. External state-machine and aggregate sources help sharpen questions, but they do not define this repo's allowed transitions.

## Review Lens
Treat state names as business language. A transition bug exists when changed code permits a forbidden move, blocks an approved move, treats a terminal state as non-terminal, changes a no-op into success or failure without authority, or hides a transition behind a side effect or retry path.

## Bad Finding Example
```text
[medium] internal/subscription/service.go:118
The transition logic is not a real state machine.
```

Why it fails: it asks for a design rewrite without proving an illegal transition or business impact.

## Good Finding Example
```text
[high] [go-domain-invariant-review] internal/subscription/service.go:118
Issue:
Approved behavior only allows `cancelled -> active` reactivation after billing has resumed, but `Reactivate` writes `StatusActive` before checking `CancelledAt` or confirming the billing resume. Non-cancelled or billing-failed subscriptions can become active.
Impact:
Customers can receive active entitlement without a valid reactivation path or resumed billing, and support will see an `active` state that contradicts the subscription lifecycle.
Suggested fix:
Validate the current state and resume billing before saving `StatusActive`, or introduce the existing pending/reactivation state if the local lifecycle already defines one.
Reference:
Local subscription lifecycle tests or spec; external state-machine guidance is calibration only.
```

## Non-Findings To Avoid
- Do not require a transition table when a small guard or method already makes legal moves clear.
- Do not flag "too many states" or "not enough states" without a local transition rule being broken.
- Do not assume every repeated event is an error; duplicates may be no-ops when the local contract says so.
- Do not conflate technical states such as `retrying` or `published` with business lifecycle states unless the product semantics depend on them.

## Smallest Safe Correction
Prefer the narrowest change that restores legal movement:
- add or restore a guard on the affected transition;
- move state assignment after preconditions and required confirmations;
- preserve terminal-state rejection or no-op behavior exactly as approved;
- centralize duplicate transition checks only when an existing local owner already exists;
- add a stale-version or current-state check at the consumer path that performs the state change.

## Escalation Cases
Escalate when:
- allowed transitions are missing or contradictory across specs, tests, and code;
- the fix needs a new lifecycle state or terminal-state policy;
- a long-running workflow needs a process manager, saga, or reconciliation model;
- changing transition behavior would alter public API or event semantics;
- several services or modules claim transition authority for the same business entity.

## Source Links From Exa
- [Microsoft Learn: Use tactical DDD to design microservices](https://learn.microsoft.com/en-us/azure/architecture/microservices/model/tactical-domain-driven-design)
- [NILUS: State Machines in Microservices Workflows](https://www.nilus.be/blog/state_machines_in_microservices_workflows/)
- [NILUS: Event Ordering Tradeoffs in Event Streaming](https://www.nilus.be/blog/event_ordering_tradeoffs_in_event_streaming/)
- [Martin Fowler: Domain Model](https://martinfowler.com/eaaCatalog/domainModel.html)

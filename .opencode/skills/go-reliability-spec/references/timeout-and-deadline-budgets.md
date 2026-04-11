# Timeout And Deadline Budgets

## Behavior Change Thesis
When loaded for deadline propagation, this file makes the model choose caller-budget-derived per-hop deadlines and bounded async handoff rules instead of likely mistake "set a timeout" with fixed values or `context.Background()`.

## When To Load
Load when the spec needs inbound deadlines, outbound per-hop budgets, context propagation, fail-fast thresholds, async detachment, server timeout policy, or shutdown deadlines.

## Decision Rubric
- Start with the user-visible end-to-end budget. If it is unknown and material, mark it as an assumption or blocker instead of inventing a number.
- For synchronous request work, outbound calls derive from the inbound context and cannot exceed remaining budget.
- Per-hop caps use `min(<per-hop cap>, <remaining inbound budget> - <reserved write/cleanup>)`.
- If remaining budget is below the fail-fast floor, skip new outbound work and return the selected timeout, degraded, or fail-closed contract.
- Detach work only after a durable accepted/deferred handoff; create a new bounded deadline, tracking ID, expiry rule, and reconciliation owner.
- Shutdown budgets must fit inside the platform grace period and cover readiness false, stop accepting work, drain, required flush/record, and hard exit.

## Imitate
- "The request budget is `<end-to-end budget>`; downstream inventory calls use `min(<inventory cap>, remaining request budget - <write/cleanup reserve>)`; below `<fail-fast floor>`, skip the call and return the specified timeout response."
- "The export endpoint returns `202 Accepted` with a tracking reference before background work starts; the worker has `<worker deadline>`, expiry, and reconciliation policy."
- "Shutdown sets readiness false, stops accepting new traffic, drains in-flight HTTP work for `<drain window>`, records drain duration, and exits before platform hard kill."

## Reject
- "Use `context.Background()` for outbound calls from a request handler." This loses request cancellation and deadline propagation unless the spec defines an async handoff first.
- "Set every downstream timeout to 5s." This can exceed caller budget, cause late writes, and amplify slow failures.
- "Let shutdown wait until requests finish." This is unbounded and can exceed the termination window.

## Agent Traps
- Do not mix a per-hop timeout with an end-to-end deadline as if both can be fully spent.
- Do not allow telemetry flush, cleanup, or response write to be squeezed out of the remaining budget.
- Do not detach work just because Go makes it easy; the spec needs accepted/deferred semantics first.

## Validation Shape
- Given caller cancellation, synchronous dependency work observes cancellation and stops before starting new outbound work.
- Given remaining inbound budget below the fail-fast floor, no outbound dependency call is attempted.
- Given a dependency exceeds its per-hop timeout, the selected timeout/degraded/fail-closed contract occurs and no later write is attempted.
- Given shutdown begins, readiness changes before new work is admitted, in-flight work is drained or timed out, and the process exits within the grace window.

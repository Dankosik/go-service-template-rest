# Problem And Behavior Delta

## Load When
Load this when the frame cannot yet say who is affected, what currently happens, what should happen instead, or which downstream design choices remain undecided.

## The Move
Name one affected actor before the mechanism — the precise one: an API client, operator, support agent, tenant admin, or automation, not "users." Separate current behavior from desired behavior as observable facts; "current behavior is missing the feature" hides the delta in a feature name. Preserve the user's concern while stripping unapproved architecture, API, storage, rollout, and task choices, and close with the boundary sentence that tells downstream specialists what must still be decided. When the problem cannot be stated without inventing facts, mark readiness `fail` and ask for the smallest missing fact.

## Inspect
Input:
```text
We should add idempotency keys to POST /orders because retries sometimes create duplicate orders.
```

Output:
```markdown
Problem
Clients that retry order creation after a timeout can create more than one order for the same user intent, leaving operators and customers with duplicate state to reconcile.

Behavior Delta
- Current behavior: A retry of the same order-create intent is treated like a new create request.
- Desired behavior: The service can recognize repeat submissions of the same create intent and return a stable outcome instead of creating a duplicate.
- Affected actor: API clients and downstream order operators.
- Boundary: This frame covers duplicate create-intent semantics only. It does not choose the idempotency-key header contract, storage design, transaction strategy, or retry policy.
```

Copy: the mechanism is preserved as context but not approved as the frame.

## Reject
```markdown
Problem
Exports need a better experience.
```

A slogan cannot route API, data, reliability, or UX decisions.

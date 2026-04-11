# Problem And Behavior Delta

Behavior Change Thesis: When loaded for the symptom "the request is a proposed mechanism, slogan, or vague improvement," this file makes the model choose an actor, current behavior, desired behavior, and design boundary instead of the likely mistake of restating the proposed implementation as the problem.

## When To Load
Load this when the frame cannot yet say who is affected, what currently happens, what should happen instead, or which downstream design choices remain undecided.

## Decision Rubric
- Name one affected actor before naming the mechanism.
- Separate current behavior from desired behavior; do not hide the delta in a feature name.
- Preserve the user's concern, but strip out unapproved architecture, API, storage, rollout, or task choices.
- The boundary sentence matters: it tells downstream specialists what must still be decided.
- If the problem cannot be stated without inventing facts, mark readiness `fail` and ask for the smallest missing fact.

## Imitate
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

Input:
```text
Make the health endpoint more useful for deploys. It should tell us if dependencies are okay.
```

Output:
```markdown
Problem
Deployment automation and operators cannot tell whether the service is ready to receive traffic or merely process-alive, so deploy gates may advance with hidden dependency failure.

Behavior Delta
- Current behavior: Health reporting is too coarse to distinguish process liveness from traffic readiness.
- Desired behavior: The service exposes enough readiness signal for deploy automation to avoid routing traffic to an instance that cannot serve its expected dependencies.
- Affected actor: deploy automation and on-call operators.
- Boundary: This frame names the readiness problem only. It does not decide probe routes, dependency checks, failure thresholds, caching, or Kubernetes configuration.
```

Copy: the output distinguishes traffic readiness from liveness without designing the probe.

Input:
```text
Users keep asking why their export is stuck. Add better export status.
```

Output:
```markdown
Problem
Users cannot distinguish an export that is still progressing from one that is blocked or failed, so they retry or ask support before the system has a clear next action.

Behavior Delta
- Current behavior: Export progress is opaque after the request is accepted.
- Desired behavior: Users and support can see a clear export lifecycle state and know whether to wait, retry, or investigate.
- Affected actor: export requesters and support operators.
- Boundary: This frame covers status transparency. It does not decide event storage, polling vs push, background worker design, or UI layout.
```

Copy: the behavior delta is user-visible and testable, not a UI or worker design.

## Reject
Bad:
```markdown
Problem
Implement idempotency keys for POST /orders.
```

Why: this turns the requested mechanism into the problem and leaves actor, duplicate semantics, and downstream decisions implicit.

Bad:
```markdown
Problem
Exports need a better experience.
```

Why: this is a slogan. It cannot route API, data, reliability, or UX decisions.

## Agent Traps
- Do not say "current behavior is missing the feature"; name the observable behavior gap.
- Do not approve a header, endpoint, cache, table, worker, or UI choice in the boundary.
- Do not use "users" if the real affected actor is an API client, operator, support agent, tenant admin, or automation.

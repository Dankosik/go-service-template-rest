# Scope Constraints And Non-Goals

## Load When
Load this when the request bundles adjacent ideas, contains vague words like "fast" or "compliant," or risks treating downstream implementation and rollout details as already approved.

## The Move
Keep only behavior framing in scope and route architecture, API, data, security, and reliability details downstream. Write non-goals as the plausible adjacent work a model might otherwise include — each one means "not approved for this pass," not a forbidden future, and a short pointed list beats a thoroughness display. Keep a constraint only when it changes downstream choices, stated as the requirement rather than a hidden decision ("GDPR" surfaces the retention conflict; it does not mean "hard delete immediately"). When "fast," "simple," "compliant," or "secure" lacks an acceptance threshold or policy owner, make it a question, and admit a delivery commitment only when it materially constrains the frame and has an owner.

## Inspect
Input:
```text
Add tenant-aware export limits, plus maybe cache export files and show admin analytics while we are in there.
```

Output:
```markdown
Scope
In scope:
- Frame the tenant-aware export-limit behavior and how users or operators should experience limit enforcement.
- Identify which tenant/account boundaries matter to the limit decision.
- Capture the success signal: repeatable enforcement that prevents one tenant from exhausting export capacity for others.

Non-Goals
- No export-file cache design.
- No admin analytics feature.
- No storage or queue implementation choice.
- No specific rate-limit algorithm.

Constraints
- Tenant isolation is a hard constraint.
- The framing must preserve fairness across tenants without deciding the enforcement mechanism.
- Existing export semantics must not be redefined unless later specification explicitly approves it.
```

Copy: the frame protects the tenant fairness problem while rejecting cache and analytics creep.

## Reject
```markdown
Constraints
It must be secure, scalable, compliant, and fast.
```

Generic wishes constrain nothing until tied to a policy, actor, threshold, or downstream design consequence.

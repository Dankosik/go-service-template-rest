# Scope Constraints And Non-Goals

Behavior Change Thesis: When loaded for the symptom "scope is expanding or constraints are mixed with delivery promises," this file makes the model choose explicit in-scope work, non-goals, and design-shaping constraints instead of the likely mistake of carrying adjacent features or implementation commitments into the handoff.

## When To Load
Load this when the request bundles adjacent ideas, contains vague words like "fast" or "compliant," or risks treating downstream implementation and rollout details as already approved.

## Decision Rubric
- Put only behavior framing in scope; route architecture/API/data/security/reliability details downstream.
- Use non-goals to exclude plausible adjacent work, not to repeat obvious absences.
- Constraints must change downstream choices. If a constraint does not affect design, omit it or demote it to context.
- If a word like "fast," "simple," "compliant," or "secure" lacks an acceptance threshold or policy owner, make it a question.
- Do not turn a non-goal into a forbidden future. It only says "not approved for this pass."

## Imitate
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

Input:
```text
We need account deletion to be GDPR-compliant. Delete all user data everywhere and make it fast.
```

Output:
```markdown
Scope
In scope:
- Frame the account-deletion behavior from user request through externally visible completion or rejection.
- Identify data-retention and audit constraints that could prevent immediate deletion.
- Separate user-visible deletion semantics from internal cleanup mechanics.

Non-Goals
- No final legal policy decision.
- No database deletion strategy.
- No cross-service saga design.
- No performance SLO selection.

Constraints
- Privacy and retention requirements may conflict and must be surfaced as planning-critical questions.
- "Fast" is not a usable requirement until a stakeholder defines the acceptance threshold.
```

Copy: legal policy, retention, and performance are exposed as constraints/questions, not invented decisions.

Input:
```text
Can we make background jobs observable and maybe also rewrite the worker pool?
```

Output:
```markdown
Scope
In scope:
- Frame the operator problem: background job failures are hard to diagnose and correlate with user-visible outcomes.
- Identify which job lifecycle states or failure classes must be visible to support later design.

Non-Goals
- No worker-pool rewrite.
- No tracing library choice.
- No metric names or log schema.
- No queue replacement.

Constraints
- The frame must preserve existing job semantics unless later specification approves behavior change.
- Observability signals should be tied to operator questions, not generic telemetry expansion.
```

Copy: observability scope is tied to operator questions; worker-pool redesign is excluded.

## Reject
Bad:
```markdown
Scope
Build tenant-aware limits, cached export files, and admin analytics.
```

Why: it accepts "while we are in there" as approval and loses the first behavior problem.

Bad:
```markdown
Constraints
It must be secure, scalable, compliant, and fast.
```

Why: these are generic wishes until tied to a policy, actor, threshold, or downstream design consequence.

## Agent Traps
- Do not use a huge "out of scope" list to look thorough. Name only adjacent work a model might otherwise include.
- Do not turn constraints into hidden decisions, such as "GDPR means hard delete immediately."
- Do not include delivery commitments like "two-week rollout" unless they materially constrain the frame and have an owner.

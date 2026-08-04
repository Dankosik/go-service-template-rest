# Readiness Decision Examples

## Behavior Change Thesis
When loaded for the symptom "the pass/fail readiness decision is close, blocked, or drifting into ready-enough language," this file makes the model emit a decisive readiness outcome with next handoff or minimum missing data instead of approving a vague frame.

## When To Load
Load this when the pass/fail readiness decision is close, blocked, or drifting into a generic "ready enough" statement.

## The Move
`pass` means the frame can move into challenge/spec/design without reopening the basic actor, behavior delta, scope, constraints, and known unknowns; it may carry accepted risks only as tracked downstream blockers with owner and unblock condition, and it approves no architecture, API, data, security, reliability, rollout, or task decision. A caveat that can change the basic problem or scope makes the decision `fail`, and `fail` names the minimum missing discriminator rather than a discovery questionnaire — the test is whether the next session would have to invent material framing facts. Even low-risk work is ready only when the handoff says what the next specialist should challenge or design.

## Imitate
Input:
```text
Make uploads faster.
```

Output:
```markdown
Readiness Decision
fail

Why
"Make uploads faster" does not state the affected actor, upload type, current latency, desired threshold, dominant pain, or whether the problem is throughput, tail latency, reliability, UX feedback, or backend capacity.

Minimum Data Needed
- Which upload workflow is affected.
- Current observed latency or failure mode.
- Desired user or operator outcome.
- Whether the performance issue blocks product behavior, operational safety, or user satisfaction.
```

Copy: the fail asks for discriminators, not a full spec.

Input:
```text
Add soft delete for projects so accidental deletes can be recovered.
```

Output:
```markdown
Readiness Decision
pass

Why
The frame can pass if it preserves the recovery problem and explicitly leaves delete-state modeling, API contract, retention, authorization, and cleanup policy for downstream design.

Accepted Risks
- Existing delete behavior and restore authorization remain open questions, but they are tracked as design blockers with owners and unblock conditions.

Next Handoff
Run pre-spec challenge because delete semantics are stateful and may hide authorization, retention, and consistency risks.
```

Copy: a pass can carry unresolved specialist questions only when they are explicitly owned.

## Reject
```markdown
Readiness Decision
pass - Implement soft delete with deleted_at and a restore endpoint.
```

It approves an implementation path, not frame readiness.

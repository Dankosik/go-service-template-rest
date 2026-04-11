# Readiness Decision Examples

Behavior Change Thesis: When loaded for the symptom "the pass/fail readiness decision is close, blocked, or drifting into ready-enough language," this file makes the model emit a decisive readiness outcome with next handoff or minimum missing data instead of the likely mistake of approving a vague frame.

## When To Load
Load this when the pass/fail readiness decision is close, blocked, or drifting into a generic "ready enough" statement.

## Decision Rubric
- `pass` means the problem frame can move into challenge/spec/design without reopening the basic actor, behavior delta, scope, constraints, and known unknowns.
- `fail` means the next session would have to invent material framing facts.
- A pass may include accepted risks only if they are tracked as downstream blockers with owner and unblock condition.
- A fail must name the minimum data needed, not a broad questionnaire.
- Readiness does not approve architecture, API, data, security, reliability, rollout, or implementation tasks.

## Imitate
Input:
```text
Expose audit events to admins so they can investigate suspicious account changes.
```

Output:
```markdown
Readiness Decision
pass

Why
The frame names the operator, current gap, desired investigation outcome, high-risk seams, and open questions that specialist lanes can own. It is not yet an API or storage design, but it is specific enough for pre-spec challenge.

Next Handoff
Run pre-spec challenge with emphasis on authorization boundaries, tenant visibility, event completeness, retention expectations, and support-safe redaction.
```

Copy: the pass is not design approval; it routes the risky seams.

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
Bad:
```markdown
Readiness Decision
pass - Looks clear enough.
```

Why: the decision has no proof of actor, behavior, scope, unknowns, or handoff readiness.

Bad:
```markdown
Readiness Decision
pass - Implement soft delete with deleted_at and a restore endpoint.
```

Why: it approves an implementation path, not frame readiness.

## Agent Traps
- Do not use `pass with caveats` if the caveat can change the basic problem or scope. That is a fail.
- Do not fail with a giant discovery questionnaire. Ask for the smallest missing discriminator.
- Do not mark low-risk work ready if the handoff still cannot say what the next specialist should challenge or design.

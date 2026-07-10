# Workflow Plan Update Examples

## Behavior Change Thesis
When loaded for master `workflow-plan.md` updates, this file makes the model record cross-phase planning state and handoff facts in the master artifact instead of leaving them in chat or only in `workflow-plans/planning.md`.

## When To Load
Load when repairing or writing the master `workflow-plan.md` planning status, artifact status, pending task-review/readiness handoff, adequacy challenge packet status, blockers, or next-session handoff.

## Decision Rubric
- Keep `workflow-plan.md` cross-phase: status, artifact inventory, blockers, pending task-review/readiness state, challenge packet state, boundary, and next session.
- Do not copy `tasks.md`, `spec.md`, or design details into the master file.
- Master and `workflow-plans/planning.md` must agree on canonical `phase_state=not_started|active|complete|blocked|reopened`; lifecycle, routing validity, session boundary, and handoff readiness remain separate fields.
- Record a `Next session context bundle` every time: either say default resume order is sufficient or list the task-specific file bundle for the next implementation or reopen session.
- If planning is blocked, `Next session starts with` points to the reopen target, not a coding task.
- When formal adequacy is triggered, its typed gate state must say whether blocking findings were reconciled or still block handoff; when no `ADEQUACY-*` condition is true, record the local deterministic matrix audit instead of a waiver.

## Imitate
```markdown
Current phase: planning
phase_state: complete
session_boundary: reached
handoff_readiness: ready
Next session starts with: task-review/readiness
Next session context bundle: `spec.md` for decisions; `design/overview.md` and required design maps for technical constraints; `tasks.md` for ledger review and proof mapping.

Artifact state:
- `spec.md`: artifact_expectation=expected, artifact_state=approved, record_validity=current
- `design/`: artifact_expectation=expected, artifact_state=approved, record_validity=current
- `tasks.md`: artifact_expectation=expected, artifact_state=review_ready, record_validity=current
- `test-plan.md`: artifact_expectation=not_expected, artifact_state=absent, record_validity=current, waiver_disposition=none; the test-design owner trigger resolved to not expected because proof obligations fit directly in `tasks.md`
- `rollout.md`: artifact_expectation=not_expected, artifact_state=absent, waiver_disposition=none; no migration or delivery choreography
- `workflow-plans/planning.md`: artifact_expectation=expected, artifact_state=complete, record_validity=current
- review/validation phase-control files: artifact_expectation=not_expected, artifact_state=absent; `tasks.md` is sufficient for the task-review/readiness packet

Task-review/readiness procedural_gate_state: pending
Task-review/readiness review_verdict: pending
Workflow plan adequacy procedural_gate_state: pending when any ADEQUACY-* condition is true; otherwise local deterministic matrix audit: complete
Blockers: none
```

Copy this shape: it makes the cross-phase state scannable without duplicating the plan.

If the default resume order is enough, still keep the field:

```markdown
Next session context bundle: default resume order is sufficient; no task-specific additions.
```

```markdown
Current phase: planning
phase_state: blocked
session_boundary: reached
handoff_readiness: ready
Next session starts with: go-code-ownership-design
Next session context bundle: `spec.md`, current `design/overview.md`, and the blocked design artifact that owns the missing decision.

Artifact state:
- `tasks.md`: artifact_expectation=expected, artifact_state=blocked, record_validity=current
- `workflow-plans/planning.md`: artifact_expectation=expected, artifact_state=blocked, record_validity=current

Task-review/readiness procedural_gate_state: blocked
Task-review/readiness review_verdict: pending
Blocker: implementation order depends on a missing ownership decision in the design bundle.
phase_state: reopened
reopen_target: go-code-ownership-design
```

Copy this shape: the blocked master update routes upstream instead of implying implementation can start.

## Reject
```markdown
Current phase: planning
phase_state: complete
Task-review/readiness review_verdict: <non-pending verdict assigned by planning>
```

Failure: planning is assigning readiness. The next session should normally be `task-review/readiness`.

```markdown
Workflow plan adequacy challenge: done.
```

Failure: it hides whether blocking findings existed and whether they were reconciled.

## Agent Traps
- Letting master and phase-local files contradict each other.
- Recording `tasks.md: approved` before the separate task-review/readiness gate passes.
- Treating `workflow-plan.md` as a full planning document.
- Omitting canonical `session_boundary`, `Next session starts with`, or the always-present `Next session context bundle`, forcing the next agent to infer state from chat.

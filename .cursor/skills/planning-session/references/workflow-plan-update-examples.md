# Workflow Plan Update Examples

## Behavior Change Thesis
When loaded for master `workflow-plan.md` updates, this file makes the model record cross-phase planning state and handoff facts in the master artifact instead of leaving them in chat or only in `workflow-plans/planning.md`.

## When To Load
Load when repairing or writing the master `workflow-plan.md` planning status, artifact status, readiness status, adequacy challenge status, blockers, or next-session handoff.

## Decision Rubric
- Keep `workflow-plan.md` cross-phase: status, artifact inventory, blockers, readiness, challenge state, boundary, and next session.
- Do not copy `tasks.md`, optional `plan.md`, `spec.md`, or design details into the master file.
- Master and `workflow-plans/planning.md` must agree on lifecycle `Phase status` such as `pending`, `in_progress`, `blocked`, or `complete`; use a separate routing state for reopened handoffs.
- Record a `Next session context bundle` every time: either say default resume order is sufficient or list the task-specific file bundle for the next implementation or reopen session.
- If readiness is `FAIL`, `Next session starts with` points to the reopen target, not an implementation phase.
- Adequacy challenge status must say whether blocking findings were reconciled, waived under an eligible rationale, or still block handoff.

## Imitate
```markdown
Current phase: planning
Phase status: complete
Session boundary reached: yes
Ready for next session: yes
Next session starts with: T001
Next session context bundle: `spec.md` for decisions; `design/overview.md` and required design maps for technical constraints; `tasks.md` for T001 entry and proof; `workflow-plans/implementation-phase-1.md` only if named routing created it.

Artifact status:
- `spec.md`: approved
- `design/`: approved
- `tasks.md`: approved
- `plan.md`: not expected; no separate multi-checkpoint strategy
- `test-plan.md`: not expected; proof obligations fit in `tasks.md`
- `rollout.md`: not expected; no migration or delivery choreography
- `workflow-plans/planning.md`: complete
- post-code phase-control files: not expected; `tasks.md` is sufficient for the next implementation session

Implementation readiness: PASS
Workflow plan adequacy challenge: completed; blocking findings reconciled
Blockers: none
```

Copy this shape: it makes the cross-phase state scannable without duplicating the plan.

If the default resume order is enough, still keep the field:

```markdown
Next session context bundle: default resume order is sufficient; no task-specific additions.
```

```markdown
Current phase: planning
Phase status: blocked
Session boundary reached: no
Ready for next session: no
Next session starts with: technical-design
Next session context bundle: `spec.md`, current `design/overview.md`, and the blocked design artifact that owns the missing decision.

Artifact status:
- `tasks.md`: blocked
- `plan.md`: not expected
- `workflow-plans/planning.md`: blocked

Implementation readiness: FAIL
Blocker: implementation order depends on a missing ownership decision in the design bundle.
Reopen target: technical-design
Routing state: reopen technical-design
```

Copy this shape: the blocked master update routes upstream instead of implying implementation can start.

## Reject
```markdown
Current phase: implementation-phase-1
Planning status: complete
Implementation readiness: PASS
```

Failure: the planning session has not stopped yet; the next session starts implementation later.

```markdown
Workflow plan adequacy challenge: done.
```

Failure: it hides whether blocking findings existed and whether they were reconciled.

## Agent Traps
- Letting master and phase-local files contradict each other.
- Recording `tasks.md: approved` while the task ledger is missing or still speculative.
- Treating `workflow-plan.md` as a full planning document.
- Omitting `Session boundary reached`, `Next session starts with`, or the always-present `Next session context bundle`, forcing the next agent to infer state from chat.

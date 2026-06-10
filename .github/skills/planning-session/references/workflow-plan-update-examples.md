# Workflow Plan Update Examples

## Behavior Change Thesis
When loaded for master `workflow-plan.md` updates, this file makes the model record cross-phase planning state and handoff facts in the master artifact instead of leaving them in chat or only in `workflow-plans/planning.md`.

## When To Load
Load when repairing or writing the master `workflow-plan.md` planning status, artifact status, pending task-review/readiness handoff, adequacy challenge packet status, blockers, or next-session handoff.

## Decision Rubric
- Keep `workflow-plan.md` cross-phase: status, artifact inventory, blockers, pending task-review/readiness state, challenge packet state, boundary, and next session.
- Do not copy `tasks.md`, `spec.md`, or design details into the master file.
- Master and `workflow-plans/planning.md` must agree on lifecycle `Phase status` such as `pending`, `in_progress`, `blocked`, or `complete`; use a separate routing state for reopened handoffs.
- Record a `Next session context bundle` every time: either say default resume order is sufficient or list the task-specific file bundle for the next implementation or reopen session.
- If planning is blocked, `Next session starts with` points to the reopen target, not a coding task.
- Adequacy challenge status must say whether blocking findings were reconciled, waived under an eligible rationale, or still block handoff.

## Imitate
```markdown
Current phase: planning
Phase status: complete
Session boundary reached: yes
Ready for next session: yes
Next session starts with: task-review/readiness
Next session context bundle: `spec.md` for decisions; `design/overview.md` and required design maps for technical constraints; `tasks.md` for ledger review and proof mapping.

Artifact status:
- `spec.md`: approved
- `design/`: approved
- `tasks.md`: draft_review_ready
- `test-plan.md`: not expected; proof obligations fit in `tasks.md`
- `rollout.md`: not expected; no migration or delivery choreography
- `workflow-plans/planning.md`: complete
- review/validation phase-control files: not expected; `tasks.md` is sufficient for the task-review/readiness packet

Task ledger review: pending_task_review
Implementation readiness: pending_task_review
Workflow plan adequacy challenge packet: ready for review, or not expected with rationale
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
Next session starts with: go-code-ownership-design
Next session context bundle: `spec.md`, current `design/overview.md`, and the blocked design artifact that owns the missing decision.

Artifact status:
- `tasks.md`: blocked
- `workflow-plans/planning.md`: blocked

Task ledger review: pending_task_review
Implementation readiness: pending_task_review
Blocker: implementation order depends on a missing ownership decision in the design bundle.
Reopen target: go-code-ownership-design
Routing state: reopen go-code-ownership-design
```

Copy this shape: the blocked master update routes upstream instead of implying implementation can start.

## Reject
```markdown
Current phase: planning
Planning status: complete
Implementation readiness: <non-pending verdict>
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
- Omitting `Session boundary reached`, `Next session starts with`, or the always-present `Next session context bundle`, forcing the next agent to infer state from chat.

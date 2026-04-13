# Phase Control File Examples

## Behavior Change Thesis
When loaded for phase-control file work, this file makes the model create only routing-focused files for named phases instead of creating just-in-case controls or duplicating `tasks.md`, optional `plan.md`, `spec.md`, or `design/`.

## When To Load
Load when writing or repairing `workflow-plans/planning.md`, or when planning must create future implementation, review, or validation phase-control files for named multi-session routing.

## Decision Rubric
- `workflow-plans/planning.md` records phase-local orchestration: status, outputs, blockers, readiness, adequacy challenge, stop rule, next action.
- Future phase-control files are allowed only when named multi-session routing requires them before implementation starts.
- Future files are pending routing skeletons, not execution logs and not new decision records.
- If a future phase-control file needs design facts that do not exist, block planning and reopen upstream instead of filling the gap.
- Put executable tasks and the implementation handoff in `tasks.md`, optional supplemental strategy in `plan.md` only when justified, test depth in triggered `test-plan.md`, and rollout choreography in triggered `rollout.md`.

## Imitate
```markdown
Phase: planning
Phase status: complete
Completion marker: `tasks.md` approved; readiness gate recorded.
Allowed writes used: `tasks.md`, `workflow-plan.md`, `workflow-plans/planning.md`.
Implementation readiness: PASS.
Workflow plan adequacy challenge: completed; blocking findings reconciled.
Stop rule: do not begin implementation in this session.
Next action: start T001 in a later session.
```

Copy this shape: the planning file stays phase-local and handoff oriented.

```markdown
Phase: implementation-phase-1
Phase status: pending
Consumes: `spec.md`, `design/`, `tasks.md`, and optional planning artifacts named in `tasks.md`.
Entry condition: implementation readiness remains PASS or eligible CONCERNS from planning.
Allowed future writes: code/test/config artifacts named by `tasks.md`, plus existing control/progress artifact updates.
Stop rule: do not create new workflow/process artifacts; stop and reopen planning or technical design if required context is missing.
Next action: implement T001 through the checkpoint named in `tasks.md`.
```

Copy this shape: a future phase skeleton gives routing constraints without pretending work already happened.

## Reject
```markdown
Phase: validation-phase-1
Phase status: pending
Created because validation is usually useful.
```

Failure: later phase files need an approved named phase, not a generic habit.

```markdown
Phase: implementation-phase-1
Decision: use polling because it seems simpler than the design's async path.
```

Failure: phase-control files do not make new technical decisions.

## Agent Traps
- Turning `workflow-plans/planning.md` into a duplicate of `tasks.md` or optional `plan.md`.
- Creating review or validation phase files "for completeness."
- Writing an implementation phase file that describes code already changed during planning.
- Creating a missing phase-control file during implementation instead of reopening planning.

# Planning Session Readiness Examples

## When To Load
Load this reference when deciding whether a `planning-session` may begin, when checking required inputs, or when a planning attempt exposes missing context before `plan.md` or `tasks.md` writes.

This file gives examples only. `AGENTS.md` and `docs/spec-first-workflow.md` remain authoritative.

## Good Session Outcomes
- The session confirms task-local `workflow-plan.md` points to `planning`, `workflow-plans/planning.md` is active or repairable, `spec.md` is approved, and required `design/` artifacts are approved before planning work begins.
- The session identifies triggered conditional design artifacts before task breakdown, such as `design/data-model.md` or `design/contracts/`, and confirms they exist when they affect sequencing, validation, or rollout.
- The session treats an explicit recorded design-skip rationale as a narrow exception only for eligible tiny or direct-path work.
- The session produces or repairs `plan.md` and expected `tasks.md` from approved `spec.md + design/`, then records implementation readiness and stops.

## Bad Session Outcomes
- Planning starts from `spec.md` alone for non-trivial work after the design-bundle stage is expected.
- The session invents an ownership, data, API, or rollout answer inside `tasks.md` because a design artifact is missing.
- The session writes implementation files after sketching the first task.
- The session treats a missing `workflow-plans/planning.md` as harmless chat memory instead of repairing the phase-control file during planning.

## Example Handoff Notes
Good handoff note:

```markdown
Planning status: complete.
Inputs: `spec.md` approved; core `design/` approved; no triggered conditional design artifacts are missing.
Outputs: `plan.md` approved; `tasks.md` approved; `test-plan.md` not expected; `rollout.md` not expected.
Implementation readiness: PASS.
Next session starts with: `implementation-phase-1`.
Stop rule: do not begin implementation in this planning session.
```

Concern handoff note:

```markdown
Planning status: complete with CONCERNS.
Accepted risk: one performance budget is estimated from current design evidence.
Proof obligation: first implementation phase must include a benchmark or trace-backed measurement before expanding scope.
Next session starts with: `implementation-phase-1`.
Stop rule: implementation starts in a later session only.
```

## Blocker Handling
- If `spec.md` is missing, draft, or has a planning-critical open question, mark planning blocked, record the upstream blocker in `workflow-plan.md` and `workflow-plans/planning.md`, and stop.
- If required core design artifacts are missing without an eligible design-skip rationale, mark planning blocked with reopen target `technical design`; do not create or edit `design/` here.
- If task breakdown requires a missing rollout, data, security, reliability, or ownership decision, record the blocker and stop instead of hiding the gap in optimistic task wording.
- If an expected `tasks.md` is missing after implementation has already started, planning should be reopened in a new planning session rather than invented in a post-code phase.

## Exa Calibration Source Links
Found through Exa MCP search before these examples were written. Use these links only for calibration; local repo guidance wins.

- arc42 documentation: https://arc42.org/documentation/
- arc42 method: https://arc42.org/method
- Martin Fowler on Architecture Decision Records: https://martinfowler.com/bliki/ArchitectureDecisionRecord.html
- Asana implementation plan guide: https://www.asana.com/resources/implementation-plan


# Phase Control File Examples

## When To Load
Load this reference when planning must write or repair `workflow-plans/planning.md` or create pre-code phase-control files for later implementation, review, or validation phases named by the approved phase structure.

This file gives examples only. `AGENTS.md` and `docs/spec-first-workflow.md` remain authoritative.

## Good Session Outcomes
- `workflow-plans/planning.md` records only planning-phase orchestration, outputs, blocker state, readiness gate result, adequacy challenge status, stop rule, and next action.
- Later phase-control files are created during planning only when `plan.md` names those phases.
- Later phase-control files are placeholders for future sessions, not execution logs for work that has not started.
- Each named implementation phase has a narrow handoff surface and stop rule so the next session does not re-plan from scratch.

## Bad Session Outcomes
- `workflow-plans/planning.md` becomes a second `plan.md` or `tasks.md`.
- `workflow-plans/implementation-phase-1.md` records code already changed during planning.
- `workflow-plans/review-phase-1.md` or `workflow-plans/validation-phase-1.md` is created "just in case" without being named by the approved phase structure.
- A later phase-control file contains new technical decisions that belong in `spec.md` or `design/`.

## Example Handoff Notes
Planning phase-control update:

```markdown
Phase: planning
Phase status: complete
Completion marker: `plan.md` and `tasks.md` approved; readiness gate recorded.
Allowed writes used: `plan.md`, `tasks.md`, `workflow-plan.md`, `workflow-plans/planning.md`, `workflow-plans/implementation-phase-1.md`.
Implementation readiness: PASS.
Workflow plan adequacy challenge: completed; blocking findings reconciled.
Stop rule: do not begin implementation in this session.
Next action: start `implementation-phase-1` in a later session.
```

Implementation phase-control skeleton created during planning:

```markdown
Phase: implementation-phase-1
Phase status: pending
Consumes: `spec.md`, `design/`, `plan.md`, `tasks.md`, optional planning artifacts named in `plan.md`.
Entry condition: implementation readiness remains PASS or eligible CONCERNS from planning.
Allowed future writes: code/test/config artifacts named by `tasks.md`, plus existing control/progress artifact updates.
Stop rule: do not create new workflow/process artifacts; stop and reopen planning or technical design if required context is missing.
Next action: implement T001 through the checkpoint named in `plan.md`.
```

Review phase-control skeleton created during planning:

```markdown
Phase: review-phase-1
Phase status: pending
Consumes: completed implementation checkpoint and existing planning bundle.
Entry condition: implementation-phase-1 is complete.
Allowed future writes: none, unless the orchestrator later records reconciliation work in the proper phase.
Stop rule: review stays read-only and advisory.
Next action: run the review lanes named by `plan.md`.
```

Validation phase-control skeleton created during planning:

```markdown
Phase: validation-phase-1
Phase status: pending
Consumes: implementation and reconciliation results plus existing planning bundle.
Entry condition: implementation/reconciliation checkpoint is complete.
Allowed future writes: existing closeout surfaces only.
Stop rule: do not create new workflow/process artifacts during validation.
Next action: run the proof path named by `plan.md` or `test-plan.md`.
```

## Blocker Handling
- If the approved phase structure does not name a later phase, do not create its phase-control file; record it as not expected.
- If a later phase-control file cannot be made specific enough without a missing design decision, block planning and name the reopen target.
- If review or validation details exceed what belongs in `workflow-plans/<phase>.md`, put strategy in `plan.md` or a triggered `test-plan.md`; keep phase-control files routing-only.
- If implementation has already started and a required phase-control file is missing, record a planning reopen target instead of creating the file mid-implementation.

## Exa Calibration Source Links
Found through Exa MCP search before these examples were written. Use these links only for calibration; local repo guidance wins.

- arc42 documentation: https://arc42.org/documentation/
- arc42 method: https://arc42.org/method
- Martin Fowler on Architecture Decision Records: https://martinfowler.com/bliki/ArchitectureDecisionRecord.html
- Asana implementation plan guide: https://www.asana.com/resources/implementation-plan


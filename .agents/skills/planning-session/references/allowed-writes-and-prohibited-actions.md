# Allowed Writes And Prohibited Actions Examples

## When To Load
Load this reference when the planning session needs concrete examples of what may be written, what must stay untouched, or how to respond when the user asks to combine planning with coding, review, validation, or spec/design changes.

This file gives examples only. `AGENTS.md` and `docs/spec-first-workflow.md` remain authoritative.

## Good Session Outcomes
- The session writes or repairs only planning-phase artifacts: `plan.md`, `tasks.md`, optional `test-plan.md`, optional `rollout.md`, `workflow-plan.md`, `workflow-plans/planning.md`, and pre-code phase-control files already called for by the approved phase structure.
- The session creates `workflow-plans/implementation-phase-N.md`, `workflow-plans/review-phase-N.md`, or `workflow-plans/validation-phase-N.md` only when the approved plan expects those later phases.
- The session records blockers and reopen targets in workflow-control artifacts without editing `spec.md`, `design/`, production code, tests, migrations, generated output, or runtime configuration.
- The session stops after implementation-readiness handoff instead of starting the first implementation task.

## Bad Session Outcomes
- Creating `internal/.../*.go`, `*_test.go`, migrations, generated artifacts, configs, or runtime files during the planning session.
- Editing `spec.md` decisions or writing new `design/` artifacts from inside planning to make task breakdown easier.
- Creating review or validation phase-control files "just in case" when the phase structure does not call for them.
- Recording `Current phase: implementation` in `workflow-plan.md` before the planning session has stopped and the later implementation session has begun.
- Marking `tasks.md` approved while it still contains speculative decisions that should belong to `spec.md` or `design/`.

## Example Handoff Notes
Allowed-write handoff:

```markdown
Planning writes completed:
- `plan.md` approved.
- `tasks.md` approved.
- `workflow-plan.md` updated with implementation-readiness PASS.
- `workflow-plans/planning.md` updated with stop rule and next-session handoff.
- `workflow-plans/implementation-phase-1.md` created because `plan.md` names that phase.

No code, test, migration, generated, runtime config, `spec.md`, or `design/` writes were performed.
```

Prohibited request response:

```markdown
Planning is ready, but coding the first task is outside this session. I recorded `Next session starts with: implementation-phase-1` and stopped at the planning boundary.
```

## Blocker Handling
- If the user asks for a combined planning-and-code pass, finish the planning handoff and stop; implementation belongs to a later implementation session unless an eligible upfront direct/local waiver already exists.
- If planning needs a missing spec or design decision, record the blocker and reopen target; do not edit `spec.md` or `design/` inside this planning session.
- If a later phase-control file seems useful but the approved phase structure does not call for it, record it as not expected instead of creating a surprise file.
- If the allowed write set is ambiguous, prefer the narrower planning-only write and leave a blocker note rather than broadening scope silently.

## Exa Calibration Source Links
Found through Exa MCP search before these examples were written. Use these links only for calibration; local repo guidance wins.

- arc42 documentation: https://arc42.org/documentation/
- arc42 method: https://arc42.org/method
- Martin Fowler on Architecture Decision Records: https://martinfowler.com/bliki/ArchitectureDecisionRecord.html
- Asana implementation plan guide: https://www.asana.com/resources/implementation-plan


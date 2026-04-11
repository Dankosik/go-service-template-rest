# Workflow Plan Update Examples

## When To Load
Load this reference when repairing the task-local master `workflow-plan.md`, aligning it with `workflow-plans/planning.md`, or making planning status, artifact status, readiness, adequacy challenge status, and next-session handoff explicit.

This file gives examples only. `AGENTS.md` and `docs/spec-first-workflow.md` remain authoritative.

## Good Session Outcomes
- `workflow-plan.md` remains the cross-phase control artifact and does not absorb `plan.md`, `tasks.md`, `spec.md`, or `design/` content.
- Planning status, artifact status, blocker status, implementation-readiness status, adequacy challenge resolution, session boundary, and next-session start point are explicit.
- The master file and `workflow-plans/planning.md` agree on whether planning is complete, blocked, reopened, or still in progress.
- The update records later phase-control files as created, not expected, or blocked on a reopen.

## Bad Session Outcomes
- `workflow-plan.md` says planning is complete while `workflow-plans/planning.md` still says in progress.
- The master file records `implementation-readiness: PASS` while expected `tasks.md` is missing.
- The master file starts implementation by changing `Current phase` to `implementation-phase-1` before the planning session has stopped.
- Adequacy challenge findings are summarized as "done" without recording whether blocking findings were reconciled or waived.

## Example Handoff Notes
Complete planning master update:

```markdown
Current phase: planning
Phase status: complete
Session boundary reached: yes
Ready for next session: yes
Next session starts with: implementation-phase-1

Artifact status:
- `spec.md`: approved
- `design/`: approved
- `plan.md`: approved
- `tasks.md`: approved
- `test-plan.md`: not expected
- `rollout.md`: not expected
- `workflow-plans/planning.md`: complete
- `workflow-plans/implementation-phase-1.md`: created

Implementation readiness: PASS
Workflow plan adequacy challenge: completed; blocking findings reconciled
Blockers: none
```

Blocked planning master update:

```markdown
Current phase: planning
Phase status: blocked
Session boundary reached: no
Ready for next session: no
Next session starts with: technical-design

Artifact status:
- `plan.md`: draft
- `tasks.md`: blocked
- `workflow-plans/planning.md`: blocked

Implementation readiness: FAIL
Blocker: implementation order depends on a missing ownership decision in the design bundle.
Reopen target: technical-design
```

CONCERNS master update:

```markdown
Current phase: planning
Phase status: complete
Session boundary reached: yes
Ready for next session: yes
Next session starts with: implementation-phase-1
Implementation readiness: CONCERNS
Accepted risks: benchmark threshold is estimated from existing design evidence.
Proof obligations: `implementation-phase-1` must record benchmark evidence before phase completion.
Workflow plan adequacy challenge: completed; no blocking findings remain.
```

## Blocker Handling
- If master and phase-local status conflict, repair both in the planning session before marking handoff ready.
- If implementation-readiness is `FAIL`, set the next session to the named reopen target, not an implementation phase.
- If adequacy challenge findings are blocking, keep phase status blocked or in progress until reconciled.
- If a direct/local skip rationale is eligible, record it in the master update instead of leaving missing artifacts unexplained.

## Exa Calibration Source Links
Found through Exa MCP search before these examples were written. Use these links only for calibration; local repo guidance wins.

- arc42 documentation: https://arc42.org/documentation/
- arc42 method: https://arc42.org/method
- Martin Fowler on Architecture Decision Records: https://martinfowler.com/bliki/ArchitectureDecisionRecord.html
- Asana implementation plan guide: https://www.asana.com/resources/implementation-plan


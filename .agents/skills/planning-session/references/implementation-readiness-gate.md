# Implementation Readiness Gate Examples

## When To Load
Load this reference when assigning or checking implementation-readiness status, when recording `PASS`, `CONCERNS`, `FAIL`, or `WAIVED`, or when deciding whether a planning handoff can point to an implementation phase.

This file gives examples only. `AGENTS.md` and `docs/spec-first-workflow.md` remain authoritative.

## Good Session Outcomes
- `PASS` is used only after required spec, design, planning, task-ledger, optional validation or rollout artifacts, pre-created phase-control files, proof path, and blockers are resolved.
- `CONCERNS` names each accepted risk and adds proof obligations that the first implementation phase can verify without re-planning.
- `FAIL` names the earlier phase to reopen and keeps planning blocked rather than passing ambiguity forward.
- `WAIVED` is used only for an eligible tiny, direct-path, or prototype-scoped exception with rationale and scope.
- The readiness status is recorded in `workflow-plan.md`, the gate result and stop/handoff rule are recorded in `workflow-plans/planning.md`, a compact summary is added to `plan.md`, and `tasks.md` carries a short reference when useful.

## Bad Session Outcomes
- `PASS` while `tasks.md` is missing for non-trivial work.
- `CONCERNS` with vague wording such as "some risk remains" and no proof obligation.
- `FAIL` followed by creating the missing design artifact inside the same planning session.
- `WAIVED` used as a convenience for normal non-trivial work.
- The gate result appears only in chat and not in workflow-control artifacts.

## Example Handoff Notes
PASS:

```markdown
Implementation readiness: PASS.
Gate result: approved for `implementation-phase-1` in a later session.
Proof path: task-level verification is listed in `tasks.md`; phase checkpoint verification is listed in `plan.md`.
Stop rule: planning session ends here.
```

CONCERNS:

```markdown
Implementation readiness: CONCERNS.
Accepted risk: cache invalidation proof depends on first-phase integration test evidence.
Proof obligation: `implementation-phase-1` must add the named integration test before marking phase complete.
Gate result: implementation may start in the next session with this proof obligation visible.
```

FAIL:

```markdown
Implementation readiness: FAIL.
Reopen target: technical design.
Reason: `tasks.md` cannot order the migration safely because ownership of the backfill source of truth is not settled in `design/`.
Gate result: implementation must not start.
```

WAIVED:

```markdown
Implementation readiness: WAIVED.
Rationale: direct-path prototype note update; no code, data, API, or runtime-sequence change; separate `tasks.md` would add no useful execution control.
Scope: one documented planning note only.
Gate result: no implementation phase is opened by this planning session.
```

## Blocker Handling
- For unresolved high-impact decisions that could change correctness, ownership, rollout, or validation, use `FAIL`, name the reopen target, and stop.
- For accepted but bounded risks, use `CONCERNS` only when the risk can be carried by named proof obligations during implementation.
- For missing required artifacts, keep readiness at `FAIL` or blocked; do not downgrade to `CONCERNS` to force handoff.
- For tiny/direct-path exceptions, record the waiver rationale and scope clearly enough that a later session does not guess whether artifacts were forgotten.

## Exa Calibration Source Links
Found through Exa MCP search before these examples were written. Use these links only for calibration; local repo guidance wins.

- arc42 documentation: https://arc42.org/documentation/
- arc42 method: https://arc42.org/method
- Martin Fowler on Architecture Decision Records: https://martinfowler.com/bliki/ArchitectureDecisionRecord.html
- Asana implementation plan guide: https://www.asana.com/resources/implementation-plan


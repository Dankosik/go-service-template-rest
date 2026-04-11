# Session Boundary And Stop Rule Examples

## When To Load
Load this reference when closing a planning session, resolving whether a planning phase is complete or blocked, or writing stop rules that prevent drift into implementation, review, validation, or silent spec/design changes.

This file gives examples only. `AGENTS.md` and `docs/spec-first-workflow.md` remain authoritative.

## Good Session Outcomes
- The session marks `Session boundary reached: yes` only after planning artifacts, workflow-control artifacts, readiness status, and required adequacy challenge findings agree.
- The session sets `Ready for next session: yes` only when implementation may start under `PASS`, eligible `CONCERNS`, or eligible `WAIVED`.
- The session sets `Next session starts with` to the first named implementation phase or to a reopen target when readiness fails.
- The final planning action is a handoff update, not a code, review, or validation action.

## Bad Session Outcomes
- The session says "planning complete" and then begins T001.
- The session marks `Session boundary reached: yes` while readiness is `FAIL`.
- The session leaves `Next session starts with` blank, forcing the next agent to infer the phase from chat.
- The session records "continue implementation" when no implementation phase has started yet.
- The session updates `spec.md` or `design/` to clear a blocker instead of stopping at the planning boundary.

## Example Handoff Notes
Complete stop:

```markdown
Planning phase complete.
Session boundary reached: yes.
Ready for next session: yes.
Next session starts with: implementation-phase-1.
Stop rule: do not perform code, test, migration, review, validation, rollout execution, or closeout work in this planning session.
```

Blocked stop:

```markdown
Planning phase blocked.
Session boundary reached: no.
Ready for next session: no.
Next session starts with: technical-design.
Stop rule: do not create implementation tasks that depend on the missing ownership decision.
```

CONCERNS stop:

```markdown
Planning phase complete with CONCERNS.
Session boundary reached: yes.
Ready for next session: yes, with named proof obligations.
Next session starts with: implementation-phase-1.
Stop rule: the next session must satisfy the proof obligations before widening scope.
```

## Blocker Handling
- If completion criteria are not met, leave planning `in_progress` or `blocked`; do not mark the boundary reached.
- If the next step is a reopen target, name that target and stop rather than repairing upstream artifacts in the same planning session.
- If the user asks to continue into implementation immediately, answer with the recorded handoff and stop unless an eligible upfront waiver already covers same-session phase collapse.
- If an adequacy challenge has unresolved blocking findings, keep planning open or blocked and record the exact additions needed before handoff.

## Exa Calibration Source Links
Found through Exa MCP search before these examples were written. Use these links only for calibration; local repo guidance wins.

- arc42 documentation: https://arc42.org/documentation/
- arc42 method: https://arc42.org/method
- Martin Fowler on Architecture Decision Records: https://martinfowler.com/bliki/ArchitectureDecisionRecord.html
- Asana implementation plan guide: https://www.asana.com/resources/implementation-plan


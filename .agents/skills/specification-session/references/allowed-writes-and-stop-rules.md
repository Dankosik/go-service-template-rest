# Allowed Writes And Stop Rules Examples

## When To Load
Load this immediately before writing files, or when the user asks the session to include technical design, planning, tests, or implementation.
It is a boundary checklist for `specification-session`, not a replacement for the main protocol.

## Good Session Outcomes
- Writes stay limited to task-local `spec.md`, `workflow-plan.md`, and `workflow-plans/specification.md`.
- `workflow-plans/` is created only when needed to hold `workflow-plans/specification.md`.
- The phase file records out-of-scope surfaces instead of creating them.
- The session ends after specification artifacts agree on approval, blockers, and next session routing.

## Bad Session Outcomes
- Treating `design/overview.md` as a writable "starter" handoff note.
- Treating `plan.md`, `tasks.md`, `test-plan.md`, or `rollout.md` as writable during specification.
- Treating tests, implementation files, migrations, or staging as proof of spec quality.
- Updating a later phase-control file that does not already belong to the active specification session.

## Blocker Handling
When a request crosses the boundary, do not partially comply downstream.
Either complete the specification checkpoint or leave it blocked with the boundary violation recorded.

Example refusal:

```text
Cannot write `design/overview.md` in this session.
Reason: specification-session is allowed to update only `spec.md`, `workflow-plan.md`, and `workflow-plans/specification.md`.
Action: finish or block specification, then route the next session to `technical-design` if `spec.md` is approved.
```

## Workflow Update Examples
Allowed writes note in `workflow-plans/specification.md`:

```text
Allowed writes confirmed: spec.md, workflow-plan.md, workflow-plans/specification.md
Out of scope: design/, plan.md, tasks.md, tests, migrations, implementation files
Stop rule: stop after specification artifacts agree on state and next session routing.
```

Boundary block in `workflow-plan.md`:

```text
Current phase: specification
Current phase status: blocked
spec.md status: draft
Clarification gate: not run
Blockers: user request bundled technical design and implementation into the specification session.
Ready for next session: no
Next session starts with: specification after scope is corrected
```

## Exa Source Links
Exa MCP was attempted before these examples were authored, but `web_search_exa` and `web_fetch_exa` returned a 402 credits-limit error on 2026-04-11. These links are retained only as external calibration targets; `AGENTS.md` and `docs/spec-first-workflow.md` define the repository contract.

- [Atlassian - What is a Product Requirements Document?](https://www.atlassian.com/agile/requirements)
- [IBM - What is requirements management?](https://www.ibm.com/think/topics/what-is-requirements-management)
- [NASA - 4.2 Technical Requirements Definition](https://www.nasa.gov/reference/4-2-technical-requirements-definition/)

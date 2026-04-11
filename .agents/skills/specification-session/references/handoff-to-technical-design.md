# Handoff To Technical Design Examples

## When To Load
Load this after `spec.md` is approved or when deciding whether the specification session may route the next session to `technical-design`.
Use it to shape the handoff without starting technical design.

## Good Session Outcomes
- `spec.md` is approved with stable decisions, visible assumptions, and validation consequences.
- `workflow-plan.md` records `Current phase: specification`, phase completion, `Session boundary reached: yes`, and `Next session starts with: technical-design`.
- `workflow-plans/specification.md` records why the clarification gate is resolved or explicitly waived.
- The handoff names blockers, accepted assumptions, and reopen conditions that technical design must honor.
- The session stops before any `design/` artifact exists because of this session.

## Bad Session Outcomes
- Writing `design/overview.md` as the handoff.
- Creating a design checklist, implementation slice, or task ledger in the specification session.
- Leaving accepted assumptions only in chat.
- Routing to technical design while the clarification gate is still blocked.

## Blocker Handling
If design cannot safely begin, do not hand off to `technical-design`.

```text
Handoff status: blocked
Reason: accepted assumptions do not cover API idempotency behavior, and the clarification gate classified it as blocks_spec_approval.
Next session starts with: targeted API research or specification after the answer exists.
Stop rule: no technical design until the approval blocker is reconciled.
```

If design can begin with an explicit accepted risk:

```text
Handoff status: ready with accepted risk
Accepted risk: exact retry backoff values are deferred to technical design under the constraint that retry budget remains bounded and observable.
Spec location: `Open Questions / Assumptions` and `Validation`
Next session starts with: technical-design
```

## Workflow Update Examples
Ready handoff in `workflow-plan.md`:

```text
Current phase: specification
Current phase status: complete
Session boundary reached: yes
Ready for next session: yes
Next session starts with: technical-design
Artifacts: spec.md approved; design/ missing; plan.md missing; tasks.md missing
Clarification gate: complete and reconciled
Blockers: none
Reopen conditions: reopen specification if technical design finds a scope or acceptance contradiction.
```

Ready handoff in `workflow-plans/specification.md`:

```text
Phase status: complete
Completion marker: spec.md approved; clarification gate reconciled; master workflow plan updated
Stop rule: stop here; do not create design/, plan.md, tasks.md, tests, or implementation changes
Next action: start a new technical-design session from approved spec.md and workflow-plan.md
```

## Exa Source Links
Exa MCP was attempted before these examples were authored, but `web_search_exa` and `web_fetch_exa` returned a 402 credits-limit error on 2026-04-11. These links are retained only as external calibration targets; `AGENTS.md` and `docs/spec-first-workflow.md` define the repository contract.

- [Atlassian - What is a Product Requirements Document?](https://www.atlassian.com/agile/requirements)
- [IBM - What is requirements management?](https://www.ibm.com/think/topics/what-is-requirements-management)
- [NASA - 4.2 Technical Requirements Definition](https://www.nasa.gov/reference/4-2-technical-requirements-definition/)

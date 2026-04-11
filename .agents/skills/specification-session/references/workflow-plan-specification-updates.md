# Workflow Plan Specification Updates Examples

## When To Load
Load this when writing or repairing `workflow-plan.md` and `workflow-plans/specification.md` during a specification-only session.
Use it to keep master routing separate from phase-local orchestration.

## Good Session Outcomes
- `workflow-plan.md` records cross-phase state: current phase, artifact status, blockers, clarification gate status, session boundary, and next session routing.
- `workflow-plans/specification.md` records only phase-local details: readiness, sources, challenge lane, resolution, phase status, completion marker, stop rule, next action, blockers, and parallelizable work.
- The two files agree on whether `spec.md` is approved, draft, or blocked.
- Downstream artifact status can be recorded as missing or expected without creating downstream artifacts.

## Bad Session Outcomes
- Putting the full spec decisions in `workflow-plans/specification.md`.
- Putting implementation task order into `workflow-plan.md`.
- Marking `Ready for next session: yes` while `spec.md` is still draft and clarification is blocked.
- Recording "technical design next" in chat but leaving the master workflow plan stale.

## Blocker Handling
When blocked, make the next reopen point explicit in both files.
Avoid generic blockers like "needs more info" when a specific missing decision is known.

## Workflow Update Examples
Approved specification in `workflow-plan.md`:

```text
Current phase: specification
Current phase status: complete
Session boundary reached: yes
Ready for next session: yes
Next session starts with: technical-design
Phase workflow plans: specification complete
Artifacts: spec.md approved; design/ missing; plan.md missing; tasks.md missing
Clarification gate: complete and reconciled
Blockers: none
```

Approved specification in `workflow-plans/specification.md`:

```text
Phase status: complete
Readiness outcome: spec-ready
Input sources used: workflow-plan.md, research summary, candidate decisions, existing spec.md
Clarification challenge: complete and reconciled
Completion marker: spec.md approved and master routing updated
Stop rule: stop before technical design, planning, tests, or implementation
Next action: begin technical-design in a new session
Parallelizable work: none in this phase
```

Blocked specification in `workflow-plan.md`:

```text
Current phase: specification
Current phase status: blocked
Session boundary reached: yes
Ready for next session: no
Next session starts with: targeted research
Artifacts: spec.md blocked; design/ missing; plan.md missing; tasks.md missing
Clarification gate: blocked by domain question
Blockers: API idempotency semantics are unresolved and can change acceptance criteria.
```

## Exa Source Links
Exa MCP was attempted before these examples were authored, but `web_search_exa` and `web_fetch_exa` returned a 402 credits-limit error on 2026-04-11. These links are retained only as external calibration targets; `AGENTS.md` and `docs/spec-first-workflow.md` define the repository contract.

- [Atlassian - What is a Product Requirements Document?](https://www.atlassian.com/agile/requirements)
- [IBM - What is requirements management?](https://www.ibm.com/think/topics/what-is-requirements-management)
- [NASA - 4.2 Technical Requirements Definition](https://www.nasa.gov/reference/4-2-technical-requirements-definition/)

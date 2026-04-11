# Specification Session Readiness Examples

## When To Load
Load this before deciding whether a dedicated `specification-session` may draft, repair, or approve `spec.md`.
Use it when the phase context is incomplete, the task looks under-researched, or the caller asks for `spec.md` approval from partly settled decisions.

## Good Session Outcomes
- The orchestrator reads `workflow-plan.md` before `workflow-plans/specification.md`, confirms the current phase is `specification`, and loads only the smallest upstream evidence needed.
- The behavior delta, scope cuts, constraints, risk hotspots, and validation expectations are explicit enough to write stable `Decisions`.
- Remaining uncertainty is either harmless enough for `Open Questions / Assumptions` or is recorded as a blocker before approval.
- Non-trivial approval is held until the clarification gate is complete, reconciled, or explicitly waived by an eligible direct/local exception.
- If the task is not spec-ready, the session updates routing and stops without writing downstream artifacts.

## Bad Session Outcomes
- Approving `spec.md` because the user requested momentum while SAML vs OIDC, tenant boundary, ownership, or acceptance semantics are still unresolved.
- Treating the clarification challenge as a way to discover the basic product direction instead of challenging candidate decisions.
- Treating `design/`, `plan.md`, `tasks.md`, tests, or implementation changes as in-scope while checking readiness.
- Leaving workflow routing stale so the next session must reconstruct the real state from chat.

## Blocker Handling
Use blockers when a missing answer could change scope, correctness, ownership, rollout, or validation.

Example blocker decision:

```text
Spec readiness: blocked
Reason: candidate decisions do not define the tenant boundary for export job visibility.
Reopen target: targeted research or domain clarification before specification approval.
Spec state: draft only; do not approve.
```

Acceptable assumption handling:

```text
Spec readiness: ready with recorded assumption
Assumption: admin-only controls follow the repository's existing admin authorization pattern.
Risk: if research later contradicts that pattern, reopen specification before technical design.
```

## Workflow Update Examples
Ready path in `workflow-plan.md`:

```text
Current phase: specification
Current phase status: in_progress
spec.md status: draft
Clarification gate: pending
Ready for next session: no
Next session starts with: specification
Blockers: none known
```

Not-ready path in `workflow-plans/specification.md`:

```text
Phase status: blocked
Readiness outcome: not spec-ready
Input gap: core tenant visibility rule is unresolved and could change scope.
Completion marker: not met
Stop rule: stop before approval and before technical design.
Next action: reopen targeted research or upstream clarification.
```

## Exa Source Links
Exa MCP was attempted before these examples were authored, but `web_search_exa` and `web_fetch_exa` returned a 402 credits-limit error on 2026-04-11. These links are retained only as external calibration targets; `AGENTS.md` and `docs/spec-first-workflow.md` define the repository contract.

- [Atlassian - What is a Product Requirements Document?](https://www.atlassian.com/agile/requirements)
- [IBM - What is requirements management?](https://www.ibm.com/think/topics/what-is-requirements-management)
- [NASA - 4.2 Technical Requirements Definition](https://www.nasa.gov/reference/4-2-technical-requirements-definition/)

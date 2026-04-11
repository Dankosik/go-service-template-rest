# Spec Clarification Gate Flow Examples

## When To Load
Load this before approving non-trivial `spec.md`, when a clarification challenge result exists, or when the session needs to record why the gate is blocked, clear, or waived.

## Good Session Outcomes
- Candidate decisions exist before the challenge runs.
- The orchestrator prepares a compact bundle: problem frame, scope, non-goals, candidate decisions, constraints, validation expectations, assumptions, open questions, and relevant research links.
- One read-only lane runs exactly one skill: `spec-clarification-challenge`.
- The orchestrator reconciles every returned question before approval or records why approval is blocked.
- Only final resolved outcomes go into `spec.md`; raw challenge transcript stays out.
- If material decisions changed, the challenge is rerun once after the reopened seam is resolved.

## Bad Session Outcomes
- Running the challenge before the candidate decisions are clear enough to inspect.
- Treating `non_blocking_but_record` questions as invisible because they do not block approval.
- Pasting the challenge transcript into `spec.md`.
- Using `defer_to_design` for a question that would change scope or acceptance semantics.
- Starting `technical-design` immediately after the gate clears inside the same non-trivial session.

## Blocker Handling
Map challenge classifications to session outcomes:

```text
blocks_spec_approval: leave spec.md draft or blocked until answered, accepted as risk, or routed upstream.
blocks_specific_domain: reopen one targeted expert lane or targeted research path; record the reopen and stop.
non_blocking_but_record: record the constraint, assumption, or validation consequence before approval.
requires_user_decision: leave spec.md blocked or partially draft; do not invent the product/business answer.
```

## Workflow Update Examples
Resolved gate in `workflow-plans/specification.md`:

```text
Clarification challenge: complete
Lane: challenger-agent with spec-clarification-challenge
Resolution: all approval-changing questions answered from existing evidence
Targeted research reopened: no
Approval rationale: spec.md decisions now cover scope, constraints, validation, and accepted assumptions.
Phase status: complete
Stop rule: stop before technical design.
```

Blocked gate in `workflow-plan.md`:

```text
Current phase: specification
Current phase status: blocked
spec.md status: draft
Clarification gate: blocked by requires_user_decision
Blockers: retention policy choice cannot be derived from repository evidence.
Ready for next session: no
Next session starts with: specification after user decision or targeted policy research
```

## Exa Source Links
Exa MCP was attempted before these examples were authored, but `web_search_exa` and `web_fetch_exa` returned a 402 credits-limit error on 2026-04-11. These links are retained only as external calibration targets; `AGENTS.md` and `docs/spec-first-workflow.md` define the repository contract.

- [Atlassian - What is a Product Requirements Document?](https://www.atlassian.com/agile/requirements)
- [IBM - What is requirements management?](https://www.ibm.com/think/topics/what-is-requirements-management)
- [NASA - 4.2 Technical Requirements Definition](https://www.nasa.gov/reference/4-2-technical-requirements-definition/)

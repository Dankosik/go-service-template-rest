# Planning Readiness Gap Examples

## When To Load
Load this only when the active phase is `planning` or a planning-phase handoff controls whether implementation may start. Focus on implementation-readiness status, accepted risks, proof obligations, and reopen routing.

## Authoritative Inputs
- `AGENTS.md`: implementation readiness is the exit gate inside planning; allowed statuses are `PASS`, `CONCERNS`, `FAIL`, and `WAIVED`.
- `docs/spec-first-workflow.md`: `workflow-plan.md` records readiness status; `workflow-plans/planning.md` records gate result and stop or handoff rule; `plan.md` carries only a compact summary.

## Good Findings
- `Gap`: Planning handoff says "ready to code" but no implementation-readiness status is recorded.
  `Why It Matters`: Implementation could start without proving spec, design, plan, tasks, conditional artifacts, blockers, and validation path are ready or explicitly waived.
  `Classification`: `blocks_phase_handoff`.
  `Recommended Action`: `clarify_readiness_status`.
  `Exact Orchestrator Addition`: In `workflow-plan.md`, add `Implementation readiness: FAIL; route: planning repair before implementation`; in `workflow-plans/planning.md`, add `Gate result: FAIL because readiness status was missing; Stop rule: do not start implementation until PASS, eligible CONCERNS, or eligible WAIVED is recorded`.
- `Gap`: Readiness is `CONCERNS`, but accepted risks and proof obligations are not named.
  `Why It Matters`: The next session cannot tell which risk was accepted or what evidence must be produced.
  `Classification`: `blocks_phase_handoff`.
  `Recommended Action`: `clarify_readiness_status`.
  `Exact Orchestrator Addition`: Add `Implementation readiness: CONCERNS; accepted risk: <bounded risk>; proof obligation: <specific validation evidence>; handoff rule: implementation may start only if phase 1 verifies <proof> before broader changes`.
- `Gap`: Readiness is `WAIVED` for non-trivial work without tiny, direct-path, or prototype rationale.
  `Why It Matters`: `WAIVED` becomes a bypass around the non-trivial planning chain.
  `Classification`: `blocks_phase_handoff`.
  `Recommended Action`: `record_skip_or_accepted_risk`.
  `Exact Orchestrator Addition`: Replace with `Implementation readiness: FAIL; reopen planning to approve plan.md and tasks.md`, or record an eligible waiver with scope and rationale if the work truly qualifies.

## Bad Findings
- "Implementation readiness PASS after adding the missing field." Bad because this skill cannot approve readiness.
- "Copy all task IDs from `tasks.md` into `workflow-plans/planning.md`." Bad because the task ledger owns executable work.
- "Set WAIVED to move faster." Bad because waiver is narrow and must be justified by scope.

## Blocker Classification Examples
- `blocks_phase_handoff`: readiness missing, `FAIL` without reopen route, `CONCERNS` without named risks and proof, or ineligible `WAIVED`.
- `blocks_specific_lane`: a generated implementation phase-control file lacks handoff rule, but readiness can stay blocked only for that phase file repair.
- `non_blocking_but_record`: readiness is `PASS`, but the master should also record where implementation starts for resume reliability.

## Exact Orchestrator Additions
- `workflow-plan.md`: `Implementation readiness: PASS|CONCERNS|FAIL|WAIVED; Concerns: <named accepted risks and proof obligations or none>; Reopen target: <earlier phase if FAIL>; Next session starts with: <implementation phase or planning repair>`.
- `workflow-plans/planning.md`: `Gate result: <status and reason>; Stop or handoff rule: <do not implement|implementation may start with named concerns|waiver scope>; Required phase-control files: <created|not expected|missing>`.
- `plan.md`: compact summary only, for example `Implementation readiness: CONCERNS; proof obligation captured in workflow-plans/planning.md`.

## Exa Source Links
Exa MCP was attempted before authoring, but this environment returned a 402 credits-limit error. These fallback links are calibration only; repository-local docs remain authoritative.
- [Scrum.org Definition of Ready discussion](https://www.scrum.org/resources/blog/ready-or-not-demystifying-definition-ready-scrum) for readiness as contextual criteria with risks of over-rigid gatekeeping.
- [Atlassian project risk management](https://www.atlassian.com/work-management/project-management/project-risk-management) for risk reviews, escalation, and action-oriented next steps.
- [Asana progress report template](https://asana.com/templates/progress-report) for documenting issues, risks, owners, and next steps.

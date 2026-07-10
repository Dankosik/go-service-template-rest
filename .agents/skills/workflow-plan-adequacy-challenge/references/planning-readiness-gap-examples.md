# Planning Readiness Gap Examples

## Behavior Change Thesis
When loaded for symptom planning says implementation may start but task-ledger review or readiness is weak, this file makes the model route the gap through `PASS`, `CONCERNS`, `FAIL`, or `WAIVED` repair instead of likely mistake accepting "ready to code" prose.

## When To Load
Load this only when the active phase is `planning` or a planning-phase handoff controls whether implementation may start. Focus on task-ledger review status, implementation-readiness status, accepted risks, proof obligations, and reopen routing.

## Decision Rubric
- `PASS` means implementation may start only after required spec, design, plan, task ledger, conditional artifacts, blockers, and validation path are ready, and the ledger has been checked against that artifact chain.
- `CONCERNS` must name accepted risks and proof obligations.
- `FAIL` must name the earlier phase or planning repair target to reopen.
- `WAIVED` is eligible only for a recorded prototype-scoped ledger-review waiver and must state rationale and scope. Current `SHAPE-DIRECT` never enters task-ledger review; its absent ledger/review artifacts are `artifact_expectation=not_expected` with `waiver_disposition=none`.
- The current task-review/readiness carrier records `procedural_gate_state`, `review_verdict`, and `record_validity`; `workflow-plans/planning.md` records the pending handoff plus stop rule, and `tasks.md` carries only a short reference when useful.

## Imitate
### No readiness status
`Gap`: Planning handoff says "ready to code" but no task-review/readiness procedural state, verdict, or validity is recorded.

Why to copy: implementation could start without proving the planning exit gate.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `clarify_readiness_status`
- `Exact Orchestrator Addition`: In the owning readiness record, add `procedural_gate_state=blocked; review_verdict=FAIL; record_validity=current; reopen_target=planning`; in `workflow-plans/planning.md`, record `session_boundary=reached; handoff_readiness=ready` for the actionable planning-repair session and keep implementation unauthorized until a fresh `PASS`, eligible `CONCERNS`, or eligible prototype-scoped `WAIVED` is recorded.

### Concerns without proof obligation
`Gap`: Readiness is `CONCERNS`, but accepted risks and proof obligations are not named.

Why to copy: the next session cannot tell which risk was accepted or what evidence must be produced.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `clarify_readiness_status`
- `Exact Orchestrator Addition`: Add `procedural_gate_state=complete; review_verdict=CONCERNS; record_validity=current; accepted risk: <bounded risk>; proof obligation: <specific validation evidence>; handoff_readiness=ready only for the named obligation`.

### Ineligible waiver
`Gap`: Readiness is `WAIVED` without an eligible prototype-scoped ledger-review rationale.

Why to copy: `WAIVED` can otherwise become a bypass around the non-trivial planning chain.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `record_skip_or_accepted_risk`
- `Exact Orchestrator Addition`: Replace with `procedural_gate_state=blocked; review_verdict=FAIL; reopen_target=planning`, or record an eligible separate waiver disposition with scope, rationale, evidence, and reopen trigger if the work truly qualifies.

## Reject
- "Task-ledger review PASS after adding the missing field." This skill cannot approve readiness.
- "Copy all task IDs from `tasks.md` into `workflow-plans/planning.md`." The task ledger owns executable work.
- "Set WAIVED to move faster." Waiver is narrow and must be justified by scope.

## Agent Traps
- Do not infer `PASS` from optimistic wording.
- Do not let `CONCERNS` stand without named accepted risks and proof obligations.
- Do not route `FAIL` to implementation; it reopens the named earlier phase or planning repair.

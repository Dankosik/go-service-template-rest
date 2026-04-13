# Planning Readiness Gap Examples

## Behavior Change Thesis
When loaded for symptom planning says implementation may start but readiness is weak, this file makes the model route the gap through `PASS`, `CONCERNS`, `FAIL`, or `WAIVED` repair instead of likely mistake accepting "ready to code" prose.

## When To Load
Load this only when the active phase is `planning` or a planning-phase handoff controls whether implementation may start. Focus on implementation-readiness status, accepted risks, proof obligations, and reopen routing.

## Decision Rubric
- `PASS` means implementation may start only after required spec, design, plan, task ledger, conditional artifacts, blockers, and validation path are ready.
- `CONCERNS` must name accepted risks and proof obligations.
- `FAIL` must name the earlier phase or planning repair target to reopen.
- `WAIVED` is eligible only for tiny, direct-path, or prototype-scoped work and must state rationale and scope.
- The master records readiness status; `workflow-plans/planning.md` records gate result plus stop or handoff rule; `tasks.md` carries only a short reference when useful.

## Imitate
### No readiness status
`Gap`: Planning handoff says "ready to code" but no implementation-readiness status is recorded.

Why to copy: implementation could start without proving the planning exit gate.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `clarify_readiness_status`
- `Exact Orchestrator Addition`: In `workflow-plan.md`, add `Implementation readiness: FAIL; route: planning repair before implementation`; in `workflow-plans/planning.md`, add `Gate result: FAIL because readiness status was missing; Stop rule: do not start implementation until PASS, eligible CONCERNS, or eligible WAIVED is recorded`.

### Concerns without proof obligation
`Gap`: Readiness is `CONCERNS`, but accepted risks and proof obligations are not named.

Why to copy: the next session cannot tell which risk was accepted or what evidence must be produced.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `clarify_readiness_status`
- `Exact Orchestrator Addition`: Add `Implementation readiness: CONCERNS; accepted risk: <bounded risk>; proof obligation: <specific validation evidence>; handoff rule: implementation may start only if phase 1 verifies <proof> before broader changes`.

### Ineligible waiver
`Gap`: Readiness is `WAIVED` for non-trivial work without tiny, direct-path, or prototype rationale.

Why to copy: `WAIVED` can otherwise become a bypass around the non-trivial planning chain.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `record_skip_or_accepted_risk`
- `Exact Orchestrator Addition`: Replace with `Implementation readiness: FAIL; reopen planning to approve tasks.md`, or record an eligible waiver with scope and rationale if the work truly qualifies.

## Reject
- "Implementation readiness PASS after adding the missing field." This skill cannot approve readiness.
- "Copy all task IDs from `tasks.md` into `workflow-plans/planning.md`." The task ledger owns executable work.
- "Set WAIVED to move faster." Waiver is narrow and must be justified by scope.

## Agent Traps
- Do not infer `PASS` from optimistic wording.
- Do not let `CONCERNS` stand without named accepted risks and proof obligations.
- Do not route `FAIL` to implementation; it reopens the named earlier phase or planning repair.

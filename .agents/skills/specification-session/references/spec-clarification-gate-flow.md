# Spec Clarification Gate Flow

## Behavior Change Thesis
When loaded before marking a formally challenged `spec.md` review-ready or reconciling a clarification result, this file makes the model choose challenge-first reconciliation with final outcomes recorded in `spec.md` instead of the likely mistake of treating the gate as optional ceremony, pasting the transcript, or deferring approval-changing questions to design. Lean-local specs without escalation triggers use inline `Risk Challenge` instead.

## When To Load
Load this before approval when `execution_shape=full_orchestrated`, relevant `FULL-*` evidence is true or approval-relevant unknown, the decision is hard-to-reverse or cross-domain, `agent_request=substantive`, or an explicit deep challenge triggers formal clarification; also load it when a `spec-clarification-challenge` result exists or the session must record why the gate became not expected after guarded reclassification.

## Decision Rubric
- Run the challenge only after candidate decisions are concrete enough to inspect.
- Prepare a compact bundle: problem frame, scope, non-goals, candidate decisions, constraints, validation expectations, assumptions, open questions, and relevant research links.
- Use one focused read-only challenger lane, preferably `challenger-agent` with `spec-clarification-challenge`, for the formal gate. Add lanes only for additional concrete independent approval questions that can change review-readiness and materially benefit from separate context. Each lane still uses one skill.
- Default to no more than three concurrently active subagent lanes. Exceed three only with a task-specific reason why the extra question cannot wait, merge, or run sequentially.
- Keep `Lens` as coverage metadata. Lane outputs still use existing `spec-clarification-challenge` classifications.
- Reconcile every returned question before review-ready handoff, including `non_blocking_but_record` items that must become constraints, assumptions, or validation consequences.
- If a question requires expert work, record the reopen and stop. No waiver permits phase collapse or waives expert work required by a clarification finding.
- Paste final resolved outcomes into `spec.md`; do not paste the raw challenge transcript.
- Rerun the challenge once only when material decisions changed or a major seam was reopened and resolved.

## Imitate
Classification mapping:

```text
blocks_spec_approval: leave spec.md draft or blocked until answered, accepted as risk, or routed upstream.
blocks_specific_domain: reopen one targeted expert lane or targeted research path; record the reopen and stop.
non_blocking_but_record: record the constraint, assumption, or validation consequence before approval.
requires_user_decision: leave spec.md blocked or partially draft; do not invent the product/business answer.
```

Resolved gate in `workflow-plans/specification.md`:

```text
Clarification challenge procedural_gate_state: complete
Clarification challenge record_validity: current
Lanes: one challenger-agent with spec-clarification-challenge
Question: Can any unresolved scope, invariant, ownership, contract, or proof issue change review-readiness?
Additional lanes: none; root synthesis can reconcile the dependent checks
Resolution: all approval-changing questions answered from existing evidence
phase_state: complete
reopen_target: none
Review-ready rationale: spec.md decisions now cover scope, constraints, validation, and accepted assumptions.
Stop rule: stop before technical design.
```

Copy the classification-to-action mapping, especially the difference between recordable questions and approval blockers.

## Reject
Gate as decoration:

```text
Clarification challenge: skipped; spec already looks reasonable.
```

This fails for formally challenged work because approval requires a reconciled gate unless recorded shape reclassification proves the formal trigger no longer applies. Lean-local work should record inline `Risk Challenge` and its subagent gate decision rather than pretending no challenge happened.

Transcript dumping:

```text
Decisions: [full challenge transcript pasted here]
```

This fails because `spec.md` stores orchestrator-owned final outcomes, not subagent raw output.

## Agent Traps
- Running the challenge too early, before the challenger has actual decisions to pressure-test.
- Treating `non_blocking_but_record` as no-op.
- Using `defer_to_design` for a question that changes scope, acceptance, ownership, or validation.
- Starting triggered `system-integration-design` or `go-code-ownership-design` inside the same specification session after the gate clears.

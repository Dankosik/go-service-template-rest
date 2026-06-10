# Spec Clarification Gate Flow

## Behavior Change Thesis
When loaded before approving a formally challenged `spec.md` or reconciling a clarification result, this file makes the model choose challenge-first reconciliation with final outcomes recorded in `spec.md` instead of the likely mistake of treating the gate as optional ceremony, pasting the transcript, or deferring approval-changing questions to design. Lean-local specs without escalation triggers use inline `Risk Challenge` instead.

## When To Load
Load this before full-orchestrated, high-risk, protected-domain, or otherwise formally challenged spec approval, when a `spec-clarification-challenge` result exists, or when the session needs to record why the formal gate is clear, blocked, or not expected after recorded shape reclassification.

## Decision Rubric
- Run the challenge only after candidate decisions are concrete enough to inspect.
- Prepare a compact bundle: problem frame, scope, non-goals, candidate decisions, constraints, validation expectations, assumptions, open questions, and relevant research links.
- Use read-only challenger lane(s), preferably `challenger-agent` with `spec-clarification-challenge`. Broad or multi-domain full-orchestrated, protected-domain, high-risk, cross-domain, hard-to-reverse, or user-requested deep challenge work should use five distinct lenses by default: scope/spec coherence; domain invariants/edge cases; architecture ownership/dependency boundaries; API/data/compatibility/source-of-truth; security/reliability/delivery/validation proof.
- Use one lane only when the approval risk is narrowly concentrated and the scoped-down rationale lists default lenses considered, the retained lane, and why omitted lenses cannot change approval. Each lane still uses one skill.
- Keep `Lens` as coverage metadata. Lane outputs still use existing `spec-clarification-challenge` classifications.
- Reconcile every returned question before approval, including `non_blocking_but_record` items that must become constraints, assumptions, or validation consequences.
- If a question requires expert work, record the reopen and stop. A direct/lean waiver may affect phase collapse only; it cannot waive expert work required by a clarification finding.
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
Clarification challenge: complete
Lanes: challenger-agent with spec-clarification-challenge
Lenses: scope/spec coherence; domain invariants/edge cases; architecture ownership; API/data/compatibility; security/reliability/delivery/validation
Scoped-down rationale: N/A; broad default lens set used
Resolution: all approval-changing questions answered from existing evidence
Targeted research reopened: no
Approval rationale: spec.md decisions now cover scope, constraints, validation, and accepted assumptions.
Phase status: complete
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

# Spec Clarification Gate Flow

## Behavior Change Thesis
When loaded before approving a non-trivial `spec.md` or reconciling a clarification result, this file makes the model choose challenge-first reconciliation with final outcomes recorded in `spec.md` instead of the likely mistake of treating the gate as optional ceremony, pasting the transcript, or deferring approval-changing questions to design.

## When To Load
Load this before non-trivial spec approval, when a `spec-clarification-challenge` result exists, or when the session needs to record why the gate is clear, blocked, or legitimately waived.

## Decision Rubric
- Run the challenge only after candidate decisions are concrete enough to inspect.
- Prepare a compact bundle: problem frame, scope, non-goals, candidate decisions, constraints, validation expectations, assumptions, open questions, and relevant research links.
- Use one read-only lane, preferably `challenger-agent`, with exactly one skill: `spec-clarification-challenge`.
- Reconcile every returned question before approval, including `non_blocking_but_record` items that must become constraints, assumptions, or validation consequences.
- If a question requires expert work, record the reopen and stop unless an upfront direct/local waiver permits same-session collapse.
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
Lane: challenger-agent with spec-clarification-challenge
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

This fails for non-trivial work because approval requires either a reconciled gate or an eligible direct/local waiver.

Transcript dumping:

```text
Decisions: [full challenge transcript pasted here]
```

This fails because `spec.md` stores orchestrator-owned final outcomes, not subagent raw output.

## Agent Traps
- Running the challenge too early, before the challenger has actual decisions to pressure-test.
- Treating `non_blocking_but_record` as no-op.
- Using `defer_to_design` for a question that changes scope, acceptance, ownership, or validation.
- Starting `technical-design` inside the same non-trivial specification session after the gate clears.

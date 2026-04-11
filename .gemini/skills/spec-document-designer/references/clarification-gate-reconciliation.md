# Clarification-Gate Reconciliation

## Behavior Change Thesis
When loaded after a `spec-clarification-challenge` pass, this file makes the model reconcile findings into final spec sections and gate status instead of pasting the challenge transcript or approving through unresolved blockers.

## When To Load
Load this after challenge findings arrive, when checking whether non-trivial `spec.md` approval is legitimate, or when a draft contains reviewer comments rather than resolved outcomes.

## Decision Rubric
- Treat the challenge as advisory. The orchestrator owns final decisions.
- For `answer_from_existing_evidence`, write the stable answer in `Decisions` and any proof consequence in `Validation`.
- For design-shaped detail, record `[defer_to_design]` only when the behavior decision is already stable.
- For external product or policy decisions, record `[requires_user_decision]` and keep the spec draft or blocked if the answer changes planning.
- For under-evidenced technical claims, record `[targeted_research]` or reopen research instead of guessing.
- If a material decision changes or a major seam reopens and then resolves, rerun the clarification challenge once before approval.
- Do not copy raw findings, reviewer names, or back-and-forth into `spec.md`.

## Imitate

Challenge finding:

```text
blocks_spec_approval: The spec does not decide whether failed webhook delivery is retried or surfaced only in logs.
next_action: answer_from_existing_evidence
```

Repo-native reconciliation:

```markdown
## Decisions
- Failed webhook delivery uses the existing retry policy owned by the outbound delivery component; this change does not introduce a new retry budget.

## Validation
- Tests prove the new webhook path delegates failures to the existing retry policy.
```

Copy this: the raw challenge is not pasted into the spec; the final decision and proof consequence are placed where downstream design needs them.

```markdown
## Open Questions / Assumptions
- [defer_to_design] The exact retry metric label belongs in `design/overview.md` and observability design; the spec only decides that existing retry semantics are preserved.
```

Copy this: design detail is deferred only after the behavior decision is stable.

## Reject

```markdown
## Clarification Challenge Transcript
Reviewer: What about webhook retry?
Orchestrator: Maybe existing retry?
Reviewer: Please check.
```

Failure: transcripts are not final decisions and force later phases to infer authority from conversation.

```markdown
## Outcome
Spec approved.

## Open Questions / Assumptions
- Need to decide retry behavior later.
```

Failure: the unresolved item changes correctness and design, so non-trivial spec approval is blocked unless an explicit accepted risk and proof consequence exist.

## Agent Traps
- Treating the challenge as a second author of `spec.md`.
- Recording every reviewer concern even after it is resolved.
- Calling a design-owned detail a blocker when behavior semantics are already stable.
- Asking the human for questions that repository evidence or targeted research can answer.

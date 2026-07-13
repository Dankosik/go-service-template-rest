---
name: pre-spec-challenge
description: "Pressure-test a candidate synthesis before specification or planning with a few discriminating questions about hidden assumptions, ownership, failure semantics, and rollout risk. Use when research/framing exists and an independent challenge could change decisions; skip blank-page framing, ordinary spec review, and one isolated high-impact spec question."
---

# Pre-Spec Challenge

## Outcome

Return the smallest set of unresolved seams that could still make planning unsafe, together with severity and a concrete resolution route. The orchestrator owns decisions; this skill stays advisory and read-only.

Use `spec-clarification-challenge` for one concrete high-impact specification question and `specification-review` for readiness of a fixed spec revision.

## Challenge Method

1. Confirm the input contains a problem frame, candidate decisions, material constraints/assumptions, and relevant evidence.
2. Extract only assumptions carrying the candidate path and try to falsify each: what breaks in production if it is false?
3. Keep a question only when a different answer changes scope, correctness, ownership, failure semantics, rollout, or planning order.
4. Classify the seam and choose its next action before wording the final question.
5. Stop when remaining uncertainty is tracked, low-value, or ordinary downstream elaboration.

## Question Rules

- Prefer `3-5` strong questions; use fewer when fewer material forks exist.
- Name the challenged assumption or seam, why it matters, and what changes.
- Ask about security, performance, failure, or rollout only through a concrete seam already present.
- Do not reopen settled decisions without evidence, ask the user directly, author replacement design, or produce a competing spec.

Use blocker levels: `blocks_planning`, `blocks_specific_domain`, `non_blocking`.

Use next actions: `answer`, `re-research`, `ask_user`, `defer`, `accept_risk`. For `re-research`, name the specialist/fact pattern and whether challenge must rerun.

## Reference Selector

Load at most one reference by default; load more only for independent seams.

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| The bundle may be too thin to challenge. | [input-sufficiency-and-challenge-readiness.md](references/input-sufficiency-and-challenge-readiness.md) | Route to framing/research instead of inventing questions. |
| Convenience assumptions carry the candidate path. | [falsifying-candidate-assumptions.md](references/falsifying-candidate-assumptions.md) | Ask failure-discriminating questions instead of category prompts. |
| Source of truth, actor authority, side effects, async, cache, or destructive seams are unclear. | [ownership-boundary-and-failure-seams.md](references/ownership-boundary-and-failure-seams.md) | Challenge durable ownership and recovery. |
| Migration, compatibility, flags, canary, backfill, or rollback matters. | [rollout-compatibility-and-risk-questions.md](references/rollout-compatibility-and-risk-questions.md) | Test mixed-version and rollback states. |
| Severity or resolution route is unclear. | [triage-and-next-action-routing.md](references/triage-and-next-action-routing.md) | Classify by planning impact and choose one action. |

## Output

Return `Challenge Summary`, `Questions`, `Escalations / Re-research`, and `Confidence`. Each question includes `Question / Challenged Assumption`, `Why It Matters`, `What Changes`, `Blocker Level`, `Next Action`, and `Research Reopen` only when applicable.

## Stop Or Reject

Stop as sufficiently reconciled when unresolved items no longer change planning safety. Reject or route upstream when the request is trivial, asks for nitpicks, lacks candidate synthesis, or needs problem framing/integration design before challenge can add value.

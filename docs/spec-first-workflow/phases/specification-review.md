# Specification Review Phase

Detailed phase companion for `docs/spec-first-workflow.md`. Read this when reviewing a completed non-trivial `spec.md` before technical design, planning, or implementation.

## Read When

- A completed non-trivial `spec.md` is marked review-ready.
- Downstream technical design, compact tasking, planning, or implementation is waiting on a spec-review verdict.
- A repaired spec after `FAIL` needs a fresh or explicitly updated follow-up verdict.

## Inputs

- Completed `spec.md`, workflow-control state, preserved research, clarification fan-in, and linked source-of-truth artifacts.
- Read-only specification-review lane summaries, or a valid scoped-down/local-only rationale.

## Outputs

- `PASS`, `CONCERNS`, or `FAIL` with lens coverage, findings, orchestrator resolution, accepted risks, proof obligations, readiness consequence, and reopen target.

## Stop Rule

Keep specification review read-only. If findings require content changes, route to specification repair and require a follow-up review before downstream phases start.

## Specification Review

Specification review is the mandatory post-spec gate for non-trivial work. It is not the same thing as `spec-clarification-challenge`: clarification finds approval-changing questions while candidate decisions are being finalized; specification review inspects the completed `spec.md` for breadth, depth, decision coverage, assumptions, proof obligations, and downstream readiness.

Run specification review after the specification session records `spec.md` as review-ready and before any of these start:

- compact lean tasking;
- separate technical design;
- planning;
- implementation.

Specification review must be read-only and falsification-oriented:

- inspect the completed `spec.md`, workflow-control state, preserved research, formal clarification fan-in, and any linked source-of-truth artifacts;
- check scope/non-goals, behavior/contract delta, product or operator expectations, domain invariants, edge cases, public/API/data/source-of-truth effects, dependency/OSS diligence, Pattern Fit Diligence, legacy-surface handling, security/reliability/delivery implications, validation proof obligations, and downstream handoff clarity;
- verify that every material decision is explicit enough for design or planning without rediscovering product meaning;
- distinguish missing spec decisions from design-owned mechanism choices and planning-owned task ordering;
- treat `defer_to_design` as `FAIL` when it hides owner/source-of-truth, observable contract, reliability, security, rollout, validation, dependency/OSS, Pattern Fit, or legacy-cleanup decisions required to select the production-ready solution;
- report only findings that can change approval, require a named proof obligation, or prevent the next phase from starting honestly.

Use the review to try to disprove downstream readiness, not to improve prose. Ask:

- What would make this spec unsafe to plan or design from?
- Which caller, operator, invariant, owner, source-of-truth, or proof claim could be wrong or absent?
- Which stated non-goal is actually required target-state work for the accepted scope?
- Which accepted risk lacks a boundary, owner, downstream carry target, or reopen trigger?
- Which proof obligation is being used to postpone a spec-owned decision?

## Lens Selection And Coverage

For non-trivial work, use at least one distinct read-only specification-review lane. Use multiple lanes by default when independent review lenses could change approval, including product/scope coherence, domain invariants, API/data/source-of-truth, architecture ownership, security/reliability/delivery, validation/QA, dependency/OSS, Pattern Fit, and legacy cleanup.

Select lenses from the spec's actual approval risks, not from a fixed checklist. Start with every domain that appears in scope, non-goals, behavior delta, decisions, compact design constraints, diligence sections, accepted risks, or proof obligations. Add a lens when a missing fact in that domain could change `PASS`, `CONCERNS`, or `FAIL`. Merge lenses only when they share the same evidence, same owner, and same downstream-readiness question.

A scoped-down review must list candidate lenses considered and why omitted lenses cannot change review readiness. Local-only specification review is valid only for explicit direct-path/prototype waiver or when read-only lane execution is unavailable and the workflow records the consequence as `scoped_down` or blocked.

Specification review must include a compact lens coverage table:

```text
Lens | Trigger/source | Owned readiness question | Falsification check | Status | Evidence pointer | Reason | Disposition
```

`Trigger/source` is the spec section, behavior delta, decision, accepted risk, proof obligation, diligence section, or scoped-down candidate that caused the lens to be considered. `Falsification check` states what would make that lens fail and whether the reviewed evidence exposed it.

Status is `covered`, `not_applicable`, `concern`, or `fail`. `covered` means the reviewer tried to disprove readiness for that lens and the reviewed spec plus linked evidence answered the readiness question. It is not enough to find related prose. `not_applicable` requires a trigger-based reason. `concern` means the spec is coherent but must carry a named accepted risk or proof obligation into a downstream artifact. `fail` means the review found a missing or contradictory decision, evidence gap, unresolved owner, hidden scope cut, or unavailable proof path that blocks downstream start.

`PASS` is not valid until every readiness-critical lens has a recorded status. Omitted lenses require the scoped-down rationale; unexamined plausible lenses are not `not_applicable`.

Each surviving finding must use this minimum shape:

```text
Finding: <short title>
Spec anchor: <spec section/path>
Evidence: <artifact/source pointer>
Impact: <downstream readiness consequence>
Decision owner: <specification | research | user/specialist | system-integration-design | go-code-ownership-design | planning | downstream-proof>
Primary classification: <classification below>
Owner/reopen target: <artifact, phase, user/specialist decision, or none>
Why not stronger/weaker: <why this is not PASS-only context, CONCERNS-only carry-forward, or FAIL-blocking repair>
Required disposition: <repair | user decision | accepted risk | proof obligation | record only>
```

Specification review gate status:

- `PASS`: technical design, compact lean tasking, or planning may start from the reviewed spec. Use only when every readiness-critical lens is `covered` or justified `not_applicable`, all material decisions are explicit, and no accepted risk or proof obligation is required beyond the spec's normal validation obligations.
- `CONCERNS`: the next phase may start only with named accepted risks and proof obligations carried into design, planning, `tasks.md`, `test-plan.md`, or `rollout.md`. Use only when the spec is coherent and downstream-ready, and the remaining concern is a bounded risk or proof requirement that does not change the selected scope, behavior, owner, source of truth, dependency/OSS choice, Pattern Fit outcome, legacy-surface decision, or validation strategy.
- `FAIL`: downstream phases must not start; reopen specification, research, targeted specialist review, or user decision. Use when any readiness-critical lens is `fail`, a material decision is missing or contradictory, evidence is insufficient to make an honest spec decision, a non-goal hides in-scope target-state work, an accepted risk lacks boundaries, `defer_to_design` hides a production-readiness decision, or proof obligations are too vague or impossible to plan from. Repair alone is not enough to continue; the revised spec needs a follow-up review verdict of `PASS` or `CONCERNS`.

Use this decision order:

1. If any lens is `fail`, any required spec decision is absent, or any blocker classification remains unresolved, the verdict is `FAIL`.
2. Otherwise, if remaining items are bounded accepted risks or proof obligations that downstream artifacts can carry without changing the spec decision, the verdict is `CONCERNS`.
3. Otherwise, if every readiness-critical lens is `covered` or justified `not_applicable`, the verdict is `PASS`.

Classify findings by strongest downstream-readiness impact:

- `blocks_spec_approval`: the spec cannot become downstream-ready until the issue is resolved.
- `reopens_specification`: `spec.md` must change before review can pass.
- `reopens_research`: missing evidence prevents an honest spec decision.
- `requires_user_decision`: the missing decision is external product, business, policy, or legal judgment.
- `accepted_risk_candidate`: the orchestrator may proceed only by naming the accepted risk and boundary.
- `proof_obligation`: the spec is coherent, but later artifacts must carry a named proof.
- `record_only`: useful context that does not affect downstream entry.

Use a simple discriminator for ownership. If the answer defines what must be true, who owns it, what external behavior changes, what source of truth wins, what risk is accepted, or what evidence must exist, it is spec/research/user-owned. If the answer chooses system behavior or runtime mechanism inside already-approved facts, it is system/integration-design-owned. If the answer chooses package/file ownership, focused responsibility, local abstraction, cleanup, or test ownership without changing those facts, it is Go-code-ownership-design-owned. If the answer chooses when or in what slice to execute already-decided work, it is planning-owned.

`system-integration-design` is the owner only for mechanism choices that cannot change review-ready scope, observable contract, ownership/source-of-truth policy, reliability/security boundary, rollout policy, validation feasibility, dependency/OSS approval, Pattern Fit approval, or legacy-cleanup decisions. `go-code-ownership-design` is the owner only for package/file placement, dependency direction, local abstraction, cleanup/removal, and test-ownership choices that cannot change observable behavior or source-of-truth policy. `planning` is the owner only for task order, checkpoint placement, evidence collection sequence, and task slicing after the spec and design decisions are already complete. If classifying a finding as design-owned or planning-owned would require those phases to choose product meaning, source of truth, external behavior, accepted risk boundaries, or proof feasibility, classify it as `reopens_specification`, `reopens_research`, or `requires_user_decision` instead.

Proof obligations must name the claim, downstream artifact that must carry it, expected fresh evidence, freshness or negative-proof requirement when relevant, and the reopen condition if proof fails. Do not write the design answer, implementation task, or validation transcript inside specification review. Do not use `proof_obligation` to launder a missing spec decision into `CONCERNS`; if the decision itself is absent, the verdict is `FAIL`.

Record the review result in the active workflow-control surface: `workflow-plan.md`, `workflow-plans/specification-review.md` when a dedicated review phase needs durable routing, or the lean-local `spec.md` when no workflow-control artifact exists. The record must name the reviewed `spec.md`, reviewer or lanes, scope, lens coverage table, findings in the required shape, orchestrator resolution, final gate status, accepted risks, proof obligations, readiness consequence, and reopen target. Review subagents do not edit `spec.md`; if findings require content changes, route to specification repair and run a follow-up review after the repair.

Follow-up review after repair is a fresh read-only verdict, not a rubber stamp. It must identify the prior `FAIL` or blocking concerns, the repaired `spec.md` version or evidence anchor, the changed sections reviewed, any unchanged sections relied on, and whether the repair introduced new contradictions or stale proof obligations. Include a closure table:

```text
Prior finding | Repair/evidence anchor | Rechecked lenses | Closure status | Residual proof obligation/reopen target
```

The follow-up may be narrower than the original review only when the record explains why untouched lenses cannot change the new verdict; otherwise rerun the affected lens coverage before returning `PASS` or `CONCERNS`.

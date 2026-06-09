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
- report only findings that can change approval, require a named proof obligation, or prevent the next phase from starting honestly.

For non-trivial work, use at least one distinct read-only specification-review lane. Use multiple lanes by default when independent review lenses could change approval, including product/scope coherence, domain invariants, API/data/source-of-truth, architecture ownership, security/reliability/delivery, validation/QA, dependency/OSS, Pattern Fit, and legacy cleanup. A scoped-down review must list candidate lenses considered and why omitted lenses cannot change review readiness. Local-only specification review is valid only for explicit direct-path/prototype waiver or when read-only lane execution is unavailable and the workflow records the consequence as `scoped_down` or blocked.

Specification review must include a compact lens coverage table. Each considered lens is marked `covered`, `not_applicable`, `concern`, or `fail`, with an evidence pointer and short reason. `PASS` is not valid until every readiness-critical lens has a recorded status; omitted lenses require the scoped-down rationale.

Each surviving finding must use this minimum shape:

```text
Finding: <short title>
Spec anchor: <spec section/path>
Evidence: <artifact/source pointer>
Impact: <downstream readiness consequence>
Classification: <classification below>
Required disposition: <repair | user decision | accepted risk | proof obligation | record only>
```

Specification review gate status:

- `PASS`: technical design, compact lean tasking, or planning may start from the reviewed spec.
- `CONCERNS`: the next phase may start only with named accepted risks and proof obligations carried into design, planning, `tasks.md`, `test-plan.md`, or `rollout.md`.
- `FAIL`: downstream phases must not start; reopen specification, research, targeted specialist review, or user decision. Repair alone is not enough to continue; the revised spec needs a follow-up review verdict of `PASS` or `CONCERNS`.

Classify findings by strongest downstream-readiness impact:

- `blocks_spec_approval`: the spec cannot become downstream-ready until the issue is resolved.
- `reopens_specification`: `spec.md` must change before review can pass.
- `reopens_research`: missing evidence prevents an honest spec decision.
- `requires_user_decision`: the missing decision is external product, business, policy, or legal judgment.
- `accepted_risk_candidate`: the orchestrator may proceed only by naming the accepted risk and boundary.
- `proof_obligation`: the spec is coherent, but later artifacts must carry a named proof.
- `record_only`: useful context that does not affect downstream entry.

Record the review result in the active workflow-control surface: `workflow-plan.md`, `workflow-plans/specification-review.md` when a dedicated review phase needs durable routing, or the lean-local `spec.md` when no workflow-control artifact exists. The record must name the reviewed `spec.md`, reviewer or lanes, scope, lens coverage table, findings in the required shape, orchestrator resolution, final gate status, accepted risks, proof obligations, readiness consequence, and reopen target. Review subagents do not edit `spec.md`; if findings require content changes, route to specification repair and run a follow-up review after the repair.

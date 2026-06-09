# Technical Design Review Phase

Detailed phase companion for `docs/spec-first-workflow.md`. Read this when reviewing triggered technical design before planning.

## Read When

- Separate design depth was triggered and the design bundle is review-ready.
- Planning is waiting on technical-design-review status.
- A repaired design after `FAIL` needs a fresh or explicitly updated follow-up verdict.

## Inputs

- Specification-review-approved `spec.md`, design entrypoint, triggered design artifacts, and conditional `test-plan.md` or `rollout.md` when present.
- Workflow-control paths that define current phase, blockers, expected review result, and `Design fan-out` status.
- Relevant specialist outputs and `docs/repo-architecture.md` when boundaries matter.

## Outputs

- `PASS`, `CONCERNS`, or `FAIL` with findings, orchestrator resolution, planning-input obligations, accepted risks, proof obligations, and reopen target.

## Stop Rule

Keep technical design review read-only. Do not solve design defects inside review and do not start planning until the review status permits it.

## Technical Design Review

Technical design review is mandatory whenever separate design depth is triggered. It is the special pre-planning gate that tests whether the design bundle is coherent enough for executable planning.

This gate is not required for direct path work or for lean-local work whose design stays inside `spec.md` `Compact Design`; the inline `Risk Challenge` covers that smaller path. It is required when lean local creates a separate `design/overview.md`, and it is required for full-orchestrated triggered design.

Run technical design review after the design bundle and any triggered conditional artifacts are written, but before `tasks.md` or the task-ledger review/readiness handoff is approved.

If technical design review returns `FAIL`, the next action is a reopen of technical design or specification. After the repair, planning still waits for a follow-up technical design review verdict on the revised packet. The follow-up may be targeted to the failed findings and changed artifacts when the repair is narrow, but it must still check that adjacent design assumptions remain valid and record a new or explicitly updated gate status.

The review packet must be explicit enough that the reviewer does not rediscover phase state from scratch:

- specification-review-approved `spec.md`;
- design entrypoint and triggered design artifacts, with status and trigger rationale;
- triggered `test-plan.md`, `rollout.md`, or explicit not-expected rationale when those surfaces matter;
- workflow-control paths that define the current phase, blockers, and expected review result;
- known assumptions, accepted trade-offs, non-goals, and reopen conditions.

The review must be read-only and risk-driven:

- inspect specification-review-approved `spec.md`, the design bundle, triggered conditional artifacts, `docs/repo-architecture.md` when boundaries matter, and relevant specialist outputs;
- check source-of-truth ownership, dependency direction, runtime sequence, failure behavior, conditional artifact triggers, validation/rollout handoff, dependency/OSS due diligence, Pattern Fit Diligence, and accidental complexity;
- separate design defects from implementation preferences;
- identify any live fork where two plausible design options would materially change ownership, interfaces, data shape, async or sync semantics, operability, rollout, or validation, and verify the design has selected one with a rejection reason for the other;
- challenge the design from the first safe implementation slice: ask whether planning can create executable tasks without adding architecture, ownership, contract, sequencing, rollout, or validation policy;
- choose the strongest justified gate status, avoiding both over-blocking on proof-only concerns and under-blocking on missing ownership, contract, sequencing, rollout, or validation decisions;
- explain why the status is not stronger or weaker, especially for `CONCERNS` versus `FAIL`;
- when recommending `FAIL`, name the smallest reopen target, the decision or artifact that must change, and the concrete condition that a follow-up review should verify;
- return findings as advisory evidence for orchestrator reconciliation.

Technical design review is not a second design pass. If a finding requires a new decision, rewrite of the design bundle, or changed approval boundary, route it back to technical design or specification instead of solving it inside review.

For lean local with one `design/overview.md`, a local read-only orchestrator review is acceptable only when a recorded local-only rationale explains why no independent review lane would materially improve correctness. The checkpoint and result must be recorded before `tasks.md`.

For full orchestrated triggered design, use at least one distinct read-only review lane. Add specialist lanes when independent API, data, security, reliability, observability, delivery, performance, or QA design risks are real. A design-integrator lane is the default fit when the hard part is coherence across specialist concerns. A local-only review requires an explicit scoped-down rationale and cannot be used when independent design questions remain.

When the design packet records a `Design fan-out rationale`, review that rationale first. Do not require retroactive specialist lanes solely because multiple domains are mentioned; require them only when the rationale misses an unresolved live fork, domain-owned decision, or planning-critical proof gap.

Review gate status:

- `PASS`: planning may start from the reviewed design context.
- `CONCERNS`: planning may start only with named accepted design risks and proof obligations.
- `FAIL`: planning must not start; reopen technical design or specification. Repair alone is not enough to enter planning; the revised packet needs a follow-up review verdict of `PASS` or `CONCERNS`.

Gate decision discipline:

- Use `PASS` only when the reviewer has tried to falsify the design against source-of-truth ownership, sequence/failure behavior, validation, rollout, dependency/OSS due diligence, Pattern Fit Diligence, and artifact-trigger expectations and found no planning blocker.
- Use `CONCERNS` only when the design is coherent enough for planning and the remaining risk can be carried as a named accepted risk or proof obligation without asking implementation to choose a missing design decision.
- Use `FAIL` when planning would have to choose between live design options, repair ownership or dependency direction, define missing contract/data/rollout/failure semantics, or resolve a spec/design contradiction.
- Use `record_only` or no finding for cleaner-code preferences unless the issue changes planning safety or production-readiness proof.

Classify findings by strongest planning impact:

- `blocks_planning`: planning would invent or hide an important decision if it started now.
- `reopens_design`: the design bundle must change before review can pass.
- `reopens_spec`: the approved problem frame, invariant, scope, or contract must change.
- `accepted_risk_candidate`: the risk may be accepted only if the orchestrator names the reason and boundary.
- `proof_obligation`: planning may proceed only if the obligation is carried into `tasks.md`, `test-plan.md`, or `rollout.md`.
- `record_only`: useful context that does not affect planning entry.

Record the review result in the active workflow-control surface: `workflow-plan.md`, `workflow-plans/technical-design-review.md` when a dedicated review phase needs durable routing, or the lean-local artifact that owns the review checkpoint. The record must name the reviewed packet, reviewer or lane, scope, findings, orchestrator resolution, final gate status, and planning-input obligations. Follow-up review after `FAIL` must also name the prior failed review, the revised artifacts or decisions, which blockers were closed, any remaining accepted risks or proof obligations, and the new gate status. `CONCERNS` is valid only when every accepted risk and proof obligation is named for planning. Post-code discovery of a missing required technical design review reopens the earlier phase instead of creating a new review artifact after implementation starts.

# Technical Design Review Phase

Detailed phase companion for `docs/spec-first-workflow.md`. Read this when reviewing triggered system/integration and Go code ownership design before test design or planning.

## Read When

- Separate design depth was triggered and the system/integration plus Go code ownership design packet is review-ready, or one checkpoint is explicitly not expected.
- Test design or planning is waiting on technical-design-review status.
- A repaired design after `FAIL` needs a fresh or explicitly updated follow-up verdict.

## Inputs

- Specification-review-approved `spec.md`, design entrypoint, triggered system/integration artifacts, triggered Go code ownership artifacts, and conditional `test-design` or `rollout.md` trigger decisions when present.
- Workflow-control paths that define current phase, blockers, expected review result, and checkpoint-scoped `Design fan-out` status.
- Relevant specialist outputs and `docs/repo-architecture.md` when boundaries matter.

## Outputs

- `PASS`, `CONCERNS`, or `FAIL` with findings, orchestrator resolution, test-design and planning-entry consequence, accepted risks, proof obligations, carrying destination, and reopen target.

## Stop Rule

Keep technical design review read-only. Do not solve design defects inside review and do not start test design or planning until the review status permits it.

## Technical Design Review

Technical design review is mandatory whenever separate design depth is triggered. It is the special pre-test-design and pre-planning gate that tests whether the system/integration and Go code ownership design packet is coherent enough for scenario design and executable planning.

This gate is not required for direct path work or for lean-local work whose design stays inside `spec.md` `Compact Design`; the inline `Risk Challenge` covers that smaller path. It is required when lean local creates a separate `design/overview.md`, and it is required for full-orchestrated triggered design.

Run technical design review after the triggered system/integration design, triggered Go code ownership design, and any triggered conditional artifacts or phase decisions are written or explicitly not expected, but before `test-plan.md`, `tasks.md`, or the task-ledger review/readiness handoff is approved.

If technical design review returns `FAIL`, the next action is a reopen of system/integration design, Go code ownership design, specification, or specification review according to the failed decision owner. After the repair, test design and planning still wait for a follow-up technical design review verdict on the revised packet. The follow-up may be targeted to the failed findings and changed artifacts when the repair is narrow, but it must still check that adjacent design assumptions remain valid and record a new or explicitly updated gate status.

The review packet must be explicit enough that the reviewer does not rediscover phase state from scratch:

- specification-review-approved `spec.md`;
- design entrypoint, triggered system/integration artifacts, triggered Go code ownership artifacts, and trigger rationale;
- triggered `test-design`, `rollout.md`, or explicit not-expected rationale when those surfaces matter;
- workflow-control paths that define the current phase, blockers, and expected review result;
- checkpoint-scoped `Design fan-out` records with candidate seams, lane table or local-only rationale, collapsed and escalation seams, fan-in outcome, and unresolved lane blockers or material conflicts;
- prior specification-review or design-authoring proof obligations that the design claims to consume;
- known assumptions, accepted trade-offs, non-goals, and reopen conditions.

The review must be read-only and risk-driven:

- inspect specification-review-approved `spec.md`, system/integration design, Go code ownership design, triggered conditional artifacts, `docs/repo-architecture.md` when boundaries matter, and relevant specialist outputs;
- verify that checkpoint-scoped `Design fan-out` is present and eligible before treating the design as review-ready;
- check source-of-truth ownership, dependency direction, runtime sequence, failure behavior, conditional artifact triggers, validation/rollout handoff, package/file ownership, source responsibility audit, focused responsibilities, cleanup/removal, test ownership, dependency/OSS due diligence, Pattern Fit Diligence, and accidental complexity;
- when system/integration design is triggered, falsify the system mechanism closure for each planning-critical mechanism: selected or preserved behavior, source-of-truth owner, affected runtime or failure branch, code-carrying constraint, rejected live alternative and closure rule, proof carrier, and reopen trigger must be present or explicitly not applicable with evidence;
- separate design defects from implementation preferences;
- identify any live fork where two plausible design options would materially change ownership, interfaces, data shape, async or sync semantics, operability, rollout, or validation, and verify the design has selected one with a rejection reason for the other;
- challenge the design from the first safe implementation slice: ask whether test design and planning can create scenario obligations and executable tasks without adding architecture, system behavior, ownership, package/file boundaries, contract, sequencing, failure behavior, rollout, validation policy, cleanup policy, or test ownership;
- choose the strongest justified gate status, avoiding both over-blocking on proof-only concerns and under-blocking on missing ownership, contract, sequencing, rollout, or validation decisions;
- explain why the status is not stronger or weaker, especially for `CONCERNS` versus `FAIL`;
- when recommending `FAIL`, name the smallest reopen target, the decision or artifact that must change, and the concrete condition that a follow-up review should verify;
- return findings as advisory evidence for orchestrator reconciliation.

Technical design review is not a second design pass. If a finding requires a new decision, rewrite of the design bundle, or changed approval boundary, route it back to the owning design checkpoint or specification instead of solving it inside review.

Use a simple discriminator for ownership. If the missing answer changes accepted scope, invariant, observable contract, source of truth, or approval boundary, it is specification-owned. If it changes service behavior, contracts, source of truth, runtime sequence, data/cache shape, failure behavior, rollout, validation handoff, or conditional artifact triggers inside the approved scope, it is system/integration-design-owned. If it changes package/file ownership, focused responsibility, dependency direction, local abstraction shape, cleanup/removal, generated/manual boundary, or test ownership without changing observable behavior, it is Go-code-ownership-design-owned. If it chooses scenario classes, proof levels, pass/fail observables, fail-before expectations, or quality gates after behavior and ownership are already approved, it is test-design-owned. If it only chooses task order, checkpoint placement, proof sequencing, or diff slicing after the design and test-design decisions are already complete, it is planning-owned.

Test-design and planning readiness are the core falsification tests. Reviewers should try to draft the first few scenario and task sources mentally, not to create `test-plan.md` or `tasks.md`, but to expose whether later phases would have to invent architecture. Test design or planning is not ready when scenarios or the ledger would need to decide any of these before they can name scenario sources, task sources, proof level, task order, evidence, or stop conditions:

- the authoritative owner, generated or hand-written source, package boundary, owner file, adapter, composition root, or source-of-truth surface;
- the selected runtime sequence, sync or async boundary, failure behavior, cleanup, retry/no-retry, degraded-mode, or rollback semantics;
- the selected data, contract, dependency, Pattern Fit, code ownership, local abstraction, cleanup/removal, test ownership, or conditional-artifact shape;
- the rejected alternative for a live fork that still has material planning consequences;
- the proof claim, carrying artifact, freshness or negative-proof rule, and reopen condition for an accepted design risk.

If any item above is missing, classify the finding as `blocks_planning`, `reopens_design`, or `reopens_spec`; do not downgrade it to a planning detail or proof obligation.

When the missing item belongs to system/integration design, cite the missing or contradictory system handoff field, not only the downstream planning symptom. A task-ledger symptom such as "planning cannot name proof owner" should point back to the absent mechanism, source-of-truth, rejected-alternative, proof-carrier, or reopen-trigger field that made planning unsafe.

For Go code ownership, use this smoke test before `PASS` or `CONCERNS`: can planning fill `Files:`, `Owner package/file:`, `Placement evidence:`, cleanup owner, and test owner from the design packet without opening source to make a design choice? If not, return `FAIL` to `go-code-ownership-design`.

For lean local with one `design/overview.md`, a local read-only orchestrator review is acceptable only when a recorded local-only rationale explains why no independent review lane would materially improve correctness. The checkpoint and result must be recorded before `tasks.md`.

For full orchestrated triggered design, use at least one distinct read-only review lane. Add specialist lanes when independent API, data, security, reliability, observability, delivery, performance, QA, package ownership, Go simplification, or code maintainability risks are real. A design-integrator lane is the default fit when the hard part is coherence across specialist concerns. A local-only review requires an explicit scoped-down rationale and cannot be used when independent design questions remain.

When the design packet records a `Design fan-out` rationale, review that rationale first. Do not require retroactive specialist lanes solely because multiple domains are mentioned; require them only when the rationale misses an unresolved live fork, domain-owned decision, or planning-critical proof gap.

Review the `Design fan-out` rationale as an approval input, not as ceremony:

- Confirm the record uses an eligible status: `complete`, valid `scoped_down`, eligible `local_only`, or `blocked` with the smallest unblock route.
- Confirm candidate seams cover the planning-critical frontier from `system-integration-design.md` and `go-code-ownership-design.md`, including system ownership, source of truth, sequence, failure behavior, rollout, validation, dependency/OSS, Pattern Fit, package/file ownership, cleanup/removal, and test ownership seams when those surfaces are plausible.
- For `complete`, verify each unresolved live fork or domain-owned design decision had a read-only lane, the lane result was reconciled, and no lane blocker or material severity conflict remains.
- For `scoped_down`, verify at least one needed lane ran when full-orchestrated, protected-domain, high-impact, or user-requested agent-backed design is in scope, and that omitted lanes have evidence-backed reasons why they cannot change design correctness, test-design readiness, or planning readiness.
- For `local_only`, verify the work is eligible for local-only authoring and the rationale lists candidate lanes, evidence checked, why each omitted lane cannot change readiness, and the seam that would reopen fan-out.
- For `blocked`, verify the review does not proceed to `PASS` or `CONCERNS`; route to the smallest unblock path.

Look for hidden live forks by comparing the reviewed spec, specification-review obligations, authoring fan-in, design alternatives, and not-expected conditional artifacts. Wording such as "either", "could", "may", "TBD", "future", or "implementation decides" is not automatically a blocker, but it is a prompt to ask whether two viable options would change ownership, interfaces, data shape, sequence, package/file placement, cleanup, operability, rollout, validation, tests, or proof. If yes, the design must select one option or reopen the owner before test design or planning.

Before assigning `PASS` or `CONCERNS`, first check the handoff and readiness criteria in `system-integration-design.md` and `go-code-ownership-design.md` for every triggered checkpoint. Missing artifact-trigger decisions, unresolved planning-critical seams, vague proof obligations, or generic source-of-truth ownership are `FAIL`, not planning concerns.

Invalid shallow-review passes include:

- accepting `not affected`, `covered by tests`, `use existing mechanism`, or `not expected` without a source evidence pointer and falsification handle;
- treating a rejected live alternative as closed without a closure rule or reopen trigger;
- accepting generic validation or rollout proof without a carrying artifact, owner phase, pass/fail signal, and reopen target;
- allowing test design or planning to decide source of truth, runtime sequence, failure policy, rollout mechanism, owner package/file, cleanup owner, proof owner, or test owner.

Separate design defects from implementation preferences:

- A design defect changes planning safety or production-readiness proof: missing ownership, ambiguous source of truth, unselected generated versus manual authority, unclear cross-service contract, missing data or failure semantics, missing package/file responsibility, unreviewed dependency/pattern choice, unresolved rollout or validation handoff, unclear cleanup/test ownership, or a conditional artifact trigger that is real but undecided.
- An implementation preference is a cleaner local name, exact task slicing, small refactor style, or proof sequence after the owning design decision is already complete. Record it only when useful; do not block planning on taste.
- A code-quality concern becomes design-level when it changes package/file responsibility, public or generated surface ownership, dependency direction, lifecycle/failure behavior, cleanup/removal, test ownership, or the ability to prove the accepted design. Optional naming or style remains `record_only`.
- If changing the recommendation would change a task source, owner file/package, import direction, cleanup task, proof owner, or reopen target, treat it as design-level. Otherwise classify it as `record_only` or no finding.

Review gate status:

- `PASS`: test design or planning may start from the reviewed design context.
- `CONCERNS`: test design or planning may start only with named accepted design risks and proof obligations.
- `FAIL`: test design and planning must not start; reopen system/integration design, Go code ownership design, specification, or specification review. Repair alone is not enough to enter test design or planning; the revised packet needs a follow-up review verdict of `PASS` or `CONCERNS`.

Gate decision discipline:

- Use `PASS` only when the reviewer has tried to falsify the design against source-of-truth ownership, sequence/failure behavior, validation, rollout, dependency/OSS due diligence, Pattern Fit Diligence, and artifact-trigger expectations and found no test-design or planning blocker.
- Use `CONCERNS` only when the design is coherent enough for test design or planning and the remaining risk can be carried as a named accepted risk or proof obligation without asking test design, planning, or implementation to choose a missing design decision.
- Use `FAIL` when test design or planning would have to choose between live design options, repair system or code ownership, repair dependency direction, define missing contract/data/rollout/failure semantics, decide package/file placement, invent missing source responsibility audit or rejected owner locations, decide cleanup/test ownership, or resolve a spec/design contradiction.
- Use `record_only` or no finding for cleaner-code preferences unless the issue changes planning safety or production-readiness proof.

Use this decision order:

1. If test design or planning would have to invent or choose a design/spec decision, the verdict is `FAIL`.
2. Otherwise, if remaining items are bounded accepted risks or proof obligations that test design or planning can carry without changing the design decision, the verdict is `CONCERNS`.
3. Otherwise, if the reviewed design survives the risk-driven checks above, the verdict is `PASS`.

Classify findings by strongest planning impact:

- `blocks_planning`: test design or planning would invent or hide an important decision if either started now.
- `reopens_design`: the design bundle must change before review can pass.
- `reopens_spec`: the approved problem frame, invariant, scope, or contract must change.
- `accepted_risk_candidate`: the risk may be accepted only if the orchestrator names the reason and boundary.
- `proof_obligation`: test design or planning may proceed only if the obligation is carried into `test-plan.md`, `tasks.md`, or `rollout.md`.
- `record_only`: useful context that does not affect planning entry.

Carry-forward proof obligations in review-ready form. A `CONCERNS` result must tell planning exactly where each concern goes:

```text
Finding or accepted risk | Carrying artifact | Required evidence | Freshness or negative proof | Reopen target if proof fails
```

Use `test-plan.md` for scenario and layered validation obligations, `tasks.md` for executable proof tasks, and `rollout.md` for deployment, migration, compatibility, rollback, failback, or operator readback proof. If the review cannot name the carrying artifact and required evidence without deciding more design, the status is `FAIL`, not `CONCERNS`.

When `FAIL` is the right result, choose the smallest reopen target that can change the failing fact:

- `specification` when the approved scope, invariant, observable contract, non-goal, accepted risk boundary, source-of-truth policy, or proof feasibility must change.
- `specification review` when a required specification-review verdict is missing, stale after spec repair, or contains unresolved blocking findings that the design relies on.
- `system-integration design` when the approved scope is sound but the system design bundle or authoring fan-out must decide or rewrite contracts, source of truth, dependency direction, sequence, data shape, failure behavior, rollout, validation handoff, Pattern Fit, dependency/OSS, conditional artifact triggers, or `Design fan-out`.
- `Go code ownership design` when the approved system behavior is sound but the code ownership bundle or authoring fan-out must decide or rewrite package/file ownership, focused responsibility, dependency direction, generated/manual boundary, local abstraction shape, cleanup/removal, test ownership, or `Design fan-out`.
- both `system-integration design` and `Go code ownership design` when the failed decision spans the two checkpoints; name both owners instead of using a generic umbrella target.
- `technical design review` only for a follow-up review after a repaired packet, stale review verdict, unresolved review-lane conflict, or incomplete review record when the design packet itself does not need a new authoring decision.
- `user/specialist decision` only when the missing fact belongs outside repository artifacts; record the exact question and the phase that must resume after the answer.

Do not route a `FAIL` to test design or planning when the failed fact belongs to specification or design. Test design can choose scenarios only after behavior is decided; planning can repair task coverage, ordering, and evidence mapping after design review and any triggered test design pass; neither phase can author missing architecture.

Record the review result in the active workflow-control surface: `workflow-plan.md`, `workflow-plans/technical-design-review.md` when a dedicated review phase needs durable routing, or the lean-local artifact that owns the review checkpoint. The record must name the reviewed packet, reviewer or lane, scope, findings, orchestrator resolution, final gate status, and test-design or planning-input obligations. When the review uses lanes, scoped-down review, or local-only review, include a `Subagent Gate Audit` or pointer with lane result summary, fan-in, gate result, readiness consequence, and unresolved conflicts or proof obligations. Follow-up review after `FAIL` must also name the prior failed review, the revised artifacts or decisions, which blockers were closed, any remaining accepted risks or proof obligations, and the new gate status. Include a closure table:

```text
Prior finding | Repair/evidence anchor | Rechecked areas | Closure status | Residual proof obligation/reopen target
```

Every technical-design-review record must include this test-design/planning-entry smoke test before test design or planning may start:

```text
Test-design/planning-entry test: pass | fail
Reason: <why test design or planning can or cannot draft scenarios/tasks without choosing architecture, system behavior, package/file ownership, source responsibility audit, sequencing, rollout, validation policy, cleanup, or test ownership>
Hidden design work: <none | finding id and reopen target>
```

A repair note from the design author is evidence for this table, not the verdict. The follow-up may be narrower than the original review only when the record explains why untouched areas cannot change test-design or planning readiness; otherwise rerun the affected review coverage before returning `PASS` or `CONCERNS`.

`CONCERNS` is valid only when every accepted risk and proof obligation is named for planning. Post-code discovery of a missing required technical design review reopens the earlier phase instead of creating a new review artifact after implementation starts.

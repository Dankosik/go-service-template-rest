---
name: planning-and-task-breakdown
description: "Turn specification-review-approved `spec.md` plus required compact or split design context and triggered test-design context into a dependency-ordered, verifiable draft or repaired `tasks.md` ledger for this repository, ready for the planning session's internal task-review/readiness checkpoint. Use after `spec.md` is stable, mandatory specification review is reconciled for non-trivial work, lean compact design is sufficient or triggered system/integration and Go code ownership artifacts are approved, mandatory technical design review is reconciled when separate design depth exists, triggered test design is approved or its owner trigger resolved it to `not_expected`, and required challenge gates are reconciled, whenever implementation should be driven from planning artifacts rather than improvised from the decision/design record. Reach for this when executable task order, checkpoints, or parallelism are not obvious. Skip unresolved architecture/API/data/security/reliability/package-ownership/test-design decisions and skip actual coding."
---

# Planning And Task Breakdown

## Trigger And Scope

Use this skill to turn a stable, reviewed decision bundle into one dependency-ordered, proof-bound `tasks.md` ledger for non-trivial `lean_local` or `full_orchestrated` implementation. Plan the accepted production-ready target state, required cleanup, generated/mirrored work, tests, checkpoints, validation, and any approved rollout obligations without restating the spec or design.

Current `SHAPE-DIRECT` work stays outside this skill with inline proof. If direct work grows to ledger scale, reclassify it instead of producing a planning waiver.

## Approved Input And Planning Boundary

Planning starts from specification-review-approved `spec.md` plus every artifact selected by the current design-depth and test-design decisions:

- lean `Compact Design` or approved `design/overview.md`, or triggered `design/system-integration.md`, `design/go-code-ownership.md`, and their required companion artifacts;
- current technical-design-review `PASS` or eligible `CONCERNS` when separate design depth exists;
- approved `test-plan.md` with `TD-*` scenarios when test design was triggered, or an explicit owner decision that it is `not_expected`;
- current rollout, dependency/OSS, Pattern Fit, generated-source, cleanup, and review proof obligations when triggered.

Stop if a required artifact is missing, stale, contradictory, `FAIL`, blocked, or still leaves architecture, contract, source-of-truth, package/file ownership, cleanup/test ownership, sequence, proof level, or rollout policy for implementation to choose. Do not create or repair `test-plan.md` during planning. Do not write production code, tests, migrations, or generated output.

## Planning Invariants

1. **Artifacts keep distinct ownership.** `spec.md` owns approved decisions, compact or split design owns technical context, `test-plan.md` owns triggered scenario design, and `tasks.md` owns executable order and evidence. Planning consumes; it does not recreate prior phases.
2. **Every task is traceable and executable.** Bind material work to `Source:` anchors, stable task IDs, one reviewable action, owner file/package or approved placement rule, dependencies, preserved constraints, and planned proof.
3. **Order follows authority and risk.** Put source-of-truth, schema/contract/generator inputs, dependency-establishing work, migrations, and proof-first seams before derived consumers. Use `[P]` only for truly disjoint work after shared prerequisites.
4. **Slices are small but complete.** Prefer vertical or invariant-preserving reviewable slices. Split broad “and” tasks, but keep tightly coupled edits together when separation creates an unsafe half-state.
5. **Proof is designed, not improvised.** Carry accepted `CONCERNS`, `TD-*` scenario IDs, fail-before expectations, generated drift, race/integration needs, and negative cleanup proof into the exact task or checkpoint that owes them.
6. **Target-state cleanup is part of readiness.** Missing in-scope legacy cleanup is a planning-readiness failure. Task removal/refactor, or record approved retention with owner, reason, proof, and exit condition; do not leave “cleanup later.”
7. **Implementation retains local coding freedom, not design freedom.** Name owner surfaces, invariants, forbidden regressions, and proof; do not prescribe harmless local syntax. Reopen the owning phase when a task would need to choose architecture, contract, pattern, dependency, or ownership.
8. **Planning stops before approval and coding.** Leave task-ledger review/readiness procedural state and verdict pending. `handoff_readiness=ready` points only to the named review session or actionable reopen; implementation remains unauthorized until that distinct gate records an eligible verdict.

Confirm Pattern Fit Diligence decisions are explicit for any approved architecture, workflow, integration, resilience, consistency, data-flow, or material abstraction shape. Confirm stdlib, repository-pattern, mature OSS, and custom-code due diligence is explicit for a new runtime dependency, custom infrastructure, or meaningful helper. Planning carries the selected constraints and proof; it does not redo or invent those choices.

## Symptom-Driven Reference Selector

Use this entrypoint as the router. Before loading a reference, state what planning choice it will change. Load at most one by default and more only for independent pressures, such as dependency order plus checkpoint/reopen wording. Repository workflow docs and approved task-local artifacts outrank examples.

| Symptom or decision pressure | Load | Behavior change |
| --- | --- | --- |
| Session boundaries, target-state versus phased work, or review/validation checkpoints are unclear. | [phase-strategy-examples.md](references/phase-strategy-examples.md) | Choose one target-state ledger with real risk stops instead of partial delivery or ceremony-only phases. |
| Task order, `[P]`, task schema, migrations, generated artifacts, or source-of-truth-first sequencing is unclear. | [dependency-ordered-task-ledgers.md](references/dependency-ordered-task-ledgers.md) | Derive a stable, owner-first executable ledger instead of false parallelism or derived-first work. |
| Tasks are broad, horizontal, vague, coupled, or difficult to review and prove in one session. | [task-sizing-and-slicing.md](references/task-sizing-and-slicing.md) | Produce bounded reviewable slices without inventing design or postponing integration. |
| Acceptance, proof commands, evidence fields, `TD-*` binding, or accepted concerns are vague. | [acceptance-criteria-and-proof-obligations.md](references/acceptance-criteria-and-proof-obligations.md) | State task-specific truths and matching proof instead of “run tests” or optimistic progress. |
| Goal contract, stop points, blockers, task-review handoff, or reopen targets need exact wording. | [checkpoints-and-reopen-conditions.md](references/checkpoints-and-reopen-conditions.md) | Separate successful completion from blocked stop and route missing decisions to their owning phase. |
| A draft looks plausible but may contain invented decisions, duplicate authority, false readiness, vague proof, or artifact misuse. | [planning-anti-patterns.md](references/planning-anti-patterns.md) | Falsify readiness with focused smell triage instead of checklist momentum. |

## Required Ledger Evidence

Before writing tasks, inspect the reviewed spec, review verdicts and obligations, required design ownership/sequence/source-of-truth surfaces, approved test scenarios, and actual repository command/generation surfaces. Record what must land first, what can run in parallel, what old surfaces must disappear or remain, and what condition reopens each earlier phase.

The ledger must contain, in proportion to scope:

- a compact Goal contract for resumable work: objective, successful completion condition, separate blocked-stop condition, read context, preserved constraints, progress/resume rule, and blocker behavior;
- stable checkbox IDs, dependency/checkpoint labels, owner surfaces, `Source:` anchors, implementation obligations, proof-first or waiver, planned verification, and evidence fields;
- checkpoint gates only where risk, dependency, review, rollout, or validation state changes;
- exact `TD-*` mappings when test design exists and explicit proof carriers when it does not;
- a legacy cleanup audit for replacement work and owner-first generation/sync order for derived artifacts;
- a final self-check showing every approved behavior, concern, pattern/dependency constraint, cleanup owner, test owner, validation/rollout obligation, and reopen condition has a task or explicit non-task disposition.

No ready ledger contains `TBD`, open questions, unresolved decision gates, hidden design work, generic `implement the feature`, vague owner placement, or proof broader/weaker than the task claim.

## Success, Escalation, And Stop Conditions

Success means `tasks.md` reaches `artifact_state=review_ready` through dependency-ordered, reviewable, source-traceable tasks; proof and checkpoints match risk; required cleanup and derived artifacts are owned; and the planning root can invoke internal task-review/readiness without re-planning. This authoring method returns to the planning root; it does not end the user session.

Escalate to specification, technical design, test design, or rollout ownership when its decision or approval is missing. Do not paper over the gap with a task. Missing runtime capability is not a valid `Ledger-review fan-out rationale:`; use the configured independent fallback before blocking instead of silently declaring local readiness.

Reject a ledger that duplicates the spec/design, invents protected-domain work, hides a missing owner or scenario, marks unsafe parallelism, postpones integration or cleanup, creates implementation-time design choices, or implies coding may start before task review.

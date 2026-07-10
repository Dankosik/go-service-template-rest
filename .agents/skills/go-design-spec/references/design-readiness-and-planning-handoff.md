# Design Readiness And Review Handoff

## Behavior Change Thesis
When loaded before technical design review or planning handoff, this file makes the model choose an honest readiness verdict with artifact status, accepted risks, review-gate needs, and reopen conditions instead of likely mistakes like "done enough" claims that force review or planning to rediscover unresolved design.

## When To Load
Load this when the symptom is readiness uncertainty: `design/overview.md` claims readiness, workflow-control artifacts need a handoff summary, technical design review or `planning-and-task-breakdown` is about to consume the bundle, blockers or accepted risks may affect review/planning, or `test-plan.md` and `rollout.md` expectations are unclear.

Do not load this to write `tasks.md`. This reference only checks whether the design handoff is honest.

## Decision Rubric
- Review-ready means required core artifacts are present, consistent, and `artifact_state=review_ready`; conditional artifacts use their typed expectation instead of a flat waiver or absence claim.
- Conditional artifacts must be either present with `artifact_expectation=expected` or absent with `artifact_expectation=not_expected`, `artifact_state=absent`, `waiver_disposition=none`, and trigger evidence. For contract surfaces, this means `design/contracts/` is review-ready, compact contract design is explicitly sufficient, or a trigger test proves REST/API, event, generated, and material internal-interface shape is unchanged.
- `design/overview.md` should make that status scannable in its artifact index when technical design review is the next session, instead of forcing review or planning to rediscover conditional triggers from separate files.
- Accepted risks must have proof obligations and reopen conditions.
- A planning-critical blocker cannot live only in chat.
- If planning would need to choose contract shape, package surfaces, owner files, source responsibility evidence, source-of-truth ownership, runtime ownership, rollout sequence, or validation strategy, design is not ready.
- The handoff summary should tell the next session where to start and where not to drift.

## Imitate

Technical design review handoff with artifact status, accepted risk, and reopen conditions:

```markdown
## Technical Design Review Handoff

Review must inspect:
- specification-review-approved `spec.md`
- `design/overview.md`
- `design/component-map.md`
- `design/sequence.md`
- `design/ownership-map.md`
- `design/data-model.md` because persisted state changes
- `rollout.md` because migration compatibility is release-critical

Typed conditional artifacts:
- `design/contracts/`: artifact_expectation=not_expected, artifact_state=absent, record_validity=current, waiver_disposition=none; trigger test found no REST/API, event, generated-contract, or material internal-interface change.
- `design/dependency-graph.md`: artifact_expectation=not_expected, artifact_state=absent, record_validity=current, waiver_disposition=none; dependency direction stays within existing package boundaries.
- `test-plan.md`: artifact_expectation=not_expected, artifact_state=absent, record_validity=current, waiver_disposition=none; validation obligations fit in `tasks.md`.

Accepted risks:
- Backfill duration is unknown; planning must include a verification checkpoint before contract cleanup.

- If review or planning cannot task the migration without inventing source-of-truth or rollback policy, reopen `system-integration-design`.
- If API behavior must change to expose migration state, reopen specification and API design.
```

Workflow-control update summary:

```markdown
phase_state: complete
session_boundary: reached
handoff_readiness: ready
Design artifacts: required core artifact_expectation=expected, artifact_state=review_ready, record_validity=current; data-model and rollout use their recorded typed states; contracts, dependency-graph, and test-plan use artifact_expectation=not_expected, artifact_state=absent, waiver_disposition=none with trigger evidence.
Technical-design-review procedural_gate_state: pending
Technical-design-review review_verdict: pending
Technical-design-review record_validity: current
Next session starts with: technical design review.
Stop rule: do not begin review, `tasks.md`, or implementation in this session.
```

## Reject

Handoff with hidden blockers:

```markdown
Technical design review can start. Open issue: we still need to decide whether the worker or HTTP handler owns retries.
```

Why it is bad: retry ownership affects sequence, ownership, and reliability. It blocks planning.

Handoff that makes planning rediscover design:

```markdown
Review or planning should inspect the repo and decide which packages change.
```

Why it is bad: `design/component-map.md` or `design/go-code-ownership.md` should already identify the relevant packages, source responsibility evidence, rejected owner locations, and approved placement rule when exact files are design-time unknown.

Readiness claim without artifact status:

```markdown
Design is done enough.
```

Why it is bad: the next session needs artifact status, blockers, accepted risks, and reopen conditions.

## Agent Traps
- `design/overview.md` says review-ready while triggered core artifacts are missing, draft, or contradictory.
- `workflow-plan.md` says current phase is technical design review or planning, but `workflow-plans/go-code-ownership-design.md` says design is blocked.
- `tasks.md` exists before every triggered design checkpoint is approved or explicitly resolved as `not_expected`; lean compact design must already be closed in its owning artifact, while `SHAPE-DIRECT` never enters this design/task chain.
- `tasks.md` is expected but design handoff leaves package surfaces, owner files, placement rules, source responsibility audit, or ownership unresolved.
- `tasks.md` is expected but design handoff leaves REST/OpenAPI resource shape, status/error semantics, retry/idempotency, async/freshness, compatibility, or generated/manual authority unresolved.
- `rollout.md` is not expected while a migration, backfill, mixed-version window, or failback rule is planning-critical.
- `test-plan.md` is not expected while validation obligations are too layered for `tasks.md`.
- A blocker is recorded only in chat, not in the design or workflow artifacts that the next session will read.

## Validation Shape
Before handoff, produce a compact typed readiness record: `phase_state`, required and conditional artifact expectation/lifecycle/validity/waiver fields, separate technical-design-review procedure and verdict fields, `session_boundary`, `handoff_readiness`, blockers, accepted risks, reopen conditions, next session start point, and stop rule. If any field changes correctness, ownership, rollout, or validation, block technical design review or planning.

## Escalation Rules
- Keep the owning design checkpoint open when a planning-critical decision, artifact, or contradiction remains unresolved.
- Route to specification when the missing decision changes scope, external behavior, or accepted risk.
- Route to a specialist when the missing detail is domain-owned and cannot be safely integrated from existing evidence.
- Route to technical design review only when separate design depth is triggered and every required design artifact is review-ready; compact lean design that resolves separate depth to `not_expected` does not create or waive the mandatory review gate.
- Route to planning only after mandatory technical design review is reconciled when separate design depth exists.
- If planning starts and exposes a missing design decision, reopen `system-integration-design` or `go-code-ownership-design` according to the missing decision owner instead of inventing the decision in `tasks.md`.

## Repo Pointers
- `docs/spec-first-workflow.md`: planning-entry gate, planning gate, implementation-readiness gate, session-boundary gate, and design-bundle rules.
- `.agents/skills/technical-design-session/SKILL.md`: technical-design-review handoff and phase-local stop condition.
- `.agents/skills/planning-and-task-breakdown/SKILL.md`: what planning consumes and what it must not invent.

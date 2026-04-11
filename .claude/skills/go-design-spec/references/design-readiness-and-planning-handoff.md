# Design Readiness And Planning Handoff

## Behavior Change Thesis
When loaded before planning handoff, this file makes the model choose an honest readiness verdict with artifact status, accepted risks, and reopen conditions instead of likely mistakes like "done enough" claims that force planning to rediscover unresolved design.

## When To Load
Load this when the symptom is readiness uncertainty: `design/overview.md` claims readiness, workflow-control artifacts need a handoff summary, `planning-and-task-breakdown` is about to consume the bundle, blockers or accepted risks may affect planning, or `test-plan.md` and `rollout.md` expectations are unclear.

Do not load this to write `plan.md` or `tasks.md`. This reference only checks whether the design handoff is honest.

## Decision Rubric
- Planning-ready means required core artifacts are present, consistent, and approved or explicitly waived by an eligible rationale.
- Conditional artifacts must be either present with reasons or marked not expected with reasons.
- Accepted risks must have proof obligations and reopen conditions.
- A planning-critical blocker cannot live only in chat.
- If planning would need to choose package surfaces, source-of-truth ownership, runtime ownership, rollout sequence, or validation strategy, design is not ready.
- The handoff summary should tell the next session where to start and where not to drift.

## Imitate

Planning handoff with artifact status, accepted risk, and reopen conditions:

```markdown
## Planning Handoff

Planning may start with:
- approved `spec.md`
- `design/overview.md`
- `design/component-map.md`
- `design/sequence.md`
- `design/ownership-map.md`
- `design/data-model.md` because persisted state changes
- `rollout.md` because migration compatibility is release-critical

Not expected:
- `design/contracts/`; public REST schema stays unchanged.
- `design/dependency-graph.md`; dependency direction stays within existing package boundaries.
- `test-plan.md`; validation obligations fit in `plan.md`.

Accepted risks:
- Backfill duration is unknown; planning must include a verification checkpoint before contract cleanup.

Reopen conditions:
- If planning cannot task the migration without inventing source-of-truth or rollback policy, reopen technical design.
- If API behavior must change to expose migration state, reopen specification and API design.
```

Workflow-control update summary:

```markdown
Technical design status: complete.
Design artifacts: required core approved; data-model and rollout approved; contracts, dependency-graph, and test-plan not expected.
Next session starts with: planning.
Stop rule: do not begin `plan.md`, `tasks.md`, or implementation in this session.
```

## Reject

Handoff with hidden blockers:

```markdown
Planning can start. Open issue: we still need to decide whether the worker or HTTP handler owns retries.
```

Why it is bad: retry ownership affects sequence, ownership, and reliability. It blocks planning.

Handoff that makes planning rediscover design:

```markdown
Planning should inspect the repo and decide which packages change.
```

Why it is bad: `design/component-map.md` should already identify the relevant packages or explain why exact files are design-time unknown.

Readiness claim without artifact status:

```markdown
Design is done enough.
```

Why it is bad: the next session needs artifact status, blockers, accepted risks, and reopen conditions.

## Agent Traps
- `design/overview.md` says planning-ready while required core artifacts are missing, draft, or contradictory.
- `workflow-plan.md` says current phase is planning, but `workflow-plans/technical-design.md` says design is blocked.
- `plan.md` exists before design readiness when no direct/local waiver was recorded.
- `tasks.md` is expected but design handoff leaves package surfaces or ownership unresolved.
- `rollout.md` is not expected while a migration, backfill, mixed-version window, or failback rule is planning-critical.
- `test-plan.md` is not expected while validation obligations are too layered for `plan.md`.
- A blocker is recorded only in chat, not in the design or workflow artifacts that the next session will read.

## Validation Shape
Before handoff, produce a compact readiness verdict: status, required artifacts, triggered and non-triggered conditional artifacts, blockers, accepted risks, reopen conditions, next session start point, and stop rule. If any field changes correctness, ownership, rollout, or validation, block planning.

## Escalation Rules
- Keep the technical-design phase open when a planning-critical decision, artifact, or contradiction remains unresolved.
- Route to specification when the missing decision changes scope, external behavior, or accepted risk.
- Route to a specialist when the missing detail is domain-owned and cannot be safely integrated from existing evidence.
- Route to planning only when required design artifacts are approved or explicitly waived by an eligible direct/local rationale.
- If planning starts and exposes a missing design decision, reopen technical design instead of inventing the decision in `plan.md` or `tasks.md`.

## Repo Pointers
- `docs/spec-first-workflow.md`: planning-entry gate, planning gate, implementation-readiness gate, session-boundary gate, and design-bundle rules.
- `.agents/skills/technical-design-session/SKILL.md`: planning handoff and phase-local stop condition.
- `.agents/skills/planning-and-task-breakdown/SKILL.md`: what planning consumes and what it must not invent.

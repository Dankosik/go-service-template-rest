# Design Readiness And Planning Handoff

## When To Load
Load this before marking a technical-design bundle planning-ready, especially when:
- `design/overview.md` claims readiness
- `workflow-plan.md` or `workflow-plans/technical-design.md` needs a handoff summary
- `planning-and-task-breakdown` is about to consume the bundle
- blockers, assumptions, accepted risks, or reopen conditions may affect planning
- triggered `test-plan.md` or `rollout.md` expectations are unclear

Do not load this to write `plan.md` or `tasks.md`. This reference only checks whether the design handoff is honest.

## Good Examples

Good planning handoff:

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

Good workflow-control update summary:

```markdown
Technical design status: complete.
Design artifacts: required core approved; data-model and rollout approved; contracts, dependency-graph, and test-plan not expected.
Next session starts with: planning.
Stop rule: do not begin `plan.md`, `tasks.md`, or implementation in this session.
```

## Bad Examples

Bad handoff with hidden blockers:

```markdown
Planning can start. Open issue: we still need to decide whether the worker or HTTP handler owns retries.
```

Why it is bad: retry ownership affects sequence, ownership, and reliability. It blocks planning.

Bad handoff that makes planning rediscover design:

```markdown
Planning should inspect the repo and decide which packages change.
```

Why it is bad: `design/component-map.md` should already identify the relevant packages or explain why exact files are design-time unknown.

Bad readiness claim without artifact status:

```markdown
Design is done enough.
```

Why it is bad: the next session needs artifact status, blockers, accepted risks, and reopen conditions.

## Contradictions To Detect
- `design/overview.md` says planning-ready while required core artifacts are missing, draft, or contradictory.
- `workflow-plan.md` says current phase is planning, but `workflow-plans/technical-design.md` says design is blocked.
- `plan.md` exists before design readiness when no direct/local waiver was recorded.
- `tasks.md` is expected but design handoff leaves package surfaces or ownership unresolved.
- `rollout.md` is not expected while a migration, backfill, mixed-version window, or failback rule is planning-critical.
- `test-plan.md` is not expected while validation obligations are too layered for `plan.md`.
- A blocker is recorded only in chat, not in the design or workflow artifacts that the next session will read.

## Escalation Rules
- Keep the technical-design phase open when a planning-critical decision, artifact, or contradiction remains unresolved.
- Route to specification when the missing decision changes scope, external behavior, or accepted risk.
- Route to a specialist when the missing detail is domain-owned and cannot be safely integrated from existing evidence.
- Route to planning only when required design artifacts are approved or explicitly waived by an eligible direct/local rationale.
- If planning starts and exposes a missing design decision, reopen technical design instead of inventing the decision in `plan.md` or `tasks.md`.

## Repo-Native Sources
- `docs/spec-first-workflow.md`: planning-entry gate, planning gate, implementation-readiness gate, session-boundary gate, and design-bundle rules.
- `.agents/skills/technical-design-session/SKILL.md`: planning handoff and phase-local stop condition.
- `.agents/skills/planning-and-task-breakdown/SKILL.md`: what planning consumes and what it must not invent.

## Source Links Gathered Through Exa
- arc42 architecture decisions: https://docs.arc42.org/section-9/
- arc42 quality requirements: https://docs.arc42.org/section-10/
- arc42 risks and technical debt: https://docs.arc42.org/section-11/
- ISO/IEC/IEEE 42010 architecture descriptions: http://www.iso-architecture.org/ieee-1471/ads/
- Michael Nygard, "Documenting Architecture Decisions": http://thinkrelevance.com/blog/2011/11/15/documenting-architecture-decisions


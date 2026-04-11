# Design Bundle Assembly

## When To Load
Load this when the task needs help shaping the integrated `design/` bundle itself:
- writing or repairing `design/overview.md`
- deciding which required artifacts must exist
- deciding whether conditional artifacts such as `design/data-model.md`, `design/dependency-graph.md`, `design/contracts/`, `test-plan.md`, or `rollout.md` are genuinely triggered
- preventing `spec.md`, `design/`, `plan.md`, and `tasks.md` from absorbing each other's jobs

Do not load this when the question is only a domain-specific decision. Use the specialist skill for that domain and bring the result back into this integrator pass.

## Good Examples

Good `design/overview.md` entrypoint:

```markdown
# Design Overview

## Chosen Approach
The feature follows the existing HTTP request path: OpenAPI contract -> generated `internal/api` bindings -> `internal/infra/http` adapter -> `internal/app/orders` use case. The repository-wide dependency direction stays unchanged.

## Artifact Index
- `design/component-map.md`: package responsibilities and stable surfaces.
- `design/sequence.md`: create-order request flow, validation failure, dependency timeout, and persistence failure.
- `design/ownership-map.md`: OpenAPI, app behavior, migration, and adapter ownership.
- `design/data-model.md`: expected because persisted order state changes.
- `rollout.md`: expected because migration compatibility and backfill order affect release safety.
- `test-plan.md`: not expected; validation fits in `plan.md`.

## Readiness Summary
Planning may start after the persistence failure policy is confirmed in `design/sequence.md`. No implementation tasks are defined here.
```

Good conditional-artifact note:

```markdown
`design/contracts/` is not expected. The public REST shape is unchanged and `api/openapi/service.yaml` remains the only runtime contract authority.
```

Good design-scope boundary:

```markdown
Planning must preserve the selected data ownership and failure policy, but execution order and task IDs belong in `plan.md` and `tasks.md`.
```

## Bad Examples

Bad overview that replaces `spec.md`:

```markdown
## Product Decisions
We will now support refunds, adjust customer-facing states, and allow partial settlement.
```

Why it is bad: final scope and behavior changes belong in `spec.md`; design can only consume already-approved decisions.

Bad overview that replaces planning:

```markdown
## Implementation Steps
T001 edit OpenAPI, T002 regenerate code, T003 add handler tests, T004 run migrations.
```

Why it is bad: task sequencing belongs to `planning-and-task-breakdown`.

Bad conditional-artifact sprawl:

```markdown
Create `data-model.md`, `dependency-graph.md`, `contracts/`, `test-plan.md`, and `rollout.md` for completeness.
```

Why it is bad: conditional artifacts need real triggers. Filler artifacts make planning rediscover what is actually relevant.

## Contradictions To Detect
- `spec.md` says no public contract change, but `design/contracts/` declares a new REST payload authority.
- `design/overview.md` says no persisted-state change, but `design/data-model.md` defines schema evolution.
- `design/sequence.md` introduces async work, but `design/ownership-map.md` has no owner for durable retries, DLQ, or reconciliation.
- `workflow-plan.md` says planning can start, but `design/overview.md` still lists planning-critical blockers.
- `rollout.md` is marked not expected while the design requires `expand -> backfill/verify -> contract`.
- `test-plan.md` is created even though validation obligations are small enough for `plan.md`.

## Escalation Rules
- Escalate to specification when the design needs a new behavior, scope, invariant, acceptance policy, or external product decision.
- Escalate to the relevant specialist when the open issue is primarily API, data, security, reliability, observability, delivery, or QA rather than integration.
- Escalate to `technical-design-session` when the session boundary, allowed writes, or workflow artifact status is unclear.
- Escalate to `planning-and-task-breakdown` only after the design bundle is stable; do not write plan/task content from this skill.
- Keep design blocked when a required artifact would be filler or when a contradiction changes correctness, ownership, rollout, or validation.

## Repo-Native Sources
- `docs/spec-first-workflow.md`, especially the design-bundle section and artifact ownership rules.
- `docs/repo-architecture.md`, especially stable component boundaries, source-of-truth ownership, dependency direction, and runtime flows.
- `.agents/skills/technical-design-session/SKILL.md` for the session wrapper, allowed writes, stop rule, and planning handoff.

## Source Links Gathered Through Exa
- arc42 building block view: https://docs.arc42.org/section-5
- arc42 runtime view: https://docs.arc42.org/section-6/
- arc42 architecture decisions: https://docs.arc42.org/section-9/
- arc42 quality requirements: https://docs.arc42.org/section-10/
- arc42 risks and technical debt: https://docs.arc42.org/section-11/
- C4 component diagram: https://c4model.com/diagrams/component
- C4 dynamic diagram: https://c4model.com/diagrams/dynamic
- Michael Nygard, "Documenting Architecture Decisions": http://thinkrelevance.com/blog/2011/11/15/documenting-architecture-decisions


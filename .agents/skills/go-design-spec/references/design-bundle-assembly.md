# Design Bundle Assembly

## Behavior Change Thesis
When loaded for a fuzzy or overgrown `design/` bundle, this file makes the model choose a minimal, indexed bundle with real artifact triggers instead of likely mistakes like filler artifacts, `spec.md` rewrites, or task sequencing inside design.

## When To Load
Load this when the symptom is bundle-shape confusion: `design/overview.md` is missing or bloated, required artifacts are unclear, conditional artifacts are being created "for completeness", or `spec.md`, `design/`, `plan.md`, and `tasks.md` are starting to absorb each other's jobs.

Do not load this when the question is only a domain-specific decision. Use the specialist skill for that domain and bring the result back into this integrator pass.

## Decision Rubric
- Core artifacts are expected for non-trivial design: `overview.md`, `component-map.md`, `sequence.md`, and `ownership-map.md`.
- Conditional artifacts need behavior-changing triggers: persisted state, dependency shape, contracts, layered validation, migration choreography, or release safety.
- `design/overview.md` is the entrypoint and artifact index. It should point to details, not repeat every artifact.
- `spec.md` owns final behavior, scope, invariants, and accepted risk. Design consumes those decisions.
- `plan.md` and `tasks.md` own sequencing, task IDs, checkpoints, and implementation ordering. Design may name planning constraints, not write the plan.
- If an artifact would be empty or generic, mark it not expected and say why.

## Imitate

Entrypoint that records selected approach, artifact index, and readiness without stealing other artifacts' jobs:

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

Conditional-artifact note with a real non-trigger:

```markdown
`design/contracts/` is not expected. The public REST shape is unchanged and `api/openapi/service.yaml` remains the only runtime contract authority.
```

Design-to-planning boundary:

```markdown
Planning must preserve the selected data ownership and failure policy, but execution order and task IDs belong in `plan.md` and `tasks.md`.
```

## Reject

Overview that replaces `spec.md`:

```markdown
## Product Decisions
We will now support refunds, adjust customer-facing states, and allow partial settlement.
```

Why it is bad: final scope and behavior changes belong in `spec.md`; design can only consume already-approved decisions.

Overview that replaces planning:

```markdown
## Implementation Steps
T001 edit OpenAPI, T002 regenerate code, T003 add handler tests, T004 run migrations.
```

Why it is bad: task sequencing belongs to `planning-and-task-breakdown`.

Conditional-artifact sprawl:

```markdown
Create `data-model.md`, `dependency-graph.md`, `contracts/`, `test-plan.md`, and `rollout.md` for completeness.
```

Why it is bad: conditional artifacts need real triggers. Filler artifacts make planning rediscover what is actually relevant.

## Agent Traps
- `spec.md` says no public contract change, but `design/contracts/` declares a new REST payload authority.
- `design/overview.md` says no persisted-state change, but `design/data-model.md` defines schema evolution.
- `design/sequence.md` introduces async work, but `design/ownership-map.md` has no owner for durable retries, DLQ, or reconciliation.
- `workflow-plan.md` says planning can start, but `design/overview.md` still lists planning-critical blockers.
- `rollout.md` is marked not expected while the design requires `expand -> backfill/verify -> contract`.
- `test-plan.md` is created even though validation obligations are small enough for `plan.md`.

## Validation Shape
Before handoff, prove the bundle by naming required artifacts, triggered conditional artifacts, non-triggered artifacts with reasons, unresolved blockers, and the next artifact owner. Do not claim readiness if the proof relies on chat-only context.

## Escalation Rules
- Escalate to specification when the design needs a new behavior, scope, invariant, acceptance policy, or external product decision.
- Escalate to the relevant specialist when the open issue is primarily API, data, security, reliability, observability, delivery, or QA rather than integration.
- Escalate to `technical-design-session` when the session boundary, allowed writes, or workflow artifact status is unclear.
- Escalate to `planning-and-task-breakdown` only after the design bundle is stable; do not write plan/task content from this skill.
- Keep design blocked when a required artifact would be filler or when a contradiction changes correctness, ownership, rollout, or validation.

## Repo Pointers
- `docs/spec-first-workflow.md`, especially the design-bundle section and artifact ownership rules.
- `docs/repo-architecture.md`, especially stable component boundaries, source-of-truth ownership, dependency direction, and runtime flows.
- `.agents/skills/technical-design-session/SKILL.md` for the session wrapper, allowed writes, stop rule, and planning handoff.

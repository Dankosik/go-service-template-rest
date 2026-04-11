# Spec Patterns Reference

## Behavior Change Thesis
When loaded for foreign-template pressure or coverage gaps, this file makes the model translate only behavior-changing coverage concerns into repo-native `spec.md` sections instead of copying PRD, BMAD, Spec Kit, Superpowers, or SDD headings wholesale.

## When To Load
Load this when the prompt mentions an external spec framework, the draft has template-shaped section sprawl, or the spec feels under-covered but not because of a narrower artifact-placement, depth, clarification, or validation problem.

## Decision Rubric
- Start from the repository shape: `Context`, `Scope / Non-goals`, `Constraints`, `Decisions`, `Open Questions / Assumptions`, `Plan Summary / Link`, `Validation`, `Outcome`.
- Borrow a concern only when it changes planning, design, acceptance semantics, or proof. Do not borrow a heading because a framework names it.
- Translate user journeys into affected actors or acceptance semantics only when they change behavior.
- Translate requirement lists into stable decisions plus validation hooks; do not preserve `FR-001` numbering unless it is already an external contract.
- Translate NFRs into `Constraints` only when they shape design or validation. Omit irrelevant categories.
- Translate architecture, roadmap, task, or rollout material into `design/`, `plan.md`, `tasks.md`, or `rollout.md`; keep only the stable decision or link in `spec.md`.
- Preserve rejected alternatives only when planning is likely to revive the wrong path without them.

## Imitate

Foreign prompt:

```text
Fill out personas, user journeys, FR/NFR sections, architecture, and sprint tasks for async usage export.
```

Repo-native translation:

```markdown
## Context
Admins need usage exports without synchronous request timeouts.

## Scope / Non-goals
- In scope: admin-only async export for usage data.
- Non-goal: self-serve user exports.

## Constraints
- Export access remains workspace-scoped.
- Export status must survive process restarts.

## Decisions
- Export jobs use `queued -> running -> ready -> failed` states.
- Each workspace may have one active export job at a time.

## Validation
- Tests prove workspace-scoped download access and restart-stable job status.
```

Copy this: the useful actor, state, scope, and proof concerns were retained, while personas, FR numbering, architecture, and sprint tasks were not.

## Reject

```markdown
## Personas
## User Journeys
## FR-001 Through FR-019
## NFR Catalog
## Architecture
## Sprint Plan
```

Failure: the spec now inherits a foreign artifact model and forces later phases to rediscover which parts are final decisions.

## Agent Traps
- Treating "coverage" as permission to add every possible section.
- Copying framework vocabulary into headings instead of translating the concern.
- Filling NFR categories with generic availability, performance, or security text that does not constrain this task.
- Keeping architecture or task breakdown because it was useful in the source prompt.
- Omitting non-goals after collapsing a foreign "future work" section.

---
name: spec-first-brainstorming
description: "Structure and de-risk feature requests before spec design in this repository's spec-first workflow. Use when starting new feature/refactor/behavior-change work and you need a clear problem frame, scope/non-goals, assumptions, and prioritized open questions before Phase 0. Skip when the task is active bug debugging, code review, implementation on an already approved coder plan, or a pure informational question without artifact changes."
---

# Spec-First Brainstorming

## Purpose
Create a clear, bounded, and reviewable pre-spec input package before specification design starts. Success means the task exits with an explicit `B0` decision (`pass` or `fail`) and a concrete handoff to `go-architect-spec` when ready.

Workflow position:
- this skill is expected after message-level routing (`M0`) classifies the request as `new_feature_or_behavior_change`
- successful `B0 pass` hands off directly to `go-architect-spec` for Phase 0 initialization

## Scope And Boundaries
In scope:
- normalize raw feature/refactor/behavior-change requests into a precise problem frame
- define scope, non-goals, constraints, and success criteria
- capture explicit `[assumption]` entries and their validation path
- seed prioritized open questions with owner and unblock condition
- decide pre-spec readiness via `B0` gate

Out of scope:
- architecture decisions owned by `*-spec` roles in Phase 1/2
- endpoint-level contract design, physical data modeling, or migration design
- security/reliability/observability hardening decisions
- code implementation, test implementation, or domain-scoped review

## Hard Skills
### Brainstorming Core Instructions

#### Mission
- reduce ambiguity before Phase 0 without leaking into architecture/design decisions
- produce a reliable start state for `specs/<feature-id>/00/10/80`
- fail fast with explicit blockers when readiness is not met

#### Default Posture
- be strict about scope and unknowns
- keep statements concrete and testable
- prefer explicit blockers over hidden assumptions
- do not solve design topics that belong to downstream `*-spec` skills

#### Problem Framing Competency
- rewrite the request into one concise problem statement
- separate requested outcome from proposed implementation
- define expected behavior change and affected actors/systems

#### Scope And Constraint Competency
- define in-scope and out-of-scope explicitly
- capture constraints from product, architecture, compliance, and delivery context
- flag scope conflicts early instead of carrying them into spec phases

#### Assumption And Unknown Competency
- mark every critical unknown as `[assumption]`
- for each assumption, add risk and a concrete validation path
- reject implicit assumptions hidden in narrative text

#### Open Question Seeding Competency
- produce a prioritized question list
- each question must include owner and unblock condition
- separate "nice to know" from "blocking for Phase 0"

#### B0 Gate Competency
- `B0 pass` only when:
  - problem and expected behavior change are unambiguous
  - scope and non-goals are conflict-free
  - critical unknowns are explicitly tracked as assumptions
  - open questions are prioritized
  - no hidden architecture decisions are made in brainstorming
- `B0 fail` when:
  - goals or boundaries are still ambiguous
  - critical constraints are unknown and not tracked
  - open questions are missing owner or unblock condition
- never report readiness without explicit gate outcome

#### Handoff Competency
- for `B0 pass`, prepare direct handoff to `go-architect-spec` with normalized input and priority questions
- for `B0 fail`, provide minimal next data needed to re-run brainstorming

#### Review Blockers For This Skill
- no explicit `B0` decision
- assumptions are present but not marked or not actionable
- open questions missing owner or unblock condition
- architecture decisions introduced during brainstorming
- output too generic to initialize `00/10/80` artifacts

## Working Rules
1. Confirm the request belongs to feature/refactor/behavior-change framing. If not, mark `not_applicable` and route to the appropriate skill path.
2. Normalize the request into one problem statement.
3. Extract goals, success criteria, non-goals, and hard constraints.
4. Build an explicit `[assumption]` register with risk and validation path.
5. Seed prioritized open questions with owner and unblock condition.
6. Check for early conflicts with current repository invariants and process guardrails.
7. Evaluate readiness against `B0`.
8. If `B0 pass`, prepare a Phase 0 handoff package for `go-architect-spec`.
9. If `B0 fail`, provide blockers and minimum required clarifications for re-entry.
10. If a feature folder already exists, ensure minimum updates are reflected in `specs/<feature-id>/00-input.md`, `10-context-goals-nongoals.md`, and `80-open-questions.md`.
11. Keep output process-level; defer architecture and domain design to downstream skills.

## Output Expectations
Use this section order:

```text
Problem
Scope
Constraints
Assumptions
Open Questions
B0 Decision
Handoff
```

Output rules:
- `B0 Decision` must be explicit: `pass` or `fail`
- `Problem` must include:
  - one normalized problem statement
  - business/user impact
  - success criteria
- `Handoff` must include one explicit status:
  - `Ready for Phase 0`
  - `Blocked before Phase 0`
- for `B0 pass`, `Handoff` must include:
  - normalized input package for Phase 0 start
  - prioritized questions for `go-architect-spec`
- for `B0 fail`, include minimum required clarifications to reach `pass`
- assumptions must be labeled as `[assumption]`

## Definition Of Done
- all required output sections are present
- `B0` decision is explicit and justified
- handoff package is actionable for `go-architect-spec`
- no architecture/contract decisions are made inside brainstorming

## Anti-Patterns
- mixing brainstorming with architecture or implementation decisions
- mixing active bugfix/debug workflow with feature brainstorming
- broad narratives without concrete scope boundaries
- "open questions" without owner or unblock condition
- marking readiness when critical ambiguity is still unresolved
- hiding blockers as implied assumptions

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when problem framing, scope boundaries, constraints, and blocker questions are unambiguous.

Always load:
- `docs/spec-first-workflow.md`:
  - read `2. Core Principles`, `3. Artifacts`, `Phase 0` section, and related gate criteria first
- `AGENTS.md`:
  - read dynamic loading policy and execution loop sections
- `docs/skills/spec-first-brainstorming-spec.md`
- if `specs/<feature-id>/` already exists:
  - `specs/<feature-id>/00-input.md`
  - `specs/<feature-id>/10-context-goals-nongoals.md`
  - `specs/<feature-id>/80-open-questions.md`

Load by trigger:
- API-shape or HTTP behavior uncertainty:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- SQL/cache/data consistency implications:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- service-boundary, interaction-style, or distributed-flow uncertainty:
  - `docs/llm/architecture/10-service-boundaries-and-decomposition.md`
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- resilience/degradation/rollout concerns:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- trust-boundary/auth concerns:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- testability or acceptance ambiguity:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`

Unknowns:
- if critical facts are missing, continue with bounded assumptions labeled as `[assumption]`
- unresolved blockers must be surfaced in `Open Questions` and reflected in `B0 Decision`

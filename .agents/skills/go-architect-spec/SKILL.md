---
name: go-architect-spec
description: "Design architecture-first specifications for Go services in a spec-first workflow. Use when planning new features, refactors, or behavior changes before coding and you need boundaries, decomposition, interaction style, consistency model, resilience assumptions, and an implementation-ready architecture plan. Skip when the task is a local code fix, low-level API/DB/security implementation, test-case authoring, or CI/container configuration."
---

# Go Architect Spec

## Purpose
Create a clear, reviewable architecture specification for Go service changes before implementation. Success means architecture decisions are explicit, defensible, and directly translatable into coding tasks. Keep workflow control in `docs/spec-first-workflow.md`; focus this skill on architecture expertise.

## Scope And Boundaries
In scope:
- define service or module boundaries, ownership, and dependency direction
- decide component decomposition and seams
- decide sync or async interaction style and command or event intent
- decide consistency model (local transaction, eventual consistency, outbox or saga frame)
- define resilience shape (failure domains, degradation, rollout safety)
- produce architecture deliverables that remove "decide later" gaps

Out of scope:
- endpoint-level API payload, status, and error details
- physical SQL modeling, DDL details, and migration scripts
- concrete cache key, TTL, and invalidation policies
- detailed security control catalog and hardening checklists
- detailed telemetry schemas, SLI or SLO targets, and alert thresholds
- concrete CI or CD pipeline and container runtime hardening setup
- detailed test matrix design
- benchmark or profile plans and performance tuning details

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and pass goal before drafting decisions. Keep decision scope aligned to that phase.
2. Set phase-specific output targets before drafting decisions:
   - Phase 0: `00/10/80` and skeleton readiness for `15..90`
   - Phase 1: `20/60/80/90`
   - Phase 2 and later: `20/60/80/90` plus impacted `30/40/50/55/70`
3. Load context using the dynamic loading rules in this file and stop loading when coverage is sufficient.
4. Frame the architecture problem: constraints, ownership boundaries, and non-negotiables.
5. For each non-trivial decision, evaluate at least two options and select one explicitly.
6. Assign a decision ID and owner for each major decision.
7. Record trade-offs and cross-domain impact (API, data, security, operability) for each selected option.
8. Mark missing critical facts as `[assumption]` and keep assumptions bounded.
9. If an uncertainty blocks a decision, record it in `80-open-questions.md` with owner, unblock condition, and next step.
10. Produce the required deliverables in the required structure.
11. Check internal consistency: no conflicts and no hidden architectural decisions deferred to coding.
12. Keep focus on architecture expertise and do not turn the output into workflow management notes.

## Architectural Decision Protocol
For every non-trivial architectural decision, document:
1. decision ID (`ARCH-###`) and current phase
2. owner role
3. context and problem
4. options (minimum two)
5. selected option with rationale
6. at least one rejected option with explicit rejection reason
7. trade-offs (gains and losses)
8. impact on API, data, security, and operability
9. risks and control mechanisms
10. reopen conditions
11. affected artifacts and linked open-question IDs (if any)

## Output Expectations
- Response format: architecture specification package with these artifacts.
- Phase-specific minimum artifacts:
  - Phase 0:
    - `00-input.md`: normalized problem statement, scope, non-goals, constraints, assumptions
    - `10-context-goals-nongoals.md`: context and success frame
    - `80-open-questions.md`: initial architecture blockers and owners
    - skeleton readiness for `15..90` according to `docs/spec-first-workflow.md`
  - Phase 1:
    - `20-architecture.md`: context and constraints, boundaries and ownership, dependency rules, interaction style, consistency choices, architecture risks and trade-offs
    - `60-implementation-plan.md`: architecture-safe implementation sequence with no hidden "decision later"
    - `80-open-questions.md`: architecture-only uncertainties and blockers
    - `90-signoff.md`: decisions accepted in the current pass with rationale and reopen criteria
  - Phase 2 and later:
    - `20-architecture.md`
    - `60-implementation-plan.md`
    - `80-open-questions.md`
    - `90-signoff.md`
- Conditional alignment artifacts (update when architecture decisions affect them):
  - `30-api-contract.md`: contract-level architecture implications only.
  - `40-data-consistency-cache.md`: consistency frame and data-boundary implications.
  - `50-security-observability-devops.md`: architecture-level security and operability constraints.
  - `55-reliability-and-resilience.md`: architecture-level timeout/retry/degradation/shutdown policy frame.
  - `70-test-plan.md`: architecture-driven test obligations only.
- Conditional artifact format for `30/40/50/55/70`:
  - include one explicit status per file: `Status: updated` or `Status: no changes required`
  - when `Status: no changes required`, add one sentence with justification and linked decision IDs
  - when `Status: updated`, list changed sections and linked decision IDs
- Language: match the user language when possible.
- Detail level: concrete and reviewable, with explicit decisions and explicit trade-offs.
- Constraint: do not drift into low-level implementation details that are out of scope.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.

Always load:
- `docs/spec-first-workflow.md`:
  - read only sections `2. Core Principles`, `3. Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only if a decision cannot be made without them
- `docs/project-structure-and-module-organization.md`:
  - read only sections relevant to boundaries, ownership, and dependency direction first
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/llm/architecture/10-service-boundaries-and-decomposition.md`

Load by trigger:

Sync request-reply style, API hop rules, or deadline propagation decisions:
- `docs/llm/architecture/20-sync-communication-and-api-style.md`

Eventing, async workflows, queue semantics, or outbox/inbox decisions:
- `docs/llm/architecture/30-event-driven-and-async-workflows.md`

Cross-service consistency, saga choreography/orchestration, or compensation decisions:
- `docs/llm/architecture/40-distributed-consistency-and-sagas.md`

Failure-domain, degradation, startup/shutdown, retry budget, or rollout safety decisions:
- `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`

API surface impact:
- `docs/llm/api/10-rest-api-design.md`
- `docs/llm/api/30-api-cross-cutting-concerns.md`

Data, store, or caching impact:
- `docs/llm/data/10-sql-modeling-and-oltp.md`
- `docs/llm/data/20-sql-access-from-go.md`
- `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
- `docs/llm/data/50-caching-strategy.md`
- `docs/llm/data/30-nosql-and-columnar-decision-guide.md`

Security or identity impact:
- `docs/llm/security/10-secure-coding.md`
- `docs/llm/security/20-authn-authz-and-service-identity.md`

Operability or delivery impact:
- `docs/llm/operability/10-observability-baseline.md`
- `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
- `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- `docs/llm/delivery/10-ci-quality-gates.md`
- `docs/llm/platform/10-containerization-and-dockerfile.md`
- `docs/build-test-and-development-commands.md`
- `docs/ci-cd-production-ready.md`

Deep trade-off support:
- only when core loaded docs are insufficient for a disputed trade-off or when the user requests evidence
- use minimal additional repo sources and cite exact file names in decision rationale

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict still remains, preserve the latest accepted decision in `90-signoff.md` and record a reopen item in `80-open-questions.md` with owner and unblock condition.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- Phase 0 pass is complete when `00/10/80` are updated and skeleton readiness for `15..90` is confirmed.
- Phase 1 pass is complete when `20/60/80/90` are updated and consistent.
- Phase 2 and later pass is complete when `20/60/80/90` are updated and each affected `30/40/50/55/70` file has explicit status (`updated` or `no changes required`) with decision links.
- Architecture frame is internally consistent across all impacted artifacts.
- Every major decision includes decision ID, owner, selected option, and at least one rejected option with reason.
- No hidden architectural decisions are deferred to coding.
- Key trade-offs, risks, assumptions, and constraints are explicitly documented.
- Blockers are closed or explicitly recorded with clear owner and next step.
- Decisions are testable in review without reinterpretation.

## Anti-Patterns
- replacing technical position with generic workflow management
- making vague decisions without trade-off analysis
- pushing architectural uncertainty to coding phase
- mixing architecture scope with low-level implementation details
- copying requirements without explicit architectural choice
- loading full documents by default when section-level loading is enough

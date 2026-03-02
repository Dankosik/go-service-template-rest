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
1. Load context using the dynamic loading rules in this file and stop loading when coverage is sufficient.
2. Frame the architecture problem: constraints, ownership boundaries, and non-negotiables.
3. For each non-trivial decision, evaluate at least two options and select one explicitly.
4. Record trade-offs and cross-domain impact (API, data, security, operability) for each selected option.
5. Mark missing critical facts as `[assumption]` and keep assumptions bounded.
6. Produce the required deliverables in the required structure.
7. Check internal consistency: no conflicts and no hidden architectural decisions deferred to coding.
8. Keep focus on architecture expertise and do not turn the output into workflow management notes.

## Architectural Decision Protocol
For every non-trivial architectural decision, document:
1. context and problem
2. options (minimum two)
3. selected option with rationale
4. trade-offs (gains and losses)
5. impact on API, data, security, and operability
6. risks and control mechanisms
7. reopen conditions

## Output Expectations
- Response format: architecture specification package with these artifacts.
- `20-architecture.md`: context and constraints, boundaries and ownership, dependency rules, interaction style, consistency choices, architecture risks and trade-offs.
- `60-implementation-plan.md`: architecture-safe implementation sequence with no hidden "decision later".
- `80-open-questions.md`: architecture-only uncertainties and blockers.
- `90-signoff.md`: final architecture decisions, rationale, risk notes, and reopen criteria.
- Language: match the user language when possible.
- Detail level: concrete and reviewable, with explicit decisions and explicit trade-offs.
- Constraint: do not drift into low-level implementation details that are out of scope.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.

Always load:
- `docs/spec-first-workflow.md`
- `docs/project-structure-and-module-organization.md`
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/llm/architecture/10-service-boundaries-and-decomposition.md`
- `docs/llm/architecture/20-sync-communication-and-api-style.md`
- `docs/llm/architecture/30-event-driven-and-async-workflows.md`
- `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`

Load by trigger:

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
- relevant files from `docs/researches/`

Conflict resolution:
- The more specific document is the decisive rule for that topic.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.

## Definition Of Done
- Required artifacts are present and complete.
- Architecture frame is internally consistent across `20/40/50/55/60/70` when those files are part of the spec package.
- No hidden architectural decisions are deferred to coding.
- Key trade-offs and risks are explicitly documented.
- Assumptions and constraints are explicit.
- Blockers are closed or explicitly recorded with clear owner and next step.
- Decisions are testable in review without reinterpretation.

## Anti-Patterns
- replacing technical position with generic workflow management
- making vague decisions without trade-off analysis
- pushing architectural uncertainty to coding phase
- mixing architecture scope with low-level implementation details
- copying requirements without explicit architectural choice

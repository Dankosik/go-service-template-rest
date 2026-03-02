---
name: go-design-spec
description: "Design specification-integrity passes for Go services in a spec-first workflow. Use when a draft spec needs an integrated pre-coding pass to enforce simplicity, maintainability, and cross-artifact consistency across `15/20/30/40/50/55/60/70`. Skip when the task is a local code fix, endpoint/schema-only editing, implementation coding, code-review execution, or CI/container setup."
---

# Go Design Spec

## Purpose
Create a clear, reviewable design-integrity specification pass before implementation. Success means the spec package is coherent across artifacts, accidental complexity is controlled, and implementation can proceed without hidden design decisions.

## Scope And Boundaries
In scope:
- enforce design integrity across `15/20/30/40/50/55/60/70`
- identify and reduce accidental complexity (unnecessary layers, indirections, speculative abstractions)
- define maintainability-oriented constraints (locality of change, explicit seams, predictable impact radius)
- ensure implementation plan has no deferred system-level design decisions
- register design blockers and unresolved complexity risks with owners
- produce design decisions that are testable and reviewable in later phases

Out of scope:
- primary ownership of service decomposition and ownership boundaries
- primary ownership of endpoint-level API payload/status/error contracts
- primary ownership of physical SQL modeling, migration mechanics, and datastore selection
- primary ownership of cache key/TTL/invalidation policy and SQL access discipline
- primary ownership of security controls, observability SLI/SLO policy, and CI/CD container hardening
- primary ownership of performance budgets and benchmark protocol
- writing production code, writing tests, or performing code-review role tasks

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets:
   - Phase 1: perform architecture sanity-check and complexity baseline alignment.
   - Phase 2 and later: run integrated design pass on the full spec package and reconcile cross-artifact inconsistencies.
3. Load context using this skill's dynamic loading rules and stop when four design axes are source-backed: artifact consistency, complexity profile, maintainability constraints, and implementation readiness.
4. Normalize the design problem: where complexity grows, where ownership/terminology diverges, and where change impact becomes unpredictable.
5. For each nontrivial design decision, compare at least two options and select one explicitly.
6. Assign decision ID (`DES-###`) and owner for each major design decision.
7. Record trade-offs and cross-domain impact (architecture/API/data/security/operability/reliability/testing).
8. Preserve specialist ownership: express design constraints and integration decisions without replacing domain-specific decisions owned by other `*-spec` roles.
9. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate in current pass or move to `80-open-questions.md` with owner and unblock condition.
10. If uncertainty blocks coherent design closure, record it in `80-open-questions.md` with concrete next step.
11. Keep design outputs integration-first: resolve contradictions between artifacts before introducing new abstractions.
12. Verify internal consistency: no hidden design choices are deferred to coding.

## Design Decision Protocol
For every major design decision, document:
1. decision ID (`DES-###`) and current phase
2. owner role
3. context and complexity symptom
4. options (minimum two)
5. selected option with rationale
6. at least one rejected option with explicit rejection reason
7. trade-offs (`simplicity`/`flexibility`/cost/risk/change-impact)
8. cross-domain impact and affected artifacts
9. control measures and reopen conditions
10. linked open-question IDs (if any)

## Output Expectations
- Response format:
  - `Decision Register`: accepted `DES-###` decisions with rationale and trade-offs
  - `Artifact Update Matrix`: `20/60/80/90` and conditional `30/40/50/55/70` with `Status: updated|no changes required` and linked `DES-###`
  - `Assumptions`: active `[assumption]` items and resolution path
  - `Open Blockers`: unresolved design blockers for `80-open-questions.md` with owner and unblock condition
  - `Sign-Off Delta`: what must be appended to `90-signoff.md` in this pass
- Primary artifacts:
  - `20-architecture.md`:
    - design-integrity findings
    - simplification decisions
    - explicit complexity boundaries
  - `60-implementation-plan.md`:
    - complexity-safe sequencing
    - integration-risk reduction order
    - no hidden "decide later" design gaps
  - `80-open-questions.md`:
    - unresolved complexity/design blockers with owner
  - `90-signoff.md`:
    - accepted design decisions and reopen criteria
- Conditional alignment artifacts (update when impacted):
  - `30-api-contract.md`
  - `40-data-consistency-cache.md`
  - `50-security-observability-devops.md`
  - `55-reliability-and-resilience.md`
  - `70-test-plan.md`
- Conditional artifact status format for `30/40/50/55/70`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `DES-###`
  - for `updated`, list changed sections and linked `DES-###`
- Language: match user language when possible.
- Detail level: concrete and reviewable with explicit simplification choices and integration impacts.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when four design axes are covered with source-backed inputs: artifact consistency, complexity profile, maintainability constraints, and implementation readiness.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only when unresolved design decisions require them
- `docs/project-structure-and-module-organization.md`
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/llm/architecture/10-service-boundaries-and-decomposition.md`

Load by trigger:
- Sync request-reply and boundary interaction design implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
- Event-driven or async workflow coupling/complexity implications:
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
- Cross-service consistency and saga complexity implications:
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Degradation/startup-shutdown/rollback complexity implications:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- API-level simplicity and behavioral consistency impact:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Data/cache coupling and evolution impact:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Maintainability dispute or unclear simplification trade-off:
  - `docs/llm/go-instructions/70-go-review-checklist.md`
- Security/operability/delivery complexity impact:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/delivery/10-ci-quality-gates.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and add reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by source validation in current pass or by promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- Major design conflicts between impacted artifacts are resolved or explicitly tracked.
- Every major decision includes `DES-###`, owner, selected option, and at least one rejected option with reason.
- `20/60/80/90` are synchronized and contain no hidden design deferrals.
- Impacted `30/40/50/55/70` artifacts have explicit status with linked `DES-###`.
- Critical complexity risks are reduced with explicit rationale or tracked in `80-open-questions.md` with owner and unblock condition.
- No system-level design uncertainty is silently carried into coding phase.

## Anti-Patterns
Use these preferred patterns to avoid anti-pattern drift:
- reduce complexity through explicit scope/boundary choices with measurable simplification outcomes
- avoid speculative abstractions unless a concrete extension pressure is documented
- resolve cross-artifact contradictions before adding new structure
- keep design ownership clear and coordinate with specialist skills instead of duplicating their domain decisions
- track unresolved design uncertainty explicitly in `80-open-questions.md`

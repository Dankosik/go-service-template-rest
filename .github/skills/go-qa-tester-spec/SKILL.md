---
name: go-qa-tester-spec
description: "Design test-strategy-first specifications for Go services in a spec-first workflow. Use when planning or revising test strategy before coding and you need explicit unit/integration/contract/e2e-smoke test obligations, traceability to invariants and reliability fail-paths, quality-gate expectations, and an implementation-ready `70-test-plan.md`. Skip when the task is writing test code, reviewing a diff, fixing a local implementation bug, or making architecture/API/data/security decisions as a primary domain."
---

# Go QA Tester Spec

## Purpose
Create a clear, reviewable testing specification package before implementation. Success means testing obligations are explicit, defensible, and directly translatable into implementation tasks for `go-qa-tester` and validation in review.

## Scope And Boundaries
In scope:
- define test strategy for affected behavior (`unit`, `integration`, `contract`, `e2e-smoke`)
- define traceable test obligations for domain invariants and acceptance criteria
- define traceable fail-path obligations for reliability contracts (`timeout`, `retry`, `degradation`, `shutdown`)
- define contract-level test obligations for API behavior, idempotency, and error semantics
- define data and cache related testing obligations (consistency, migration compatibility, invalidation risks)
- define security and observability verification obligations when they affect correctness
- define quality-check expectations for implementation readiness
- produce test deliverables that remove hidden "decide later" gaps

Out of scope:
- writing production code or test code in the repository
- executing code-review duties of `go-qa-review`
- service decomposition, ownership topology, or architecture shape decisions as primary domain
- endpoint/resource design and full API semantics as primary domain
- SQL schema design or migration mechanics as primary domain
- security control catalog design as primary domain
- SLI/SLO target and alert policy design as primary domain
- CI/CD pipeline architecture and container hardening as primary domain

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting testing decisions.
2. Set phase-specific output targets:
   - Phase 0: record only critical testing assumptions/blockers in `80-open-questions.md`.
   - Phase 1: add architecture-shaping testability constraints and initial testing obligations.
   - Phase 2 and later: maintain full `70-test-plan.md`, sync impacted artifacts, and close test blockers.
3. Load context using this skill's dynamic loading rules and stop when four testing axes are source-backed: test levels, invariant coverage, fail-path coverage, and quality checks.
4. Normalize the testing problem: changed behavior, risk profile, trust boundaries, and retry/consistency semantics.
5. Choose the smallest sufficient test level first (`unit` -> `integration` -> `contract` -> `e2e-smoke`) and escalate only when lower levels cannot prove the requirement.
6. For each nontrivial testing decision, compare at least two candidate levels/approaches and select one explicitly.
7. Assign decision ID (`TST-###`) and owner for each major testing decision.
8. Record trade-offs and cross-domain impact (architecture, API, data, security, reliability, observability).
9. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate in the current pass or convert to blockers in `80-open-questions.md` with owner and unblock condition.
10. If uncertainty blocks test design quality, record it in `80-open-questions.md` with concrete next step.
11. Keep `70-test-plan.md` as primary artifact and synchronize impacted `15/30/40/50/55/60/90` sections.
12. Verify internal consistency: no contradictions between `70` and related artifacts, and no hidden test decisions deferred to coding.

## Test Decision Protocol
For every major testing decision, document:
1. decision ID (`TST-###`) and current phase
2. owner role
3. context and risk/invariant under test
4. options (minimum two for nontrivial cases)
5. selected option with rationale
6. at least one rejected option with explicit rejection reason
7. required scenarios (`happy path`, `fail path`, `edge cases`, plus `idempotency/retry/concurrency` where relevant)
8. preconditions, test data, and environment assumptions
9. pass/fail criteria and observable expected outcomes
10. traceability to decision IDs and spec artifacts
11. residual risks, coverage gaps, reopen conditions, linked open-question IDs (if any)

## Output Expectations
- Primary artifact:
  - `70-test-plan.md` with mandatory sections:
    - `Scope And Test Levels`
    - `Test-Level Selection Rationale`
    - `Traceability To Invariants And Decisions`
    - `Scenario Matrix (Happy/Fail/Edge/Abuse)`
    - `Reliability And Failure-Mode Coverage`
    - `Contract/API Coverage`
    - `Data/Cache Consistency And Migration Coverage`
    - `Security/Observability Verification Obligations`
    - `Quality Checks And Execution Expectations`
    - `Residual Risks And Reopen Criteria`
- Required core artifacts per pass:
  - `80-open-questions.md` with testing blockers/unknowns
  - `90-signoff.md` with accepted testing decisions and reopen criteria
- Conditional alignment artifacts (update when impacted):
  - `15-domain-invariants-and-acceptance.md`
  - `30-api-contract.md`
  - `40-data-consistency-cache.md`
  - `50-security-observability-devops.md`
  - `55-reliability-and-resilience.md`
  - `60-implementation-plan.md`
- Conditional artifact status format for `15/30/40/50/55/60`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `TST-###`
  - for `updated`, list changed sections and linked `TST-###`
- Language: match user language when possible.
- Detail level: concrete and reviewable with explicit scenario expectations and testability criteria.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when all four testing axes are covered with source-backed inputs: test levels, invariant coverage, fail-path coverage, quality checks.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only if unresolved testing decisions require them
- `docs/llm/go-instructions/40-go-testing-and-quality.md`

Load by trigger:
- Error behavior, timeout/cancellation contracts, and wrapped-error expectations:
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- API contract changes and retry/idempotency semantics:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Sync/async architecture and distributed workflow implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Data/migration/cache behavior changes:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Security-sensitive flows and negative-path requirements:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Quality gate and execution baseline alignment:
  - `docs/llm/delivery/10-ci-quality-gates.md`
  - `docs/build-test-and-development-commands.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and add reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by source validation in current pass or by promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- `70-test-plan.md` contains all mandatory sections from this skill.
- All major testing decisions include `TST-###`, owner, selected option, and at least one rejected option with reason.
- Every scenario in the matrix has test level, rationale, and explicit pass/fail criteria.
- Invariant and reliability fail-path coverage are explicitly mapped to `15` and `55`.
- Critical API/data/security/observability impacts are reflected as test obligations where relevant.
- Test blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- Impacted `15/30/40/50/55/60` artifacts have explicit status with decision links and no contradictions.
- No hidden testing decisions are deferred to coding.

## Anti-Patterns
- Prefer concrete scenario matrices with traceability over generic "add unit and integration tests" guidance.
- Prefer balanced coverage (`happy`, `fail`, `edge`, and risk-driven abuse paths) over happy-path-only planning.
- Keep strategy at specification level and separate from production implementation details.
- Add testing rationale when referencing architecture/API/data/security decisions.
- Assign an explicit owner and reopen condition for each nontrivial residual risk.
- Resolve critical testing decisions in spec artifacts or track them as blockers before coding starts.

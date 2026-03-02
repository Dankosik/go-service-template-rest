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

## Hard Skills
### QA Tester Spec Core Instructions

#### Mission
- Convert changed behavior into explicit, testable obligations before coding starts.
- Protect Gate `G2` quality by preventing hidden testing decisions and deferred risk.
- Ensure `70-test-plan.md` is execution-ready for `go-qa-tester` and review-ready for `go-qa-review`.

#### Default Posture
- Test strategy is risk-first and evidence-first, not checklist-first.
- Prefer the smallest test level that proves the requirement (`unit -> integration -> contract -> e2e-smoke`).
- Treat missing fail-path coverage as a blocker, not as optional polish.
- Treat untestable requirements as specification defects to be escalated, not coded around.
- Keep strategy repository-realistic: quality checks and environments must match project commands and CI gates.

#### Spec-First Workflow Competency
- Enforce `docs/spec-first-workflow.md` constraints for Phase 0/1/2 and target gate:
  - no coding-time test decisions hidden in implementation phase;
  - explicit synchronization between `70` and impacted `15/30/40/50/55/60/90`;
  - unresolved ambiguity tracked in `80-open-questions.md` with owner and unblock condition.
- Treat invariant and reliability coverage as mandatory for `G2`.
- Treat contradictions between artifacts as blockers until resolved or formally reopened.

#### Test-Level Selection Competency
- For each major risk, compare at least two candidate levels and document why one is rejected.
- Use level selection rules:
  - `unit`: deterministic logic and local invariants with no external boundary proof needed;
  - `integration`: DB/cache/network/process boundary behavior, transaction or timeout/cancel interactions;
  - `contract`: transport-visible semantics (status codes, headers, error model, idempotency behavior);
  - `e2e-smoke`: minimal critical-path confidence across composed runtime edges.
- Escalate level only when lower level cannot prove behavior with acceptable confidence.

#### Scenario Matrix Completeness Competency
- Every `TST-###` must include required scenario classes:
  - `happy path`
  - `fail path`
  - `edge cases`
  - `abuse/negative` where trust boundary or misuse risk exists
  - `idempotency/retry/concurrency` where side effects or parallelism exist
- Each scenario must have explicit preconditions, data shape, expected observable outcome, and pass/fail rule.
- Scenario outcomes must be externally meaningful (state change, HTTP/gRPC response, persisted effect, emitted message), not only internal branch execution.

#### Invariant And Acceptance Traceability Competency
- Map every critical invariant from `15-domain-invariants-and-acceptance.md` to explicit test obligations.
- Map each acceptance criterion to at least one proving scenario and level rationale.
- Distinguish:
  - local hard invariants requiring strict commit-time checks;
  - cross-service process invariants requiring eventual convergence and reconciliation evidence.

#### Reliability And Failure-Mode Competency
- Include mandatory test obligations for reliability contracts from `55-reliability-and-resilience.md`:
  - timeout/deadline propagation;
  - bounded retry behavior and no-retry conditions;
  - backpressure/load-shedding outcomes;
  - degradation mode transitions and fallback semantics;
  - graceful startup/shutdown behavior where relevant.
- Ensure retry/idempotency obligations are tied to explicit conflict semantics and duplicate-suppression behavior.
- For async/retryable flows, include failure classification coverage (`retryable`, `non-retryable`, `poison`) plus DLQ/escalation expectations where applicable.

#### Error And Context Competency
- Require test obligations that prove:
  - error contracts are explicit and inspectable (`%w`, `errors.Is/As` when relevant);
  - cancellation and deadline errors remain recognizable (`context.Canceled`, `context.DeadlineExceeded`);
  - request context is propagated instead of replaced with `context.Background()`;
  - derived-context cancel behavior is not leaked.
- Avoid brittle string-based error assertions unless exact text is part of external contract.

#### API Contract And Cross-Cutting Competency
- When API surface is affected, include contract-level obligations for:
  - method/status semantics and error model consistency (`application/problem+json` by default);
  - retry classification and idempotency-key behavior (`24h` dedup window by default unless overridden);
  - conflict semantics for same-key/different-payload;
  - async `202 + operation resource` behavior and completion state transitions;
  - request validation/normalization/input-limit behavior at boundary;
  - correlation/request ID propagation and visibility.
- Include negative-path obligations for rate limiting (`429`), payload-limit violations (`413/414/431`), malformed input, and auth context failures where relevant.

#### Data, Migration, And Cache Competency
- Define integration obligations for data correctness:
  - transaction boundary behavior and optimistic/pessimistic conflict handling;
  - deterministic pagination guarantees;
  - N+1/chatty query risk coverage where changed path is data-heavy.
- For schema evolution, include expand/backfill/contract compatibility obligations:
  - mixed-version compatibility window;
  - idempotent/resumable backfill behavior;
  - verification-gate expectations before contract phase.
- For cache-sensitive paths, include obligations for:
  - hit/miss/fallback correctness;
  - TTL and jitter expectations;
  - stampede protection/concurrency behavior;
  - fail-open degraded cache behavior;
  - tenant-safe keying and stale/negative-cache semantics.

#### Security And Identity Negative-Path Competency
- For trust-boundary changes, include mandatory negative scenarios for:
  - strict boundary validation and size limits;
  - authorization fail-closed and object-level checks;
  - tenant mismatch and cross-tenant denial;
  - invalid/forged/expired token handling;
  - SSRF/path/file-upload misuse controls when relevant.
- Ensure security obligations are verifiable at API boundary behavior, not only internal implementation assumptions.

#### Async And Distributed Consistency Competency
- For event-driven/distributed flows, include obligations for:
  - outbox/inbox-idempotency expectations;
  - dedup key semantics and replay safety;
  - ack-after-durable-state ordering expectations;
  - ordering-boundary assumptions and out-of-order tolerance;
  - compensation/forward-recovery and reconciliation-triggered correctness checks.
- Ensure cross-service consistency tests prove documented staleness and convergence contracts.

#### Quality Gates And Execution Competency
- Test strategy must specify project-executable validation path:
  - `make test`
  - `make test-race` when concurrency risk exists
  - `make test-integration` for boundary/integration obligations
  - `go vet ./...` / `make lint` when required by change scope
  - OpenAPI and migration checks when affected (`make openapi-check`, `make migration-validate`)
- Align local expectations with CI gate policy in `docs/llm/delivery/10-ci-quality-gates.md` and `docs/build-test-and-development-commands.md`.
- Do not define test obligations that cannot be executed in repository tooling/CI context.

#### Evidence Threshold Competency
- Every nontrivial decision requires `TST-###` with:
  1. owner and phase/gate context
  2. risk/invariant/contract under test
  3. at least two options
  4. selected option plus rejected option rationale
  5. scenario set with pass/fail observables
  6. artifact traceability (`15/30/40/50/55/60/70/80/90`)
  7. residual risk and reopen conditions
- Evidence must be specific enough that another engineer can implement tests without reinterpretation.

#### Assumption And Uncertainty Discipline
- Mark missing facts as bounded `[assumption]`.
- Resolve assumptions in current pass when possible; otherwise escalate to `80-open-questions.md` with owner/unblock condition.
- Never hide uncertainty in generic statements like "cover in tests later."

#### Review Blockers For This Skill
- `70-test-plan.md` missing required sections or missing `TST-###` rationale for major decisions.
- Happy-path-only planning without explicit fail/edge/abuse coverage.
- Missing traceability to critical invariants (`15`) or reliability contracts (`55`).
- Missing idempotency/retry/concurrency obligations where side effects exist.
- API/data/security/cache/distributed behavior changed without corresponding test obligations.
- Quality-check expectations not aligned with repository commands/CI gates.
- Critical test decisions deferred to coding without blocker tracking.

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting testing decisions.
2. Set phase-specific output targets:
   - Phase 0: record only critical testing assumptions/blockers in `80-open-questions.md`.
   - Phase 1: add architecture-shaping testability constraints and initial testing obligations.
   - Phase 2 and later: maintain full `70-test-plan.md`, sync impacted artifacts, and close test blockers.
3. Load context using this skill's dynamic loading rules and stop when four testing axes are source-backed: test levels, invariant coverage, fail-path coverage, and quality checks.
4. Apply `Hard Skills` defaults from this file; any deviation must be explicit in decision rationale or residual risks.
5. Normalize the testing problem: changed behavior, risk profile, trust boundaries, and retry/consistency semantics.
6. Choose the smallest sufficient test level first (`unit` -> `integration` -> `contract` -> `e2e-smoke`) and escalate only when lower levels cannot prove the requirement.
7. For each nontrivial testing decision, compare at least two candidate levels/approaches and select one explicitly.
8. Assign decision ID (`TST-###`) and owner for each major testing decision.
9. Record trade-offs and cross-domain impact (architecture, API, data, security, reliability, observability).
10. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate in the current pass or convert to blockers in `80-open-questions.md` with owner and unblock condition.
11. If uncertainty blocks test design quality, record it in `80-open-questions.md` with concrete next step.
12. Keep `70-test-plan.md` as primary artifact and synchronize impacted `15/30/40/50/55/60/90` sections.
13. Verify internal consistency: no contradictions between `70` and related artifacts, and no hidden test decisions deferred to coding.

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
12. explicit link to the governing `Hard Skills` competency for this decision (for example `Reliability And Failure-Mode Competency`)

## Output Expectations
- Response format:
  - `Decision Register`: accepted `TST-###` decisions with selected/rejected options and risk rationale
  - `Scenario Coverage Matrix`: `happy/fail/edge/abuse` coverage and chosen test levels
  - `Artifact Update Matrix`: required updates for `70/80/90` and status for impacted `15/30/40/50/55/60`
  - `Assumptions`: active `[assumption]` items and resolution path
  - `Open Blockers`: unresolved testing blockers with owner and unblock condition
  - `Sign-Off Delta`: what must be appended to `90-signoff.md`
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
Stop condition: stop loading when core testing axes are source-backed (test levels, invariant coverage, fail-path coverage, quality checks) and all triggered domain axes are source-backed (API, data/cache/migrations, security/identity, async/distributed consistency) for the current change scope.

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
- Each major `TST-###` is explicitly mapped to a governing `Hard Skills` competency.
- Every scenario in the matrix has test level, rationale, and explicit pass/fail criteria.
- Invariant and reliability fail-path coverage are explicitly mapped to `15` and `55`.
- Critical API/data/security/observability impacts are reflected as test obligations where relevant.
- Test blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- Impacted `15/30/40/50/55/60` artifacts have explicit status with decision links and no contradictions.
- No active `Review Blockers For This Skill` remain unresolved.
- No hidden testing decisions are deferred to coding.

## Anti-Patterns
- Generic guidance like "add unit and integration tests" without scenario matrix and pass/fail criteria.
- Happy-path-only planning without explicit fail/edge/abuse coverage rationale.
- Strategy text that mixes spec obligations with production implementation details.
- Cross-domain references (architecture/API/data/security) without explicit testing rationale.
- Residual risk entries without explicit owner and reopen condition.
- Deferring critical testing decisions to coding phase instead of resolving/tracking blockers in spec artifacts.

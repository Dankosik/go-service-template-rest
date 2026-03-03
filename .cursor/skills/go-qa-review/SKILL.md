---
name: go-qa-review
description: "Review Go code changes for test quality in a spec-first workflow. Use when auditing pull requests or diffs for test completeness against approved `70-test-plan.md`, assertion quality, suite determinism, and traceability to critical scenarios with spec-reopen escalation. Skip when designing specs, implementing code/tests, or performing architecture/security/performance/concurrency/DB reviews as the primary focus."
---

# Go QA Review

## Purpose
Deliver domain-scoped code review findings for test quality during Phase 4 review. Success means implemented tests are complete against approved test obligations, assertions are meaningful, flaky risks are controlled, and spec mismatches are escalated before `Gate G4`.

## Scope And Boundaries
In scope:
- review test implementation against approved `specs/<feature-id>/70-test-plan.md`
- verify required coverage exists for `unit/integration/contract` obligations in changed scope
- verify critical scenarios are tested, including required fail-path and edge-path cases from approved plan
- review assertion strength and failure diagnostics quality
- review test-suite determinism, isolation, and reproducibility signals
- produce actionable findings with exact `file:line`, impact, and fix
- escalate spec mismatches through `Spec Reopen`

Out of scope:
- redesigning test strategy during code review without explicit `Spec Reopen`
- editing spec artifacts in Phase 4
- performing primary-domain architecture, idiomatic style, domain invariant, performance, concurrency, DB/cache, reliability, or security review
- blocking PRs using subjective comments without concrete QA risk

## Hard Skills
### QA Review Core Instructions

#### Mission
- Protect `Gate G4` by proving changed behavior is covered with deterministic, meaningful tests mapped to approved obligations.
- Detect false-confidence test suites (happy-path-only, weak assertions, flaky timing dependence) before merge.
- Convert QA risks into the smallest safe corrective test changes without redesigning approved architecture/spec intent.

#### Default Posture
- Start from changed behavior and mandatory obligations in `specs/<feature-id>/70-test-plan.md`.
- Evaluate behavior protection quality, not test-count or line-coverage vanity metrics.
- Treat missing critical fail-path coverage as blocking until resolved or explicitly escalated.
- Keep QA-review domain ownership strict; use explicit handoff for primary non-QA issues.

#### Spec-First QA Workflow Competency
- Enforce Phase 4 constraints from `docs/spec-first-workflow.md`:
  - domain-scoped review comments only;
  - exact `file:line` anchors;
  - practical corrective action;
  - explicit `Spec Reopen` for spec-intent conflicts.
- Treat unresolved `critical/high` QA findings as merge blockers for `Gate G4`.
- Never change approved behavior implicitly via review comments.

#### Coverage And Traceability Competency
- Require explicit mapping from changed behavior to test obligations (prefer `TST-*` IDs when available).
- Verify required test layers (`unit/integration/contract`) are present for changed scope as defined in `70-test-plan.md`.
- Verify that required invariant and fail-path obligations from `15`/`55` are represented via approved test plan links.
- Flag orphan tests with no contract obligation and critical obligations with no validating tests.

#### Assertion Strength And Failure Diagnostics Competency
- Assertions must validate observable behavior/contract semantics, not only "no panic/no error".
- Require verification of key outputs, side effects, state transitions, and error class/shape where contract requires it.
- Require idiomatic error checks (`errors.Is`/`errors.As`) when wrapping semantics matter; reject brittle string matching unless contract-bound.
- Require failure output that localizes cause quickly (clear case names, targeted assertion messages, deterministic fixtures).

#### Determinism And Isolation Competency
- Flag tests that depend on uncontrolled time, random seeds, scheduling luck, shared mutable global state, or external nondeterministic systems.
- Flag sleep-based synchronization where deterministic coordination primitives are required.
- Require explicit control/isolation for time, randomness, environment variables, and external dependencies.
- Require `t.Parallel()` only when data isolation and side-effect safety are explicit.
- Require race-safety validation recommendation (`go test -race`/`make test-race`) when concurrent test/code paths are touched.

#### API Contract Scenario Competency (Trigger)
- For API contract changes, require coverage for:
  - method/status semantics (`200/201/202/204/4xx/5xx`) and contract-specific edge statuses;
  - `PUT` full-replacement and `PATCH` partial-update semantics, including unknown/immutable field handling;
  - deterministic pagination/filter/sort behavior and rejection of unsupported query options;
  - idempotency-key contract (`same key + same payload`, conflict on payload mismatch, required-key enforcement);
  - optimistic concurrency/preconditions (`ETag`, `If-Match`, `412`, `428`) where required;
  - async `202 + Location` flow and operation-state transitions where applicable;
  - stable `application/problem+json` error shape.

#### API Cross-Cutting Scenario Competency (Trigger)
- For cross-cutting API behavior changes, require coverage for:
  - boundary validation pipeline ordering and strict decode behavior (unknown fields/trailing JSON);
  - request size/transport guard semantics (`413/414/431`) where limits are contractually enforced;
  - auth context + tenant propagation + object-level authorization fail paths;
  - retry classification and rate-limit behavior (`429`, `Retry-After`) where relevant;
  - correlation/request ID propagation observability hooks when defined by contract;
  - upload/webhook/async cross-cutting guarantees when those surfaces are changed.

#### Data Modeling And SQL Access Scenario Competency (Trigger)
- For data/SQL-impacting changes, require coverage for:
  - DB-encoded invariants (constraints/uniqueness/nullability/fk assumptions) on affected behavior;
  - transaction semantics (`Commit`/`Rollback`) and conflict handling in changed flows;
  - optimistic-concurrency conflict paths where concurrent updates are possible;
  - context deadline/cancellation propagation across DB calls in request flows;
  - deterministic pagination behavior and tenant-isolation rules where applicable;
  - bounded query-path expectations for critical endpoints (avoid silent query-per-item regressions).

#### Migration And Evolution Scenario Competency (Trigger)
- For schema evolution/migration-impacting changes, require evidence for:
  - mixed-version compatibility expectations during rollout (`expand -> backfill -> contract`);
  - backfill idempotency/resumability and verification conditions where rollout depends on transformed data;
  - explicit handling of rollback limitations for destructive steps;
  - consistency-safe event publication expectations (no cross-system dual-write assumptions).

#### Cache Correctness And Degradation Scenario Competency (Trigger)
- For cache behavior changes, require tests for:
  - hit/miss/expired/evicted/negative/error/stale paths in scope;
  - fallback behavior on cache timeout/error (fail-open vs approved fail-closed exception);
  - stampede protection and bounded origin calls under concurrency;
  - cache-up and cache-degraded integration modes;
  - key-safety expectations (tenant/scope/version dimensions) when affected by change.
- Treat missing cache reliability coverage as a QA risk when cache behavior changed.

#### Security Negative-Case Test Competency (Trigger)
- For security-sensitive behavior, require negative-case coverage for:
  - strict input validation and size limits at boundary;
  - injection/SSRF/path-traversal/file-handling defenses when relevant;
  - sanitized client-facing error behavior and no sensitive leakage expectations;
  - abuse-resistance controls (timeouts/limits/concurrency guards) for expensive paths.
- Keep primary threat-depth ownership with `go-security-review`; QA role verifies test coverage presence/quality.

#### Performance And Concurrency Signal Competency (Trigger)
- When changed tests/code rely on concurrency primitives, verify deterministic coordination and race-check suitability.
- When performance claims justify test changes, require evidence path (benchmark/profile/trace) rather than speculative assertions.
- Hand off deep concurrency/performance correctness to `go-concurrency-review` / `go-performance-review`.

#### Command And Quality-Gate Competency
- Align verification recommendations with repository commands:
  - `make test`
  - `make test-race` when concurrency-sensitive behavior is touched
  - `make test-integration` when integration behavior changes
  - `make lint` and `go vet ./...` for quality baseline when relevant
  - `make openapi-check` when API contract/runtime behavior changes
- If commands were not executed in the review context, explicitly call out the verification gap in `Residual Risks`.

#### Evidence Threshold And Severity Calibration Competency
- Every finding must include:
  - exact `file:line`;
  - concrete missing/weak obligation (`70-test-plan.md`, prefer `TST-*`);
  - regression-leakage impact;
  - smallest safe fix path.
- Severity reflects merge risk, not style preference:
  - `critical`: missing critical obligations or systemic nondeterminism invalidating quality gates;
  - `high`: significant required branch/scenario gaps or assertions failing to validate required behavior;
  - `medium`: meaningful edge/fail-path or maintainability weakness with bounded near-term risk;
  - `low`: local diagnostic/readability improvements.

#### Assumption And Uncertainty Discipline
- If critical facts are missing, proceed with bounded `[assumption]` and reduced certainty.
- If required spec artifacts are unavailable, mark `[assumption: missing-spec-artifacts]`.
- Surface unresolved safety-impact assumptions in `Residual Risks` or escalate via `Spec Reopen`.

#### Review Blockers For This Skill
- Critical test obligations from approved `70-test-plan.md` are missing.
- Test behavior is systemically nondeterministic/flaky and undermines CI trust.
- Assertions are too weak to verify required behavior/contract outcomes.
- Required fail-path coverage for approved invariant/reliability/API/data/cache/security scenarios is absent in changed scope.
- QA-significant spec mismatch is observed but not escalated as `Spec Reopen`.

## Working Rules
1. Confirm review unit from context (`single task` or `bounded task scope`) and determine changed scope.
2. Determine `feature-id` from review context, changed paths, or task metadata. If it cannot be identified, continue with bounded `[assumption]` and reduced certainty.
3. Load review context using this skill's dynamic loading rules.
4. Apply `Hard Skills` defaults from this file; any deviation must be explicit in findings or residual risks.
5. Review tests first, then map findings to approved test obligations.
6. Evaluate five QA axes for changed scope:
   - `Coverage Conformance`
   - `Critical Scenario Verification`
   - `Assertion Quality`
   - `Stability And Determinism`
   - `Test Maintainability`
7. For touched trigger surfaces (API/data/cache/security/concurrency/performance), run corresponding QA scenario checks and record explicit handoff when deep analysis belongs to another reviewer skill.
8. Record only evidence-backed findings with concrete code location and specific obligation reference from `70-test-plan.md` (prefer `TST-*` or equivalent IDs when present).
9. Classify severity by merge risk for regression leakage (`critical/high/medium/low`).
10. Provide the smallest safe corrective action for each finding.
11. If fix requires spec-level decision change, create `Spec Reopen` in `reviews/<feature-id>/code-review-log.md`.
12. Keep comments strictly in QA-review domain and hand off cross-domain risks to the corresponding reviewer role.
13. If no findings exist, state this explicitly and include residual QA risks.

## Output Expectations
- Findings-first output ordered by severity.
- Match output language to the user language when practical.
- Use this exact finding format:

```text
[severity] [go-qa-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

- After findings, include:
  - `Handoffs`: cross-domain risks and owner skill.
  - `Spec Reopen`: `required` or `not required` with reason.
  - `Residual Risks`: non-blocking QA risks or verification gaps.
  - `Validation commands`: minimal command set to verify fixes and reduce residual uncertainty.
- Keep section order stable: `Findings`, `Handoffs`, `Spec Reopen`, `Residual Risks`, `Validation commands`.
- Keep all sections present; if a section is empty, write `none` and one short reason.
- If there are no findings, output `No QA findings.` and still include `Residual Risks` and `Validation commands`.

Severity guide:
- `critical`: required critical test obligations are missing, or test behavior is systemically non-deterministic and invalidates quality gates.
- `high`: significant missing coverage on required branches or assertions do not validate required behavior.
- `medium`: meaningful edge/fail-path or maintainability weakness with bounded short-term risk.
- `low`: local test readability/diagnostic improvements.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading once all five QA review axes and all touched trigger-scenario obligations can be evaluated with code evidence and approved spec references.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Phase 4`, `Reviewer Focus Matrix`, `Review Findings Format`, and `Gate G4` criteria first
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/go-instructions/40-go-testing-and-quality.md`
- review artifacts:
  - `specs/<feature-id>/70-test-plan.md`
  - `reviews/<feature-id>/code-review-log.md` (if present)

Load by trigger:
- Invariant-driven scenario obligations:
  - `specs/<feature-id>/15-domain-invariants-and-acceptance.md`
- Reliability/fail-path obligations:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
- API/contract scenario obligations:
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Data/consistency/cache scenario obligations:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Command and validation baseline:
  - `docs/build-test-and-development-commands.md`
- If test failures indicate concurrency/security/performance risks, load only relevant docs for accurate handoff:
  - concurrency: `docs/llm/go-instructions/20-go-concurrency.md`
  - security: `docs/llm/security/10-secure-coding.md`
  - performance: `docs/llm/go-instructions/60-go-performance-and-profiling.md`

Conflict resolution:
- Approved decisions in `specs/<feature-id>/90-signoff.md` override generic guidance unless `Spec Reopen` is raised.
- The more specific document is the decisive rule for that topic.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- If required spec artifacts are unavailable, mark `[assumption: missing-spec-artifacts]` and reduce certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated as `Spec Reopen`.

## Definition Of Done
- QA review output stays within QA-review domain boundaries.
- Findings are evidence-backed and use exact `file:line` references.
- Every finding has impact, fix path, and spec reference.
- All `critical/high` QA findings are either resolved or clearly escalated.
- No spec-level mismatch remains implicit.
- Output includes explicit `Validation commands` section aligned with changed scope.
- If no findings, output explicitly states `No QA findings.` and includes residual risk and validation notes.

## Anti-Patterns
- prioritize behavior validation quality over raw test-count checks
- tie every finding to a concrete regression-risk statement
- keep QA-domain ownership explicit and hand off deep cross-domain issues
- prefer the smallest safe test-layer correction before broader changes
- escalate unresolved critical test gaps through `Spec Reopen`
- omit verification command guidance after behavior-changing findings

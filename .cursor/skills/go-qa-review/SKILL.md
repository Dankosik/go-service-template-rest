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

## Working Rules
1. Confirm the task is code review and determine changed scope.
2. Determine `feature-id` from review context, changed paths, or task metadata. If it cannot be identified, continue with bounded `[assumption]` and reduced certainty.
3. Load review context using this skill's dynamic loading rules.
4. Review tests first, then map findings to approved test obligations.
5. Evaluate five QA axes for changed scope:
   - `Coverage Conformance`
   - `Critical Scenario Verification`
   - `Assertion Quality`
   - `Stability And Determinism`
   - `Test Maintainability`
6. Record only evidence-backed findings with concrete code location and specific obligation reference from `70-test-plan.md` (prefer `TST-*` or equivalent IDs when present).
7. Classify severity by merge risk for regression leakage (`critical/high/medium/low`).
8. Provide the smallest safe corrective action for each finding.
9. If fix requires spec-level decision change, create `Spec Reopen` in `reviews/<feature-id>/code-review-log.md`.
10. Keep comments strictly in QA-review domain and hand off cross-domain risks to the corresponding reviewer role.
11. If no findings exist, state this explicitly and include residual QA risks.

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
- Keep section order stable: `Findings`, `Handoffs`, `Spec Reopen`, `Residual Risks`.
- Keep all sections present; if a section is empty, write `none` and one short reason.
- If there are no findings, output `No QA findings.` and still include `Residual Risks`.

Severity guide:
- `critical`: required critical test obligations are missing, or test behavior is systemically non-deterministic and invalidates quality gates.
- `high`: significant missing coverage on required branches or assertions do not validate required behavior.
- `medium`: meaningful edge/fail-path or maintainability weakness with bounded short-term risk.
- `low`: local test readability/diagnostic improvements.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading once all five QA review axes can be evaluated with code evidence and approved spec references.

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
- If no findings, output explicitly states `No QA findings.` and includes residual risk note.

## Anti-Patterns
Use these preferred patterns to avoid anti-pattern drift:
- prioritize behavior validation quality over raw test-count checks
- tie every finding to a concrete regression-risk statement
- keep QA-domain ownership explicit and hand off deep cross-domain issues
- prefer the smallest safe test-layer correction before broader changes
- escalate unresolved critical test gaps through `Spec Reopen`

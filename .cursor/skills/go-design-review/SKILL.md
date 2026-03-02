---
name: go-design-review
description: "Review Go code changes for architecture and design integrity in a spec-first workflow. Use when reviewing implementation against approved specs and you need findings on architecture alignment, complexity control, and maintainability drift with spec-reopen escalation rules. Skip when planning specs before coding, writing code/tests, or performing deep domain reviews for performance, security, DB/cache, concurrency, or QA as primary focus."
---

# Go Design Review

## Purpose
Validate that code changes stay aligned with approved architecture and design decisions in spec-first workflow reviews. Success means actionable findings prevent architecture drift and unresolved spec conflicts before `Gate G4`.

## Scope And Boundaries
In scope:
- compare implementation with approved spec artifacts (`20/60` and relevant `15/30/40/50/55/70/90`)
- detect boundary violations, dependency-direction breaks, and hidden cross-layer coupling
- detect accidental complexity and maintainability regressions
- verify code does not introduce unapproved architecture-level decisions
- raise `Spec Reopen` when implementation requires spec-level changes
- produce domain-scoped findings with concrete `file:line`, impact, and fix

Out of scope:
- redesigning architecture from scratch without an explicit spec conflict
- editing any spec artifact during Phase 4 review
- deep primary-domain review for idiomatic/style, QA/testing, performance, concurrency, DB/cache, reliability, or security topics
- blocking changes by subjective preference without concrete design impact

## Working Rules
1. Confirm the task is a Phase 4 code review and identify the target feature and diff scope.
2. Determine `feature-id` from task context, changed paths, or review metadata. If `feature-id` cannot be identified, continue with bounded `[assumption]` and state reduced certainty.
3. Load approved feature artifacts and review context using dynamic loading rules from this skill.
4. Review changed code first, then map each risky change to relevant approved spec decisions.
5. Evaluate exactly five design axes:
   - `Architecture Compliance`
   - `Plan Conformance`
   - `Complexity Control`
   - `Maintainability`
   - `Spec Consistency`
6. Report only evidence-backed findings where changed code and spec reference are both explicit.
7. Classify each finding severity (`critical/high/medium/low`) by architecture integrity and merge risk.
8. For every finding, provide the smallest safe correction and explicit spec reference.
9. If a required correction changes approved spec decisions, record a `Spec Reopen` entry in `reviews/<feature-id>/code-review-log.md`.
10. Keep review comments strictly in design-review domain and route non-design concerns to the corresponding reviewer role.
11. If no findings exist, state that explicitly and include residual design risks or verification gaps.

## Output Expectations
- Present findings first, sorted by severity (highest first).
- Match output language to the user language when possible.
- Use this exact format for each finding:

```text
[severity] [go-design-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

- After findings, include:
  - `Open Questions`: unresolved design uncertainties only.
  - `Spec Reopen`: `required` or `not required` with reason.
  - `Residual Risks`: short list of remaining non-blocking design risks.
- If there are no findings, output `No design findings.` and still include `Residual Risks`.

Severity guide:
- `critical`: merge-blocking architecture or boundary violation, or change that invalidates approved design without reopen.
- `high`: significant architecture drift or complexity growth that materially raises regression or change cost risk.
- `medium`: maintainability issue that should be corrected but has bounded near-term risk.
- `low`: local design cleanup that improves clarity and future change safety.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when the five review axes can be assessed with concrete code evidence and at least one relevant approved spec source each.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Phase 4`, `Reviewer Focus Matrix`, `Review Findings Format`, and `Gate G4` sections first
- `docs/llm/go-instructions/70-go-review-checklist.md`
- review target artifacts:
  - `specs/<feature-id>/20-architecture.md`
  - `specs/<feature-id>/60-implementation-plan.md`
  - `specs/<feature-id>/90-signoff.md`
  - `reviews/<feature-id>/code-review-log.md` (if present)

Load by trigger:
- Invariant and acceptance behavior impact:
  - `specs/<feature-id>/15-domain-invariants-and-acceptance.md`
- API contract or cross-cutting API behavior impact:
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Data/consistency/cache seam impact:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Security/observability/delivery control impact:
  - `specs/<feature-id>/50-security-observability-devops.md`
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/delivery/10-ci-quality-gates.md`
- Reliability/failure/degradation behavior impact:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Testability obligations impact:
  - `specs/<feature-id>/70-test-plan.md`
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`

Conflict resolution:
- Approved feature decisions in `specs/<feature-id>/90-signoff.md` override generic guidance unless `Spec Reopen` is initiated.
- The more specific document is the decisive rule for that topic.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- If `specs/<feature-id>` artifacts are missing, continue with available approved sources and mark `[assumption: missing-spec-artifacts]`.
- Any unresolved assumption that can affect merge safety must be recorded as an open question or `Spec Reopen` candidate.

## Definition Of Done
- Output contains only design-review-domain findings with concrete `file:line` references.
- Every finding has explicit impact and actionable fix.
- Every finding links to a concrete spec reference.
- No spec-level conflict remains unmarked.
- All `critical/high` design conflicts are either resolved or escalated through `Spec Reopen`.
- If no findings, output explicitly states `No design findings.` and includes residual risks.

## Anti-Patterns
Use these preferred patterns to avoid anti-pattern drift:
- write findings as concrete architecture-impact statements, not abstract cleanliness advice
- keep recommendations aligned with approved spec decisions, or escalate through `Spec Reopen`
- keep comments in design-review domain and route deep domain checks to the corresponding reviewer
- include both `file:line` and `Spec reference` for every finding
- treat architecture alignment as a required gate even when tests are green

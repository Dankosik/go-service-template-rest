---
name: go-domain-invariant-review
description: "Review Go code changes for domain-invariant correctness in a spec-first workflow. Use when auditing pull requests or diffs for business-invariant preservation, state-transition correctness, and acceptance behavior conformance against approved domain specs with spec-reopen escalation. Skip when designing specifications, implementing features, or performing primary architecture/performance/concurrency/DB/reliability/security/QA reviews."
---

# Go Domain Invariant Review

## Purpose
Deliver domain-scoped code review findings for business-invariant correctness during Phase 4 review. Success means changed code preserves approved business invariants, forbidden transitions remain blocked, acceptance behavior stays consistent across happy-path and fail-path scenarios, and spec conflicts are escalated before `Gate G4`.

## Scope And Boundaries
In scope:
- review changed code against approved domain decisions in `specs/<feature-id>/15-domain-invariants-and-acceptance.md`
- verify invariant preservation across code paths and guard placement
- verify state-transition correctness (`allowed`, `forbidden`, preconditions, postconditions)
- verify acceptance behavior conformance (happy-path, fail-path, corner-case semantics)
- verify invariant-violation handling behavior and side-effect safety
- verify test traceability for critical invariants and transitions against `specs/<feature-id>/70-test-plan.md`
- detect unapproved domain behavior decisions and raise `Spec Reopen` when needed
- produce actionable findings with exact `file:line`, impact, and minimal safe fix

Out of scope:
- redesigning domain model/spec in Phase 4 without explicit `Spec Reopen`
- editing spec artifacts during code review
- deep primary-domain review for idiomatic/style, architecture integrity, performance, concurrency, DB/cache, reliability, security, or QA strategy
- blocking PRs using preference-only comments without concrete invariant impact

## Hard Skills
### Domain Invariant Review Core Instructions

#### Mission
- Protect merge safety by surfacing implementation changes that violate approved business invariants.
- Keep domain review evidence-based, traceable to approved spec artifacts, and bounded to Phase 4 ownership.
- Convert each confirmed invariant risk into the smallest safe corrective action.

#### Default Posture
- `15-domain-invariants-and-acceptance.md` is the primary source of truth for domain behavior.
- Treat hidden transition logic and implicit business assumptions as risks until proven aligned with approved decisions.
- Require observable behavior consistency, not only internal state shape consistency.
- Keep domain ownership strict; hand off deep non-domain root causes to the corresponding review skill.

#### Spec-First Review Competency
- Enforce Phase 4 rules from `docs/spec-first-workflow.md`:
  - domain-scoped findings only;
  - exact `file:line` anchors;
  - practical fix path;
  - explicit `Spec Reopen` for spec-intent conflicts.
- Treat unresolved `critical/high` domain findings as `Gate G4` blockers.
- Never redefine approved domain behavior implicitly through code-review commentary.

#### Invariant Preservation Competency
- Verify each affected critical invariant (`DOM-*`) remains enforced at runtime.
- Detect bypass paths where invariant guards can be skipped through alternative flows.
- Reject "eventual correction later" assumptions for hard business invariants without approved process-level guarantees.
- Require clear ownership and enforcement point alignment with approved domain decisions.

#### State Transition Correctness Competency
- Verify changed logic permits only approved transitions and blocks forbidden transitions.
- Verify preconditions are checked before side effects and postconditions are guaranteed after transition completion.
- Detect hidden transition paths created by retries, duplicates, or reordered operations.
- Treat incorrect transition guards as domain defects even when tests currently pass.

#### Acceptance Behavior Conformance Competency
- Validate externally observable behavior against approved acceptance criteria:
  - happy-path results,
  - fail-path behavior,
  - corner-case outcomes.
- Verify domain errors are deterministic and contract-consistent for invariant violations.
- Flag implementation behavior that changes domain semantics without approved sign-off.

#### Invariant Violation Semantics Competency
- Verify invariant violations produce predictable failure behavior and do not silently continue.
- Verify partial side effects are prevented, compensated, or explicitly handled according to approved behavior.
- Treat silent corruption, silent data loss, or inconsistent side-effect outcomes as blocker-level domain risk.

#### Corner-Case And Fail-Path Competency
- Evaluate retry, duplicate, reorder, delay, and dependency-failure paths when defined by spec.
- Verify these paths preserve domain correctness under realistic execution conditions.
- Flag gaps where fail-path handling can move entities into invalid or undefined domain states.

#### Test Traceability Competency
- Verify critical invariants and transition rules are mapped to concrete test obligations in `70-test-plan.md`.
- Flag gaps between required invariant checks and implemented tests.
- Require explicit coverage signals for fail-path and corner-case domain scenarios when behavior changed.

#### Spec Consistency And Reopen Competency
- Detect code-level domain decisions that are absent from approved `15/30/40/55/70/90` artifacts.
- If safe resolution needs spec changes, require `Spec Reopen` entry in `reviews/<feature-id>/code-review-log.md`.
- Do not resolve spec conflicts by ad hoc code-review compromise.

#### Cross-Domain Handoff Competency
- Route deep non-domain causes while preserving domain impact:
  - `go-db-cache-review` for transaction/query/cache paths affecting invariant preservation;
  - `go-reliability-review` for retry/timeout/degradation paths affecting invariant outcomes;
  - `go-security-review` for authz/tenant/object-ownership dependency of invariants;
  - `go-qa-review` for missing or weak invariant-test coverage;
  - `go-design-review` for architecture drift causing invariant risks.
- Keep handoffs concrete and limited to the minimum needed for ownership transfer.

#### Evidence Threshold And Severity Calibration Competency
- Every finding must include:
  - exact `file:line`;
  - violated invariant/transition/acceptance rule;
  - domain impact;
  - smallest safe corrective action;
  - explicit spec reference.
- Severity is based on domain-correctness risk and merge safety:
  - `critical`: confirmed critical invariant violation or forbidden transition allowed;
  - `high`: high-likelihood invariant break in fail-path/corner-case;
  - `medium`: bounded but meaningful domain behavior risk;
  - `low`: local traceability/clarity improvement with limited direct risk.

#### Assumption And Uncertainty Discipline
- Mark unknown critical facts as bounded `[assumption]`.
- If required artifacts are missing, mark `[assumption: missing-spec-artifacts]` and reduce certainty.
- Surface unresolved assumptions that affect merge safety in `Residual Risks` or `Spec Reopen`.

#### Review Blockers For This Skill
- Critical invariant is not enforced in changed runtime path.
- Forbidden state transition is possible in changed logic.
- Invariant violation behavior is inconsistent, silent, or side-effect unsafe.
- Acceptance behavior regresses against approved domain criteria in changed scope.
- Critical invariant/transition test obligations are missing for changed paths.
- Spec conflict exists but no `Spec Reopen` is initiated.

## Working Rules
1. Confirm review unit from context (`single task` or `bounded task scope`), then identify changed domain-sensitive code paths.
2. Determine `feature-id` from review context, changed paths, or task metadata. If unknown, continue with bounded `[assumption]`.
3. Load context using this skill's dynamic loading rules.
4. Apply `Hard Skills` defaults from this file; any deviation must be explicit in findings or residual risks.
5. Evaluate changed code in this order:
   - `Invariant Preservation`
   - `State Transition Correctness`
   - `Acceptance Behavior Conformance`
   - `Invariant Violation Semantics`
   - `Corner-Case And Fail-Path Coverage`
   - `Test Traceability`
6. Record only evidence-backed findings with explicit invariant/transition reference.
7. Classify severity by domain risk and merge safety (`critical/high/medium/low`).
8. Provide the smallest safe correction for each finding.
9. Keep findings strictly in domain-invariant ownership and route cross-domain depth to `Handoffs`.
10. If correction needs approved spec change, mark `Spec Reopen` as required and add rationale.
11. Do not edit spec files during Phase 4 review.
12. If no findings exist, state this explicitly and still include `Handoffs`, `Spec Reopen`, and `Residual Risks`.

## Output Expectations
- Findings-first output ordered by severity.
- Match output language to user language when practical.
- Use this exact finding format:

```text
[severity] [go-domain-invariant-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

- After findings, include:
  - `Handoffs`: cross-domain risks and owner skill.
  - `Spec Reopen`: `required` or `not required` with reason.
  - `Residual Risks`: non-blocking domain risks, assumptions, or verification gaps.
- Keep section order stable: `Findings`, `Handoffs`, `Spec Reopen`, `Residual Risks`.
- Keep all sections present; if empty, write `none` and one short reason.
- If there are no findings, output `No domain-invariant findings.` and still include `Residual Risks`.

Severity guide:
- `critical`: implementation allows critical invariant violation or forbidden transition with merge-blocking impact.
- `high`: strong evidence of major fail-path/corner-case invariant break risk.
- `medium`: meaningful but bounded domain inconsistency risk.
- `low`: local improvement for traceability or review clarity.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading once invariant preservation, transition correctness, acceptance conformance, and test traceability are assessable with code evidence and approved references.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Phase 4`, `Reviewer Focus Matrix`, `Review Findings Format`, and `Gate G4` first
- `docs/llm/go-instructions/70-go-review-checklist.md`
- review artifacts:
  - `specs/<feature-id>/15-domain-invariants-and-acceptance.md`
  - `specs/<feature-id>/70-test-plan.md`
  - `specs/<feature-id>/90-signoff.md`
  - `reviews/<feature-id>/code-review-log.md` (if present)

Load by trigger:
- API-visible acceptance behavior changed:
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Data consistency/cache path affects invariant safety:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Async/saga/reconciliation behavior affects invariant outcomes:
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Retry/timeout/degradation paths affect invariant outcomes:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Authz/tenant boundaries affect invariant enforcement:
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Invariant-test quality verification is needed:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`

Conflict resolution:
- Approved feature decisions in `specs/<feature-id>/90-signoff.md` override generic guidance unless `Spec Reopen` is initiated.
- The most specific approved artifact for the affected rule is decisive.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- If required feature artifacts are missing, mark `[assumption: missing-spec-artifacts]`.
- If uncertainty can change merge safety, include it in `Residual Risks` or escalate via `Spec Reopen`.

## Definition Of Done
- Critical invariants and transition rules were reviewed against approved `15` decisions.
- Findings are formatted with `file:line`, impact, suggested fix, and spec reference.
- All `critical/high` domain findings are either fixed or explicitly escalated through `Spec Reopen`.
- No untracked spec-level domain conflict remains.
- Review output remains strictly within domain-invariant scope.

## Anti-Patterns
Use these preferred patterns to avoid drift:
- keep findings tied to explicit invariant or transition rules, not generic quality comments
- include `file:line` and concrete domain impact in every finding
- avoid happy-path-only review; include fail-path and corner-case reasoning
- route deep cross-domain causes through explicit handoffs instead of scope expansion
- always escalate unresolved spec conflicts via `Spec Reopen`

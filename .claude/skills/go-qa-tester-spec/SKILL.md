---
name: go-qa-tester-spec
description: "Design test-strategy-first specifications for Go services. Use when planning or revising testing before coding and you need explicit unit, integration, contract, and e2e-smoke obligations, traceability to invariants and reliability fail-paths, quality-gate expectations, and an implementation-ready test strategy. Skip when the task is writing test code, reviewing a diff, fixing a local implementation bug, or making architecture/API/data/security decisions as the primary domain."
---

# Go QA Tester Spec

## Purpose
Turn changed behavior into explicit, risk-based test obligations before coding so that implementation and review do not need to invent coverage later.

## Specialist Stance
- Treat test strategy as risk selection and proof design, not a coverage checklist.
- Choose the smallest level that can honestly prove each invariant, contract, failure mode, or regression risk.
- Make scenarios executable, deterministic, and traceable to approved behavior.
- Hand off domain, API, data, reliability, security, or performance decisions when test obligations depend on unresolved semantics.

## Scope
Use this skill to define or review risk-based test strategy: level selection, scenario matrix, invariant traceability, fail-path obligations, contract coverage, and executable quality checks.

## Boundaries
Do not:
- write test code or review implementation details as the primary output
- default to broad test coverage when a smaller level can prove the behavior
- allow happy-path-only planning or untestable acceptance criteria
- define obligations that repository tooling or CI cannot actually run

## Escalate When
Escalate if critical invariants are not traceable to test obligations, side effects lack idempotency/retry/concurrency coverage, reliability behavior is unprovable, or the design is not testable without first changing the design itself.

## Core Defaults
- Test strategy is risk-first and evidence-first, not checklist-first.
- Prefer the smallest level that proves the requirement: `unit -> integration -> contract -> e2e-smoke`.
- Treat missing fail-path coverage as a blocker.
- Treat untestable requirements as design defects that must be escalated.
- Keep validation realistic: use repository commands and CI-compatible environments.

## Expertise

### Test-Level Selection
- Compare at least two candidate levels for each major risk when the right level is not obvious.
- Use level-selection rules:
  - unit for deterministic logic and local invariants
  - integration for DB/cache/network/process-boundary behavior
  - contract for transport-visible semantics
  - e2e-smoke for minimal critical-path confidence across composed runtime edges
- Escalate level only when a lower level cannot prove behavior with sufficient confidence.

### Scenario Matrix Completeness
- Every major risk needs explicit happy path, fail path, and edge-case scenarios.
- Add abuse/negative scenarios when trust boundaries or misuse risk exist.
- Add idempotency/retry/concurrency scenarios whenever side effects or parallelism exist.
- Every scenario should define preconditions, data shape, expected observable outcome, and pass/fail rule.
- Outcomes must be externally meaningful: response, persisted effect, emitted message, or visible state transition.

### Invariant And Acceptance Traceability
- Map every critical domain invariant to explicit test obligations.
- Map every acceptance criterion to at least one proving scenario and explain why the chosen level is sufficient.
- Distinguish local hard invariants from cross-service process invariants; the latter require convergence and reconciliation evidence.

### Reliability And Failure Modes
- Include timeout/deadline propagation, bounded retries, no-retry conditions, backpressure/load shedding, degradation, and graceful startup/shutdown where relevant.
- Tie retry/idempotency checks to explicit conflict semantics and duplicate-suppression behavior.
- For async flows, include `retryable`, `non-retryable`, and `poison` failure-class coverage plus DLQ or escalation expectations.

### Error, Context, And Contract Semantics
- Verify wrapped errors are inspectable where that matters.
- Keep cancellation and deadline errors recognizable.
- Verify request context is propagated rather than replaced.
- Avoid brittle string-based assertions unless exact text is part of the public contract.
- When API behavior changes, cover status codes, problem details, idempotency keys, conflict semantics, async `202 + operation resource` behavior, validation, limits, and request/correlation IDs.

### Data, Cache, Security, And Distributed Concerns
- Cover transaction behavior, optimistic/pessimistic conflicts, deterministic pagination, and N+1/chatty query risk when the change is data-heavy.
- For schema evolution, cover mixed-version compatibility, idempotent/resumable backfill behavior, and verification gates before destructive steps.
- For cache-sensitive behavior, cover hit/miss/fallback correctness, staleness, stampede protection, tenant-safe keying, and degraded cache behavior.
- For security-sensitive flows, cover strict validation, auth fail-closed behavior, tenant mismatch denial, invalid/expired credentials, and misuse paths.
- For distributed flows, cover outbox/inbox expectations, dedup semantics, replay safety, ack-after-durable-state behavior, ordering assumptions, compensation/forward recovery, and convergence/reconciliation.

### Quality Gates And Execution
- Express validation through real repository-executable checks such as unit tests, race tests, integration tests, lint/vet, contract checks, and migration validation when relevant.
- Keep local and CI expectations aligned.
- Do not define obligations that the repository cannot actually execute.
- Make residual risks, coverage limits, and reopen conditions explicit.

## Decision Quality Bar
For every major testing recommendation, include:
- the risk, invariant, or contract under test
- at least two viable levels or approaches when the choice is nontrivial
- the selected option and at least one explicit rejection reason
- scenario classes and pass/fail observables
- preconditions, data, and environment assumptions
- traceability to invariants, contracts, reliability behavior, and other affected domains
- residual risks, blockers, and reopen conditions

## Deliverable Shape
When writing the test strategy or review, cover:
- scope and chosen test levels
- level-selection rationale
- traceability to invariants and major decisions
- scenario matrix for happy, fail, edge, abuse, and retry/concurrency behavior
- reliability and failure-mode coverage
- API/contract coverage
- data, cache, security, and distributed-consistency coverage where relevant
- quality checks and execution expectations
- residual risks and reopen criteria

## Escalate Or Reject
- happy-path-only planning
- missing traceability to critical invariants or reliability contracts
- missing idempotency, retry, or concurrency coverage where side effects exist
- API, data, security, cache, or distributed behavior changed without matching test obligations
- quality-check expectations that do not match repository tooling or CI
- critical testing decisions deferred to implementation

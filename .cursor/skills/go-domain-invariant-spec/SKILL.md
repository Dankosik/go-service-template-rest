---
name: go-domain-invariant-spec
description: "Design domain-invariant-first specifications for Go services. Use when planning or revising behavior and you need explicit business invariants, state-transition rules, acceptance criteria, corner-case handling, and traceability into API, data, reliability, and testing concerns. Skip when the task is a local code fix, low-level implementation, endpoint schema-only design, SQL/migration mechanics, or CI/container setup."
---

# Go Domain Invariant Spec

## Purpose
Turn product behavior into explicit, falsifiable domain rules so that invariants, state transitions, and acceptance behavior are testable before implementation begins.

## Specialist Stance
- Treat the domain model as the source of allowed behavior, not a shadow of transport or storage shape.
- Name invariants, actors, states, transitions, side effects, and rejection rules explicitly.
- Prefer crisp acceptance semantics and forbidden-state handling over broad “business logic” prose.
- Hand off API encoding, data modeling, reliability, and test strategy when they become implementation of the domain decision rather than the decision itself.

## Scope
Use this skill to define or review business invariants, state-transition rules, acceptance criteria, corner cases, and the traceability of those rules into API, data, reliability, and testing decisions.

## Boundaries
Do not:
- jump into transport, schema, or infrastructure mechanics before the business rules are explicit
- allow ambiguous terms, hidden state transitions, or contradictory acceptance behavior
- collapse process invariants and local hard invariants into the same rule set
- leave edge cases or invalid transitions to implementation guesswork

## Escalate When
Escalate if core business terms are undefined, invariants conflict across actors or states, acceptance criteria cannot be made observable, or downstream design depends on unresolved domain semantics.

## Core Defaults
- Start from business invariants and lifecycle semantics first; map them to API, data, and reliability concerns second.
- Keep one explicit invariant register with owner and enforcement point for every critical rule.
- Classify invariants as `local_hard_invariant` or `cross_service_process_invariant`.
- Use explicit state machines for nontrivial lifecycles; do not rely on event-sequence prose.
- Treat invariant violations as explicit outcomes such as reject, deny, compensate, forward-recover, or manual intervention.
- Prefer compatibility-first invariant evolution across mixed-version rollout.

## Expertise

### Invariant Modeling And Ownership
- Define each critical invariant as one falsifiable rule with:
  - name
  - owner service
  - invariant type
  - enforcement point
  - observable pass/fail condition
- Require explicit source-of-truth ownership per invariant-related entity.
- Represent identity, tenant, and authorization constraints as domain invariants when they affect correctness.
- Allowed enforcement points include API validation, domain transition guards, persistence constraints, process step contracts, and reconciliation.
- Reject ownerless or descriptive-only invariants.

### State Transitions And Process Semantics
- Model lifecycles as states and transitions, not narrative prose.
- For each transition define:
  - trigger
  - preconditions
  - postconditions
  - allowed next states
  - forbidden next states
- Require monotonic, version-checked process transitions for distributed state.
- Require one active process instance per business key when concurrency can violate invariants.
- Make timeout and stuck-state behavior explicit.
- If compensation is impossible, mark pivot placement and forward recovery explicitly.

### Acceptance Criteria And Corner Cases
- Define acceptance as observable behavior, not as internal implementation hints.
- Cover happy path, forbidden path, fail path, and corner or edge conditions for every critical invariant.
- Include duplicate, replay, and out-of-order behavior when async processing is involved.
- Define idempotency conflict behavior where retries are possible:
  - same key + same payload => equivalent outcome
  - same key + different payload => explicit conflict
- For eventual reads, define freshness and staleness boundaries.
- For long-running side effects, require honest async acknowledgement semantics rather than fake immediate completion.

### Invariant Violation Semantics
- Map each violation to a deterministic outcome:
  - reject
  - deny
  - deferred async processing
  - compensate
  - forward-recover
  - manual intervention
- Keep one stable external error model per API surface.
- Never return success when an invariant check failed.
- Do not mask cancellation or timeouts as business success.
- Keep authorization, tenant, and object-ownership violations fail-closed.

### API, Distributed, And Persistence Alignment
- When invariants affect external behavior, make method semantics, status codes, idempotency, optimistic concurrency, and consistency disclosure explicit.
- For state changes that emit messages, require outbox-equivalent atomic linkage.
- Require consumer idempotency, durable dedup, bounded retries, and reconciliation for cross-service invariants.
- Use DB constraints for DB-enforceable invariants rather than app-only checks.
- Keep transaction boundaries local to one service-owned datastore.
- Require migration compatibility strategy and objective invariant verification before destructive steps.

### Cache, Identity, And Reliability Impact
- Treat cache as acceleration, not invariant authority.
- Define staleness contract before allowing cached data to influence invariant-sensitive decisions.
- Require tenant, scope, and version-safe keying when cache affects domain behavior visibility.
- Require explicit `AuthContext` expectations when identity influences domain invariants.
- Derive tenant from verified identity or trusted internal credentials, not caller-supplied headers.
- Classify dependency failure modes where invariant outcomes depend on downstream systems, and make degradation impact explicit.

### Test Traceability
- Every critical invariant should map to explicit positive, negative, and edge-case tests.
- Require contract tests where invariant behavior crosses API or async boundaries.
- Require replay/idempotency tests for async invariants.
- Keep timeout, cancellation, and fail-path semantics testable without reinterpretation.

## Decision Quality Bar
For every major domain recommendation, include:
- the business rule or lifecycle problem
- the invariant statement in falsifiable form
- at least two viable options when the design is nontrivial
- the selected option and at least one explicit rejection reason
- transition rules, violation behavior, and duplicate/replay semantics where relevant
- cross-domain impact on API, data, distributed design, reliability, security, and testing
- rollout compatibility, rollback limits, assumptions, blockers, and reopen conditions

## Deliverable Shape
When writing the domain behavior spec or review, cover:
- domain terms and scope
- invariant register
- state transition rules
- acceptance criteria
- corner cases and edge conditions
- invariant-violation semantics
- traceability into API, data, distributed, reliability, and testing concerns

## Escalate Or Reject
- a critical invariant without owner, enforcement point, or falsifiable pass/fail condition
- state transitions defined only as prose with no forbidden paths
- unspecified or contradictory invariant-violation behavior
- missing retry/idempotency semantics where repeated execution is possible
- cross-service invariants without outbox, idempotency, and reconciliation stance
- invariant-sensitive decisions based on stale projections with no freshness contract
- identity or tenant correctness implied but not explicitly enforced
- migration or rollout plans that can silently break invariant semantics

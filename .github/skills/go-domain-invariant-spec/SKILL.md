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
- When another domain is only affected, record the consequence as `constraint_only`, `proof_only`, or explicit `no new decision required` instead of widening the design.

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

## Source Authority
- Prefer the current task's product notes, `spec.md`, workflow artifacts, repository docs, and existing domain code as the source of truth for behavior.
- Keep this pass domain-decision-first. Do not choose transport status codes, SQL constraints, infra topology, or implementation mechanics until the business rule and violation semantics are explicit.
- Use general DDD, aggregate, state-machine, and idempotency knowledge only as background vocabulary. Do not import external sample domains as task decisions.

## Reference Loading
References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Load at most one reference by default: choose the narrowest file that matches the current symptom. Load a second only when the task clearly spans independent decision pressures, such as ambiguous domain vocabulary plus retry/replay semantics.

| Reference | Load For Symptom | Behavior Change |
| --- | --- | --- |
| `references/domain-language-and-boundaries.md` | Domain terms, actors, ownership, approval, "done", "active", tenant, or source-of-truth vocabulary is ambiguous before rules are written. | Makes the model define the local policy boundary before writing invariants instead of encoding vague nouns or transport/storage labels as business rules. |
| `references/invariant-register-patterns.md` | A spec needs invariant statements, owner assignment, source-of-truth authority, enforcement-point choices, or review of descriptive-only rules. | Makes the model write falsifiable owner-backed rules instead of broad "business logic" bullets with no pass/fail signal. |
| `references/state-machine-and-transition-rules.md` | Lifecycle states, phase boundaries, terminal states, transition guards, invalid transitions, timeout, or stuck-state behavior matters. | Makes the model define legal movement and forbidden paths instead of narrating event order or implementation progress. |
| `references/acceptance-criteria-and-corner-cases.md` | Domain rules exist but acceptance behavior, edge cases, or proof obligations are too vague for planning or QA handoff. | Makes the model produce observable positive, negative, and edge-case outcomes instead of happy-path prose or implementation hints. |
| `references/invariant-violation-semantics.md` | A rule says what must be true but not what happens when it is false. | Makes the model choose reject, deny, defer, compensate, recover, manual intervention, or accepted risk instead of returning false success or "handle error". |
| `references/idempotency-replay-and-async-domain-rules.md` | Retries, duplicate requests, replay, out-of-order events, async commands, side effects, eventual consistency, or reconciliation are in scope. | Makes the model define domain sameness, effect boundaries, and replay policy before transport keys, queues, or dedupe tables. |
| `references/api-data-reliability-test-traceability.md` | Stable domain decisions need downstream handoff into API, data, distributed, reliability, security, observability, or tests. | Makes the model preserve traceability from invariant IDs to downstream obligations instead of letting API/schema/retry/test details become competing sources of truth. |

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
  - same key + same domain intent or documented request fingerprint => equivalent outcome
  - same key + different domain intent => explicit conflict
  - in-progress duplicate => pending, retry-later, or conflict; never a second side effect
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
- For state changes that emit messages, require outbox-equivalent atomic linkage and make duplicate delivery a domain proof obligation.
- Require consumer idempotency, durable dedup, bounded retries, and reconciliation for cross-service invariants.
- Use DB constraints for DB-enforceable invariants rather than app-only checks.
- Prefer transaction boundaries local to one service-owned datastore; escalate distributed transactions or external-resource locks as explicit architecture decisions.
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
- whether a real `live fork` exists
- when a `live fork` exists, the viable options, the selected option, and at least one explicit rejection reason
- transition rules, violation behavior, and duplicate/replay semantics where relevant
- only the downstream API, data, distributed, reliability, security, or testing effects that force a new decision, handoff, or proof obligation now
- rollout compatibility, rollback limits, assumptions, blockers, and reopen conditions

## Deliverable Shape
When writing the domain behavior spec or review, cover:
- domain terms and scope
- invariant register
- state transition rules
- acceptance criteria
- corner cases and edge conditions
- invariant-violation semantics
- traceability into API, data, distributed, reliability, and testing concerns only when they force current decisions or proof; otherwise use `no new decision required in <domain>`

## Escalate Or Reject
- a critical invariant without owner, enforcement point, or falsifiable pass/fail condition
- state transitions defined only as prose with no forbidden paths
- unspecified or contradictory invariant-violation behavior
- missing retry/idempotency semantics where repeated execution is possible
- cross-service invariants without outbox, idempotency, and reconciliation stance
- invariant-sensitive decisions based on stale projections with no freshness contract
- identity or tenant correctness implied but not explicitly enforced
- migration or rollout plans that can silently break invariant semantics

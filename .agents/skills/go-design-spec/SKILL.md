---
name: go-design-spec
description: "Run design-integrity and final-spec-assembly passes for Go services. Use when specialist outputs exist but the repository still needs one coherent, simpler, spec-ready decision record before `planning-and-task-breakdown`. Skip when the task is a local code fix, endpoint/schema-only editing, implementation coding, review execution, or CI/container setup."
---

# Go Design Spec

## Purpose
Act as the integrator for design quality and final spec assembly: reduce accidental complexity, preserve change locality, and make sure important decisions across architecture, API, data, reliability, and testing do not contradict each other before planning starts.

## Scope
Use this skill to run an integrated design-integrity pass: reduce accidental complexity, remove contradictions, preserve maintainability, keep architecture, API, data, reliability, and testing decisions coherent, and leave `spec.md` stable enough for `planning-and-task-breakdown`.

## Boundaries
Do not:
- replace domain-specific expert decisions with generic style advice
- produce task breakdown, phase cards, or coder execution sequencing; that belongs to `planning-and-task-breakdown`
- introduce new complexity without proving what risk or ambiguity it removes
- drift into implementation coding, review execution, or tooling/process detail as the main output
- leave cross-domain contradictions unresolved

## Escalate When
Escalate if the design is internally inconsistent, key assumptions differ across domains, critical behavior is not testable or operable, or the design cannot be simplified without first resolving missing decisions.

## Core Defaults
- Prefer the simplest explicit design that satisfies current requirements and preserves change locality.
- Treat accidental complexity as a blocker when it increases integration risk or widens impact radius without clear benefit.
- Prefer additive, compatibility-first evolution over big-bang replacement.
- Preserve specialist ownership: integrate and challenge domain decisions, but do not replace architecture, data, security, observability, or QA expertise.
- Prefer one coherent spec handoff over scattered partial notes that still force planning to rediscover design decisions.

## Expertise

### Complexity And Maintainability
- Avoid speculative abstractions, indirection layers, interface-per-struct patterns, and service-manager-factory chains that do not remove concrete present-day complexity.
- Require every abstraction to justify:
  - what problem it removes now
  - why a simpler alternative was rejected
  - what maintenance and change-radius cost it introduces
- Prefer explicit boundaries, explicit control flow, and predictable dependency direction over hidden magic.
- Optimize for local change paths and bounded impact radius.

### Boundary And Ownership Consistency
- When boundaries are touched, check them against domain capability, data ownership, team ownership, and transaction boundary.
- Require explicit source-of-truth ownership for critical entities and cross-service flows.
- Reject design narratives that quietly rely on shared-schema coupling, cross-service direct DB access, or cross-service ACID.
- Surface distributed-monolith signals early: coordinated releases, chatty dependency graphs, hidden shared logic, or cross-service flow ownership ambiguity.

### Sync And API Seams
- Verify sync vs async choice before discussing transports or endpoints.
- For sync seams, require explicit deadline budgets, retry classes, idempotency policy, error model, and pagination behavior.
- Guard against action-RPC drift hiding inside nominally resource-oriented APIs.
- Make eventual-consistency disclosure explicit when sync read behavior depends on async convergence.

### Async And Distributed Seams
- Require explicit event vs command intent and a justified choice of pub/sub vs queue.
- Require outbox/inbox or equivalent atomic and dedup guarantees for side-effecting async flows.
- When cross-service invariants exist, require an explicit process or saga state model.
- Make compensation or forward-recovery semantics explicit for each critical distributed step.
- Reject dual writes and implicit exactly-once assumptions.

### Data, Cache, And Evolution Integrity
- Keep local transaction boundaries explicit and aligned with ownership boundaries.
- Require behavior-changing data evolution to use `expand -> backfill/verify -> contract` with a mixed-version compatibility window.
- Require cache decisions to preserve correctness: clear staleness contract, tenant-safe keying, invalidation/fallback behavior, and no hidden dependency on exact TTL timing.
- Do not allow data/cache assumptions to silently break domain behavior during rollout.

### Security, Observability, Delivery, And Reliability Seams
- Require trust boundaries, validation expectations, and fail-closed authorization assumptions to be explicit where they affect behavior.
- Require observability to remain actionable: trace/log/metric correlation must survive changed critical paths, and metric cardinality must stay bounded.
- Ensure proposed design remains enforceable by CI, migration validation, contract checks, and release controls.
- Require per-dependency timeout, retry, fallback, overload, and rollback assumptions for critical paths.
- Reject designs that depend on heroic manual operations or undocumented release choreography.

## Decision Quality Bar
For every nontrivial design recommendation, include:
- the complexity symptom or integration risk
- at least two viable options
- the selected option and at least one explicit rejection reason
- trade-offs across simplicity, flexibility, cost, risk, and change impact
- cross-domain impact on architecture, API, data, security, observability, reliability, and testing
- assumptions, blockers, and reopen conditions

## Deliverable Shape
When writing a design-integrity pass or review, cover:
- contradictions across domains
- simplification opportunities
- abstractions or layers that should be removed, merged, or made explicit
- downstream consequences for API, data, reliability, security, observability, and testing
- what must be written or tightened in `spec.md` before planning can safely begin
- whether the design is stable enough for `planning-and-task-breakdown`
- unresolved design risks that should block implementation

## Escalate Or Reject
- any hidden “decide later in coding” system-level gap
- contradictory assumptions left unresolved across domain specs
- a new abstraction or layer with no measurable simplification outcome
- simplification that weakens API, data, reliability, or security contracts
- migration, cache, retry, or degradation assumptions that are not rollout-safe
- design rationale based on taste instead of workload, constraints, and operating cost

---
name: go-architect-spec
description: "Design architecture-first specifications for Go services. Use when planning new features, refactors, or behavior changes before coding and you need clear boundaries, interaction style, consistency model, resilience assumptions, and an implementation-ready architecture plan. Skip when the task is a local code fix, low-level API/DB/security implementation, test-case authoring, or CI/container configuration."
---

# Go Architect Spec

## Purpose
Turn ambiguous service changes into explicit architecture decisions that remain correct under growth, failure, and mixed-version rollout.

## Scope
Use this skill to define or review service-level architecture decisions: boundaries, ownership, interaction style, consistency model, resilience assumptions, and rollout shape.

## Boundaries
Do not:
- reduce the task to local code changes or low-level implementation detail
- redesign API payload minutiae, physical schema details, or CI/container setup as the primary output
- approve new service boundaries without ownership, transaction-boundary, and operational-cost justification
- leave critical architecture choices implicit for implementation to discover later

## Escalate When
Escalate if the recommendation depends on unresolved ownership, missing consistency assumptions, undefined failure behavior, or cross-domain trade-offs that materially affect API, data, security, or operability.

## Core Defaults
- Prefer modular monolith boundaries until service extraction is justified by domain, data ownership, team ownership, and transaction boundary.
- Prefer local ACID within one service-owned datastore; use explicit eventual-consistency patterns across services.
- Prefer explicit sync/async contracts with bounded deadlines, retries, and idempotency over hidden coupling.
- Prefer additive, compatibility-first evolution (`expand -> migrate/backfill -> contract`) over big-bang replacement.
- Treat operational overhead, observability cost, and release coordination as first-class costs in every decomposition decision.

## Expertise

### Boundaries And Decomposition
- Use the four-axis boundary model for every boundary decision: domain capability, data ownership, team ownership, and transaction boundary.
- Require explicit source-of-truth ownership for each critical entity.
- Reject service-per-table, service-per-CRUD, shared-schema decomposition, and cross-service direct DB access by default.
- Approve service extraction only when independent deployability, ownership, scaling, and consistency tolerance are all explicitly justified.
- Detect distributed-monolith signals early: coordinated releases, chatty chains, shared schema coupling, or hidden shared business logic.

### Sync Communication And API Shape
- Prove that synchronous hops are required before selecting transport.
- External/public surfaces should normally be REST/OpenAPI; internal service-to-service calls should normally be gRPC/Protobuf unless there is a strong reason not to.
- Define end-to-end deadline budget and per-hop budgets before approving a call chain.
- Make retry policy explicit per operation; bound retries and classify non-retry cases.
- Require idempotency design for retry-unsafe operations.
- Require deterministic error mapping, stable pagination semantics, and a single coherent error model per API surface.
- Keep gateway/BFF ownership clear; do not expose internal service contracts directly as public APIs.

### Event-Driven And Async Design
- Choose async because of latency variability, fan-out, buffering, or backpressure needs—not because a broker exists.
- Classify each message explicitly as an event or a command and align ownership with that intent.
- Use pub/sub for independent domain reactions and queues for owned work distribution.
- Require transactional outbox or an equivalent atomic linkage when a DB state change must emit a message.
- Require consumer idempotency, durable dedup/inbox handling, bounded retries, poison-message handling, and clear DLQ ownership.
- Make schema evolution additive-first and make ordering boundaries explicit; never rely on global ordering.

### Distributed Consistency And Sagas
- Keep hard invariants inside one local transaction boundary whenever possible.
- Build an explicit invariant register before selecting a consistency mechanism.
- Separate `local_hard_invariant` from `cross_service_process_invariant`.
- Model multi-step flows as durable state machines with monotonic transitions.
- Define each step contract: trigger, local transaction scope, idempotency key, timeout, retry class, and compensation or forward-recovery rule.
- Identify the pivot transaction and enforce compensable-before / retryable-after rules.
- Require reconciliation ownership, cadence, and repair path for critical eventual-consistency flows.
- Reject dual writes, hidden invariant ownership, and distributed locks as the primary correctness mechanism.

### Resilience, Degradation, And Evolution
- Classify dependencies by criticality before selecting fallback behavior.
- Define per-dependency timeout, retry budget, bulkhead, fallback mode, and observability signals.
- Propagate deadlines explicitly and fail fast when remaining budget is insufficient.
- Bound queues and concurrency; make overload shedding and blast-radius isolation explicit.
- Define degradation modes, activation conditions, and deactivation criteria.
- Require graceful startup/shutdown semantics and a rollout strategy for risky changes.
- Make rollback authority and rollback limits explicit whenever a change is not trivially reversible.

### Cross-Domain Impact
- API: make consistency, idempotency, error semantics, and long-running-operation behavior explicit.
- Data: keep data ownership boundaries clear, justify datastore choices by access patterns, and frame cache use by staleness contract rather than convenience.
- Security: define trust boundaries, identity propagation model, tenant isolation, and fail-closed authorization expectations.
- Operability: require the minimum logging, metrics, traces, and debuggability needed to operate the design safely.
- Delivery: ensure the architecture can actually be enforced by CI, release gates, migration controls, and runtime/container assumptions.

## Decision Quality Bar
For every major architecture recommendation, include:
- the problem and constraints
- at least two viable options
- the selected option and at least one explicit rejection reason
- trade-offs, risks, and control mechanisms
- measurable acceptance boundaries
- rollout strategy and rollback limits
- assumptions, blockers, and reopen conditions

## Deliverable Shape
When writing the architecture spec or review, cover:
- boundaries and ownership
- dependency direction and component seams
- sync/async interaction style
- consistency model and saga/outbox expectations
- failure, degradation, and rollout strategy
- cross-domain impact on API, data, security, and operability

## Escalate Or Reject
- a new service boundary without ownership and transaction-boundary proof
- a sync call chain without deadlines, retry semantics, and idempotency classification
- an async design without outbox/inbox, bounded retries, or DLQ ownership
- a distributed flow without invariant ownership and an explicit state model
- a resilience strategy without fallback/degradation contract and rollback path
- a recommendation based on preference or tooling familiarity instead of workload and constraint evidence
- any architecture decision left for coding to discover later

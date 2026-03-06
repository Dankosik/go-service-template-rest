---
name: go-data-architect-spec
description: "Design data-architecture-first specifications for Go services. Use when planning or revising SQL and data modeling, consistency boundaries, datastore choice, schema evolution, migration rollout, and data reliability before coding. Skip when the task is a local code fix, endpoint-level API contract design, pure service decomposition work, CI/container setup, or low-level implementation tuning."
---

# Go Data Architect Spec

## Purpose
Turn ambiguous data requirements into explicit decisions about ownership, invariants, consistency, schema evolution, and recovery before implementation begins.

## Scope
Use this skill to define or review data ownership, invariants, OLTP modeling, datastore fit, schema evolution, migration safety, retention, deletion, and recovery expectations.

## Boundaries
Do not:
- drift into endpoint contract design, generic service decomposition, or low-level code tuning as the primary output
- treat cache as a source of truth without an explicit correctness contract
- approve new datastores without workload-fit and operational-readiness evidence
- leave schema evolution, backfill, or rollback safety unspecified

## Escalate When
Escalate if ownership is unclear, invariants cross service boundaries without an explicit consistency model, datastore choice is underconstrained, or destructive migration risk lacks a safe rollout and verification plan.

## Core Defaults
- Default to SQL OLTP (`PostgreSQL`-compatible) as the primary system of record for service business data.
- Default to one service-owned schema or database boundary and local ACID within that boundary.
- Default cross-service consistency to eventual consistency with explicit outbox, idempotency, and reconciliation.
- Default schema evolution to `expand -> migrate/backfill -> contract`.
- Treat cache as an accelerator, not as a source of truth.
- Treat new datastore engines as exceptions that require workload-fit evidence and operational readiness.

## Expertise

### Data Ownership And Boundaries
- Require explicit source-of-truth ownership per critical entity.
- Keep service data private: no direct cross-service table access, no cross-service foreign keys, and cross-service references by ID only.
- Make cross-service read strategy explicit: API composition, replicated read model, or event-driven projection.
- Reject shared-schema coupling and service-per-table decomposition by default.
- State ownership and transaction-boundary impact for every major model change.

### SQL And OLTP Modeling
- Start from normalized schema (`~3NF`) unless denormalization has an explicit source of truth, sync path, staleness contract, and reconciliation owner.
- Treat constraints as contracts:
  - primary key on every table
  - `UNIQUE` for business uniqueness
  - `NOT NULL` by default for required fields
  - row-local `CHECK` constraints where they materially protect correctness
- Keep referential integrity inside one service boundary.
- Build index policy from real or explicitly expected access patterns; avoid speculative index growth.
- Require stable ordering and a unique tie-breaker for pagination; default operational lists to keyset pagination.
- Make delete and history policy explicit: hard delete by default, soft delete or temporal history only when restore, compliance, or audit needs justify it.
- For pooled multi-tenancy, require a tenant-safe model such as `tenant_id` plus centralized isolation controls like RLS where appropriate.

### Consistency, Transactions, And Concurrency
- Keep transactions local and short-lived.
- Reject cross-service global ACID assumptions.
- Prefer optimistic concurrency for mutable entities; make conflict semantics explicit.
- Use pessimistic locking only when contention or serialization needs are proven.
- Treat stricter isolation levels as exception paths that must come with retry design and deadlock/serialization testing.
- Bound retries and make write paths idempotency-aware.

### Go SQL Access Contract
- Prefer explicit SQL plus generated access layers (`sqlc`) as the production contract.
- Prefer `pgx/v5 + pgxpool` for PostgreSQL-first systems; use `database/sql` only when portability is a real requirement.
- Keep SQL as the source of truth; generated code is derivative and not hand-edited.
- Make transaction ownership explicit at the use-case boundary.
- Require end-to-end context propagation, DB deadlines, and pool-capacity assumptions.
- Prevent `N+1` and chatty access patterns on hot paths.
- Require parameterized values and allow-listed dynamic identifiers.
- Make slow-query thresholds, pool saturation, and stable query identity observable.

### Datastore Selection
- Keep SQL OLTP as the default until another engine is justified by access-pattern evidence.
- Require an access-pattern catalog before selecting document, key-value, wide-column, time-series, or columnar storage.
- For partitioned NoSQL, require partition-key, skew, growth, and hot-partition mitigation plans.
- Make consistency model, conflict behavior, retention, deletion, archival, and downsampling explicit before launch.
- Evaluate cache before adopting a new datastore when the real problem is repeated read amplification with safe cacheability.
- Reject datastore adoption when on-call, backup/restore, observability, or runbook readiness is absent.

### Schema Evolution And Migration Safety
- Enforce mixed-version compatibility across rollout.
- Use one controlled migration runner with immutable, versioned migrations and separate runtime vs migration roles.
- Require migration safety budgets such as lock and statement timeouts.
- For behavior-changing changes, use phased rollout:
  - additive expand
  - idempotent, resumable, throttled migrate/backfill
  - verification-gated contract
- Require backfill checkpoints, bounded retries, throttling, and abort thresholds.
- Verify before contract with row parity, aggregate parity, deterministic sample diff, and domain-invariant checks.
- Declare rollback class (`safe`, `conditional`, or `restore-based`) and call out irreversible steps.
- Reject cross-system dual writes for DB state change plus message emission; require an outbox-equivalent linkage.

### Reliability, Lifecycle, And Recovery
- Treat backups as incomplete without restore drills.
- Require explicit RPO and RTO by data class and tested restore procedures.
- Make retention, archival, deletion triggers, and PII deletion/anonymization process explicit.
- Call out backup and archive residual-retention limits when hard deletion is required.
- Define disaster-recovery posture for destructive migration, regional outage, and corruption or backfill bug scenarios.
- Align published data semantics and downstream consumers before contracting schema or event shapes.

### Data vs Cache Boundary
- Keep correctness and source-of-truth guarantees in the authoritative datastore.
- Classify reads as strict-consistency or cacheable-with-staleness-budget.
- Make staleness, fallback, and cache correctness assumptions explicit whenever cache affects observable behavior.
- Reject cache designs that introduce tenant leakage or make correctness depend on cache behavior by default.

### Cross-Domain Impact
- API: make consistency, freshness, async materialization, idempotency, and concurrency semantics explicit when data decisions change external behavior.
- Distributed flows: require invariant ownership, step contracts, idempotency, retry semantics, and forward recovery for cross-service updates.
- Security: require parameterization, least-privilege roles, tenant propagation, fail-closed object access, and no secret/PII leakage in telemetry.
- Operability: require DB latency/error metrics, pool saturation visibility, migration/backfill/reconciliation progress, and bounded-cardinality telemetry.

## Decision Quality Bar
For every major data recommendation, include:
- the data problem and constraints
- at least two viable options
- the selected option and at least one explicit rejection reason
- compatibility class, rollout sequence, and rollback limitations
- consistency and transaction semantics
- evidence for ownership/modeling, transactions, evolution, and recovery
- assumptions, blockers, and reopen conditions

## Deliverable Shape
When writing the data spec or review, cover:
- data ownership and source-of-truth boundaries
- model shape, keys, constraints, and indexing policy
- transaction and concurrency rules
- datastore choice rationale
- schema evolution, migration, and backfill plan
- reliability, retention, deletion, and recovery expectations
- data vs cache responsibility boundary

## Escalate Or Reject
- a recommendation without explicit trade-offs and a rejected option
- shared-schema or cross-service DB coupling without an explicit exception and strong justification
- schema evolution without mixed-version compatibility
- migration or backfill plans without phased rollout, verification gates, or rollback limits
- datastore-class changes without access-pattern evidence and operational readiness
- cache proposals that quietly become correctness-critical
- cross-system dual writes used as the default consistency mechanism
- critical unknowns left implicit instead of being called out as blockers or assumptions

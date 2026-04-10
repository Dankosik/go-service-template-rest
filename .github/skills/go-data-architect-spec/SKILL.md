---
name: go-data-architect-spec
description: "Design data-architecture-first specifications for Go services. Use when planning or revising SQL and data modeling, data ownership, multi-tenant isolation, schema evolution, migration rollout, retention/deletion, ledger/history/projection boundaries, or datastore choice before coding. Reach for this whenever the hard part is deciding source-of-truth shape, invariant locality, OLTP vs append/projection split, or safe online evolution. Skip when the task is a local code fix, endpoint-level API contract design, pure service decomposition work, CI/container setup, or low-level implementation tuning."
---

# Go Data Architect Spec

## Purpose
Turn ambiguous data requirements into explicit decisions about source of truth, invariant locality, schema shape, lifecycle, online evolution, and recovery before implementation begins.

## Specialist Stance
- Treat persisted state as source-of-truth ownership first, storage mechanics second.
- Separate OLTP rows, append-only facts, audit logs, events, and projections instead of flattening them into “data”.
- Prefer online, compatibility-safe evolution and explicit retention/deletion semantics over schema convenience.
- Hand off endpoint contracts, service decomposition, cache implementation, and delivery mechanics when they become primary.

## Scope
Use this skill to define or review:
- data ownership and source-of-truth boundaries
- relational or hybrid model shape
- identity, tenancy, temporal, and money-type decisions
- transaction and concurrency rules
- read topology, projections, and derived-data boundaries
- datastore fit
- schema evolution, migration safety, and backfill strategy
- retention, deletion, archival, and recovery expectations

## Boundaries
Do not:
- drift into endpoint contract design, generic service decomposition, or low-level code tuning as the primary output
- invent domain invariants that have not been approved; translate known invariants into data consequences and escalate when the business rule is still unclear
- let query-shape tuning, pool math, cache behavior, or fallback policy take over the output; that belongs primarily to `go-db-cache-spec`
- let workflow topology or cross-service orchestration design take over the output; that belongs primarily to `go-architect-spec` or `go-distributed-architect-spec`
- approve new datastores, projections, or history models without explicit ownership, lifecycle, and recovery consequences
- leave schema evolution, backfill, deletion, or rollback safety unspecified

## Escalate When
Escalate if ownership is unclear, hard invariants are not yet approved, workload or retention evidence is materially missing, datastore choice is underconstrained, or destructive migration risk lacks a safe rollout and verification plan.

## Core Defaults
- Default to SQL OLTP (`PostgreSQL`-compatible) as the primary system of record for service business data.
- Default to one service-owned schema or database boundary and local ACID within that boundary.
- Default cross-service consistency to eventual consistency with explicit outbox, idempotency, and reconciliation.
- Default schema evolution to `expand -> migrate/backfill -> contract`.
- Treat caches, search indexes, analytics stores, and projections as derived surfaces, not sources of truth.
- Treat new datastore engines and event-sourced truth models as exceptions that require workload-fit evidence and operational readiness.

## Data Facts To Lock First
Before recommending schema or datastore shape, make these facts explicit:
- which entities or facts are authoritative, and which views are derived-only
- which invariants are hard, especially uniqueness, non-negative balance, scarce-capacity, one-active-per-scope, and retention or legal-hold rules
- which identifiers exist: internal IDs, public references, partner references, idempotency keys, and tenant keys
- which time semantics matter: event time, effective time, processed time, and user-local business date
- what dominates workload shape: point lookups, operational lists, append-heavy ingest, upserts, scans, hot tenants, hot keys, or bulk retention deletes
- what evidence exists for growth, cardinality, retention horizon, replay needs, and recovery objectives
- which rollout constraints already exist: mixed-version windows, downtime tolerance, backfill budget, and irreversible cutover points

If these facts are missing, mark them as assumptions or blockers instead of inventing them.

## Expertise

### Data Ownership And Truth Surfaces
- Require explicit source-of-truth ownership per critical entity and per invariant-bearing process.
- Keep service data private: no direct cross-service table access, no cross-service foreign keys, and cross-service references by ID only.
- Separate current-state tables, immutable history, audit logs, outbox or CDC feeds, and read projections on purpose. They solve different problems and should not be conflated.
- Treat an audit log, domain event stream, and read model as different artifacts:
  - audit log explains who changed what and when
  - event stream captures domain facts for downstream reaction or replay
  - read model exists to answer queries efficiently
- Do not let partner payloads or external statuses become the local lifecycle truth by accident; normalize them into local state.
- State ownership and transaction-boundary impact for every major model change.

### Model The Hard Parts First
- Model the invariant-bearing facts before optimizing tables by taste.
- For scarce resources or reservations, require an explicit hold or lease model, hold expiry owner, release path, and reconciliation owner.
- For balances, credits, or money movement, prefer immutable entries plus a verifiable balance or snapshot model when explainability, replay, or audit matters. Do not make a mutable balance column the only authoritative evidence unless that trade-off is explicit.
- Separate pending, committed, reversed, expired, or refunded states when the business process needs them; do not compress materially different states into one generic status.
- For long-running or operator-repaired workflows, a current-state row may need a transition history or append-only facts beside it. State which surface is authoritative.
- For mutable domain objects with restore or support needs, current-state tables may coexist with version or history tables. Make the truth surface explicit instead of implying that all history is authoritative.

### Identity, Time, Tenancy, And Domain Types
- Prefer stable surrogate primary keys. Keep natural or business keys explicit through `UNIQUE` constraints instead of using mutable business identifiers as the primary key.
- Distinguish internal identity, public reference, partner reference, idempotency key, and correlation ID. They are not interchangeable.
- Every unique, foreign-key, and index decision should say whether it is tenant-scoped.
- For shared-table multi-tenancy, require `tenant_id` to participate in the relevant uniqueness and indexing strategy. Use centralized isolation controls such as RLS where appropriate.
- Use a version column or equivalent optimistic-concurrency token for mutable rows that are updated by competing writers.
- Model time deliberately:
  - use timestamp-with-time-zone semantics for real instants
  - use a separate business date or effective timestamp when policy depends on local dates or retroactive effect
  - if event time and processing time differ, define late or out-of-order handling explicitly
- Use exact numeric types for money, rates, quotas, or billable usage. Do not use floating-point types for money.
- Choose enum, lookup-table, or constrained-text modeling based on change cadence and compatibility needs. Do not let unstable partner statuses leak into the canonical domain state.
- `JSONB` is acceptable for sparse or adjunct attributes with weak relational invariants and bounded query needs. It is a poor default for invariant-bearing fields, multi-column uniqueness, or heavily filtered operational data.

### SQL And Physical Relational Design
- Start from normalized schema (`~3NF`) unless denormalization has an explicit source of truth, sync path, staleness contract, and rebuild owner.
- Treat constraints as contracts:
  - primary key on every table
  - `UNIQUE` or composite `UNIQUE` for business uniqueness
  - `NOT NULL` by default for required fields
  - `CHECK` constraints where row-local correctness materially matters
  - partial `UNIQUE` indexes when uniqueness applies only to active or current rows
  - exclusion constraints when interval or overlap rules are central
- Keep referential integrity inside one service boundary.
- Build index policy from actual or explicitly expected access patterns:
  - align composite index order with filter then sort usage
  - remember every index adds write cost, storage cost, and migration cost
  - use partial indexes when hot predicates are sparse and stable
- Require stable ordering and a unique tie-breaker for pagination; default operational lists to keyset pagination.
- Partition only when retention pruning, bulk deletes, write volume, or tenant-isolation needs justify the added operational complexity. Align the partition key with dominant prune and query paths.
- Make delete and history policy explicit:
  - do not default to soft delete unless restore, compliance, support, or legal workflow needs justify it
  - if soft delete exists, handle uniqueness and query discipline explicitly, often with partial unique indexes on active rows

### Consistency, Transactions, And Concurrency
- Keep transactions local and short-lived.
- Reject cross-service global ACID assumptions.
- Choose concurrency control by invariant class:
  - row uniqueness -> `UNIQUE` or partial `UNIQUE`
  - interval, overlap, or scarce allocation -> exclusion constraint, lease table, or explicit lock ownership
  - lost updates on mutable rows -> version checks or compare-and-swap semantics
  - work claiming or queue consumption -> lease semantics or `FOR UPDATE SKIP LOCKED`
  - anomalies that constraints cannot encode locally -> selective stronger isolation or explicit coordination model
- Prefer optimistic concurrency when conflicts are rare and rows are independently owned.
- Use pessimistic or advisory locks only with explicit scope, acquisition order, timeout behavior, and deadlock story.
- Treat stricter isolation levels as exception paths that come with retry semantics and serialization testing.
- Bound retries and classify them by failure class; deadlocks, serialization failures, and timeouts are not the same recovery case.
- Make write paths idempotency-aware when retries, callbacks, or duplicate partner delivery are possible.

### Read Topology, Replicas, And Derived Views
- Make cross-service read strategy explicit: API composition, replicated read model, event-driven projection, or data export. Reject ad hoc cross-service DB reads by default.
- Primary reads are the default for read-after-write or monotonic-visible-truth paths. Replica reads require an explicit lag or staleness budget.
- Projections, materialized views, search indexes, analytics stores, and exports are derived-only surfaces. Define their lag, rebuild, replay, and correction owner.
- If customer-visible numbers come from a derived view, state which correctness-critical paths must bypass that view and read authoritative data directly.
- For billing or analytics aggregates, define the raw fact store, dedupe key, late-arrival handling, replay policy, and recomputation owner.

### Datastore Selection
- Keep SQL OLTP as the default until another engine is justified by access-pattern evidence.
- Require an access-pattern catalog before selecting document, key-value, wide-column, time-series, or columnar storage.
- Document stores fit best when the natural write unit is a bounded document aggregate, cross-document invariants are weak, and secondary-query needs are narrow and explicit.
- Key-value stores fit best when single-key or prefix-key access dominates and relational joins or ad hoc predicates are not correctness-critical.
- Time-series or columnar stores fit scan-heavy append workloads and analytical windows; they are not a default replacement for OLTP truth.
- Search indexes are read accelerators only. They do not own correctness for writes.
- Event sourcing is not the default answer to auditability. Approve it only when replay, temporal reconstruction, or event-native downstream contracts justify snapshotting, projection rebuild, and evolution complexity.
- For partitioned NoSQL systems, require partition-key choice, skew analysis, hot-partition mitigation, retention plan, and conflict behavior before approval.
- Reject datastore adoption when on-call, backup/restore, observability, or runbook readiness is absent.

### Go SQL Access Compatibility Note
- Prefer explicit SQL plus generated access layers (`sqlc`) as the production contract.
- Prefer `pgx/v5 + pgxpool` for PostgreSQL-first systems; use `database/sql` only when portability is a real requirement.
- Keep SQL as the source of truth; generated code is derivative and not hand-edited.
- Make transaction ownership explicit at the use-case boundary and require end-to-end context propagation.
- Require parameterized values and allow-listed dynamic identifiers.
- Deeper guidance on query shape, pool sizing, cache behavior, and fallback policy belongs to `go-db-cache-spec`.

### Schema Evolution And Migration Safety
- Enforce mixed-version compatibility across rollout.
- Use one controlled migration runner with immutable, versioned migrations and separate runtime vs migration roles.
- Require migration safety budgets such as lock and statement timeouts.
- For PostgreSQL-class systems, call out online-DDL hazards explicitly:
  - non-concurrent index builds can take disruptive locks
  - foreign-key or check-constraint validation can block or run long if introduced carelessly
  - some defaults, type changes, or table rewrites can be more expensive than they look
  - long backfill transactions create bloat, replica lag, and rollback pain
- Prefer additive columns, concurrent index builds, phased constraint validation, and compatibility-safe reads before contract when the table is already live.
- For behavior-changing changes, use phased rollout:
  - additive expand
  - idempotent, resumable, throttled migrate or backfill
  - verification-gated contract
- Require backfills to be checkpointed, restart-safe, chunked by a stable key or time range, and bounded by load, replica-lag, and abort thresholds.
- Verify before contract with row parity, aggregate parity, deterministic sample diff, domain-invariant checks, and canary reads on the new path.
- Declare rollback class (`safe`, `conditional`, or `restore-based`) and call out irreversible steps.
- Reject cross-system dual writes for DB state change plus message emission; require an outbox-equivalent linkage.

### Retention, Deletion, And Recovery
- Treat backups as incomplete without restore drills.
- Require explicit RPO and RTO by data class and tested restore procedures.
- Make retention, archival, deletion triggers, legal hold, and PII deletion or anonymization process explicit.
- Call out residual-retention limits in backups, archives, exports, search indexes, and caches when hard deletion is required.
- Define disaster-recovery posture for destructive migration, regional outage, and corruption or backfill bug scenarios.
- Make archive or rehydrate boundaries explicit: what can be restored automatically, what requires operator repair, and what is intentionally unrecoverable.
- Align published data semantics and downstream consumers before contracting schema or event shapes.

### Data vs Cache Boundary
- Keep correctness and source-of-truth guarantees in the authoritative datastore.
- Classify reads as strict-consistency or cacheable-with-staleness-budget.
- Make staleness, fallback, and cache correctness assumptions explicit whenever cache affects observable behavior.
- Reject cache designs that introduce tenant leakage or make correctness depend on cache behavior by default.

### Cross-Domain Impact
- API: make consistency, freshness, async materialization, idempotency, and concurrency semantics explicit when data decisions change external behavior.
- Distributed flows: require invariant ownership, step contracts, idempotency, retry semantics, and forward recovery for cross-service updates.
- Security: require parameterization, tenant propagation, fail-closed object access, and no secret or PII leakage in telemetry.
- Operability: require DB latency and error metrics, replication and pool-health visibility where relevant, and migration, backfill, or reconciliation progress signals with bounded-cardinality telemetry.

## Decision Quality Bar
For every major data recommendation, include:
- the data problem and constraints
- the hard invariants and truth surfaces involved
- at least two viable options
- the selected option and at least one explicit rejection reason
- comparison on the axes that matter here: invariant locality, contention profile, tenant isolation, lifecycle or deletion semantics, mixed-version rollout risk, and recovery class
- compatibility class, rollout sequence, and rollback limitations
- consistency and transaction semantics
- evidence for ownership, model shape, evolution, and recovery
- assumptions, blockers, and reopen conditions

## Deliverable Shape
When writing the data spec or review:
- lead with the top correctness-critical or hardest-to-reverse data decisions first
- then cover ownership and truth surfaces
- model shape, keys, constraints, domain types, and indexing or partitioning policy
- transaction, concurrency, and idempotency rules
- datastore and derived-view rationale
- schema evolution, migration, and backfill plan
- retention, deletion, archival, and recovery expectations
- data vs cache responsibility boundary

Do not reward checklist completeness over strong calls. If a critical fact is missing, name the blocker or assumption instead of papering over it.

## Escalate Or Reject
- a recommendation without explicit trade-offs and a rejected option
- shared-schema or cross-service DB coupling without an explicit exception and strong justification
- schema evolution without mixed-version compatibility
- migration or backfill plans without phased rollout, verification gates, or rollback limits
- invented access-pattern, growth, retention, or recovery evidence presented as fact
- datastore-class changes without access-pattern evidence and operational readiness
- event sourcing used as a fashionable default instead of a justified truth model
- cache or projection surfaces quietly becoming correctness-critical
- runtime DB/cache tuning taking over the output instead of data-architecture decisions
- cross-system dual writes used as the default consistency mechanism
- critical unknowns left implicit instead of being called out as blockers or assumptions

# SQL/OLTP data modeling instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Designing or changing SQL schema for a microservice
  - Creating migrations, constraints, indexes, and data integrity rules
  - Designing transaction boundaries, locking strategy, and pagination
  - Designing soft delete, audit history, temporal data, or multi-tenant isolation
  - Reviewing SQL-backed CRUD/business OLTP data models
- Do not load when: The task is non-SQL, OLAP/reporting-only, or unrelated to data modeling

## Purpose
- This document defines operational defaults for SQL/OLTP data modeling in microservices.
- The goal is to make schema decisions predictable, reviewable, and safe for production CRUD/business workloads.
- Treat this as an LLM contract: apply defaults first, document deviations explicitly, and avoid speculative architecture.

## Baseline assumptions
- Default engine: PostgreSQL-compatible OLTP.
- Default architecture: database/schema owned by one service.
- Default workload: transactional CRUD and business workflows, not analytical OLAP.
- If the user does not specify DB engine, multi-tenant model, or consistency requirements:
  - assume PostgreSQL-compatible OLTP,
  - assume single-tenant,
  - assume local ACID per service,
  - and state these assumptions explicitly.

## Required inputs before generating schema
Resolve these inputs first. If missing, apply defaults and state assumptions.

- DB engine and version
- Multi-tenant model: single-tenant, silo, bridge, or pool
- Cross-service data dependencies and source-of-truth boundaries
- Business invariants that must be enforced in DB
- Read access patterns and pagination requirements
- Delete policy: hard delete, soft delete, or temporal/audit history
- Concurrency conflict model: optimistic only or selective pessimistic locks

## Service-owned schema and ownership boundaries
Default rule: `database/schema per service`.

- Model each service data store as private implementation detail.
- Never design direct table access from other services.
- Never design cross-service foreign keys.
- Cross-service references are IDs only (for example `customer_id`), validated via API/events, not FK to another service DB.
- For cross-service reads, use API composition, read-model replication, or event-driven denormalized projections.

## Normalization vs denormalization
Default rule: start normalized (about 3NF) inside service boundaries.

- Normalize by default for transactional correctness and reduced update anomalies.
- Denormalize only when there is a concrete reason:
  - lower latency,
  - fewer network calls to dependencies,
  - better resilience when another service is unavailable,
  - or simpler high-volume read paths.
- Every denormalized field must document:
  - source of truth,
  - update path (events/API/reconciliation),
  - expected staleness window,
  - owner of reconciliation job.
- Do not denormalize “just in case”.

## Keys, constraints, and index defaults
Default rule: constraints are part of the contract, not optional.

### Table and key rules
- Every table MUST have a primary key.
- PostgreSQL default PK strategy: surrogate key with `GENERATED ... AS IDENTITY`.
- Business uniqueness MUST be expressed with `UNIQUE` (or partial unique index when scoped).
- `NOT NULL` is default for domain-required columns.
- Nullable columns are allowed only when “unknown/optional” is real domain semantics.

### Constraint rules
- Use `CHECK` for row-level invariants only (ranges, enum-like status set, simple column relations).
- Never use `CHECK` for cross-row or cross-table invariants.
- Enforce cross-row uniqueness with `UNIQUE` or exclusion constraints.
- Use FK for referential integrity inside the same service boundary.

### Foreign key and index rules
- Keep FK inside service boundary only.
- Ensure referenced side has PK/UNIQUE index (usually automatic by design).
- Evaluate indexes on referencing FK columns when parent `UPDATE/DELETE` or frequent joins exist.
- Start from minimal index set:
  - PK indexes,
  - UNIQUE indexes,
  - query-driven indexes for known hot paths.
- Avoid speculative indexes.
- Use multi-column indexes only for proven query patterns; keep them small and intentional.

### NULL and uniqueness rules
- Decide and document NULL semantics for business keys.
- For PostgreSQL, remember that default UNIQUE semantics allow multiple NULL values.
- If the domain requires “NULL treated as equal”, use DB-supported semantics explicitly.

## Soft delete, audit fields, and temporal data
Default rule: hard delete unless restore/compliance/history requirements justify alternatives.

### Soft delete defaults
- Do not enable soft delete by default.
- If required, use `deleted_at TIMESTAMPTZ NULL`.
- All active-record queries must apply `deleted_at IS NULL` consistently.
- Add partial indexes to keep active-record queries efficient.
- Add partial unique indexes when uniqueness should apply only to active rows.
- Define retention/purge/archival policy; soft delete is not a substitute for lifecycle management.

### Audit field defaults
- Every business table SHOULD include:
  - `created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`
  - `updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`
- Add `created_by` and `updated_by` when actor-level accountability is required.
- For high-value actions, store append-only audit records in dedicated audit tables.
- Keep security/event logs separate from business audit trail.

### Temporal data defaults
- Use temporal history only when business/regulatory requirements need point-in-time reconstruction.
- Prefer explicit valid-time fields (`valid_from`, `valid_to`) or append-only history tables.
- If engine-native system-versioned tables are used, define:
  - retention policy,
  - storage budget,
  - query patterns for history reads.
- Do not turn on temporal history for all tables by default.

## Transaction boundaries and concurrency control
Default rule: local transactions per service, short-lived, explicit.

### Transaction boundary rules
- Keep ACID transactions inside one service database boundary.
- Never assume cross-service distributed ACID transactions as default behavior.
- For cross-service workflows, use saga/outbox/eventual consistency patterns.
- Keep transactions small and focused on required atomic state changes.

### Isolation defaults
- PostgreSQL default isolation: `READ COMMITTED`.
- Raise isolation level only for a proven invariant.
- If using stricter isolation (`REPEATABLE READ` or `SERIALIZABLE`), implement and test retry logic for serialization failures.

### Locking defaults
- Default to optimistic concurrency control.
- Use a version column (or vendor equivalent) and compare-and-swap update pattern.
- On update conflict (`rows affected = 0` for expected version), return explicit conflict error.
- Use pessimistic locking (`SELECT ... FOR UPDATE`) only when proven necessary:
  - high contention,
  - strict resource serialization,
  - unacceptable conflict retry cost.

## Pagination defaults
Default rule: deterministic ordering first, then choose keyset vs offset.

- Never use `LIMIT` without deterministic `ORDER BY`.
- Ordering for pagination must be stable and unique (add PK as tie-breaker).
- Default for next/prev flows: keyset (cursor/seek) pagination.
- Use offset pagination only when random page access is a hard requirement and depth is bounded.
- For deep/hot paths, avoid large OFFSET scans.

## Multi-tenant considerations
Default rule: single-tenant unless multi-tenancy is explicitly required.

### Model selection
- `silo`: strongest isolation, highest operational cost.
- `bridge`: shared instance with schema per tenant.
- `pool`: shared schema with `tenant_id` partition key.

### Pool model rules
- `tenant_id` is mandatory on tenant-scoped tables.
- Tenant-scoped uniqueness usually includes `tenant_id`.
- Use RLS by default for pooled PostgreSQL deployments.
- Set tenant context reliably per connection/transaction.
- Verify behavior with connection pool/proxy mode in tests.
- Account for PostgreSQL owner bypass behavior and force policy behavior when required.
- Never rely only on “developers will always add `WHERE tenant_id = ...`”.

## Default schema choices for CRUD/business OLTP workloads
Use these defaults unless a concrete requirement justifies deviation.

- Ownership:
  - one service owns one schema/database
  - no direct cross-service table access
- Data shape:
  - normalized model first
  - selective denormalized read fields with explicit sync contract
- Keys and integrity:
  - surrogate identity PK
  - business UNIQUE constraints
  - NOT NULL by default
  - CHECK for row invariants
  - FK inside service boundaries
- Indexing:
  - PK/UNIQUE base indexes
  - access-pattern indexes for hot reads and FK operations
  - no broad index proliferation
- Delete/history:
  - hard delete by default
  - soft delete only with clear restore/compliance need and partial indexes
  - append-only audit for high-value changes
- Concurrency/transactions:
  - local ACID per service
  - optimistic locking by default
  - pessimistic locking only by exception
- Pagination:
  - keyset for operational lists
  - offset only for bounded random-access UI use cases
- Tenancy:
  - single-tenant default
  - explicit silo/bridge/pool decision for multi-tenant systems

## Decision rules (when to deviate from defaults)
- If a workflow requires strict ACID across services, do not solve it with shared schema by default.
  - Keep service boundaries and use saga/outbox/eventual consistency, or explicitly re-evaluate service decomposition.
- If a read path depends on another service and that dependency hurts latency/availability:
  - denormalize required read fields,
  - define source of truth and synchronization contract,
  - define reconciliation strategy.
- If concurrent update conflicts are infrequent:
  - keep optimistic locking.
- If conflicts are frequent and retries materially hurt correctness/latency:
  - use targeted pessimistic locking on that resource only.
- If UI/API requires only next/prev navigation:
  - use keyset pagination.
- If random page jump is mandatory and result depth is bounded:
  - allow offset pagination with strict ordering and limits.
- If restore/legal retention is not required:
  - use hard delete.
- If restore/compliance/legal-hold is required:
  - allow soft delete or temporal/audit model with explicit retention and cleanup policy.
- If multi-tenant isolation must be strongest (compliance, per-tenant keys/regions):
  - prefer silo.
- If cost efficiency requires shared storage with acceptable operational complexity:
  - use pool with mandatory tenant key + enforced RLS/equivalent controls.
- If a query is on a critical path and plan shows expensive scans:
  - add a workload-driven index.
- If write amplification from indexes becomes significant:
  - remove or narrow low-value indexes instead of adding more.

## Anti-patterns to reject
- Shared DB ownership or shared schema across services.
- Cross-service foreign keys and cross-service joins as normal path.
- Missing PK/UNIQUE/FK/CHECK constraints where domain invariants exist.
- Using `CHECK` for cross-table or cross-row business rules.
- Unindexed access patterns on critical read/update/delete paths.
- Indexing every column “just in case”.
- `LIMIT/OFFSET` pagination without stable unique ordering.
- Deep offset pagination in hot paths.
- Overusing soft deletes for all entities without retention/purge strategy.
- Treating soft delete as replacement for audit history.
- Multi-tenant pool model without RLS or equivalent centralized isolation control.

## MUST / SHOULD / NEVER

### MUST
- MUST keep schema ownership per service and disallow direct cross-service DB access.
- MUST define PK for every table and explicit constraints for business invariants.
- MUST design indexes from known access patterns and verify hot queries.
- MUST use deterministic unique ordering for pagination.
- MUST keep transaction scope local to service DB boundary.
- MUST implement optimistic conflict handling where concurrent updates are possible.
- MUST document tenant model explicitly when multi-tenancy exists.
- MUST define delete and history policy (hard delete vs soft delete vs temporal/audit).

### SHOULD
- SHOULD start with normalized schema and denormalize only with clear justification.
- SHOULD default to PostgreSQL identity columns for surrogate PKs.
- SHOULD use partial indexes/partial unique indexes when soft delete is enabled.
- SHOULD include baseline audit fields (`created_at`, `updated_at`) on business tables.
- SHOULD keep multi-column indexes minimal and query-driven.
- SHOULD use keyset pagination for operational lists and APIs.

### NEVER
- NEVER propose shared-schema microservices as default architecture.
- NEVER ship schema changes with implicit invariants only in application code.
- NEVER accept unbounded, unreviewed index growth.
- NEVER rely on offset pagination as universal default.
- NEVER enable soft delete globally without explicit business need and cleanup plan.
- NEVER rely on manual tenant filters alone in pooled multi-tenant data models.

## Review checklist
Before approving SQL modeling changes, verify:

- Ownership and boundaries:
  - Data ownership is explicit per service.
  - No direct cross-service schema coupling is introduced.
- Core schema integrity:
  - Every table has PK.
  - Business uniqueness is encoded in DB constraints.
  - `NOT NULL` matches required domain fields.
  - `CHECK` constraints are row-local only.
- Referential integrity:
  - FK exists for mandatory intra-service relations.
  - No cross-service FK is present.
- Access pattern readiness:
  - Critical queries have matching indexes or explicit rationale for scan.
  - Pagination queries have stable unique ordering.
  - Hot-path lists avoid deep OFFSET.
- Concurrency and transactions:
  - Transaction boundaries are local and minimal.
  - Locking strategy is justified (optimistic default, pessimistic by exception).
  - Retry behavior is defined when strict isolation is used.
- Delete and history policy:
  - Soft delete is justified if present.
  - Partial indexes/unique constraints support active-record semantics.
  - Audit/temporal retention and operational policy are documented.
- Multi-tenant safety:
  - Tenant model is documented.
  - Pool model includes enforced isolation mechanism (RLS/equivalent) and tests.

## What good output looks like
- The schema encodes business invariants in DB constraints, not only in application logic.
- Data ownership is clear and does not create distributed-monolith coupling.
- Query patterns, indexes, and pagination are consistent and operationally safe.
- Concurrency behavior and transaction scope are explicit and testable.
- Multi-tenant and history choices are explicit, minimal, and justified by requirements.

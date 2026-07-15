---
name: go-data-architecture-spec
description: "Use when authoritative data ownership, models, schema evolution, tenant isolation, migration, retention, history, projections, or datastore choice must be decided before coding; Own data truth and lifecycle architecture; Skip when the primary decision is runtime query/transaction/cache behavior, business policy, distributed recovery, or implementation."
---

# Go Data Architecture Spec

Load the [shared specialist contract](../specialist-contract.md), then apply this data-policy boundary.

## Outcome And Boundary

Produce the logical and physical data design that preserves accepted behavior through mixed versions, migration, retention, and recovery. Own authoritative and derived data roles; models and representations; keys, constraints, indexes, query shape, and local transaction/concurrency semantics; datastore fit; history, deletion, and retention; and schema evolution/backfills.

Consume business acceptance and lifecycle meaning from `go-domain-invariant-spec`, durable cross-boundary recovery from `go-distributed-spec`, and concrete repository SQL/query/cache behavior from `go-db-cache-spec`. Do not choose their policy or repository package placement; record only forced data consequences.

Inspect current store and service authority, schemas/migrations/generated SQL sources, accepted invariants, access patterns and workload evidence, tenant/identity/time/money meaning, recovery and legal constraints, deployed versions, and rollout limits. Keep unknown growth, cardinality, lag, backfill load, retention, and data RPO/RTO as assumptions or blockers rather than invented facts.

## Data Core

1. **Keep one authoritative owner.** Keep service data private and reject direct cross-service table access and cross-service foreign keys. Give caches, search, analytics, exports, projections, audit logs, and event streams explicit non-authoritative roles plus rebuild and correction owners.
2. **Represent accepted meaning exactly.** Separate internal, public, partner, business-idempotency, and tenant identifiers; scope uniqueness and indexes by tenant; distinguish event, effective, processed, and business time; use exact numeric types for money, credits, quotas, and billable usage.
3. **Enforce hard data invariants near the write.** Select the smallest local constraint, transaction boundary, isolation/version check, lease, or lock that preserves the accepted invariant. Prefer durable local idempotency over application-only checks, cache counters, cross-system dual writes, or assumed global ACID; leave business sameness to domain and cross-boundary replay to distributed policy.
4. **Default authoritative OLTP to relational truth.** Prefer PostgreSQL-compatible OLTP, normalized schema, explicit SQL, and generated access such as `sqlc` until workload, recovery, and operational evidence justify another datastore or event-sourced truth.
5. **Tie physical design to real access paths.** Constraints encode correctness; indexes, partitioning, JSONB, denormalization, keyset pagination, and projections must name the query, write, retention, staleness, and rebuild trade-off they serve.
6. **Make evolution mixed-version safe.** Default to `expand -> migrate/backfill -> verify -> contract`; use immutable versioned migrations, short safe DDL, resumable bounded backfills, explicit cutover and rollback classes, and owner-first generated-source updates.
7. **Apply lifecycle policy to every copy.** Align retention, deletion, legal hold, anonymization, archive, backups, exports, caches, search, projections, replay inputs, PITR, and restore limits. Soft delete alone is not a deletion policy.
8. **Design local data recovery with the model.** Define late/out-of-order correction, derived rebuild, backup/restore evidence, operator repair, irreversible steps, and authoritative paths that bypass stale derived views. For a durable service boundary, expose the local data consequences and let `go-distributed-spec` own outbox/inbox, replay, redrive, and reconciliation policy.

Do not let a schema diagram silently decide business policy, distributed topology, API compatibility, or operational behavior. Defaults yield only to accepted invariant, workload, legal, or recovery evidence.

## Symptom-Driven Reference Selector

State the decision pressure and behavior-change thesis before loading.

| Symptom or decision pressure | Load | Behavior change |
| --- | --- | --- |
| Truth ownership, audit/event/history distinctions, projections, exports, caches/search, or external/provider evidence is ambiguous. | [source-of-truth-and-derived-surfaces.md](references/source-of-truth-and-derived-surfaces.md) | Name one authoritative owner and derived rebuild/correction paths instead of letting convenient reads become truth. |
| Tenant keys, identities, time semantics, money/rates, enums, JSONB, or domain types are disputed. | [tenant-identity-time-and-money-modeling.md](references/tenant-identity-time-and-money-modeling.md) | Encode business identity and exact semantics instead of inheriting mutable or lossy storage types. |
| Constraints, unique/null behavior, indexes, pagination, partitioning, overlap rules, or JSONB placement is unclear. | [sql-constraints-indexes-and-pagination.md](references/sql-constraints-indexes-and-pagination.md) | Put enforceable invariants and access-pattern physical design in SQL instead of application-only checks or broad untethered indexes. |
| Transaction scope, isolation, optimistic/pessimistic concurrency, leases, work claiming, retries, callbacks, idempotency, or outbox/inbox linkage is live. | [transactions-concurrency-and-idempotency.md](references/transactions-concurrency-and-idempotency.md) | Select the smallest local mechanism that preserves the invariant instead of blanket isolation, unscoped locks, or dual writes. |
| A new datastore, document/KV/time-series/columnar model, Redis/search truth, or event sourcing is proposed. | [datastore-fit-and-event-sourcing.md](references/datastore-fit-and-event-sourcing.md) | Require access-pattern, recovery, and operational fit before replacing relational truth. |
| Live DDL, constraints, rename/split/type change, source-of-truth cutover, index build, or backfill is needed. | [schema-evolution-and-backfills.md](references/schema-evolution-and-backfills.md) | Design expand/backfill/verify/contract with load budgets and rollback limits instead of one-shot DDL or giant transactions. |
| Retention, deletion, legal hold, PII erasure, history/archive, PITR, projection cleanup, replay, or restore is changing. | [retention-deletion-history-and-projections.md](references/retention-deletion-history-and-projections.md) | Define lifecycle and recovery across every surface instead of treating primary-row delete or “keep forever” as policy. |

## Return And Stop

Return the authoritative facts/tables and derived surfaces; model, keys, exact types, constraints, indexes, partitions, and local transaction/concurrency rules; datastore/projection rationale where live; mixed-version migration/backfill/cutover/rollback sequence and proof; lifecycle and local recovery across every copy; assumptions; forced neighbor consequences; and reopen conditions.

The design is ready when implementation can preserve every accepted invariant and owner, migrations can run with bounded load through mixed versions, derived surfaces remain non-authoritative, and deletion/recovery claims have named mechanisms and evidence. Stop when write authority, accepted business identity or invariant, tenant isolation, access shape, migration window, legal lifecycle, data recovery target, or datastore-fit evidence is unresolved; name the owning domain, distributed, DB/cache, API, security, reliability, or delivery decision without inventing it. Reject shared-schema coupling, cross-service DB reads, correctness-bearing caches/projections, unbounded backfills, fake reversible DDL, unexplained event sourcing, cross-system dual writes, and invented scale or retention facts.

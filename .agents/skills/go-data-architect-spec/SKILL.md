---
name: go-data-architect-spec
description: "Design data-architecture-first specifications for Go services. Use when planning or revising SQL and data modeling, data ownership, multi-tenant isolation, schema evolution, migration rollout, retention/deletion, ledger/history, projections, datastore selection, or transactional consistency before coding. Skip when the task is low-level query tuning, cache implementation, handler coding, or a post-code DB review."
---

# Go Data Architect Spec

## Trigger And Scope

Use this skill before implementation when a Go service changes authoritative data, schema/model shape, tenancy, constraints/indexing, transactions/concurrency, idempotency, derived views, datastore class, migration/backfill, retention/deletion, or recovery.

Own data truth, invariants, lifecycle, and evolution decisions. Do not take over low-level query/pool/cache tuning, handler code, cross-service orchestration, public API semantics, or post-code DB review; state their data consequences and hand off the primary question.

## Approved Input And Decision Boundary

Inspect current authoritative stores and service ownership, schemas/migrations/generated SQL authority, access patterns and workload evidence, hard invariants, tenant/identity/time/money semantics, consistency and recovery requirements, retention/legal constraints, existing clients/versions, and rollout limits. Missing growth, cardinality, lag, backfill, retention, or RPO/RTO facts stay labeled assumptions or blockers.

Do not let a schema diagram silently decide business invariants, cross-service ownership, compatibility, or operational policy. A design must be explicit enough that API, implementation, migration, and validation can consume it without inventing data semantics.

## Data Invariants And Defaults

1. **One owner holds authoritative truth.** Keep service data private; reject direct cross-service table access and cross-service foreign keys. Caches, search, analytics, exports, projections, audit logs, and event streams have explicit distinct roles and rebuild/correction owners.
2. **Hard invariants live near the write.** Prefer SQL constraints, one local transaction boundary, version checks, leases/locks, and durable idempotency over application-only checks, cache counters, dual writes, or global ACID assumptions.
3. **Model business identity and time deliberately.** Separate internal/public/partner/idempotency/tenant identifiers; scope uniqueness and indexes by tenant; distinguish event/effective/processed/business time; use exact numeric types for money, credits, quotas, and billable usage.
4. **Relational truth is the default.** Prefer PostgreSQL-compatible OLTP, normalized schema, explicit SQL, and generated access (`sqlc`) until workload and operational evidence justifies another engine or event-sourced truth.
5. **Physical design follows real access patterns.** Constraints encode correctness; indexes, partitioning, JSONB, denormalization, keyset pagination, and projections must name the query, write, retention, staleness, and rebuild trade-off they serve.
6. **Evolution is mixed-version safe.** Default to `expand -> migrate/backfill -> verify -> contract`; use immutable versioned migrations, short safe DDL, resumable bounded backfills, explicit cutover/rollback class, and owner-first generated updates.
7. **Lifecycle covers every copy.** Retention, deletion, legal hold, anonymization, archive, backups, exports, caches, search, projections, replay, PITR, and restore limits must agree; soft delete is not a complete policy.
8. **Recovery is part of correctness.** Name reconciliation, replay/dedup, late/out-of-order handling, backup/restore evidence, operator repair, RPO/RTO, irreversible steps, and the paths that bypass stale derived views.

Default cross-service consistency to explicit eventual consistency with outbox-equivalent atomic linkage, idempotent consumption, and reconciliation. Defaults yield only to approved workload, invariant, or operational evidence.

## Symptom-Driven Reference Selector

State the decision pressure and behavior-change thesis before loading. Load at most one reference by default; load more only for independent data decisions such as tenant modeling plus live backfill.

| Symptom or decision pressure | Load | Behavior change |
| --- | --- | --- |
| Truth ownership, audit/event/history distinctions, projections, exports, caches/search, or external/provider evidence is ambiguous. | [source-of-truth-and-derived-surfaces.md](references/source-of-truth-and-derived-surfaces.md) | Name one authoritative owner and derived rebuild/correction paths instead of letting convenient reads become truth. |
| Tenant keys, identities, time semantics, money/rates, enums, JSONB, or domain types are disputed. | [tenant-identity-time-and-money-modeling.md](references/tenant-identity-time-and-money-modeling.md) | Encode business identity and exact semantics instead of inheriting mutable or lossy storage types. |
| Constraints, unique/null behavior, indexes, pagination, partitioning, overlap rules, or JSONB placement is unclear. | [sql-constraints-indexes-and-pagination.md](references/sql-constraints-indexes-and-pagination.md) | Put enforceable invariants and access-pattern physical design in SQL instead of application-only checks or broad untethered indexes. |
| Transaction scope, isolation, optimistic/pessimistic concurrency, leases, work claiming, retries, callbacks, idempotency, or outbox/inbox linkage is live. | [transactions-concurrency-and-idempotency.md](references/transactions-concurrency-and-idempotency.md) | Select the smallest local mechanism that preserves the invariant instead of blanket isolation, unscoped locks, or dual writes. |
| A new datastore, document/KV/time-series/columnar model, Redis/search truth, or event sourcing is proposed. | [datastore-fit-and-event-sourcing.md](references/datastore-fit-and-event-sourcing.md) | Require access-pattern, recovery, and operational fit before replacing relational truth. |
| Live DDL, constraints, rename/split/type change, source-of-truth cutover, index build, or backfill is needed. | [schema-evolution-and-backfills.md](references/schema-evolution-and-backfills.md) | Design expand/backfill/verify/contract with load budgets and rollback limits instead of one-shot DDL or giant transactions. |
| Retention, deletion, legal hold, PII erasure, history/archive, PITR, projection cleanup, replay, or restore is changing. | [retention-deletion-history-and-projections.md](references/retention-deletion-history-and-projections.md) | Define lifecycle and recovery across every surface instead of treating primary-row delete or “keep forever” as policy. |

## Required Evidence And Deliverable

For each material recommendation, record the data problem, hard invariants, owner and truth surfaces, access/workload evidence, whether a real live fork exists, selected option, rejected viable options, consistency/transaction mechanism, tenant and lifecycle semantics, compatibility/mixed-version sequence, verification gates, rollback/restore class, assumptions, and reopen triggers.

Return a compact data packet with:

- authoritative facts/tables and derived surfaces;
- model, keys, constraints, exact domain types, and index/partition policy;
- transaction, concurrency, idempotency, outbox/inbox, and reconciliation rules;
- datastore and projection rationale only where live;
- migration/backfill/cutover/rollback proof;
- retention/deletion/archive/recovery policy across all copies;
- cache boundary and downstream API/security/operability proof obligations only when another owner must act now.

## Success, Escalation, And Stop Conditions

Success means implementation can preserve each hard invariant and owner, migrations can operate through mixed versions with bounded load and proof, derived surfaces remain non-authoritative, and deletion/recovery claims have named mechanisms and evidence.

Stop or escalate when write authority, tenant isolation, invariant class, access pattern, consistency, migration window, deletion/legal policy, recovery target, or operational readiness is unknown; when the only safe design requires a public-contract, domain, distributed-flow, security, or rollout decision; or when a datastore change lacks fit evidence. Reject shared-schema coupling, cross-service DB reads, correctness-bearing caches/projections, unbounded backfills, fake reversible DDL, unexplained event sourcing, cross-system dual writes, and invented scale/retention facts.

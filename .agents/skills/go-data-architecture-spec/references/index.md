# Reference Selector

State the decision pressure and behavior-change thesis before loading.

| Symptom or decision pressure | Load | Behavior change |
| --- | --- | --- |
| Truth ownership, audit/event/history distinctions, projections, exports, caches/search, or external/provider evidence is ambiguous. | [source-of-truth-and-derived-surfaces.md](source-of-truth-and-derived-surfaces.md) | Name one authoritative owner and derived rebuild/correction paths instead of letting convenient reads become truth. |
| Tenant keys, identities, time semantics, money/rates, enums, JSONB, or domain types are disputed. | [tenant-identity-time-and-money-modeling.md](tenant-identity-time-and-money-modeling.md) | Encode business identity and exact semantics instead of inheriting mutable or lossy storage types. |
| Constraints, unique/null behavior, indexes, pagination, partitioning, overlap rules, or JSONB placement is unclear. | [sql-constraints-indexes-and-pagination.md](sql-constraints-indexes-and-pagination.md) | Put enforceable invariants and access-pattern physical design in SQL instead of application-only checks or broad untethered indexes. |
| Transaction scope, isolation, optimistic/pessimistic concurrency, leases, work claiming, retries, callbacks, idempotency, or outbox/inbox linkage is live. | [transactions-concurrency-and-idempotency.md](transactions-concurrency-and-idempotency.md) | Select the smallest local mechanism that preserves the invariant instead of blanket isolation, unscoped locks, or dual writes. |
| A new datastore, document/KV/time-series/columnar model, Redis/search truth, or event sourcing is proposed. | [datastore-fit-and-event-sourcing.md](datastore-fit-and-event-sourcing.md) | Require access-pattern, recovery, and operational fit before replacing relational truth. |
| Live DDL, constraints, rename/split/type change, source-of-truth cutover, index build, or backfill is needed. | [schema-evolution-and-backfills.md](schema-evolution-and-backfills.md) | Design expand/backfill/verify/contract with load budgets and rollback limits instead of one-shot DDL or giant transactions. |
| Retention, deletion, legal hold, PII erasure, history/archive, PITR, projection cleanup, replay, or restore is changing. | [retention-deletion-history-and-projections.md](retention-deletion-history-and-projections.md) | Define lifecycle and recovery across every surface instead of treating primary-row delete or “keep forever” as policy. |

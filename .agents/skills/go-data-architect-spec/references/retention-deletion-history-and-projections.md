# Retention, Deletion, History, And Projections

## When To Load
Load this when the task involves retention windows, deletion, legal hold, PII erasure, anonymization, soft delete, history tables, audit logs, archives, PITR, partition pruning, materialized views, read projections, exports, or rebuild/replay policy.

Use it to separate lifecycle obligations from query convenience. Retention and deletion decisions must name all surfaces that may still hold data: primary tables, history, outbox, projections, search indexes, exports, backups, and archives.

## Decision Examples

### Example 1: PII deletion with derived surfaces
Context: Users can request deletion of profile PII, but support history, invoices, search indexes, and backups may still reference the user.

Selected option: Define per-field action: hard delete, anonymize, retain under legal basis, or legal hold. Propagate deletion or anonymization to derived stores through an owner-owned workflow and state residual backup/archive limits explicitly.

Rejected options:
- Treat soft delete as a completed deletion.
- Delete only the primary row and assume projections will converge.
- Keep PII in audit metadata, logs, or outbox payloads without a retention decision.

Migration and rollback consequences:
- Hard deletion is often irreversible except from backups; prove restore and correction workflows before rollout.
- Anonymization can be forward-fixed but may not reconstruct original values by design.
- Rollback must not rehydrate deleted PII from projections or backups unless policy explicitly allows it.

### Example 2: Append-only history with retention
Context: Usage facts are append-only for billing replay for 18 months and aggregated for long-term analytics.

Selected option: Keep raw facts partitioned by event or business time when retention pruning dominates. Derive aggregates or projections with rebuild rules. Detach and archive partitions before destructive drop when recovery or audit may be needed.

Rejected options:
- Keep raw facts forever because deletion is inconvenient.
- Store only aggregates when billing replay requires raw evidence.
- Partition by a key that does not match retention or dominant query pruning.

Migration and rollback consequences:
- Dropping old partitions is not a normal rollback operation; recovery depends on archive or PITR.
- Detach-before-drop provides a review and archive checkpoint.
- Derived aggregate rollback should rebuild from retained raw facts, not manual patching.

### Example 3: Materialized projection for support views
Context: Support needs a fast view across users, orders, and refund status, with 15 minutes of acceptable staleness.

Selected option: Use a derived projection or materialized view with an owner, refresh cadence, unique index requirements for concurrent refresh if needed, freshness disclosure, and rebuild procedure. Correctness-critical actions must read authoritative owner data.

Rejected options:
- Use the materialized view as the write authority for support actions.
- Promise exact current state without reading owner tables.
- Refresh in a way that blocks readers without an accepted support impact.

Migration and rollback consequences:
- Projection schema can be rebuilt if source truth remains stable.
- If refresh fails, route support to a slower safe path or disclose stale status.
- Contract projection fields only after dependent tooling is updated and rebuild scripts are compatible.

## Source Links Gathered Through Exa
- PostgreSQL, "Table Partitioning": https://www.postgresql.org/docs/current/ddl-partitioning.html
- PostgreSQL, "Continuous Archiving and Point-in-Time Recovery": https://www.postgresql.org/docs/current/continuous-archiving.html
- PostgreSQL, "CREATE MATERIALIZED VIEW": https://www.postgresql.org/docs/current/sql-creatematerializedview.html
- PostgreSQL, "REFRESH MATERIALIZED VIEW": https://www.postgresql.org/docs/current/sql-refreshmaterializedview.html
- PostgreSQL, "Row Security Policies": https://www.postgresql.org/docs/current/ddl-rowsecurity.html

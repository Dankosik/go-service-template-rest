# Retention, Deletion, History, And Projections

## Behavior Change Thesis
When loaded for retention, deletion, legal hold, PII erasure, history, archive, backup, partition pruning, export, or projection lifecycle, this file makes the model choose per-surface lifecycle and recovery rules instead of likely mistake "soft delete the primary row or keep everything forever and call it policy."

## When To Load
Load this for deletion, retention, legal hold, anonymization, soft delete, history/archive/PITR, partition pruning, residual projection data, exports, or rebuild/replay policy.

## Decision Rubric
- Name the lifecycle action for each surface: hard delete, anonymize, retain with legal basis, legal hold, archive, partition detach/drop, rebuild, or intentionally unrecoverable.
- Check every copy, not just primary tables: history, audit, outbox, projections, search indexes, exports, analytics stores, logs, backups, and archives.
- Do not default to soft delete. Use it only when restore, support, compliance, or workflow state needs justify uniqueness and query-discipline costs.
- Keep raw facts when replay, billing evidence, or audit requires them; derive aggregates and projections with rebuild rules.
- Align partition keys with retention pruning or dominant query pruning before proposing partitions.
- State residual backup/archive limits when hard deletion or PII erasure is required.

## Imitate

### PII deletion with derived surfaces
Context: Users can request deletion of profile PII, but support history, invoices, search indexes, and backups may still reference the user.

Define per-field action: hard delete, anonymize, retain under legal basis, or legal hold. Propagate deletion or anonymization to derived stores through an owner-owned workflow and state residual backup/archive limits explicitly.

Copy this because the deletion outcome is measured across all surfaces, not only the primary row.

### Append-only history with retention
Context: Usage facts are append-only for 18-month billing replay and aggregated for long-term analytics.

Keep raw facts partitioned by event or business time when retention pruning dominates. Derive aggregates or analytics projections with rebuild rules. Detach and archive partitions before destructive drop when recovery or audit may be needed.

Copy this because raw evidence and long-term aggregates have different retention and recovery jobs.

### Projection with residual sensitive data
Context: A support projection includes user email, order status, and refund state with a 15-minute freshness budget.

Treat the projection as derived. Define refresh or replay owner, PII removal/anonymization behavior, stale-data fallback, and correctness-critical paths that must read owner data.

Copy this because projection lifecycle combines staleness, rebuild, and deletion responsibilities.

## Reject
- "Soft delete means GDPR deletion is done." Soft-deleted rows and indexes still retain data.
- "Delete only the primary profile row; projections will converge eventually." The spec needs an owner and proof path for every derived copy.
- "Keep raw facts forever because deletion is inconvenient." Retention is a product/legal/operability decision, not a storage default.
- "Store only aggregates when billing replay requires raw evidence." Aggregates cannot prove corrected invoices without source facts.
- "Partition by tenant because tenants matter." If retention pruning is time-based, tenant partitioning may not help and may make deletes harder.

## Agent Traps
- Do not hide PII in audit metadata, outbox payloads, logs, or exports without a retention decision.
- Do not promise hard deletion from backups unless restore, expiration, and rehydration behavior are defined.
- Do not treat PITR as a substitute for restore drills or corruption/backfill recovery.
- Do not use projection rebuild as a hand-wave; name the source, owner, and acceptable staleness during rebuild.

## Validation Shape
- Lifecycle matrix lists each surface, retention window, delete/anonymize action, legal-hold behavior, owner, and residual-copy limit.
- Partition proof covers prune path, detach/archive/drop order, restore path, and query-pruning effect.
- Projection proof covers rebuild source, stale fallback, PII propagation, and verification that correctness-critical actions bypass derived data.

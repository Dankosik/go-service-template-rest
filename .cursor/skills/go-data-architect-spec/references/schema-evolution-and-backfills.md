# Schema Evolution And Backfills

## Behavior Change Thesis
When loaded for a live schema, constraint, index, type, source-of-truth, or backfill change, this file makes the model choose expand, migrate or backfill, verify, and contract with rollback class instead of likely mistake "run one DDL/backfill step and call rollback a down migration."

## When To Load
Load this for live schema changes, tightened constraints, column splits or renames, type changes, index creation on live tables, partitioned indexes, or any backfill.

## Decision Rubric
- Preserve mixed-version compatibility first. Old code, new code, workers, readers, migrations, and generated queries must coexist through the rollout window.
- Prefer additive expand, compatible read/write behavior, idempotent chunked migrate/backfill, verification-gated switch, then contract.
- Name rollback class: `safe`, `conditional`, or `restore-based`. Do not imply destructive or externally observed changes are trivially reversible.
- Make backfills restart-safe, checkpointed, throttled, and bounded by load, lock time, replica lag, and abort thresholds.
- Create live-table indexes and validate constraints with engine-safe phases when table size and traffic make blocking unacceptable.
- Treat not-null tightening as version-sensitive: PostgreSQL 17, this repo's default target, needs `SET NOT NULL` with a valid `CHECK` proof or a scan budget; PostgreSQL 18+ can add not-null constraints as `NOT VALID` and validate later, but only when the target engine supports it.
- For PostgreSQL concurrent indexes, ensure the migration runner can execute outside a transaction block and plan invalid-index cleanup.
- For PostgreSQL partitioned tables, do not promise `CREATE INDEX CONCURRENTLY` on the parent; plan concurrent builds on individual partitions plus the short parent metadata step when that shape fits.
- Treat validation failure as a contraction blocker, not as permission for improvised production edits.

## Imitate

### Split a name column
Context: `customers.full_name` must become `first_name` and `last_name`, with rolling deploys and background workers using old and new code for several days.

Add nullable new columns first. Ship compatible code that writes both shapes or reads both. Backfill in restart-safe chunks with parity checks. Switch reads after verification. Contract `full_name` only after old code and jobs are gone.

Copy this because it keeps both schema shapes valid during mixed-version rollout.

### Add uniqueness to a live large table
Context: `external_accounts` should be unique by `(tenant_id, provider, external_id)`, but the table accepts writes all day.

Detect duplicates, repair or quarantine them, create the unique index with a live-safe path, then attach it as a constraint if a formal constraint is useful and the engine supports the exact shape.

Copy this because duplicate cleanup and enforcement order are part of the design, not a migration afterthought.

### Tighten `NOT NULL` or `CHECK`
Context: A previously optional column is now required and must satisfy a row-local rule.

Make writers compatible first, backfill missing values, add a staged validation path where the deployed engine supports it, validate after cleanup, then remove fallback handling.

Copy this because enforcement follows evidence that existing and future rows comply.

## Reject
- "Rename or drop the old column in one deploy." Old binaries, jobs, generated code, and rollback paths can still need it.
- "Backfill all rows in one transaction." Long transactions create lock, bloat, rollback, and replica-lag risk.
- "Add a blocking unique constraint during peak traffic." Live DDL lock behavior is a delivery decision, not an implementation detail.
- "Use `IF NOT EXISTS` and assume the existing index is equivalent." Name collisions do not prove definition compatibility.
- "Rollback is just `DROP COLUMN` after clients observed the new semantics." That may destroy the only copy of data users now depend on.

## Agent Traps
- Do not bundle unrelated DDL subcommands when the strictest lock can apply to the whole statement.
- Do not skip duplicate or null detection before adding a uniqueness or required-value constraint.
- Do not infer engine support from "PostgreSQL-compatible"; name the target major version before using version-gated DDL.
- Do not leave invalid or failed concurrent index artifacts unnamed; they affect retries, write overhead, and cleanup.
- Do not hide partitioned-table index or uniqueness limitations behind a generic "concurrent index" step.
- Do not contract old fields until old code, workers, generated clients, and replay paths are drained.

## Validation Shape
- Migration proof lists expand, compatible code, backfill, verification, switch, and contract steps.
- Backfill proof includes checkpoint key, chunk size strategy, restart behavior, throttle/abort criteria, and parity checks.
- Contract proof includes old-version drain evidence, row parity or aggregate parity, deterministic sample diff where useful, and rollback class.

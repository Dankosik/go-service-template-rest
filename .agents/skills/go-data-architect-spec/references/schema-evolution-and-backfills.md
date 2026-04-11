# Schema Evolution And Backfills

## When To Load
Load this when the task changes live schema, adds or tightens constraints, renames or splits columns, changes data type or source of truth, creates indexes on live tables, introduces partitioned indexes, or requires a backfill.

Use it to keep evolution compatible across mixed versions. Prefer additive expand, idempotent migrate/backfill, verification, and contract. Name rollback class instead of implying every migration can be reverted.

## Decision Examples

### Example 1: Split a name column
Context: `customers.full_name` must become `first_name` and `last_name`, with rolling deploys and background workers using old and new code for several days.

Selected option: Add new nullable columns first. Ship code that writes both old and new shapes or reads compatibly. Backfill in restart-safe chunks with parity checks. Switch reads after verification. Contract `full_name` only after old code and jobs are gone.

Rejected options:
- Rename or drop the old column in one deploy.
- Backfill all rows in one large transaction.
- Claim rollback is just `DROP COLUMN` after new clients have observed changed data semantics.

Migration and rollback consequences:
- Expand is usually safe to roll back by ignoring the new columns.
- During dual-write, rollback must preserve whichever side is authoritative for new writes.
- After contract, rollback is restore-based or forward-fix unless old data remains recoverable.

### Example 2: Add uniqueness to a live large table
Context: `external_accounts` should be unique by `(tenant_id, provider, external_id)`, but the table is large and accepts writes all day.

Selected option: Detect duplicates, repair or quarantine them, create a unique index concurrently, then attach it as a constraint if a formal constraint is desired and PostgreSQL supports the needed shape.

Rejected options:
- Add a blocking unique constraint during peak traffic.
- Add `IF NOT EXISTS` and assume an existing index has the right definition.
- Skip duplicate cleanup and let migration failure be the first signal.

Migration and rollback consequences:
- A failed concurrent unique index can leave an invalid index that must be dropped or rebuilt.
- Unique enforcement can begin before the index is fully available, so rollout needs alerting and a conflict policy.
- Removing the constraint during rollback permits new duplicates; record who reconciles them.

### Example 3: Tighten `NOT NULL` or `CHECK`
Context: A previously optional column is now required and must satisfy a row-local rule.

Selected option: Add a compatible write path first, backfill missing values, add a `NOT VALID` check or not-null path when available for the constraint type, validate after cleanup, then contract fallback handling.

Rejected options:
- Set `NOT NULL` before all writers supply values.
- Use a `CHECK` constraint to enforce cross-row rules.
- Bundle several unrelated DDL subcommands so the strictest lock applies to all.

Migration and rollback consequences:
- Validation failures should stop contraction, not trigger improvised data edits.
- Rollback before validation can ignore the new constraint; rollback after validation may need to reallow legacy values and audit any rejected writes.
- If the constraint encodes a business invariant, reverting it requires explicit product or domain acceptance.

## Source Links Gathered Through Exa
- PostgreSQL, "ALTER TABLE": https://www.postgresql.org/docs/current/sql-altertable.html
- PostgreSQL, "CREATE INDEX": https://www.postgresql.org/docs/current/sql-createindex.html
- PostgreSQL, "Table Partitioning": https://www.postgresql.org/docs/current/ddl-partitioning.html
- PostgreSQL, "Constraints": https://www.postgresql.org/docs/current/ddl-constraints.html
- Prisma, "Expand-and-contract migrations": https://www.prisma.io/docs/guides/database/data-migration


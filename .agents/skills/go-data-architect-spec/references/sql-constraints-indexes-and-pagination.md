# SQL Constraints, Indexes, And Pagination

## Behavior Change Thesis
When loaded for a task where uniqueness, overlap, JSONB, indexes, or list pagination might be left to application code or query convenience, this file makes the model choose database-enforced invariants and access-pattern indexes instead of likely mistake "the app checks it first, JSON stores it flexibly, and offset pagination is good enough."

## When To Load
Load this for invariant-bearing SQL constraints, physical index shape, JSONB placement, partitioned uniqueness, or deterministic operational pagination.

## Decision Rubric
- Prefer SQL constraints for row-local and relation-local invariants the database can enforce: `UNIQUE`, composite `UNIQUE`, `NOT NULL`, `CHECK`, foreign keys inside one owner boundary, partial unique indexes, and exclusion constraints.
- For nullable unique keys, decide whether duplicate nulls are allowed. Use `NOT NULL` or `NULLS NOT DISTINCT` when "missing" should still be unique.
- Use application checks only as user-friendly preflight; they do not replace the write-time constraint.
- Keep invariant-bearing, join-critical, or heavily filtered fields relational. Use `jsonb` for bounded adjunct attributes with weak invariants and explicit query limits.
- Tie every index to a filter, join, sort, uniqueness, retention, or partition-pruning need. Every index adds write cost, storage cost, and rollout cost.
- Align composite index order with equality filters first, then range or sort columns, then a unique tie-breaker when pagination needs deterministic order.
- Use keyset pagination for high-churn operational lists; use offset only when the workload tolerates drift and cost.

## Imitate

### One active subscription per tenant and product
Context: A tenant can have many historical subscriptions for a product, but only one active subscription at a time.

Model current and historical state explicitly. Enforce active uniqueness with a partial unique index such as `(tenant_id, product_id) WHERE ended_at IS NULL`, or an equivalent status predicate if that is the canonical active marker.

Copy this because the database guards the one-active invariant while history remains possible.

### Time interval overlap rule
Context: Confirmed reservations must not overlap for the same tenant and resource.

Use an exclusion constraint or an explicit lease/hold model when interval overlap is the invariant. Keep pending, confirmed, expired, and canceled states separate if overlap rules differ.

Copy this because cross-row interval rules need a real concurrency story, not a row-local `CHECK`.

### Operational pagination under churn
Context: A worker or API lists recent records while new rows are inserted.

Use keyset pagination with a stable sort and unique tie-breaker, such as `(tenant_id, created_at DESC, id DESC)`, and add an index aligned with equality filters and sort order.

Copy this because it prevents duplicate or missing rows during churn and keeps the access path explicit.

## Reject
- "Check active uniqueness before insert." Two writers can pass the check unless the database enforces the invariant.
- "Put subscription state in JSONB and validate in Go." That hides uniqueness and query semantics from the database.
- "Use `CHECK` to prevent overlapping reservations." Row-local checks cannot inspect competing rows.
- "Sort by `created_at` only." Equal timestamps make cursor order unstable without a unique tie-breaker.
- "Add a covering index for every list field." Untethered indexes increase write cost and migration risk without proving selectivity.

## Agent Traps
- Do not use a plain unique constraint when historical rows require partial uniqueness.
- Do not suggest cross-service foreign keys; keep referential integrity inside one service-owned data boundary.
- Do not imply partitioned uniqueness or exclusion works globally unless the partition key participates in the constraint definition or the engine supports the exact shape.
- Do not treat index removal as always correctness-neutral if it backed uniqueness or exclusion.

## Validation Shape
- Constraint proof includes duplicate or conflict detection before creation, expected constraint definition including nullable uniqueness behavior, and what happens to legacy violating rows.
- Pagination proof includes stable sort columns, unique tie-breaker, cursor contents, and index shape.
- JSONB proof names which fields are adjunct, which fields stay relational, and which queries are intentionally unsupported or bounded.

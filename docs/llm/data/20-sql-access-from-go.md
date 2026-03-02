# SQL access from Go instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Designing or changing SQL access from Go code
  - Choosing between `pgx`, `database/sql`, `sqlc`, query builders, or ORM
  - Implementing or reviewing transactions, context deadlines, pooling, batching, scanning, and nullability handling
  - Investigating SQL performance regressions (`N+1`, chatty access, pool starvation, slow queries)
  - Defining observability and security rules for DB access
- Do not load when: The task is non-SQL, schema-only modeling without Go access code, or unrelated to runtime data access

## Purpose
- This document defines operational defaults for SQL access from Go in this template.
- Goal: predictable, safe, observable, and performant DB access with minimal LLM guesswork.
- Treat it as an LLM contract: apply defaults first, document deviations explicitly, and reject implicit behavior.

## Baseline assumptions
- Default DB: PostgreSQL-compatible OLTP.
- Default approach: query-first, explicit SQL.
- Default stack: `sqlc` + `pgx/v5` + `pgxpool`.
- Default ownership model: one service owns its DB/schema; no direct cross-service DB access.
- If user does not specify DB engine or portability requirements:
  - assume PostgreSQL,
  - assume service-owned DB,
  - assume explicit SQL with `sqlc`.

## Required inputs before changing SQL access code
Resolve these first. If missing, apply defaults and state assumptions.

- DB engine and version
- Service connection budget (`max DB connections`, max service replicas/instances)
- Query criticality (hot path vs non-critical/admin path)
- Consistency requirements (single transaction, retry policy, idempotency)
- Expected query shape (point read, list with joins, bulk write, dynamic filters)
- Observability requirements (traces/metrics/logs, slow-query thresholds)

## Default stack for template and criteria
Default rule: `sqlc` is mandatory for production code paths.

### Primary default (PostgreSQL-first)
- Use `pgx/v5` + `pgxpool` + `sqlc` (`sql_package: pgx/v5`).
- Keep SQL in `.sql` files with `-- name: <Name> <command>` annotations.
- Use generated methods as the only application DAL API.

Why default:
- Explicit SQL and predictable plans.
- Type-safe Go signatures from `sqlc`.
- Good control over transactions, batching, and Postgres-specific features (`COPY`, SQLSTATE handling).

### Supported alternative (portability-first)
- Use `database/sql` + `sqlc` (`sql_package: database/sql`) if SQL portability across engines is a hard requirement.
- Keep the same query-first discipline and review rules.

### Selection rules
- Choose `pgx` default when:
  - PostgreSQL is fixed,
  - you need max control/perf,
  - you need `COPY`, `pgx.Batch`, or richer Postgres error handling.
- Choose `database/sql` alternative when:
  - portability across engines is required,
  - you intentionally avoid Postgres-specific behavior.
- Reject manual SQL + hand-written scanning for core paths unless there is a measured reason.

## Query discipline and code organization
Default rule: SQL is source of truth, generated Go is derivative.

- Store queries under `sql/queries/*.sql`.
- Never edit generated `sqlc` files manually.
- Regenerate on each SQL/schema change.
- Keep query names stable and semantic (`GetUserByID`, `ListInvoicesByAccount`).
- Prefer explicit column lists; do not use `SELECT *` in API/business paths.
- Dynamic SQL fragments (column names, sort direction, table names) require allow-lists.

## Transaction boundaries and consistency
Default rule: transaction boundary is owned by application/use-case layer.

- Open transaction in one place, pass `tx` down, commit/rollback in the same scope.
- Standard flow: `Begin` -> `defer Rollback` -> operations -> `Commit`.
- Use `sqlc.WithTx` to reuse the same generated methods inside and outside transactions.
- Keep transactions short; no network calls or external RPC inside transaction scope.
- Never model cross-service distributed ACID as default.

Retry rules:
- Retry only full transaction blocks, never partial statements.
- Retry only for explicit transient SQLSTATE list (minimum: `40001`, `40P01`).
- Max 3 attempts with bounded backoff + jitter.
- Retried writes must be idempotent (`ON CONFLICT`, unique idempotency key, or equivalent).

Behavior differences to enforce:
- `database/sql` `BeginTx` context cancellation can trigger automatic rollback.
- `pgxpool` cancellation at `Begin/BeginTx` does not replace explicit `Commit/Rollback` responsibility.
- Therefore `defer Rollback` is mandatory for both, and critical for `pgxpool`.

## Context propagation and timeouts
Default rule: every DB call is context-bounded.

- Pass `ctx context.Context` as first parameter across handler -> service -> repository.
- Use derived context (`context.WithTimeout`) for DB operations with explicit deadlines.
- Never replace request context with `context.Background()` in request path.
- Never call context-less DB methods in production paths.
- Always call cancel function for derived contexts.

Recommended defaults:
- Point read/write DB timeout: 1-2s.
- List/report-like endpoint timeout: 2-5s.
- Any longer timeout requires explicit rationale in code comment or ADR.

## Connection pooling and budget
Default rule: one process-wide pool, explicit limits.

### `pgxpool` default config
- `pool_max_conns`: 20
- `pool_min_conns`: 4
- `pool_max_conn_lifetime`: 1h
- `pool_max_conn_idle_time`: 15m
- `pool_health_check_period`: 1m

### `database/sql` portability config
- `SetMaxOpenConns(20)`
- `SetMaxIdleConns(10)`
- `SetConnMaxLifetime(1h)`
- `SetConnMaxIdleTime(15m)`

Connection budget rule:
- `(max_open_conns_per_instance * max_instances) <= 0.8 * available_db_connections`
- If violated, reduce per-instance pool or add DB proxy/pooler strategy.

Resource return rules:
- `database/sql`: always `rows.Close()` and check `rows.Err()`.
- `pgxpool`: `Query` rows must be closed; `QueryRow` must always call `Scan`.

## Batching and bulk operations
Default rule: avoid per-row round trips.

- For bulk inserts in Postgres use `COPY` (`pgx.CopyFrom` or `sqlc :copyfrom`) by default.
- For multiple independent statements use batching (`pgx.Batch`) when it reduces round trips.
- Reject looped `INSERT` one-by-one for large loads unless measured and justified.
- For bulk updates/deletes use set-based SQL, not row-by-row repository loops.

## Null handling, scanning, and types
Default rule: nullability must be explicit in generated types and reviews.

- `sqlc` mappings are source of truth for DB->Go nullability.
- For `pgx/v5`, default `sqlc` setting: `emit_pointers_for_null_types: true`.
- For `database/sql` mode, use `sql.Null*` types or explicit pointer overrides consistently.
- Never invent non-existent nullable types (for example `sql.NullInt`).
- `QueryRow` errors are handled at `Scan`; never ignore scan result.
- Avoid ambiguous implicit casts in SQL; cast explicitly when type inference is unstable.

## SQL injection and secure query construction
Default rule: parameterize values, allow-list identifiers.

- Never compose value predicates with string concatenation.
- Use positional/named parameters only (`$1`, `sqlc.arg`, `sqlc.narg`).
- Do not try to parameterize identifiers (`ORDER BY`, table/column names); use allow-list switch.
- Keep DB credentials least-privilege:
  - app role cannot run DDL in runtime,
  - separate roles for migrations and app runtime,
  - read-only role for read-only jobs if applicable.
- Never log interpolated SQL with sensitive parameter values.

## N+1 and chatty access patterns
Default rule: query round trips are a budgeted resource.

- Point-read use case target: 1 query.
- List-with-relations target: 1 query (JOIN/CTE) or 2 queries (base list + bulk fetch by IDs).
- Never do “query in loop” over list items.
- Use bulk fetch patterns (`WHERE id = ANY($1)` / `IN (...)`) for related entities.
- Apply keyset pagination for hot/deep lists; avoid deep `OFFSET` in hot paths.

## Query observability and diagnostics
Default rule: every critical query path is observable.

- Instrument DB calls with tracing (OTel semantic conventions).
- Use stable low-cardinality identifiers in metrics/logs (query name, operation type, status).
- Do not use user IDs, emails, UUID request IDs, or raw SQL text as high-cardinality metric labels.
- Publish pool metrics (`in_use`, `idle`, `wait_count`, wait duration).
- Define slow-query thresholds:
  - warn: >=200ms,
  - error: >=1s,
  - with query name and correlation context.
- For repeated slow queries, require plan evidence (`EXPLAIN`/`EXPLAIN ANALYZE`) before redesign.

## ORM/query builder vs explicit SQL
Default rule: explicit SQL is required unless exception criteria are met.

### Query builder is acceptable when
- Query has truly dynamic filters/sorting combinations that are impractical in static SQL files.
- Builder output remains parameterized.
- Dynamic identifiers are allow-listed.
- Generated SQL is still reviewable and traceable.

### ORM is acceptable when
- Scope is non-critical admin/backoffice CRUD.
- Data volume is low and query predictability is verified.
- Lazy-loading/N+1 is explicitly disabled or controlled.
- Team accepts reduced SQL-level control for that boundary.

### Explicit SQL is required when
- Endpoint is in hot path or under strict latency SLO.
- Operation requires predictable locking/transaction semantics.
- Query needs deliberate index/plan control.
- Bulk operations or high-throughput writes are involved.
- Security-sensitive flows require auditable SQL behavior.

## Decision rules (when to deviate from defaults)
- If service must stay Postgres-first and has performance-critical flows:
  - keep `pgx + sqlc`.
- If strict multi-engine portability is required by product constraints:
  - allow `database/sql + sqlc`.
- If dynamic query surface exceeds static SQL maintainability:
  - allow query builder for that bounded component only.
- If ORM is requested for main business path:
  - require measured proof that query count, latency, and transaction behavior stay within SLO.
- If no measurement exists:
  - do not replace explicit SQL in critical paths.

## Anti-patterns to reject
- Raw SQL string formatting with user input.
- Context-less DB calls in production code.
- Missing `Rollback` safety in transactions.
- Missing `rows.Close()`/`rows.Err()` handling.
- `pgxpool.QueryRow` without `Scan`.
- Unlimited or implicit pool configuration.
- Query-per-item loops (`N+1`) in handlers/services.
- Deep `OFFSET` pagination in hot paths.
- Unbounded retries or retrying non-idempotent writes.
- Poor observability: no query names, no latency metrics, no pool metrics.
- ORM default with hidden lazy loading on critical paths.

## MUST / SHOULD / NEVER

### MUST
- MUST use query-first SQL with `sqlc` for production DAL.
- MUST parameterize all SQL value inputs.
- MUST propagate context with deadlines through all DB calls.
- MUST keep transaction boundary explicit and closed (`Commit` or `Rollback`).
- MUST configure pool limits explicitly and verify connection budget.
- MUST close rows and handle scan/iteration errors correctly.
- MUST prevent `N+1` with JOIN/bulk-fetch strategies.
- MUST instrument critical query paths with traces/metrics and slow-query logging.

### SHOULD
- SHOULD keep primary template stack as `pgx/v5 + pgxpool + sqlc`.
- SHOULD enable `emit_pointers_for_null_types` for `pgx/v5` sqlc config.
- SHOULD use `COPY`/batch operations for bulk workloads.
- SHOULD use keyset pagination for large/hot lists.
- SHOULD enforce `sqlc vet` and SQL lint checks in CI.
- SHOULD keep DB runtime role least-privilege and separate migration role.

### NEVER
- NEVER concatenate user values into SQL.
- NEVER pass dynamic identifiers without allow-list.
- NEVER run context-less DB methods in request paths.
- NEVER open long transactions around external I/O.
- NEVER retry every DB error blindly.
- NEVER ship critical DB code without query observability.
- NEVER accept hidden ORM lazy-loading behavior on hot paths.

## Review checklist
Before approving SQL access changes, verify:

- Stack and boundaries:
  - Stack choice (`pgx` vs `database/sql`) is explicit and justified.
  - Query-first discipline is preserved (`sqlc`, `.sql` sources, no manual generated edits).
- Transaction correctness:
  - Boundary is in use-case layer.
  - `defer Rollback` exists.
  - Retry policy is bounded and SQLSTATE-scoped.
  - Idempotency exists for retried writes.
- Context and timeouts:
  - Request context is propagated.
  - No `context.Background()` in request path.
  - DB operations have deadlines.
- Pooling and resources:
  - Pool limits are explicit and within connection budget.
  - `rows.Close()`/`rows.Err()` checks exist.
  - `pgx QueryRow` always ends with `Scan`.
- Query quality:
  - No `N+1` loops.
  - Round-trip count for hot path is bounded and reviewed.
  - Pagination/index strategy is explicit.
- Security:
  - All values parameterized.
  - Dynamic identifiers allow-listed.
  - Least-privilege DB roles and no sensitive data leakage in logs.
- Observability:
  - Query names, latency metrics, trace spans, and pool metrics are present.
  - Slow-query thresholds and action path are defined.

## What good output looks like
- SQL access code is explicit, deterministic, and easy to review.
- Transaction semantics, retries, and idempotency are unambiguous.
- Pool usage is bounded, leaks are prevented, and context cancellation works end-to-end.
- Query count and latency are observable; regressions are detectable before incidents.
- LLM-generated code stays inside safe defaults and avoids hidden ORM/driver behavior.

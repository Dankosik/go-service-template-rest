# Postgres Access Review

## Load When

Load when a diff changes a pgx query seam, row/error handling, transaction
composition, or deadline. Query/index/load belongs to `postgres-performance`;
stale generated SQL belongs to the generated-source owner.

## Decide

Driver code stays in depguard-approved PostgreSQL adapters. `postgres.Open`
returns the configured `*pgxpool.Pool`; use its generated query seam or acquire
and release a connection when the operation needs one. The package-level
`postgres.InTx` owns transaction completion, with generated `Queries.WithTx(tx)`
binding queries inside the callback. The current owners are
`internal/infra/postgres/postgres.go`, `transaction.go`, and `sqlcgen/db.go`.

Translate `pgx.ErrNoRows` into the caller-owned not-found identity at the adapter
edge. Hand-written iteration closes rows when early return can skip exhaustion
and always checks `rows.Err()`; configured linters do not reliably catch pgx row
leaks/errors. Generated `sqlcgen` behavior returns to query/generator source.

The PostgreSQL pool owner publishes statement and idle-transaction timeouts
and configures cancellation. Compare a proposed local timeout with those
current budgets and the caller deadline; a client cancellation may fail when
its separate PostgreSQL cancel connection cannot reach the server. A bare `pgx.Tx` still needs explicit rollback/commit;
context cancellation applied to `BEGIN` does not close later early returns.

Interpolated values belong in parameters/query source. Interpolated identifiers
need an allowlist because placeholders cannot bind identifiers. Do not recommend
per-request prepare: pgx already uses its configured statement protocol/cache.

## Smallest Safe Fix

Use the configured pool and generated query seam, compose atomic work through
`postgres.InTx`, translate not-found at the edge, and add the missing
close/iteration-error handling.

## Prove

Use the existing `pgtest` harness for not-found translation, injected iteration
error, cancellation, rollback/connection reuse, and transaction composition.

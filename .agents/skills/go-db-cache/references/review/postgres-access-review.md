# Postgres Access Review

## Load When

Load when a diff changes a pgx query seam, row/error handling, transaction
composition, or deadline. Query/index/load belongs to `postgres-performance`;
stale generated SQL belongs to the generated-source owner.

## Decide

Driver code stays in depguard-approved PostgreSQL adapters. Calls use
`postgres.Pool.Acquire` with release or `InTx`, not raw `PGX()` on a request
path. Repository methods accept `postgres.Querier` so the same implementation
composes inside a transaction.

Translate `pgx.ErrNoRows` into the caller-owned not-found identity at the adapter
edge. Hand-written iteration closes rows when early return can skip exhaustion
and always checks `rows.Err()`; configured linters do not reliably catch pgx row
leaks/errors. Generated `sqlcgen` behavior returns to query/generator source.

The pool publishes `statement_timeout` and
`idle_in_transaction_session_timeout`; config relates statement/acquire budgets
to the request timeout. A local timeout literal bypasses that validation and a
client cancellation may fail when its separate PostgreSQL cancel connection
cannot reach the server. A bare `pgx.Tx` still needs explicit rollback/commit;
context cancellation applied to `BEGIN` does not close later early returns.

Interpolated values belong in parameters/query source. Interpolated identifiers
need an allowlist because placeholders cannot bind identifiers. Do not recommend
per-request prepare: pgx already uses its configured statement protocol/cache.

## Smallest Safe Fix

Route through `Acquire`/`InTx`, type against `postgres.Querier`, translate
not-found at the edge, and add the missing close/iteration-error handling.

## Prove

Use the existing `pgtest` harness for not-found translation, injected iteration
error, cancellation, rollback/connection reuse, and transaction composition.

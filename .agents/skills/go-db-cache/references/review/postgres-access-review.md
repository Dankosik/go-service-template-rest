# Postgres Access Review

## Behavior Change Thesis
When loaded for a changed pgx access path, this file makes the model separate what
this repository enforces from what only review can catch instead of likely mistake
`assume the pgx-aware linters cover cursor and error hygiene, and spend the finding
on database/sql habits — a per-request Prepare, a hand-bound value — that the
adapter boundary and the generator already prevent`.

## When To Load
Load this when a Go diff changes how a query is issued: which seam it runs through,
how its rows and errors are handled, or how its deadline is set. Query shape, index
fit, and round-trip cost are attributed by `postgres-performance`; a changed `.sql`
file whose generated output is stale is `go-coder`'s drift finding.

## Decision Rubric
- Driver-facing code stays in the PostgreSQL adapter packages `depguard` names. It
  denies `pgx/v5`, `database/sql`, `lib/pq`, and `gorm` everywhere else and keeps
  `sqlcgen` behind that boundary, so a second adapter package fails lint rather than
  review: the finding is where the code belongs, before anything about the query
  itself. Each negation is one directory — `!**/internal/infra/postgres/*.go` does
  not reach a subdirectory, which is why `pgtest` carries its own entry — so the
  landing place is `internal/infra/postgres` itself, not a package beneath it.
- The changed call reaches the database through a seam that carries the acquire
  budget: `postgres.Pool.Acquire` with its release, or `InTx`. `PGX()` hands back
  the raw pool, whose own methods wait for a connection until the caller's whole
  request budget is gone — right at composition, a finding in a request path.
- A repository method typed against `postgres.Querier` composes into a transaction
  unchanged. One typed against `*pgxpool.Pool` cannot, which is what later forces a
  caller to reach past `InTx`.
- A single-row read translates `pgx.ErrNoRows` into the caller's own not-found
  identity at the repository edge. `pgx.ErrNoRows` wraps `sql.ErrNoRows`, so testing
  the `database/sql` sentinel does match — but importing that package outside the
  adapter is the depguard denial above, which is why `pgx.ErrNoRows` is the one the
  repository uses.
- Row iteration closes its rows and checks `rows.Err()` afterwards, and both halves
  reach review: measured against this configuration, neither `sqlclosecheck` nor
  `rowserrcheck` reports a leaked, unchecked `pgx.Rows`. Hand-written pgx iteration
  has no linter behind it.
- Generated code under `sqlcgen` is excluded from lint by path and receives
  `defer rows.Close()` plus `rows.Err()` from the generator, so a defect visible
  there belongs to its `queries/*.sql` source.
- The pool publishes `statement_timeout` and `idle_in_transaction_session_timeout`
  as session parameters on every connection, so a statement is already bounded
  server-side, and config validation ties the statement and acquire budgets to the
  request timeout so the three are one budget. A local `context.WithTimeout` literal
  escapes that validation and still does not bound the server, because pgx sends
  cancellation as a separate `CancelRequest` on a second connection — exactly what
  fails to arrive when the network is the fault.

## Reject
- A fix that ends a bare `pgx.Tx` by relying on context cancellation. pgx gives the
  context to `BEGIN` alone, so any early return leaves the transaction open;
  `pgxpool` then destroys that connection instead of reusing it, and the rows stay
  locked until the backend goes away or `idle_in_transaction_session_timeout` fires.
- "Bind it as a parameter" offered against an interpolated identifier. Values are
  bound already — the default exec mode prepares and caches every statement over
  the extended protocol — so an identifier needs an allowlist, and an interpolated
  value usually means the statement left `queries/*.sql` and should return there.

## Smallest Safe Fix
- Route the call through `Acquire`/`InTx` instead of `PGX()`.
- Type the repository parameter as `postgres.Querier`.
- Add the `rows.Err()` check after iteration, and `defer rows.Close()` only where an
  early return can skip exhaustion.
- Translate `pgx.ErrNoRows` at the repository edge rather than letting it travel.

## Validation Shape
- A not-found case proves the sentinel is translated, not returned.
- An iteration-error case proves the error reaches the caller.
- A canceled-context case proves the call returns without leaving a transaction open.
- Database-backed cases use the existing `pgtest` harness rather than a second one.

# Transaction Boundary And Commit Outcome

## Behavior Change Thesis
When loaded for write atomicity, retry eligibility, or an ambiguous commit, this
file makes the model set the boundary from business atomicity and route the three
commit outcomes through the pool's existing error identities instead of likely
mistake `retry the failed statement, and treat a commit error as a definite failure
whose outward effect may simply repeat`.

## When To Load
Load this when the change decides what must commit together, whether a failed write
may run again, or what a caller does when a commit's outcome is unknown. Schema
ownership and cross-service recovery stay with `go-data-architecture` and
`go-distributed`.

## Decision Rubric
- Derive the boundary from what must be atomically true together for the business
  rule, then run it through `postgres.Pool.InTx`. A repository method written
  against `postgres.Querier` serves inside and outside that transaction unchanged,
  which is what keeps the boundary the caller's decision.
- Classify a failed write with `postgres.Retryable` — serialization failure and
  deadlock — and replay the whole use case, because application logic chose the
  statements and values the retry has to choose again.
- Give a write an idempotency mechanism before a retry of it becomes observable
  outside the process: `ON CONFLICT`, a compare-and-set version, or a request key.
- Separate the three commit outcomes. Success and definite failure are decidable;
  `postgres.ErrCommitUnknown` means the rows may be committed, so name the status
  check or reconciliation that runs before an outward effect repeats.
- Separate `postgres.ErrSaturated` — every pooled connection busy for the whole
  acquire budget — from a spent request budget. Saturation is a retryable capacity
  signal the transport answers with 503 and `Retry-After`; a spent budget is 504.
- State an isolation level only when an invariant needs it. `pgx.TxOptions{}` sends
  a bare `begin`, so the transaction takes the server default of Read Committed,
  which re-snapshots per statement and so admits a lost update across a
  read-then-write pair that one `UPDATE` would have survived.

## Reject
- A unique violation classified as retryable with no stated reason the conflict is
  a race. Most are durable business conflicts, so retrying turns a clean rejection
  into a timeout that reports the wrong cause.
- Cache invalidation or queue publication moved inside the transaction to make it
  atomic. It cannot be — the external system takes no part in the commit — and the
  attempt holds the transaction's pooled connection, and any rows it has already
  touched, for a network round trip. A durable side effect participates as a row the
  transaction writes, not a call it makes.

## Agent Traps
- pgx `BeginTx` passes its context to the `BEGIN` command only. There is no
  auto-rollback on cancellation as in `database/sql`; `InTx` owns the rollback, and
  a bare `pgx.Tx` has to be ended explicitly on every path.
- `SendBatch` runs every queued query in one implicit transaction unless the batch
  issues its own transaction control, so a batch is already atomic and a mid-batch
  failure discards all of it.
- Repeated writes reach the server as one statement through array-per-column
  parameters — the shape `postgresoutbox.Append` uses — before they justify a batch
  or a per-row loop.

## Validation Shape
- A rollback case proves a failing later write leaves the earlier one uncommitted.
- A commit-failure case proves no success is reported and that the unknown outcome
  reaches its named reconciliation.
- A retry case proves the replayed use case leaves one effect behind, not two.
- An isolation-dependent invariant gets a concurrent case; an invariant that does
  not depend on isolation gets none.

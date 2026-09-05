# Transaction Boundary And Commit Outcome

## Load When

Load when writes must commit together, retries can repeat work, or commit outcome
can be unknown. Schema and cross-service recovery remain with their owners.

## Decide

Derive the transaction from the business facts that must become true together
and execute through `postgres.Pool.InTx`. Repository methods accept
`postgres.Querier` so callers own whether they run inside that boundary.

Retry the whole use case only for classified serialization/deadlock outcomes and
only after its observable writes are idempotent through conflict handling,
version compare-and-set, or request identity. Most unique violations are durable
business conflict, not retryable races.

Distinguish commit success, definite failure, and `postgres.ErrCommitUnknown`.
Unknown means rows may be committed; run the named status check/reconciliation
before repeating any outward effect. Distinguish pool saturation with a live
caller (503/retry guidance) from spent request budget (504).

Use stronger isolation only for an invariant that needs it. Default read
committed re-snapshots each statement and admits read-then-write lost updates
that one conditional `UPDATE` can avoid. Do not move cache/network/broker calls
inside the database transaction; only a durable row intent can participate.

Bare pgx transactions require explicit completion on every path. `SendBatch`
already uses one implicit transaction unless it contains transaction control.

## Prove

Cover rollback after a later failure, commit-unknown without success, same-key
retry leaving one effect, and a real concurrent case only when the invariant
depends on isolation.

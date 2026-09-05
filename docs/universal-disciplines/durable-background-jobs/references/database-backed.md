# Database-Backed Jobs

Load for PostgreSQL or another database-backed queue after verifying its schema
and version.

PostgreSQL row locks end with the transaction and have no wall-clock expiry.
Use `FOR UPDATE SKIP LOCKED` only in a short claim transaction: select eligible
rows in stable order, write running state, owner, attempt, increasing
generation, claim and expiry timestamps, then commit before business work.
Condition renewal, checkpoint, completion, and failure on job, owner,
generation, and current state. A sweeper applies the accepted expiry
disposition. A `locked_at` timeout is an application lease, not a PostgreSQL row
lock.

Commit business row and job row together when they share the database. Queue
deduplication keys do not replace a permanent business-effect ledger, especially
when completed jobs disappear or a locked job permits another enqueue.

Use server time for lease decisions, indexes matching eligibility and expiry,
and row-count checks that reject stale completion. Stable order plus unique
tiebreaker keeps claims explainable; `SKIP LOCKED` can starve rows, so fairness
needs a separate selection rule. Never hold the claim transaction around
network or business work.

Proof uses real PostgreSQL and at least two workers for producer rollback,
expiry recovery, stale-generation rejection, effect-before-completion, and
accepted business state with no visible runnable job.

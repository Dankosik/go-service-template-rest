# PostgreSQL Diagnostics

Load when attribution, plans, waits, I/O, WAL, vacuum, or replication can change
the decision.

Pin timestamp/window, server and PostgreSQL version, database/role/provider,
primary/standby role, application workload, and statistics reset epochs. Bound
catalog queries with selective predicates plus statement/lock timeouts; `LIMIT`
does not bound rows scanned or aggregated. Subtract cumulative snapshots only
inside one uninterrupted epoch. `EXPLAIN ANALYZE` executes the statement, so use
it only when the read/write, trigger, lock, and side-effect boundary is safe.

Time-align application latency/pool wait and host CPU/memory/storage/network
with PostgreSQL activity/waits/blockers and cumulative workload sources. Rank
`pg_stat_statements` by total time, calls, mean/tail evidence, reads, temp work,
and WAL; normalized SQL hides bind skew. A blocked query can have a fast plan,
and `idle in transaction` can retain snapshots/locks.

Read plans from leaves upward and multiply by loops. Compare estimated versus
actual rows, scan condition versus removed filters, join propagation, sort/hash
spills, heap fetches, parallel workers, triggers/JIT, planning time, I/O timing,
and representative parameter classes. Fix the earliest cardinality explosion or
estimate error. A sequential scan may be correct; high cache hit ratio, planner
cost, warm-cache speed, or database CPU alone does not identify the cause.

Use table/index stats only as bounded leads. Dead tuples are not direct bloat;
zero index scans since a reset do not prove an index is unused. Inspect slot
retention, autovacuum/analyze progress, checkpoints, WAL, replication lag,
conflicts, and host limits only when they compete with the leading explanation.

Predeclare a discriminating prediction and record the observation that would
refute it before choosing an intervention.

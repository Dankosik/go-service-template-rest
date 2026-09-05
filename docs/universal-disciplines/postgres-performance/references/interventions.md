# PostgreSQL Interventions

Load only after evidence supports a PostgreSQL cause. Stop at the first rung
that meets the target while preserving correctness and workload shape.

First remove unnecessary calls, rows, columns, round trips, sorts, transaction
time, or repeated work. Then repair query/index/statistics fit: retain sargable
predicates, stable keyset pagination where product semantics permit, statistics
for proven skew/correlation, and indexes derived from predicates, joins,
ordering, selectivity, selected columns, and write rate. Account for overlapping
indexes, storage, WAL, cache, vacuum, write latency, constraint enforcement, and
concurrent-build failure.

For contention, fix the transaction boundary or acquisition order before the
victim query. Move remote/user waits outside transactions. Bound idempotent
deadlock/serialization retries. Pool and bound application concurrency before
raising `max_connections`; budget all replicas/workers/admin reserve and verify
pooling-mode compatibility. Stop adding concurrency when latency rises without
useful throughput.

For maintenance, find long snapshots, slots, or standby feedback blocking
cleanup. Tune autovacuum per proven table need and I/O headroom. Distinguish
dead tuples, table/index bloat, and visibility coverage. `VACUUM FULL`, online
reindex, and rewrites need their own lock, disk, WAL, replica, and failure
envelope.

Treat memory as multiplicative across plan nodes, sessions, and parallel
workers; prefer session/table-local experiments to global settings. Attribute
CPU usefulness before adding parallelism. WAL/checkpoint changes preserve the
accepted durability and recovery window. Replicas need an explicit consistency,
lag, fallback, and replay-conflict contract. Partitioning earns its cost through
measured pruning, retention, lifecycle, or maintenance; sharding follows only
after simpler workload, query, pooling, replica, and schema options fail.

Proof uses the same data, binds, concurrency, cache conditions, window, and
statistics epoch. Measure the target plus protected result correctness, tail
latency/throughput, locks, pool wait, reads/temp/WAL, write cost, replica lag,
resource headroom, failure signal, and rollback. Change one variable per causal
comparison.

# PostgreSQL interventions

Read the branch matching the proven bottleneck. Treat each option as a hypothesis whose causal fit and delta must be validated.

## Query and application work

- Remove N+1 calls, duplicate reads, unused columns, unnecessary `DISTINCT`, avoidable sorts, and repeated computation before tuning the server.
- Batch compatible writes and use `COPY` for bulk ingestion when transaction, error-isolation, and durability semantics permit it.
- Filter and aggregate at the cheapest correct boundary. Preserve sargable comparisons: compatible types, indexable operators, and expressions that match expression indexes.
- Replace deep `OFFSET` pagination with keyset pagination when the product contract supports a stable ordering cursor.
- Shorten transactions and remote work inside them. A lower query time does not release locks or old snapshots held by surrounding application code.
- Cache or materialize only with an explicit freshness and invalidation contract. Measure write and refresh cost.

## Planner statistics and plan selection

- Run or schedule `ANALYZE` when modification volume made statistics stale.
- Raise statistics targets only for columns whose skew is causing material estimate errors.
- Add extended statistics for correlated predicates or groupings that single-column statistics misestimate.
- Test representative bind classes. Parameter skew can make one generic plan unsuitable; fix query or plan strategy at the application boundary when evidence supports it.
- Use planner enable/disable switches only as diagnostic experiments. Prefer accurate statistics, correct cost evidence, and a query/index shape the planner can price.
- Change global cost constants only from measured storage and workload behavior, with a cluster-wide regression check.

## Indexes

- Start from the workload: equality/range predicates, join keys, ordering, selected columns, selectivity, update rate, and competing queries.
- Use B-tree for ordered equality/range access; consider GIN for suitable multi-valued/full-text data, GiST/SP-GiST for supported geometric/search operators, and BRIN for large physically correlated ranges.
- Order multicolumn keys to support the actual predicate and ordering pattern. Validate skip-scan and bitmap behavior on the running PostgreSQL version instead of relying on slogans.
- Use a partial index when its predicate matches a stable selective subset and the query predicate can imply it at planning time.
- Use an expression index when the query uses the same immutable expression.
- Add `INCLUDE` payload columns only when index-only scans are likely and heap visibility, tuple width, and write amplification make the trade worthwhile.
- Check overlap with existing indexes. Every index consumes storage, WAL, cache, vacuum work, and write latency.
- Index referencing foreign-key columns when parent deletes/updates or joins need it; PostgreSQL does not automatically create that referencing index.
- Build production indexes with an approved lock and resource plan. `CREATE INDEX CONCURRENTLY` reduces write blocking but takes more work, lasts longer, has transaction restrictions, and can leave an invalid index after failure.

## Locks, transactions, and hot rows

- Identify the root blocker and the transaction boundary that created it. Rewrite the victim query only when its own work is also material.
- Move network calls and user think-time outside transactions; keep lock acquisition order consistent.
- Split hot counters, queues, or parent rows only after proving row-lock contention. Batch updates when semantics allow.
- Use `SKIP LOCKED` where skipping satisfies the queue contract; repair the contention source separately.
- Bound retries for deadlocks and serialization failures with jitter and idempotency. Retries consume capacity and can amplify overload.
- Use session cancellation as authorized incident mitigation after capturing blocker evidence; repair the transaction boundary for the root cause.

## Vacuum, bloat, and update cost

- Find long transactions, abandoned replication slots, and standby feedback that prevent tuple cleanup before increasing vacuum effort.
- Tune autovacuum per high-churn or very large table when global thresholds react too late. Validate worker capacity and I/O headroom.
- Use a lower table `fillfactor` when updates can become HOT updates and the extra table space is justified.
- Distinguish dead-tuple estimates, table bloat, index bloat, and visibility-map coverage; they require different evidence and remedies.
- Prefer routine `VACUUM` and sustainable autovacuum. `VACUUM FULL` rewrites the table and takes an `ACCESS EXCLUSIVE` lock; reserve it for an approved maintenance plan.
- Use `REINDEX CONCURRENTLY` or an approved online-rewrite tool only after proving index bloat and planning disk, WAL, replica, and failure costs.

## Connections and concurrency

- Budget total database connections across every application instance, worker, migration job, and admin reserve.
- Pool before raising `max_connections` when many clients are idle or bursty. More PostgreSQL backends allocate more resources and can worsen scheduling and memory pressure.
- Choose PgBouncer session or transaction pooling from application semantics. Transaction pooling is incompatible with several session-scoped features.
- Bound application concurrency and queue before PostgreSQL when overload causes collapse. Test throughput while increasing concurrency; stop where latency rises without useful throughput.
- Preserve reserved operational connections for incident access.

## Memory, CPU, and parallelism

- Treat `work_mem` as a per-operation budget multiplied by plan nodes, sessions, and parallel workers. Prefer `SET LOCAL` for a proven spilling workload before a global change.
- Size `shared_buffers` with OS cache and provider constraints in mind; validate physical reads and checkpoint/WAL effects rather than maximizing cache allocation.
- Confirm CPU is executing useful work before adding parallelism. Parallel workers can multiply CPU, memory, and I/O consumption.
- Tune JIT and parallel thresholds only for the measured query class; short OLTP queries often cannot amortize their startup cost.
- Vertical scaling is valid when a correctly shaped workload has a demonstrated resource ceiling and the cost is preferable to architectural change.

## WAL, checkpoints, and durability

- Compare timed versus requested checkpoints, write/sync time, WAL generation, full-page images, storage latency, and replica apply rate over the same window.
- Increase WAL headroom or smooth checkpoints when requested checkpoints or bursts are causal, then verify recovery objectives and disk capacity.
- Reduce avoidable index and row churn to reduce WAL at the source.
- Batch commits or use asynchronous commit only when the durability contract explicitly permits the changed loss window.
- Treat replication slots and archive failures as capacity risks: retained WAL can exhaust disk.

## Read scaling, partitioning, and distribution

- Route eligible reads to replicas only with an explicit consistency contract, lag guardrail, and fallback. Account for recovery conflicts and replay I/O.
- Partition when pruning, retention, bulk lifecycle operations, or per-partition maintenance solves a measured large-table problem. Account for planning and operational cost, and retain indexes required by each access pattern.
- Archive cold data when product and compliance rules allow it; smaller active indexes and tables can reduce working-set pressure.
- Split databases or shard only after simpler workload, pooling, replica, and schema measures cannot meet the growth envelope. Define routing, resharding, cross-shard transaction, and failure semantics first.

## Validation matrix

| Change | Primary proof | Regression proof |
|---|---|---|
| Query rewrite | Same results and lower plan work | Representative parameter classes |
| Index | Lower reads/latency for target | Write latency, WAL, size, overlapping queries |
| Statistics | Better estimates and plan | Planning/ANALYZE cost, other plan shapes |
| Pool/concurrency | Higher useful throughput, lower queue/load | Pool wait, timeout rate, session semantics |
| Memory/config | Spill/I/O/CPU delta | Peak memory, concurrency, restart/rollback |
| Vacuum/reindex | Space/visibility/scan improvement | Lock, WAL, replica lag, ongoing churn |
| Replica/partition | Primary load or pruning delta | Consistency, lag, routing, operational cost |

## Primary references

- [PostgreSQL: Indexes](https://www.postgresql.org/docs/current/indexes.html)
- [PostgreSQL: Planner statistics](https://www.postgresql.org/docs/current/planner-stats.html)
- [PostgreSQL: Query planning configuration](https://www.postgresql.org/docs/current/runtime-config-query.html)
- [PostgreSQL: Resource consumption](https://www.postgresql.org/docs/current/runtime-config-resource.html)
- [PostgreSQL: Routine vacuuming](https://www.postgresql.org/docs/current/routine-vacuuming.html)
- [PostgreSQL: WAL configuration](https://www.postgresql.org/docs/current/wal-configuration.html)
- [PostgreSQL: Table partitioning](https://www.postgresql.org/docs/current/ddl-partitioning.html)
- [PostgreSQL: Hot standby](https://www.postgresql.org/docs/current/hot-standby.html)
- [PgBouncer: pooling modes and compatibility](https://www.pgbouncer.org/features.html)

Use production case studies to inform intervention order; derive settings from the measured system:

- [GitLab: Scaling the GitLab database](https://about.gitlab.com/blog/scaling-the-gitlab-database/)
- [GitLab: Eliminating PostgreSQL subtransactions](https://about.gitlab.com/blog/why-we-spent-the-last-month-eliminating-postgresql-subtransactions/)

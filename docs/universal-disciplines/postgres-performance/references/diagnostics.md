# PostgreSQL diagnostics

Use this reference to build a bounded load profile. Adapt every query to the server version and available privileges. Prefer an existing provider dashboard or APM when it already preserves the needed history.

## Safe collection discipline

- Record timestamp, server identity, PostgreSQL version, database, role, provider, primary/standby role, and statistics-reset times.
- Tag ad-hoc sessions with a recognizable `application_name`. Apply a short `statement_timeout` and `lock_timeout` appropriate to the environment.
- Bound catalog queries with selective predicates and a statement timeout. `LIMIT` bounds returned rows, not necessarily rows scanned, aggregated, or sorted. Avoid unbounded application-table scans during an incident.
- Capture interval snapshots in separate transactions and subtract cumulative counters. Record reset boundaries; compare within one uninterrupted statistics epoch.
- Start with query identifiers and metadata. Retrieve query text for shortlisted identifiers when it is needed and allowed; redact literals, relation names, and identifiers containing secrets or personal data before sharing artifacts.
- Start with `EXPLAIN` when execution cost is unknown. `EXPLAIN ANALYZE` executes reads and writes; wrap mutating statements in `BEGIN`/`ROLLBACK` only when triggers, external side effects, and lock impact are understood.
- Treat `auto_explain.log_analyze`, per-node timing, `track_io_timing`, and broad statement logging as instrumentation with overhead. Sample and time-box them.

## Identity and reset boundaries

```sql
SELECT clock_timestamp() AS captured_at,
       version(),
       current_database(),
       pg_is_in_recovery() AS is_standby;

SELECT stats_reset
FROM pg_stat_database
WHERE datname = current_database();

SELECT stats_reset, dealloc
FROM pg_stat_statements_info;
```

The last query requires `pg_stat_statements`. A high or changing `dealloc` count means the extension is evicting entries, weakening workload rankings.

## Active work, waits, and blockers

```sql
SELECT pid,
       application_name,
       state,
       wait_event_type,
       wait_event,
       age(clock_timestamp(), xact_start) AS xact_age,
       age(clock_timestamp(), query_start) AS query_age,
       query_id
FROM pg_stat_activity
WHERE datname = current_database()
  AND pid <> pg_backend_pid()
ORDER BY xact_start NULLS LAST, query_start NULLS LAST
LIMIT 50;

SELECT a.pid AS blocked_pid,
       pg_blocking_pids(a.pid) AS blocking_pids,
       a.wait_event_type,
       a.wait_event,
       age(clock_timestamp(), a.query_start) AS blocked_for,
       a.query_id
FROM pg_stat_activity AS a
WHERE cardinality(pg_blocking_pids(a.pid)) > 0
ORDER BY a.query_start
LIMIT 50;
```

`state = 'active'` and a non-null wait event means the backend is executing but waiting. A slow blocked query may have a fast plan; attribute blocked time before rewriting it. Long `idle in transaction` sessions can retain snapshots and locks.

## Workload ranking

```sql
SELECT queryid,
       calls,
       total_exec_time,
       mean_exec_time,
       rows,
       shared_blks_hit,
       shared_blks_read,
       temp_blks_read,
       temp_blks_written,
       wal_bytes
FROM pg_stat_statements
WHERE dbid = (SELECT oid FROM pg_database WHERE datname = current_database())
ORDER BY total_exec_time DESC
LIMIT 20;
```

Re-rank the same bounded set by calls, mean time, shared reads, temp blocks, and WAL. `total_exec_time` finds load drivers; mean time alone favors rare outliers. Retrieve bounded query text for shortlisted `queryid` values when required. `pg_stat_statements` stores normalized statements and aggregates parameter values, so obtain representative binds from privacy-safe traces or logs.

## Table, index, and maintenance signals

```sql
SELECT relid::regclass AS relation,
       n_live_tup,
       n_dead_tup,
       n_mod_since_analyze,
       last_autovacuum,
       last_autoanalyze,
       autovacuum_count,
       autoanalyze_count
FROM pg_stat_user_tables
ORDER BY n_dead_tup DESC
LIMIT 20;

SELECT relid::regclass AS relation,
       indexrelid::regclass AS index_name,
       idx_scan,
       idx_tup_read,
       idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan ASC, indexrelid
LIMIT 50;
```

These are cumulative estimates, not direct bloat measurements. Fetch relation sizes for shortlisted indexes. Zero scans since a recent reset do not justify dropping an index; constraints, rare critical queries, and write-side enforcement also matter.

Inspect, when relevant:

- `pg_stat_io` for backend/object/context I/O;
- `pg_stat_database` for transactions, temp files/bytes, deadlocks, and block times;
- `pg_stat_checkpointer` and `pg_stat_wal` for checkpoint and WAL deltas;
- `pg_stat_progress_vacuum`, `pg_stat_progress_create_index`, and other progress views;
- `pg_stat_replication`, `pg_stat_wal_receiver`, replication slots, and conflict views;
- host CPU saturation, run queue, memory pressure, storage latency/IOPS/throughput, and network;
- application pool wait, request rate, retry rate, timeouts, and per-route latency.

## Plan reading

Preserve the plan as JSON when it will be compared mechanically. Read from the leaves upward and account for loops.

1. Compare estimated and actual rows at every material node. Large errors point to stale, insufficient, or independence-assuming statistics, parameter skew, or predicates the planner cannot model.
2. Multiply per-loop work by `loops`. A cheap inner node can dominate when repeated many times.
3. Separate elapsed time from I/O and waits. `BUFFERS` counts PostgreSQL buffer activity; read timing needs `track_io_timing` and still does not replace host evidence.
4. Inspect scan conditions versus filters and “Rows Removed by Filter.” A sequential scan is healthy when much of a table is needed; an index scan is costly when it causes many random heap fetches.
5. Inspect sort/hash memory, batches, disk/temp blocks, and row width. A spill is evidence for this operation, not permission for a global `work_mem` increase.
6. Inspect join order and row propagation. Fix the earliest cardinality explosion or estimate error rather than the last expensive node.
7. Inspect heap fetches for index-only scans, WAL for writes, planning time, triggers, JIT, parallel workers planned/launched, and settings that differ from defaults.
8. Compare representative parameter classes. Generic and custom plans may diverge under skew.

## Interpretation traps

- Cache-hit ratio is context, not a verdict; healthy sequential workloads can have lower ratios and pathological workloads can have high ratios.
- Database CPU can be demand, inefficient SQL, connection concurrency, spin/lock work, or maintenance. Attribute it before resizing.
- Replica reads trade primary load for lag, stale reads, replay I/O, and recovery conflicts.
- One fast warm-cache run does not predict cold-cache latency or concurrent throughput.
- Planner cost is a relative estimate in cost units, not execution milliseconds.

## Primary references

- [PostgreSQL: Monitoring Database Activity](https://www.postgresql.org/docs/current/monitoring.html)
- [PostgreSQL: `pg_stat_statements`](https://www.postgresql.org/docs/current/pgstatstatements.html)
- [PostgreSQL: Using `EXPLAIN`](https://www.postgresql.org/docs/current/using-explain.html)
- [PostgreSQL: `EXPLAIN` safety and options](https://www.postgresql.org/docs/current/sql-explain.html)
- [PostgreSQL: `auto_explain`](https://www.postgresql.org/docs/current/auto-explain.html)
- [PostgreSQL: `pgbench`](https://www.postgresql.org/docs/current/pgbench.html)

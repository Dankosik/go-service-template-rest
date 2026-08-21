# PostgreSQL Benchmarks

Load for database, pool, or query measurement.

Put real-PostgreSQL benchmarks under `test/`, use `package integration_test`,
and start the file with `//go:build integration`. Reuse
`runPostgresContainer`, the digest-pinned PostgreSQL 17 Testcontainers
dependency, migration owner, and pool construction instead of adding a mock
database or a second container harness.

The benchmark must:

1. start PostgreSQL, apply migrations, create the pool, seed the declared
   fixture size, and run `ANALYZE` before the timed loop;
2. separate row-count, selectivity, warm/cold, and pool-concurrency cases by
   sub-benchmark name;
3. keep transaction creation, network round trips, row decoding, and commit in
   the timed interval when production owns them;
4. reset write fixtures outside the timed interval, or use a fresh transaction
   and rollback when rollback is not part of the claimed production cost;
5. enforce a per-operation or benchmark-level timeout and verify the smallest
   result invariant;
6. close rows, transactions, pools, contexts, and the container through
   `b.Cleanup`.

Run and compare:

```bash
make bench-db-baseline \
  BENCH_DB_PATTERN=BenchmarkOrderRepository \
  BENCH_DB_WORKLOAD_ID=orders-100k-warm
# Apply the candidate change.
make bench-db \
  BENCH_DB_PATTERN=BenchmarkOrderRepository \
  BENCH_DB_WORKLOAD_ID=orders-100k-warm
make bench-db-compare
```

The target sets `REQUIRE_DOCKER=1`, uses the `integration` build tag, fails if
Docker, `BENCH_DB_WORKLOAD_ID`, or a matching benchmark is unavailable, and
writes results under `.artifacts/bench/db/`. Metadata records the exact
PostgreSQL image digest, a fingerprint of `migrations`, and the named
fixture/workload. The infrastructure check keeps the Testcontainers and
Compose images identical and digest-pinned.

The Testcontainers PostgreSQL dependency runs with `fsync=off`, which is what
makes a per-test database affordable. It also means these benchmarks cannot
measure anything whose cost is a WAL flush: commit durability, `synchronous_commit`,
`commit_delay`, and group commit all look free here, and every absolute number
understates a durably configured server. Take such a claim to a server
configured for durability instead of concluding from this harness.

Use `EXPLAIN (ANALYZE, BUFFERS, WAL, FORMAT JSON)` only as a diagnostic plan
artifact for a representative query. `EXPLAIN ANALYZE` executes the statement,
adds measurement overhead, and does not replace the repeated client-visible
benchmark. Run write plans in a rolled-back transaction or disposable
database. Do not extrapolate a plan from toy data or stale statistics.

## References

- Testcontainers for Go [PostgreSQL module](https://golang.testcontainers.org/modules/postgres/)
- PostgreSQL 17 [`ANALYZE`](https://www.postgresql.org/docs/17/sql-analyze.html) and [`EXPLAIN`](https://www.postgresql.org/docs/17/using-explain.html)

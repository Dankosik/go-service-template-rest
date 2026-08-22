# PostgreSQL Benchmarks

Put real-database benchmarks under `test/` with the `integration` build tag and
reuse the existing PostgreSQL 17 Testcontainers harness, migrations, and pool
construction.

Run directly:

```bash
REQUIRE_DOCKER=1 go test -run '^$' -bench BenchmarkOrderRepository \
  -benchmem -count 10 -tags=integration ./test \
  > .artifacts/bench/db/baseline.txt

# Apply the candidate and repeat to current.txt.
go tool -modfile=tools/go.mod benchstat \
  .artifacts/bench/db/baseline.txt .artifacts/bench/db/current.txt
```

Seed the declared fixture size and run `ANALYZE` before timing. Separate row
count, selectivity, warm/cold state, and concurrency by sub-benchmark. Keep
production-owned transactions, network round trips, decoding, and commit in the
timed interval; reset mutable fixtures outside it.

The test cluster uses `fsync=off`, so it cannot measure WAL flush or durable
commit cost. Use a durability-configured disposable server for those claims.
`EXPLAIN (ANALYZE, BUFFERS, WAL, FORMAT JSON)` is diagnostic evidence, not a
replacement for repeated client-visible measurement.

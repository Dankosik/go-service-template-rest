# PostgreSQL Benchmarks

Put real-database benchmarks under `test/` with the `integration` build tag and
reuse the existing PostgreSQL 17 Testcontainers harness, migrations, and pool
construction.

This template currently ships no feature-owned PostgreSQL benchmark, so live DB
benchmark execution is `not applicable`; do not add a fake repository solely to
exercise tooling. Once a real feature owns `BenchmarkXxx`, capture it through
the same evidence envelope:

```bash
export BENCH_BUDGET='<accepted database budget>'
export BENCH_RESPONSE_OWNER='<response owner>'

REQUIRE_DOCKER=1 \
BENCH_PACKAGE=./test \
BENCH_PATTERN='^Benchmark<FeatureOwner>$' \
BENCH_TAGS=integration \
BENCH_WORKLOAD_ID=<feature-fixture-id> \
BENCH_DEPENDENCY_ID='postgres:17@sha256:<reviewed-digest>' \
BENCH_SCHEMA_PATH=migrations \
BENCH_OUTPUT=.artifacts/bench/db/baseline.txt \
make benchmark-capture

# Apply the candidate, repeat with current.txt, then:
BENCH_BASELINE=.artifacts/bench/db/baseline.txt \
BENCH_CURRENT=.artifacts/bench/db/current.txt \
BENCH_COMPARE_OUTPUT=.artifacts/bench/db/comparison.txt \
make benchmark-compare
```

Seed the declared fixture size and run `ANALYZE` before timing. Separate row
count, selectivity, warm/cold state, and concurrency by sub-benchmark. Keep
production-owned transactions, network round trips, decoding, and commit in the
timed interval; reset mutable fixtures outside it.

The test cluster uses `fsync=off`, so it cannot measure WAL flush or durable
commit cost. Use a durability-configured disposable server for those claims.
`EXPLAIN (ANALYZE, BUFFERS, WAL, FORMAT JSON)` is diagnostic evidence, not a
replacement for repeated client-visible measurement.

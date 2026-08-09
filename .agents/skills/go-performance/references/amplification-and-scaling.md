# Amplification And Scaling

## When To Load

Load this before selecting or reviewing a mechanism whose work grows with
input, data cardinality, traffic, remote round trips, serialization or copies,
fan-out, retries, retained memory, or contention.

## Behavior Change Thesis

Design fails early when it chooses a topology without naming the multiplier;
review fails late when it measures only a small success-path fixture. Both miss
the maximum or failure state where repeated work dominates.

## Decision Rubric

- State the multiplier before the cost: calls, queries, iterations, bytes,
  serial work, or in-flight memory per logical operation as a function of rows,
  page size, fan-out, misses, tenants, retries, or concurrency. Show units and
  the evidence-bounded normal, maximum, and failure or recovery envelope.
- Compare the maximum the contract permits, not only a default fixture. Compound
  failure multipliers such as retries times fan-out or one origin call per item
  during a cache outage.
- Try the smallest mechanism in order: remove repeated work, reuse an existing
  result, or bound the input; then batch, stream, cache, parallelize, or queue
  only when the remaining multiplier requires it and accepted semantics permit
  it.
- When batching wins, close the maximum item and byte count, accumulation and
  flush trigger, added wait bound, ordering, atomicity or partial failure,
  retry and idempotency owner, retained memory, and backpressure. A batch-size
  sweep is proof input, not an architecture default.
- Count observable boundary crossings directly when that answers the decision.
  `otelpgx` traces each query and exports pool wait metrics, so one traced
  request can establish query amplification more directly than a broad
  benchmark.

## Reject

- Prescribing batching, caching, concurrency, or a worker pool before naming
  the multiplier and contract consequence.
- Calling a lower asymptotic or boundary-crossing count a latency or throughput
  win before comparable measurement.
- Clearing an unbounded path from a toy input or success-only benchmark.

## Validation Shape

Before implementation, record the structural boundary that rejects the losing
mechanism and the workload for later measurement. After implementation, prefer
an assertion on the growing count at the contract maximum, plus the narrowest
matching benchmark when a latency, throughput, allocation, or capacity claim is
made. A fake store proves local shape only; real database round-trip and pool
cost use `make bench-db` with a named `BENCH_DB_WORKLOAD_ID`.

Hand retry and degradation policy to `go-reliability`, concurrency correctness
to `go-concurrency`, and query, index, or plan attribution to
[`postgres-performance`](../../../../docs/universal-disciplines/postgres-performance/SKILL.md).

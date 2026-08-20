# Amplification And Scaling

## Load When

Load this before selecting or reviewing a mechanism whose work grows with
input, cardinality, traffic, round trips, copies, fan-out, retries, retained
memory, or contention.

## Decide

- Express dominant time and space complexity in workload variables. Name a
  decision-relevant worst, average, or amortized case, and expose nested scans,
  sorts, copies, membership checks, or remote work. Bind calls, queries, rows,
  bytes, serial work, and in-flight memory to the evidence-bounded normal,
  maximum, and failure or recovery envelope.
- Model the complete logical operation, not only the changed component. Include
  retained stages, nested multipliers, and recovery or replay. Sum serial work;
  bound parallel critical-path time while retaining total resource demand;
  model queues by arrival, service, backlog, and wait. Parallelism and queues do
  not erase work.
- Compare the maximum the contract permits, not only a default fixture. Compound
  failure multipliers such as retries times fan-out or one origin call per item
  during a cache outage. Reopen on a changed formula or assumption; even a
  smaller-than-order-of-magnitude input change can dominate superlinear work.
- Try the smallest mechanism in order: remove repeated work, reuse an existing
  result, or bound the input; then batch, stream, cache, parallelize, or queue
  only when the remaining multiplier requires it and accepted semantics permit
  it.
- Among contract-equivalent mechanisms, choose the simplest one that satisfies
  the envelope; when ownership and operational cost are comparable, choose the
  lower dominant complexity. Better Big-O alone does not justify new state or
  machinery for a bound the simpler path already meets.
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
- Clearing unbounded, retained, or superlinear work from a toy or success-only
  result, a bounded changed component, eventual completion, or same-order input.

## Prove

Before implementation, record the structural boundary that rejects the losing
mechanism and the workload for later measurement. After implementation, prefer
an assertion on the growing count at the contract maximum, plus the narrowest
matching benchmark when a latency, throughput, allocation, or capacity claim is
made. A fake store proves local shape only; real database round-trip and pool
cost use `make bench-db` with a named `BENCH_DB_WORKLOAD_ID`.

Hand retry and degradation policy to `go-reliability`, concurrency correctness
to `go-concurrency`, and query, index, or plan attribution to
[`postgres-performance`](../../../../docs/universal-disciplines/postgres-performance/SKILL.md).

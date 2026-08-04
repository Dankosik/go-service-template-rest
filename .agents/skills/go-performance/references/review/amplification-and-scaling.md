# Amplification And Scaling

## When To Load

Load this when changed code does work per item rather than per request: loops,
repeated encode or parse, queries or dependency calls inside iteration, fan-out,
retries, or fallback on miss.

## Behavior Change Thesis

Two opposite failures live here. Without a named multiplier, review flags a
micro-cost — `fmt.Sprintf` in a loop — that no bound makes material. With only a
success-path benchmark, review clears a change whose new work exists only on
timeout, cache miss, or partial failure, where the multiplier is largest and the
measurement never looked.

## Decision Rubric

- State the multiplier before the cost: work or calls per request as a function
  of rows, page size, fan-out width, cache misses, tenants, or retries. A
  per-item cost with no bound that can grow is not a finding; a bounded one is,
  even when the constant is small.
- Compare the maximum the contract allows, not the benchmark's default input. A
  ten-item fixture cannot show what the endpoint's maximum page size does.
- Failure-mode multipliers compound: retries times fan-out width, one origin
  call per miss during a cache outage, work that continues after the caller is
  gone. The request budget is `http.request_timeout` (8s) under
  `http.write_timeout` (10s) so the `504` can still be written — a downstream
  loop that does not carry the request context outlives the answer it was for.
- Round-trip counts are already observable: `otelpgx` traces every query and
  `otelpgx.RecordStats` exports pool acquire and wait metrics. "How many queries
  does this path make" is answerable from one traced request, which is cheaper
  and more direct than a benchmark.
- A fake store proves local code shape only. Real round-trip and pool-wait cost
  needs the integration seam: `test/`, `package integration_test`,
  `//go:build integration`, run through `make bench-db` with a
  `BENCH_DB_WORKLOAD_ID`.

## Reject

- Prescribing batching, caching, or a worker pool as the correction before the
  multiplier is named and the contract impact is stated. The smallest fix is
  often reusing data the caller already loaded or bounding what is returned.

## Validation Shape

Prefer an assertion on the count that grows — queries, dependency calls,
iterations — at the contract's maximum input, over a broad latency measurement.
Hand retry and degradation policy to `go-reliability`, and query, index, or plan
attribution to
[`postgres-performance`](../../../../../docs/universal-disciplines/postgres-performance/SKILL.md);
keep the finding on the multiplier itself.

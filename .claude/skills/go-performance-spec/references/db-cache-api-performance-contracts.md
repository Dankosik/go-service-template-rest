# DB Cache API Performance Contracts

## Behavior Change Thesis
When loaded for symptom "the bottleneck crosses DB, cache, pagination, retry, or API-visible behavior," this file makes the model surface query, pool, cache, payload, async, and retry contract handoffs instead of likely mistake hiding semantic changes inside local performance tuning.

## When To Load
Load when performance choices affect DB round trips, query shape, connection pool capacity, cache behavior, payload limits, pagination, retry/idempotency, async behavior, or API-visible degradation.

## Decision Rubric
- Choose query or payload contract first when N+1 access, unbounded pagination, optional subresources, or large responses dominate the budget.
- Choose DB pool budget only with per-instance concurrency, replica count, wait duration, timeout, and database capacity assumptions.
- Choose cache acceleration only when cacheability, key dimensions, hit ratio, staleness, stampede protection, and cache-down behavior are explicit.
- Choose a projection or read model when the query shape cannot meet the latency target and data freshness can be modeled by the data owner.
- Choose async API behavior when synchronous latency would be dishonest; hand off acceptance, retry, and status semantics to API/domain owners.
- Choose overload or backpressure response when retry amplification or dependency protection affects the envelope.

## Imitate
- List endpoint: default and max `page_size`, selected sort keys, response-size ceiling, DB round trips `<= 2`, and DB pool wait `p95 <= 5ms`. Copy the split where API owns pagination semantics and performance owns thresholds.
- Cache-backed catalog read: warm-hit, cold-miss, and cache-down budgets; cache-down has shorter timeout and bounded origin concurrency. Copy the explicit non-cacheable authoritative inventory path.
- DB pool capacity: max open connections, expected concurrent queries per instance, wait count/duration thresholds, and rollback if pool waits rise while p99 does not improve. Copy the pool math plus rollback.
- Slow bulk operation becomes async only after enqueue latency, completion latency, backlog, retry, and status polling budgets exist. Copy the honest async boundary.

## Reject
- "Add Redis for speed" with no staleness contract, key dimensions, hit-ratio expectation, or cache-down fallback budget.
- Raising `SetMaxOpenConns` without modeling database capacity, per-instance counts, deployment replicas, and wait-duration evidence.
- Optimizing serialization while the endpoint still allows unbounded `page_size` and large optional subresources by default.
- Returning success before durable acceptance without API handoff for async semantics and retry recovery.

## Agent Traps
- Letting cache or projection become a hidden source of truth.
- Treating DB pool size as an application-only knob while deployment replica count multiplies connections.
- Leaving API-visible payload limits to implementation after deciding payload size dominates performance.
- Ignoring retry amplification because retries belong to reliability; they still change the performance envelope.

## Validation Shape
Use local proof plus runtime signals when DB/cache/deployment shape matters:

```bash
go test -run='^$' -bench='BenchmarkListOrders/(page50|page200|hot_tenant)$' -benchmem -count=20 ./internal/orders > list.txt
benchstat list.txt
go test -run='^$' -bench='BenchmarkCatalogRead/(warm|cold|cache_down)$' -benchmem -count=20 ./internal/catalog > catalog.txt
benchstat catalog.txt
```

For DB pool and cache telemetry, require repository-specific metrics for DB wait count/duration, in-use and idle connections, request latency, cache hit ratio, cache timeout count, dependency error rate, and retry ratio.

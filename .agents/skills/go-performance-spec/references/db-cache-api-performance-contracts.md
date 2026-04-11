# DB Cache API Performance Contracts

## When To Load
Load this when performance choices affect DB round trips, query shape, connection pool capacity, cache behavior, payload limits, pagination, retry/idempotency, async behavior, or API-visible degradation.

Keep the output performance-first and contract-oriented. Hand off primary query design, cache correctness, API contract, reliability, or data architecture when those seams become the main decision.

## Option Comparisons
- Query or payload contract first: choose when N+1 access, unbounded pagination, overlarge payloads, or extra response fields drive the budget.
- DB pool budget: choose when saturation comes from connection waits, long-held rows, or transaction scope.
- Cache acceleration: choose only when cacheability, hit ratio, staleness, stampede protection, and cache-down behavior are explicit.
- Projection or read model: choose when query shape is fundamentally incompatible with the latency target and data freshness can be modeled.
- Async API contract: choose when synchronous latency would be dishonest or would stretch deadlines across variable work.
- Overload/backpressure response: choose when the system must shed, defer, or return `429` or `503` to preserve the envelope.

## Accepted Examples
Accepted example: a list endpoint budget includes `page_size` default and max, selected sort keys, response-size ceiling, DB round-trip count `<= 2`, and pool wait p95 `<= 5ms`. The API contract handoff owns pagination semantics; the performance spec owns the measurement thresholds.

Accepted example: a cache-backed catalog read has warm-hit, cold-miss, and cache-down budgets; cache-down has a shorter timeout and bounded origin concurrency. The spec states that stale inventory is not cacheable and must stay on the authoritative path.

Accepted example: a DB pool capacity change defines max open connections, expected concurrent queries per instance, wait count and wait duration thresholds, and a rollback rule if pool waits rise while endpoint p99 does not improve.

Accepted example: a slow bulk operation is moved to an async contract only after the performance spec states enqueue latency, completion latency, backlog, retry, and operation-status polling budgets.

## Rejected Examples
Rejected example: "add Redis for speed" with no staleness contract, key dimensions, hit-ratio expectation, or cache-down fallback budget.

Rejected example: raising `SetMaxOpenConns` without modeling database capacity, per-instance count, deployment replica count, and wait-duration evidence.

Rejected example: optimizing serialization while the endpoint still allows unbounded `page_size` and returns large optional subresources by default.

Rejected example: changing a synchronous endpoint to return success before durable acceptance without an API handoff for async semantics and retry recovery.

## Pass/Fail Rules
Pass when:
- DB round-trip, query count, pool wait, and timeout budgets are explicit for DB-heavy paths
- cache choices include hit/miss/cache-down budgets and correctness handoff for staleness
- API-visible payload, pagination, rate-limit, overload, and async behavior are surfaced as contract impacts
- performance thresholds distinguish authoritative reads from projections or caches
- validation includes both local proof and runtime telemetry when DB/cache/deployment shape matters

Fail when:
- cache or projection silently becomes a source of truth
- DB pool changes lack pool wait, database capacity, and deployment-replica math
- API limits are left to implementation while payload size dominates performance
- retry behavior can amplify load but is not part of the contract
- a performance claim depends on data/cache/API semantics that are not handed off

## Validation Commands
Use these command patterns for local proof:

```bash
go test -run='^$' -bench='BenchmarkListOrders/(page50|page200|hot_tenant)$' -benchmem -count=20 ./internal/orders > list.txt
benchstat list.txt
go test -run='^$' -bench='BenchmarkCatalogRead/(warm|cold|cache_down)$' -benchmem -count=20 ./internal/catalog > catalog.txt
benchstat catalog.txt
go test -run='^$' -bench='BenchmarkListOrders/page200$' -cpuprofile cpu.pprof -memprofile mem.pprof ./internal/orders
go tool pprof -top cpu.pprof
```

For DB pool and runtime telemetry, require a repository-specific load command plus metrics for `DB.Stats()` wait count, wait duration, in-use connections, idle connections, request latency, cache hit ratio, cache timeout count, and dependency error rate.

## Exa Source Links
- [Go database connection management](https://go.dev/doc/database/manage-connections)
- [database/sql package](https://pkg.go.dev/database/sql)
- [Go diagnostics](https://go.dev/doc/diagnostics)
- [OpenTelemetry Go instrumentation](https://opentelemetry.io/docs/languages/go/instrumentation/)
- [Google SRE: Service Level Objectives](https://sre.google/sre-book/service-level-objectives/)
- [Google SRE: Production Services Best Practices](https://sre.google/sre-book/service-best-practices/)

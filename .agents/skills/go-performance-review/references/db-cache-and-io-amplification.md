# DB, Cache, And I/O Amplification Review

## Behavior Change Thesis
When loaded for symptom "request-path performance depends on DB/cache/query count, pagination, dependency calls, or I/O inside loops," this file makes the model choose a round-trip amplification finding instead of likely mistake "treat the issue as generic DB/cache correctness or trust zero-latency unit tests."

## When To Load
Load this when the performance review touches `database/sql`, query count, DB round trips, cache hit/miss behavior, origin fallback, pagination, dependency calls, request-path HTTP/RPC fan-out, or I/O inside loops.

## Decision Rubric
- Count calls as a function of input size: rows, IDs, page size, cache misses, tenants, downstream endpoints, or retries.
- Ask whether proof includes query/dependency call count before and after the change, not just handler CPU time.
- Treat fake DB/cache benchmarks as local code proof only; they cannot prove round-trip or pool-saturation safety.
- Use DB pool stats, dependency timings, cache hit/miss/error/fallback data, or integration/load evidence when the risk is request-path latency or saturation.
- Keep ownership performance-focused. Hand off correctness, invalidation, transaction, tenant isolation, and cache-key issues when they are the primary risk.
- Use `retry-overload-and-tail-latency.md` when the dominant issue is retry or outage fallback amplification rather than normal request-path round trips.

## Imitate
```text
[high] [go-performance-review] internal/users/handler.go:103
Issue:
Axis: I/O; the changed list handler calls `LoadAvatar` once per returned user after the main query. The endpoint allows 200 users per page, so the new path adds up to 200 cache/DB round trips per request, but the PR evidence only covers a unit test with a zero-latency fake store and no query-count assertion.
Impact:
This can turn one list request into hundreds of downstream operations and saturate the DB/cache pool under normal page sizes, moving both p95 latency and dependency load.
Suggested fix:
Batch avatar loading for the returned IDs or reuse avatar data from the main query if that preserves the response contract. Add a query/dependency-count assertion for max page size and a representative integration or load measurement if batching is not possible.
Reference:
N/A
```

Copy the shape: input-to-call multiplier, why current proof cannot observe it, and a fix that preserves contract or asks for targeted proof.

## Reject
```text
Issue:
This looks like N+1.
Suggested fix:
Batch it.
```

Reject it because it does not show the amplification dimension, changed request path, evidence needed, or whether batching preserves the contract.

```text
Issue:
The benchmark is fast, so DB latency is fine.
```

Reject it when the benchmark uses a fake DB/cache or removes network and pool wait from the measured path.

## Agent Traps
- Letting "DB/cache" automatically become a DB/cache correctness finding when the actual merge risk is query count, dependency load, or p99 latency.
- Ignoring `database/sql` wait counters or dependency timing when pool wait is the suspected bottleneck.
- Missing that client-side filtering after over-fetching can be a performance contract problem, not just a local loop issue.
- Accepting endpoint CPU improvement while DB query count or downstream calls increased.
- Asking for broad load testing when a query-count assertion at max page size would expose the regression.

## Validation Shape
```bash
go test ./internal/users -run '^TestListUsersAvatarQueryCount$' -count=1
go test ./internal/users -run '^TestListUsersMaxPageDependencyCalls$' -count=1
go test -run '^$' -bench '^BenchmarkListUsers/(page=50|page=200)$' -benchmem -count=10 ./internal/users > new.txt
benchstat old.txt new.txt
go test -run '^$' -bench '^BenchmarkListUsers/page=200$' -trace trace.out ./internal/users
go tool trace trace.out
```

For live or integration proof, record workload, cache state, page size, downstream latency fixture, query count, and DB pool stats rather than only total handler time.

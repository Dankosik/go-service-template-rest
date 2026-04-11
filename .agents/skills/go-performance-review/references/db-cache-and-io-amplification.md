# DB, Cache, And I/O Amplification Review

## When To Load
Load this when a performance review touches `database/sql`, query count, DB round trips, cache hit/miss behavior, origin fallback, pagination, dependency calls, request-path HTTP/RPC fan-out, or I/O inside loops.

Keep ownership performance-focused. Hand off correctness, invalidation, transaction, tenant isolation, and cache key issues to the DB/cache or security review lanes when those are the primary risk.

## Review Smell Patterns
- A handler loops over IDs and issues one DB query, cache lookup, HTTP call, or RPC per item.
- A cache miss path fans out to origin once per item instead of batching or coalescing when the contract allows it.
- A fallback path calls the origin on every cache error and can multiply load during cache outage.
- The diff adds deep `OFFSET` pagination or client-side filtering after over-fetching.
- Repeated identical reads happen in the same request without a local reuse or explicit reason.
- `database/sql` `DBStats` wait counters or dependency timings are ignored even though connection-pool wait is the suspected bottleneck.
- A benchmark uses a fake DB or cache with no latency, so it cannot prove round-trip amplification is safe.
- The PR reports endpoint CPU improvement while DB query count or downstream calls increased.

## Evidence Required
- Query or dependency call count before and after the change, tied to input size.
- Representative dependency timing or integration/load evidence for request-path latency claims.
- DB connection pool evidence when the risk is saturation: `DBStats`, wait duration, open/in-use connections, or repo-specific DB metrics.
- Cache hit, miss, stale, error, and fallback path evidence when cache behavior drives latency.
- For API-visible pagination or payload shape, state the dataset size and max page/filter contract that bounds work.

## Bad Finding
```text
[medium] [go-performance-review] internal/users/handler.go:103
Issue:
This looks like N+1.
Impact:
It will be slow.
Suggested fix:
Batch it.
Reference:
N/A
```

Why it fails: it does not show the amplification dimension, the changed request path, the evidence needed, or whether batching preserves the contract.

## Good Finding
```text
[high] [go-performance-review] internal/users/handler.go:103
Issue:
Axis: I/O; the changed list handler calls `LoadAvatar` once per returned user after the main query. The endpoint allows 200 users per page, so the new path adds up to 200 cache/DB round trips per request, but the PR evidence only covers a unit test with a zero-latency fake store and no query-count assertion.
Impact:
This can turn one list request into hundreds of downstream operations and saturate the DB/cache pool under normal page sizes, moving both p95 latency and dependency load.
Suggested fix:
Batch avatar loading for the returned IDs or reuse avatar data from the main query if that preserves the response contract. Add a query/dependency-count assertion for max page size and a representative integration or load measurement if batching is not possible.
Reference:
Go `database/sql` context/query and `DBStats` docs, plus Go diagnostics guidance for request-path latency evidence.
```

## Validation Command Examples
```bash
go test ./internal/users -run '^TestListUsersAvatarQueryCount$' -count=1
go test ./internal/users -run '^TestListUsersMaxPageDependencyCalls$' -count=1
go test -run '^$' -bench '^BenchmarkListUsers/(page=50|page=200)$' -benchmem -count=10 ./internal/users > new.txt
benchstat old.txt new.txt
go test -run '^$' -bench '^BenchmarkListUsers/page=200$' -trace trace.out ./internal/users
go tool trace trace.out
```

For live or integration proof, record the workload, cache state, page size, downstream latency fixture, query count, and DB pool stats rather than only total handler time.

## Source Links From Exa
- [database/sql package](https://pkg.go.dev/database/sql)
- [Go SQL interface examples](https://go.dev/wiki/SQLInterface)
- [Go Diagnostics](https://go.dev/doc/diagnostics)
- [runtime/trace package](https://pkg.go.dev/runtime/trace)
- [net/http/pprof package](https://pkg.go.dev/net/http/pprof)


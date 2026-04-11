# Workload Profile And Input Shape

## When To Load
Load this when the spec must define traffic mix, data shape, input sizes, cardinality, tenant skew, hot keys, cache warm/cold behavior, dependency state, or benchmark fixtures before selecting performance options.

Keep this as a pre-coding contract. The goal is to make future measurements representative enough to trust, not to design fixture code in detail.

## Option Comparisons
- Production-derived workload: prefer when logs, traces, metrics, or customer data safely expose request mix, payload size, cardinality, and peak/burst shape.
- Synthetic representative workload: use when production data cannot be used, but the spec can encode realistic size buckets, distributions, and concurrency.
- Worst-case envelope: use when correctness or capacity must hold for documented limits, such as max page size or largest accepted upload.
- Median-only workload: reject for tail-sensitive paths unless the path is explicitly best-effort and median performance is the contract.
- Microbenchmark-only workload: useful for narrow hot functions, but insufficient for request paths with DB/cache, scheduling, network, or serialization effects.
- Blocked workload: select when input shape cannot be inferred. Record a user or upstream research decision instead of guessing.

## Accepted Examples
Accepted example: an account summary endpoint defines `small`, `typical`, and `large` fixtures from observed row-count buckets: 10, 250, and 5,000 transactions, with a separate `hot_tenant` case where one tenant owns 40% of rows. Benchmarks must report all buckets separately; the spec does not allow a geomean to hide the large tenant regression.

Accepted example: a cache budget defines warm, cold, and cache-down runs. Warm means the cache is preloaded with the exact tenant, locale, and version dimensions. Cold means the first request after deploy or invalidation. Cache-down means the cache dependency fails fast and the origin path must stay bounded.

Accepted example: a queue worker capacity contract includes arrival rate, burst duration, average and p99 job payload size, retry rate, and poison-message rate. Throughput acceptance uses backlog age rather than only jobs per second.

## Rejected Examples
Rejected example: a benchmark that measures a one-item slice for an endpoint whose public API allows `page_size=200` and whose largest tenant usually fills the page.

Rejected example: a "representative" payload built from hand-written happy-path JSON with no optional fields, no large strings, no high-cardinality dimensions, and no malformed-but-rejected negative cases.

Rejected example: one benchmark that mixes tenants, cache states, and payload sizes in a single result, then reports only a combined improvement.

Rejected example: a production profile captured from an idle replica and used as proof for peak-hour performance.

## Pass/Fail Rules
Pass when:
- workload classes are named and tied to request, message, or job semantics
- input shape includes size, cardinality, distribution, skew, cache state, dependency state, and concurrency when they affect the path
- workload fixtures and measurement labels preserve the dimensions needed for decision-making
- production-derived evidence states its source, sampling window, and privacy or sanitization assumptions
- synthetic data states why it is representative enough and when it must be revisited

Fail when:
- median request shape is treated as the only acceptance path for a tail-sensitive system
- cold start, cache miss, retry, large tenant, or degraded dependency behavior can dominate but is omitted
- fixture shape is impossible under the API contract or too small to exercise the bottleneck
- workload labels are too coarse to explain a regression or choose between options

## Validation Commands
Use command patterns like these when encoding planned proof obligations:

```bash
go test -run='^$' -bench='BenchmarkAccountSummary/(small|typical|large|hot_tenant)$' -benchmem -count=20 ./internal/accounts > workload.txt
benchstat workload.txt
go test -run='^$' -bench='BenchmarkCatalog/(warm|cold|cache_down)$' -benchmem -count=20 ./internal/catalog > cache-shape.txt
benchstat cache-shape.txt
```

For production-derived profiles, require a command or documented runbook step that captures the sampling window and replica selection, for example:

```bash
curl -o cpu.pprof 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30'
go tool pprof -top ./bin/service cpu.pprof
```

## Exa Source Links
- [Go diagnostics](https://go.dev/doc/diagnostics)
- [Go profile-guided optimization](https://go.dev/doc/pgo)
- [Go testing package benchmarks](https://pkg.go.dev/testing/)
- [benchstat command](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [Google SRE: Service Level Objectives](https://sre.google/sre-book/service-level-objectives/)

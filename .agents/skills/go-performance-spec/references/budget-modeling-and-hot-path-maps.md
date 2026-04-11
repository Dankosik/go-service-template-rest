# Budget Modeling And Hot Path Maps

## When To Load
Load this when the performance spec needs explicit budgets, an SLI or SLO tie-in, a hot-path map, a bottleneck hypothesis, or a decomposition of latency, throughput, allocation, CPU, contention, or capacity across components.

Keep the output pre-coding and contract-oriented. Do not prescribe low-level optimizations. State what must be measured, what budget must hold, and what decision changes if evidence fails.

## Option Comparisons
- Operation budget: use when one endpoint, RPC, job step, or message handler has a user-visible target. Compare percentiles, throughput, allocation, and CPU where relevant.
- Budget decomposition: use when several components share an end-to-end budget, such as `api -> domain -> db/cache -> outbound dependency -> marshal`.
- Workload-class budget: use when interactive, bulk, admin, or background workloads have different latency and throughput expectations.
- SLO-linked budget: use when the target should map to a service-level indicator, such as request latency or throughput, instead of an internal average.
- Capacity budget: use when throughput or concurrency is the risk, such as per-instance RPS, queue depth, or saturation headroom.
- Blocked budget: select when the prompt lacks user impact, workload shape, or a measurement source. Record the missing fact rather than inventing a target.

## Accepted Examples
Accepted example: `GET /v1/orders/{id}` is interactive and budgeted at `p95 <= 80ms`, `p99 <= 180ms`, `<= 8 KiB response`, and `<= 4 allocations above baseline` under `150 rps` per service instance. The hot-path map assigns `auth <= 5ms`, `domain <= 10ms`, `db primary read <= 45ms`, `cache read <= 5ms when used`, and `encode/write <= 10ms`, with `15ms` reserved for jitter. The bottleneck hypothesis is DB round-trip and row materialization, not JSON encoding, until profile evidence says otherwise.

Accepted example: an async export path uses separate objectives: enqueue response `p95 <= 250ms`, export completion `95% <= 5m` for `<= 100k rows`, backlog age `p95 <= 2m`, and queue depth below the worker saturation threshold during peak hour. The spec avoids applying an interactive request latency target to the whole export lifecycle.

Accepted example: a cache proposal defines two budgets: cache hit path `p95 <= 25ms` and cache-down fallback `p95 <= 120ms` at capped concurrency. The spec calls out that cache-down behavior may spend more latency budget but must not exceed DB pool saturation.

## Rejected Examples
Rejected example: "make checkout fast" with no user class, percentile, load, input size, or proof command. This cannot guide implementation or validation.

Rejected example: "p99 must be 10ms" copied from a current dashboard without noting that the endpoint's largest tenant payload is 50x larger than the median. Current performance is not automatically a contract.

Rejected example: a budget that sums component targets to `120ms` while the user-visible objective is `p95 <= 100ms`, with no reserve or trade-off.

Rejected example: treating average latency as the primary acceptance metric for a tail-sensitive interactive path.

## Pass/Fail Rules
Pass when:
- each affected operation has an owner, workload class, percentile or throughput target, and input-size boundary
- end-to-end budgets decompose into component budgets with reserve or an explicit contention/variance assumption
- selected SLI/SLO language measures behavior users or dependent systems care about
- hot-path maps distinguish request path, async path, cache-up/cache-down, cold/warm, and degraded dependency cases when relevant
- invented numeric targets are marked as assumptions or blockers until validated

Fail when:
- the target is only "faster", an average, or an unscoped dashboard number
- budgets omit data shape, concurrency, tenant skew, or fallback mode that can dominate the path
- a proposed budget silently requires API, DB/cache, concurrency, or reliability changes without a handoff
- the bottleneck hypothesis is a code guess instead of a measurable hypothesis

## Validation Commands
Use these as planned proof obligations, not as instructions to start optimizing while specifying:

```bash
go test -run='^$' -bench='BenchmarkOrderRead/(small|large|hot_tenant)$' -benchmem -count=20 ./internal/orders > before.txt
go test -run='^$' -bench='BenchmarkOrderRead/(small|large|hot_tenant)$' -benchmem -count=20 ./internal/orders > after.txt
benchstat before.txt after.txt
go test -run='^$' -bench='BenchmarkOrderRead/hot_tenant$' -cpuprofile cpu.pprof -memprofile mem.pprof ./internal/orders
go tool pprof -top cpu.pprof
```

For scenario or production validation, write the exact local load-test, trace, dashboard, or canary query command used by the repository. If the command is not known during specification, record it as a planning blocker.

## Exa Source Links
- [Go diagnostics](https://go.dev/doc/diagnostics)
- [Go testing package benchmarks](https://pkg.go.dev/testing/)
- [benchstat command](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [Google SRE: Service Level Objectives](https://sre.google/sre-book/service-level-objectives/)
- [Google SRE: Embracing Risk](https://sre.google/sre-book/embracing-risk/)

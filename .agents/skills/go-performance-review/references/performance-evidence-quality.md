# Performance Evidence Quality Review

## When To Load
Load this when a review needs to decide whether performance proof is sufficient, when a PR claims faster/lower-allocation/lower-latency behavior, or when the right finding is an evidence gap rather than a proven code defect.

Use this reference to keep performance review evidence-first. Do not block on folklore, personal taste, or micro-optimization preference when the changed path is not hot or the impact is not measurable.

## Review Smell Patterns
- A PR says "faster" but provides no baseline-vs-current comparison.
- A service-level latency claim is supported only by a tiny microbenchmark.
- A benchmark is run once, without `-benchmem` for allocation or GC claims.
- `benchstat` output is absent even though old and new benchmark files exist.
- A CPU profile is used to explain lock wait, network wait, or queue wait.
- A heap profile is used to prove allocation churn without checking `allocs` or `-benchmem`.
- Trace, CPU, heap, block, and mutex tools were collected together without explaining profiler interference.
- The workload shape is not stated: input size, fan-out width, cache hit rate, DB row count, or concurrency level is unknown.
- Evidence uses mocks that remove the actual bottleneck under review, such as DB or network round trips.

## Evidence Required
- Local CPU or allocation claim: old and new benchmark files, repeated runs, realistic inputs, `-benchmem` when allocation or GC is part of the claim, and `benchstat` for the comparison.
- CPU hot spot claim: a CPU profile collected under a representative workload, plus the changed code path visible in `top`, `list`, or call graph output.
- Allocation or GC pressure claim: `-benchmem`, heap or allocs profile, and the workload that makes the allocation rate relevant.
- Contention or queueing claim: block or mutex profile and often `go tool trace`, not just CPU samples.
- Request-path or service-level latency claim: representative request/load evidence, dependency timing or query-count data, and p50/p95/p99 where tail latency is the risk.
- Missing proof: state the smallest evidence that would clear the claim; do not demand broad load testing when a narrow benchmark would prove the local risk.

## Bad Finding
```text
[medium] [go-performance-review] internal/render/render.go:88
Issue:
This probably allocates too much.
Impact:
It may be slower.
Suggested fix:
Use a pool.
Reference:
N/A
```

Why it fails: the finding names no hot path, no measured allocation signal, no scale, and jumps to an optimization before proving allocations are the bottleneck.

## Good Finding
```text
[medium] [go-performance-review] internal/render/render.go:88
Issue:
Axis: Evidence; the PR replaces one shared encoder pass with per-item `json.Marshal`, but the only proof is a single `go test` run with no benchmark, no `-benchmem`, and no row-count or payload-size workload. This is a changed request hot path and the risk is allocation and serialization work per item, not a style preference.
Impact:
At the list endpoint's existing page size, the change can add one encode allocation burst per returned item and move p95 latency under load, but the PR currently has no evidence that the extra work is bounded.
Suggested fix:
Add an old-vs-new benchmark or integration measurement using the endpoint's typical and max page sizes, run it with `-benchmem -count=10`, and compare with `benchstat`. If the delta is noisy or neutral, keep the simpler implementation.
Reference:
Go diagnostics guidance on profiling/tool selection and Go benchmark/benchstat methodology.
```

## Validation Command Examples
```bash
go test -run '^$' -bench '^BenchmarkRenderList$' -benchmem -count=10 ./internal/render > new.txt
benchstat old.txt new.txt
go test -run '^$' -bench '^BenchmarkRenderList$' -cpuprofile cpu.out -memprofile mem.out ./internal/render
go tool pprof -top cpu.out
go tool pprof -top -alloc_space mem.out
```

For service-level claims, prefer the repo's existing load or integration command and record the exact workload, for example page size, concurrency, cache state, and downstream fixture size.

## Source Links From Exa
- [Go Diagnostics](https://go.dev/doc/diagnostics)
- [testing package benchmarks](https://pkg.go.dev/testing)
- [benchstat command](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [runtime/pprof package](https://pkg.go.dev/runtime/pprof)
- [Go execution trace docs](https://pkg.go.dev/runtime/trace)


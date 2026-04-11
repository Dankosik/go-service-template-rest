# Performance Evidence Quality Review

## Behavior Change Thesis
When loaded for symptom "the PR claims faster, lower-latency, or lower-allocation behavior but the proof is thin or mismatched," this file makes the model choose a precise evidence-gap finding instead of likely mistake "accept the claim because some number exists" or "demand broad load testing when a narrow proof would clear the risk."

## When To Load
Load this when proof sufficiency is the main review question and no narrower benchmark, profile, contention, allocation, DB/cache, or retry-overload reference is the better fit.

## Decision Rubric
- Ask what claim is being cleared: local CPU, allocation, contention, request-path latency, service-level latency, dependency load, or capacity.
- Match evidence to that claim. CPU samples do not prove wait reduction; heap in-use does not prove allocation churn; a microbenchmark does not prove service p99.
- Require baseline-vs-current comparison when the PR claims improvement.
- Require workload shape when the claim depends on input size, page size, fan-out width, cache state, DB row count, concurrency, or downstream latency.
- State the smallest proof that would change the decision. Do not ask for load tests when an old-vs-new benchmark with `-benchmem` proves the local claim.
- Treat missing proof as merge-risk only when the changed path is hot, the impact can be material, or the implementation adds complexity that needs to earn its keep.

## Imitate
```text
[medium] [go-performance-review] internal/render/render.go:88
Issue:
Axis: Evidence; the PR replaces one shared encoder pass with per-item `json.Marshal`, but the only proof is a single `go test` run with no benchmark, no `-benchmem`, and no row-count or payload-size workload. This is a changed request hot path and the risk is allocation and serialization work per item, not a style preference.
Impact:
At the list endpoint's existing page size, the change can add one encode allocation burst per returned item and move p95 latency under load, but the PR currently has no evidence that the extra work is bounded.
Suggested fix:
Add an old-vs-new benchmark or integration measurement using the endpoint's typical and max page sizes, run it with `-benchmem -count=10`, and compare with `benchstat`. If the delta is noisy or neutral, keep the simpler implementation.
Reference:
N/A
```

Copy the shape: claim, changed hot path, missing workload dimension, smallest clearing proof.

## Reject
```text
Issue:
This probably allocates too much.
Suggested fix:
Use a pool.
```

Reject it because it names no hot path, no measured allocation signal, no scale, and jumps to an optimization before proving allocations are the bottleneck.

```text
Issue:
The PR needs a load test before merge.
```

Reject it when the risk is local and bounded. Ask for the narrowest benchmark or profile that can prove the claim.

## Agent Traps
- Converting "no evidence" into a blocker even when the changed path is cold and the implementation is simpler.
- Treating any numeric result as proof without checking baseline, workload, sample count, variance, and metric relevance.
- Clearing service-level p99 or capacity claims with a single function benchmark.
- Asking for every diagnostic tool at once instead of the one that fits the symptom.
- Hiding residual risk in summary text instead of writing a finding when missing proof is merge-relevant.

## Validation Shape
```bash
go test -run '^$' -bench '^BenchmarkRenderList$' -benchmem -count=10 ./internal/render > new.txt
benchstat old.txt new.txt
go test -run '^$' -bench '^BenchmarkRenderList$' -cpuprofile cpu.out -memprofile mem.out ./internal/render
go tool pprof -top cpu.out
go tool pprof -top -alloc_space mem.out
```

For service-level claims, prefer the repo's existing load or integration command and record workload dimensions such as page size, concurrency, cache state, and downstream fixture size.

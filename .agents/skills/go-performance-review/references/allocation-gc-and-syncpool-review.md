# Allocation, GC, And SyncPool Review

## Behavior Change Thesis
When loaded for symptom "the diff claims or risks allocation churn, GC pressure, buffer reuse, retained backing arrays, or `sync.Pool` behavior," this file makes the model choose evidence-backed allocation and retention review instead of likely mistake "recommend pooling or manual reuse as a default optimization."

## When To Load
Load this when the review touches allocation churn, GC pressure, heap profiles, allocs profiles, `-benchmem`, buffer reuse, `sync.Pool`, object pooling, large retained backing arrays, or runtime memory metrics.

## Decision Rubric
- Separate allocation churn from retained memory: use `-benchmem` and allocs or allocation-space evidence for churn; use in-use heap evidence for retention.
- Require old-vs-new `-benchmem` when allocation, GC, buffer reuse, or `sync.Pool` claims are used to justify complexity.
- Treat `sync.Pool` as suspicious until evidence shows pooled objects are short-lived, reused often, reset safely, and reduce allocation or GC pressure under the real workload.
- Check for oversized backing arrays, request-specific data leakage, missing reset discipline, and single-threaded toy benchmarks for pooled objects.
- Prefer reducing duplicate materialization, avoiding hot-loop conversions, and changing data flow before adding pooling.
- Use `hot-path-cost-model.md` when the allocation issue is mainly repeated encode/decode or copy amplification in a loop.

## Imitate
```text
[medium] [go-performance-review] internal/encode/encode.go:57
Issue:
Axis: Allocations; the PR adds a `sync.Pool` of response buffers, but there is no `-benchmem` or allocs profile showing buffer allocation is a bottleneck. The pooled buffers are also returned without trimming oversized backing arrays after large exports.
Impact:
The pool may retain rare multi-megabyte buffers and increase steady-state heap while adding reuse complexity, so the claimed GC reduction is unproven and may reverse under mixed workloads.
Suggested fix:
First prove the allocation bottleneck with old-vs-new `-benchmem` and an allocs profile for typical and large responses. If pooling still helps, reset buffers and discard oversized ones before `Put`.
Reference:
N/A
```

Copy the shape: distinguish churn from retention, name the pool-specific hazard, and require proof before keeping the complexity.

## Reject
```text
Issue:
Use sync.Pool for this buffer.
Impact:
It will reduce GC.
```

Reject it because it prescribes pooling without proving allocation churn or GC pressure, and it ignores reset and retention risk.

```text
Issue:
The heap profile is smaller, so total allocations improved.
```

Reject it because in-use heap can improve while allocation churn remains high.

## Agent Traps
- Recommending `sync.Pool` to look performance-savvy when the simpler fix is fewer conversions or one materialization.
- Missing that rare large responses can poison a pool with oversized buffers.
- Forgetting pooled objects can carry request-specific data across requests if reset discipline is incomplete.
- Treating runtime memory metrics as proof without checking whether each metric is cumulative, instantaneous, or a distribution.
- Ignoring concurrency when a pool benchmark only covers single-threaded reuse.

## Validation Shape
```bash
go test -run '^$' -bench '^BenchmarkEncodeResponse/(size=small|size=large)$' -benchmem -count=10 ./internal/encode > new.txt
benchstat old.txt new.txt
go test -run '^$' -bench '^BenchmarkEncodeResponse/size=large$' -memprofile mem.out ./internal/encode
go tool pprof -top -alloc_space mem.out
go tool pprof -top -inuse_space mem.out
go test -run '^$' -bench '^BenchmarkEncodeResponseParallel$' -benchmem -count=10 ./internal/encode > parallel.txt
```

For runtime memory evidence, state metric names and whether rates were derived from cumulative metrics such as heap allocation counters.

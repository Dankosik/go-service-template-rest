# Allocation, GC, And SyncPool Review

## When To Load
Load this when a review touches allocation churn, GC pressure, heap profiles, allocs profiles, `-benchmem`, buffer reuse, `sync.Pool`, object pooling, large retained backing arrays, or runtime memory metrics.

Use this to separate real allocation bottlenecks from speculative allocation cleanup. Pooling and manual reuse add complexity and should earn their keep with evidence.

## Review Smell Patterns
- A PR adds `sync.Pool` without `-benchmem`, heap/allocs profile, or a measured allocation bottleneck.
- A pool stores large buffers or structs without bounding or resetting them before reuse.
- A pooled object may carry request-specific data into the next request.
- The diff converts between `[]byte` and `string` in a hot loop or materializes the same payload repeatedly.
- A handler allocates a new `bytes.Buffer`, encoder, decoder, map, slice, regex, or scratch object per item.
- A heap profile is used to prove allocation churn even though the claim is about total allocations, not live objects.
- Runtime memory metrics are sampled without explaining whether the metric is cumulative, instantaneous, or a distribution.
- A pool is benchmarked only in a single-threaded toy workload while production use is concurrent.

## Evidence Required
- Allocation claim: `-benchmem` old-vs-new results, ideally with benchstat.
- Total allocation churn: allocs profile or memory profile viewed with allocation-space/object focus, plus benchmark allocation metrics.
- Retained memory claim: heap profile with in-use focus and a workload that reaches steady state.
- GC pressure claim: runtime metrics, GC stats, heap/allocs profile, or representative load evidence that connects allocation rate to latency or CPU.
- `sync.Pool` claim: proof that pooled objects are short-lived, reused across many calls, reset safely, and reduce allocations or GC pressure under the real workload.

## Bad Finding
```text
[medium] [go-performance-review] internal/encode/encode.go:57
Issue:
Use sync.Pool for this buffer.
Impact:
It will reduce GC.
Suggested fix:
Pool buffers.
Reference:
N/A
```

Why it fails: it prescribes pooling without proving allocation churn or GC pressure, and it ignores reset and retention risk.

## Good Finding
```text
[medium] [go-performance-review] internal/encode/encode.go:57
Issue:
Axis: Allocations; the PR adds a `sync.Pool` of response buffers, but there is no `-benchmem` or allocs profile showing buffer allocation is a bottleneck. The pooled buffers are also returned without trimming oversized backing arrays after large exports.
Impact:
The pool may retain rare multi-megabyte buffers and increase steady-state heap while adding reuse complexity, so the claimed GC reduction is unproven and may reverse under mixed workloads.
Suggested fix:
First prove the allocation bottleneck with old-vs-new `-benchmem` and an allocs profile for typical and large responses. If pooling still helps, reset buffers and discard oversized ones before `Put`.
Reference:
Go `sync.Pool` purpose and runtime/pprof heap vs allocs profile guidance.
```

## Validation Command Examples
```bash
go test -run '^$' -bench '^BenchmarkEncodeResponse/(size=small|size=large)$' -benchmem -count=10 ./internal/encode > new.txt
benchstat old.txt new.txt
go test -run '^$' -bench '^BenchmarkEncodeResponse/size=large$' -memprofile mem.out ./internal/encode
go tool pprof -top -alloc_space mem.out
go tool pprof -top -inuse_space mem.out
go test -run '^$' -bench '^BenchmarkEncodeResponseParallel$' -benchmem -count=10 ./internal/encode > parallel.txt
```

For runtime memory evidence, state the metric names and whether rates were derived from cumulative metrics such as heap allocation counters.

## Source Links From Exa
- [sync.Pool docs](https://pkg.go.dev/sync#Pool)
- [runtime/pprof package](https://pkg.go.dev/runtime/pprof)
- [runtime/metrics package](https://pkg.go.dev/runtime/metrics)
- [Go Diagnostics](https://go.dev/doc/diagnostics)
- [testing package benchmarks](https://pkg.go.dev/testing)


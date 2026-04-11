# Pprof And Profile Selection Review

## When To Load
Load this when a review touches CPU, heap, allocs, goroutine, block, or mutex profiles; `runtime/pprof`; `net/http/pprof`; pprof screenshots; live profile collection; or profile-based performance claims.

Use profile selection to fit the symptom. A profile is evidence only when it was collected under a workload that exercises the changed path.

## Review Smell Patterns
- CPU profile is used to explain time spent waiting on locks, network, disk, DB, or channels.
- Heap profile is interpreted as "all allocation churn" without checking `allocs`, `-alloc_space`, or `-benchmem`.
- A pprof screenshot has no command, duration, workload, binary, commit, or profile type.
- The changed function does not appear in the profile and no caller/callee path ties it to the claimed bottleneck.
- Live `net/http/pprof` is enabled but the service path, port, or sampling duration is not recorded.
- Multiple profilers are enabled at once even though diagnostics can interfere with each other.
- The PR optimizes a line visible in `top` without checking cumulative cost, call path, or whether the cost is setup outside the hot request.
- A pprof result from synthetic data is used to clear production tail-latency risk.

## Evidence Required
- CPU-bound claim: CPU profile from representative load, with the changed stack present and enough duration to sample the hot path.
- Allocation churn claim: `-benchmem` plus heap or allocs profile, using `-alloc_space` or `-alloc_objects` when total churn matters.
- Live server claim: `net/http/pprof` collection command, profile duration, endpoint/workload, and whether the profile was collected from one representative replica.
- Contention claim: block or mutex profile, and often `go tool trace`; CPU profile alone is usually not enough.
- Memory retention claim: heap profile with `inuse_space` or `inuse_objects`, plus workload timing around GC if retention is the claim.

## Bad Finding
```text
[high] [go-performance-review] internal/cache/cache.go:119
Issue:
CPU pprof says this is bad.
Impact:
It is slow.
Suggested fix:
Rewrite the cache.
Reference:
pprof
```

Why it fails: it does not identify the profile type, workload, stack, measured cost, or why a rewrite is the smallest safe fix.

## Good Finding
```text
[high] [go-performance-review] internal/cache/cache.go:119
Issue:
Axis: Evidence; the PR claims the new cache lock improves p99 latency, but the attached CPU profile cannot prove lock wait. The changed code serializes every miss behind `mu` while calling the origin, so the fitting evidence is a mutex or block profile under concurrent misses, ideally paired with a trace.
Impact:
The change can move tail latency from origin latency to queue wait behind one critical section, and the current profile can pass even when most goroutines are parked rather than consuming CPU.
Suggested fix:
Collect a mutex or block profile under the concurrent miss workload, or move the origin call outside the lock if the existing cache contract allows it. Use the profile to verify the changed lock path no longer dominates wait time.
Reference:
Go diagnostics and runtime/pprof profile selection guidance.
```

## Validation Command Examples
```bash
go test -run '^$' -bench '^BenchmarkCacheMissParallel$' -benchmem -cpuprofile cpu.out ./internal/cache
go tool pprof -top cpu.out
go test -run '^$' -bench '^BenchmarkCacheMissParallel$' -blockprofile block.out -mutexprofile mutex.out ./internal/cache
go tool pprof -top block.out
go tool pprof -top mutex.out
curl -o cpu.out 'http://localhost:6060/debug/pprof/profile?seconds=30'
curl -o heap.out 'http://localhost:6060/debug/pprof/heap'
curl -o allocs.out 'http://localhost:6060/debug/pprof/allocs?seconds=30'
```

For live block or mutex profiles, verify the program enables the relevant runtime sampling rate before treating an empty profile as proof of no contention.

## Source Links From Exa
- [Go Diagnostics](https://go.dev/doc/diagnostics)
- [runtime/pprof package](https://pkg.go.dev/runtime/pprof)
- [net/http/pprof package](https://pkg.go.dev/net/http/pprof)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [cmd/pprof](https://pkg.go.dev/cmd/pprof)


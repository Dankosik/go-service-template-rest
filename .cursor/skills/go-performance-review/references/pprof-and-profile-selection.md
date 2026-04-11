# Pprof And Profile Selection Review

## Behavior Change Thesis
When loaded for symptom "a PR uses CPU, heap, allocs, goroutine, block, mutex, or live `pprof` output as performance evidence," this file makes the model choose symptom-matched profile review instead of likely mistake "treat any profile screenshot as proof, or use CPU profiles to explain waiting and heap profiles to explain allocation churn."

## When To Load
Load this when profile selection, collection quality, or profile interpretation is the deciding issue. Prefer `trace-block-mutex-and-contention.md` when the primary issue is a code-level lock, channel, fan-out, or queueing defect rather than a profile artifact.

## Decision Rubric
- Ask whether the profile type can observe the claimed symptom: CPU for on-CPU work, allocs or `-alloc_space` for churn, in-use heap for retention, block for waiter locations, mutex for contended critical-section holders, and trace for scheduler and blocking shape.
- Require the workload, command, duration, binary/commit, and profile type for profile evidence used to clear risk.
- Check that the changed function or its caller/callee path appears in the profile before treating it as evidence about the diff.
- Do not clear p99, lock wait, network wait, DB wait, or queue wait with CPU samples alone.
- For live `net/http/pprof`, record endpoint, sampling duration, request workload, target replica, and any runtime sampling configuration needed for block or mutex profiles.
- Avoid demanding every profile at once. Ask for the next profile that would discriminate the disputed claim.

## Imitate
```text
[high] [go-performance-review] internal/cache/cache.go:119
Issue:
Axis: Evidence; the PR claims the new cache lock improves p99 latency, but the attached CPU profile cannot prove lock wait. The changed code serializes every miss behind `mu` while calling the origin, so the fitting evidence is a mutex or block profile under concurrent misses, ideally paired with a trace.
Impact:
The change can move tail latency from origin latency to queue wait behind one critical section, and the current profile can pass even when most goroutines are parked rather than consuming CPU.
Suggested fix:
Collect a mutex or block profile under the concurrent miss workload, or move the origin call outside the lock if the existing cache contract allows it. Use the profile to verify the changed lock path no longer dominates wait time.
Reference:
N/A
```

Copy the shape: name why the supplied profile cannot observe the claim, then ask for the discriminating profile or smallest safe code correction.

## Reject
```text
Issue:
CPU pprof says this is bad.
Suggested fix:
Rewrite the cache.
```

Reject it because it does not identify profile type, workload, stack, measured cost, or why a rewrite is the smallest safe fix.

```text
Issue:
The heap profile is lower, so allocations are fixed.
```

Reject it when the claim is allocation churn. Use allocs, `-alloc_space`, or `-benchmem` before clearing churn.

## Agent Traps
- Treating a `top` line as a defect without checking cumulative cost, call path, and whether the cost is in setup outside the hot path.
- Accepting pprof screenshots that omit command, workload, commit, duration, and profile type.
- Treating an empty block or mutex profile as proof when sampling was not enabled.
- Reading mutex profile stacks as waiter locations; they point to the end of critical sections that caused contention.
- Using synthetic data profiles to clear production tail-latency risk without explaining why the workload matches.
- Ignoring profiler interference when many diagnostic modes were enabled together.

## Validation Shape
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

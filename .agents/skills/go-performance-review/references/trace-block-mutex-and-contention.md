# Trace, Block, Mutex, And Contention Review

## When To Load
Load this when a review touches locks, channels, worker pools, queueing, goroutine bursts, fan-out/fan-in, scheduler stalls, blocked goroutines, `go tool trace`, block profiles, or mutex profiles.

Use this to distinguish CPU work from waiting. Tail latency often moves because goroutines are blocked or queued, not because they are burning CPU.

## Review Smell Patterns
- A lock is held while doing network, disk, DB, cache, RPC, logging, or compression work.
- A shared mutex serializes unrelated request paths or all tenants.
- Fan-out launches one goroutine per item without bounding concurrency or proving the maximum input size.
- Fan-in waits for all downstream calls even when cancellation or first-error behavior should stop wasted work.
- A buffered channel or worker queue can grow with input size or request rate.
- A trace shows poor CPU utilization or long runnable/blocked periods, but the review treats it as a CPU optimization problem.
- A block or mutex profile is empty because profiling was not enabled, not because contention was absent.
- The PR reports average latency only, while the changed path creates queue wait or lock convoy risk at p95/p99.

## Evidence Required
- Lock contention claim: mutex profile under the concurrent workload, with the contended lock holder stack identified.
- Channel or synchronization wait claim: block profile, often with `go tool trace` to show scheduling and blocking shape.
- Fan-out latency claim: trace or request-level timing that separates downstream work time, queue wait, and fan-in wait.
- Scheduler utilization claim: execution trace, not just CPU profile, because trace records goroutine scheduling, blocking, syscalls, and GC events.
- Live service contention claim: block/mutex sampling configuration and workload concurrency must be recorded.

## Bad Finding
```text
[medium] [go-performance-review] internal/search/search.go:144
Issue:
This goroutine fan-out might be too much.
Impact:
It may be bad under load.
Suggested fix:
Use a worker pool.
Reference:
N/A
```

Why it fails: it has no bound, no workload, no measured wait signal, and prescribes a worker pool without proving fan-out is the dominant cost.

## Good Finding
```text
[high] [go-performance-review] internal/search/search.go:144
Issue:
Axis: Contention; the changed search path now starts one goroutine per shard and then holds `resultMu` while merging and sorting each shard result. With the current 64-shard production limit, fan-in can serialize large merges behind a shared mutex, but the PR only includes a CPU profile and no block, mutex, or trace evidence.
Impact:
Under concurrent searches, this can convert shard parallelism into p99 queue wait at the merge lock while average CPU remains acceptable, so the provided CPU profile does not clear the tail-latency risk.
Suggested fix:
Move sorting/normalization outside the lock or merge per-shard results after fan-in in one owner goroutine. Validate with a mutex or block profile and trace for the 64-shard workload.
Reference:
Go diagnostics guidance on mutex/block profiles and execution trace use.
```

## Validation Command Examples
```bash
go test -run '^$' -bench '^BenchmarkSearchFanout64$' -benchmem -blockprofile block.out -mutexprofile mutex.out ./internal/search
go tool pprof -top block.out
go tool pprof -top mutex.out
go test -run '^$' -bench '^BenchmarkSearchFanout64$' -trace trace.out ./internal/search
go tool trace trace.out
go tool trace -pprof=sync trace.out > sync.pprof
go tool trace -pprof=sched trace.out > sched.pprof
go tool pprof -top sync.pprof
go tool pprof -top sched.pprof
```

For live services, pair `curl -o trace.out 'http://localhost:6060/debug/pprof/trace?seconds=5'` with the request workload that reproduces the fan-out or contention path.

## Source Links From Exa
- [Go Diagnostics](https://go.dev/doc/diagnostics)
- [runtime/pprof package](https://pkg.go.dev/runtime/pprof)
- [runtime/trace package](https://pkg.go.dev/runtime/trace)
- [cmd/trace](https://go.dev/cmd/trace)
- [More powerful Go execution traces](https://go.dev/blog/execution-traces-2024)


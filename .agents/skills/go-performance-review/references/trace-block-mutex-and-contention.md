# Trace, Block, Mutex, And Contention Review

## Behavior Change Thesis
When loaded for symptom "the changed path can create lock wait, channel wait, queueing, fan-out/fan-in stalls, or scheduler pressure," this file makes the model choose a wait-shape and tail-latency finding instead of likely mistake "tune CPU work or recommend a generic worker pool without proving the wait bottleneck."

## When To Load
Load this when code shape, not just a profile artifact, introduces contention or queueing risk: locks, channels, worker pools, goroutine bursts, fan-out/fan-in, scheduler stalls, block profiles, mutex profiles, or `go tool trace`.

## Decision Rubric
- Name the wait source: mutex wait, channel send/receive wait, fan-in wait, queue wait, runnable backlog, syscall wait, or goroutine explosion.
- State the bound: request concurrency, shard count, item count, queue capacity, tenant count, or downstream call count.
- Treat locks held across network, disk, DB, cache, RPC, logging, compression, or expensive merging as tail-latency risk until bounded or measured.
- Prefer a mutex profile for lock-holder stacks, a block profile for synchronization waits, and trace for scheduler, runnable, fan-out, and wakeup shape.
- Do not prescribe worker pools by default. The smallest fix may be moving work outside a lock, single-owner merge, bounded fan-out, cancellation-aware fan-in, or avoiding shared state.
- Escalate to `go-concurrency-review` when correctness, deadlock, race, goroutine lifecycle, or shutdown safety is the primary issue.

## Imitate
```text
[high] [go-performance-review] internal/search/search.go:144
Issue:
Axis: Contention; the changed search path now starts one goroutine per shard and then holds `resultMu` while merging and sorting each shard result. With the current 64-shard production limit, fan-in can serialize large merges behind a shared mutex, but the PR only includes a CPU profile and no block, mutex, or trace evidence.
Impact:
Under concurrent searches, this can convert shard parallelism into p99 queue wait at the merge lock while average CPU remains acceptable, so the provided CPU profile does not clear the tail-latency risk.
Suggested fix:
Move sorting/normalization outside the lock or merge per-shard results after fan-in in one owner goroutine. Validate with a mutex or block profile and trace for the 64-shard workload.
Reference:
N/A
```

Copy the shape: wait source, bound, why average CPU evidence is insufficient, and the smallest fix that changes the wait shape.

## Reject
```text
Issue:
This goroutine fan-out might be too much.
Suggested fix:
Use a worker pool.
```

Reject it because it has no bound, no workload, no measured wait signal, and prescribes a worker pool without proving fan-out is the dominant cost.

```text
Issue:
The CPU profile is neutral, so the lock change is safe.
```

Reject it because goroutines can be blocked or queued while consuming little CPU.

## Agent Traps
- Looking only at average latency when the diff creates p95/p99 queue wait or lock convoy risk.
- Missing that an empty mutex or block profile can mean sampling was not enabled.
- Treating goroutine-per-item fan-out as safe because tests use small input.
- Folding performance wait findings into concurrency correctness handoff too early; keep the performance finding when the merge risk is tail latency or capacity.
- Recommending channels, pools, or locks as generic fixes without proving they reduce the specific wait source.

## Validation Shape
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

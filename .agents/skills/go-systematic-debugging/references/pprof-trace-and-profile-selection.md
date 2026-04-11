# Pprof, Trace, And Profile Selection

## When To Load
Load this reference when a Go investigation needs profile or trace evidence but the correct artifact is not obvious. It is especially useful for high CPU, memory growth, lock contention, queue wait, scheduler latency, goroutine leaks, and pprof or trace artifact interpretation.

Use it to choose one diagnostic tool at a time.

## Selection Map

| Question | First artifact | Typical command |
|---|---|---|
| Where is active CPU time spent? | CPU profile | `go test -cpuprofile cpu.out` or `/debug/pprof/profile?seconds=30` |
| What allocates or stays live? | heap profile | `go test -memprofile mem.out` or `/debug/pprof/heap` |
| Which goroutines exist now? | goroutine profile | `/debug/pprof/goroutine?debug=2` |
| Where do goroutines block on sync or channels? | block profile | `go test -blockprofile block.out -blockprofilerate=1` |
| Where is mutex contention accumulating? | mutex profile | `go test -mutexprofile mutex.out -mutexprofilefraction=1` |
| Why is timeline or scheduling odd? | execution trace | `go test -trace trace.out` |
| Which trace wait bucket dominates? | trace-derived pprof | `go tool trace -pprof=sync trace.out` |

## Commands
Test and benchmark capture:

```bash
go test ./path/to/pkg -run '^$' -bench '^BenchmarkName$' -cpuprofile cpu.out -benchmem
go test ./path/to/pkg -run '^TestName$' -count=1 -memprofile mem.out
go test ./path/to/pkg -run '^TestName$' -count=1 -blockprofile block.out -blockprofilerate=1
go test ./path/to/pkg -run '^TestName$' -count=1 -mutexprofile mutex.out -mutexprofilefraction=1
go test ./path/to/pkg -run '^TestName$' -count=1 -trace trace.out
go tool pprof -top cpu.out
go tool pprof -top mem.out
go tool pprof -top block.out
go tool pprof -top mutex.out
go tool trace trace.out
```

Trace-derived profiles:

```bash
go tool trace -pprof=sched trace.out > sched.pprof
go tool trace -pprof=sync trace.out > sync.pprof
go tool trace -pprof=syscall trace.out > syscall.pprof
go tool trace -pprof=net trace.out > net.pprof
go tool pprof -top sync.pprof
```

Service capture through `net/http/pprof`:

```bash
go tool pprof 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30'
go tool pprof 'http://127.0.0.1:6060/debug/pprof/heap'
curl -o goroutine.txt 'http://127.0.0.1:6060/debug/pprof/goroutine?debug=2'
curl -o trace.out 'http://127.0.0.1:6060/debug/pprof/trace?seconds=5'
go tool trace trace.out
```

## Evidence To Capture
- why this artifact matches the symptom
- exact capture command, duration, package, benchmark or test selector, and load level
- Go version and relevant flags when comparing profiles
- saved artifact path plus a small textual summary such as `pprof -top`
- whether profiling overhead could affect the result
- baseline and after-fix comparison when performance or growth is the claim

## Bad Debugging Moves
- collecting every profile by habit
- comparing profiles from different workloads, durations, or binary revisions
- using CPU profile to explain a goroutine that is blocked, asleep, or waiting on I/O
- using one heap profile to claim a leak without time-series evidence
- leaving pprof endpoints exposed or unauthenticated after the incident

## Good Debugging Moves
- choose CPU for active compute, block/mutex for waiting, heap for memory, goroutine for current blockers, and trace for timeline
- collect only one profile at a time when precision matters
- use trace when concurrency bottlenecks disappear from CPU samples because goroutines are not running
- keep profile artifacts outside source unless the task explicitly needs checked-in evidence
- pair performance fixes with a before/after command under the same workload

## Source Links
- [Go diagnostics](https://go.dev/doc/diagnostics)
- [runtime/pprof package](https://pkg.go.dev/runtime/pprof)
- [net/http/pprof package](https://pkg.go.dev/net/http/pprof)
- [cmd/trace tool](https://pkg.go.dev/cmd/trace)
- [runtime/trace package](https://pkg.go.dev/runtime/trace)
- [Go blog: profiling Go programs](https://go.dev/blog/pprof)
- [Go blog: execution traces](https://go.dev/blog/execution-traces-2024)

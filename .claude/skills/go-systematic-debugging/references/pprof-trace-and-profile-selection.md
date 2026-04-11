# Pprof, Trace, And Profile Selection

## Behavior Change Thesis
When loaded for ambiguous profile or trace choices, this file makes the model select the artifact that matches active CPU, retention, waiting, or timeline questions instead of collecting everything or using CPU profiles for blocked work.

## When To Load
Load when a Go investigation needs CPU, heap, goroutine, block, mutex, or execution-trace evidence, but the correct artifact is not obvious. Use `runtime-forensics-go.md` first when the main risk is losing volatile live-process evidence before restart.

## Decision Rubric
- Choose CPU only for active computation, not sleeping, channel wait, lock wait, or I/O wait.
- Choose heap for retained objects or allocation pressure; require at least two time-separated points for leak claims.
- Choose goroutine dumps for "what is blocked right now."
- Choose block or mutex profiles for accumulated wait or contention.
- Choose execution trace when ordering, wakeups, runnable bursts, or timeline relationships matter more than aggregate samples.
- Keep workload, duration, binary revision, and flags comparable before making before/after claims.

## Selection Map

| Question | First artifact | Typical command |
|---|---|---|
| Where is active CPU time spent? | CPU profile | `go test -cpuprofile cpu.out` or `/debug/pprof/profile?seconds=30` |
| What allocates or stays live? | heap profile | `go test -memprofile mem.out` or `/debug/pprof/heap` |
| Which goroutines exist now? | goroutine profile | `/debug/pprof/goroutine?debug=2` |
| Where do goroutines block on sync or channels? | block profile | `go test -blockprofile block.out -blockprofilerate=1` |
| Where is mutex contention accumulating? | mutex profile | `go test -mutexprofile mutex.out -mutexprofilefraction=1` |
| Why is scheduling or wakeup timing odd? | execution trace | `go test -trace trace.out` |
| Which trace wait bucket dominates? | trace-derived pprof | `go tool trace -pprof=sync trace.out` |

## Imitate

```bash
go test ./path/to/pkg -run '^TestName$' -count=1 -blockprofile block.out -blockprofilerate=1
go tool pprof -top block.out
```

Use this when latency is mostly waiting and the suspected owner is a channel, lock, or condition wait.

```bash
go test ./path/to/pkg -run '^TestName$' -count=1 -trace trace.out
go tool trace -pprof=sched trace.out > sched.pprof
go tool pprof -top sched.pprof
```

Use this when the unknown is scheduling or wakeup timing rather than aggregate CPU.

## Reject

```bash
go test ./path/to/pkg -run '^TestName$' -cpuprofile cpu.out -memprofile mem.out -blockprofile block.out -mutexprofile mutex.out -trace trace.out
```

This over-collects, increases overhead, and makes it hard to tie one artifact to one hypothesis.

```text
The heap profile is large, so this proves a leak.
```

One heap point can show retention or allocation pressure, but leak claims need time-series evidence under comparable load.

## Agent Traps
- Using CPU profiles to explain blocked goroutines.
- Comparing profiles from different workloads, durations, or binary revisions.
- Forgetting profile overhead can change contention-sensitive behavior.
- Leaving temporary `net/http/pprof` endpoints exposed after the incident.
- Saving profile artifacts in source unless the task explicitly asks for preserved evidence.

## Validation Shape
Record why the selected artifact matches the symptom, exact capture command, duration, package or endpoint, load level, saved artifact path, a small textual summary such as `pprof -top`, and before/after comparability when a performance or growth claim is being made.

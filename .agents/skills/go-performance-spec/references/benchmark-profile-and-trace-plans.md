# Benchmark Profile And Trace Plans

## When To Load
Load this when the spec must write concrete benchmark, profile, trace, or PGO profile plans and acceptance thresholds. Use it to make proof obligations precise without turning the skill into a low-level optimization checklist.

## Option Comparisons
- `testing.B.Loop`: prefer for new Go benchmarks when the project uses a Go version that supports it. It reduces common timer and dead-code-elimination mistakes, but every loop iteration must still do the same kind of work.
- `b.N`-style benchmark: allow for older Go versions or existing local style, but require explicit timer handling when setup or cleanup would pollute the result.
- `RunParallel`: use when the operation itself is meant to be concurrent and the benchmark needs to model concurrent callers.
- `-benchmem`: require when allocations, bytes/op, or GC pressure are part of the decision.
- `benchstat`: require for A/B comparison of repeated benchmark output.
- CPU/heap/mutex/block/goroutine profiles: choose the profile matching the symptom. Do not collect interfering profiles together unless the spec accepts lower precision.
- Runtime trace: choose when timeline, scheduler, blocking, or fan-out sequence matters.
- PGO profile plan: choose only after representative CPU-profile source and release lifecycle are explicit.

## Accepted Examples
Accepted example: a parser hot path adds a focused benchmark with `B.Loop`, fixture sizes `small`, `typical`, and `large`, `-benchmem`, and `-count=20`. The pass rule is `large` p50 ns/op not worse than baseline by more than 3%, allocations/op not worse, and no statistically significant p95 scenario regression in the API benchmark.

Accepted example: a contention hypothesis requires `go test -trace`, `go tool trace -pprof=sync`, and a mutex profile because blocked goroutines may not appear in a CPU profile.

Accepted example: a PGO plan says the first release ships without PGO, production CPU profiles are sampled from several busy replicas for the same wall duration, profiles are merged with `go tool pprof -proto`, and the next release consumes `default.pgo` only if benchmark and canary thresholds hold.

## Rejected Examples
Rejected example: a benchmark where setup allocates the dataset inside the timed loop even though the contract is supposed to measure only query evaluation.

Rejected example: a `RunParallel` benchmark used to justify adding unbounded goroutines in production. A benchmark can prove throughput, not boundedness or cancellation safety.

Rejected example: a PGO profile captured from a microbenchmark and applied to the whole service binary.

Rejected example: a trace collected because it looks detailed, while the stated hypothesis is CPU-bound compression.

## Pass/Fail Rules
Pass when:
- each benchmark names the operation, fixture dimensions, command, repeat count, and metric threshold
- benchmark setup and measured work are separated or deliberately included
- profile plans identify profile type, capture window, workload, and analysis command
- trace plans state the timeline question they must answer
- PGO plans require representative CPU profiles and a release validation loop

Fail when:
- benchmark output is single-run or best-run only
- allocation-sensitive work omits `-benchmem`
- profile type does not match the symptom
- trace/profiling overhead or tool interference is ignored for production capture
- PGO is approved without representative behavior and rollback criteria

## Validation Commands
Use these command patterns in the spec, adjusted to the repository package and benchmark names:

```bash
go test -run='^$' -bench='BenchmarkParser/(small|typical|large)$' -benchmem -count=20 ./internal/parser > old.txt
go test -run='^$' -bench='BenchmarkParser/(small|typical|large)$' -benchmem -count=20 ./internal/parser > new.txt
benchstat old.txt new.txt
go test -run='^$' -bench='BenchmarkParser/large$' -cpuprofile cpu.pprof -memprofile mem.pprof ./internal/parser
go tool pprof -top cpu.pprof
go test -run='^$' -bench='BenchmarkFanout$' -trace trace.out ./internal/fanout
go tool trace trace.out
go tool trace -pprof=sched trace.out > sched.pprof
go tool pprof -top sched.pprof
go tool pprof -proto profile-a.pprof profile-b.pprof > merged.pprof
go build -pgo=merged.pprof ./cmd/service
```

## Exa Source Links
- [Go testing package benchmarks](https://pkg.go.dev/testing/)
- [More predictable benchmarking with testing.B.Loop](https://go.dev/blog/testing-b-loop)
- [benchstat command](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [Go diagnostics](https://go.dev/doc/diagnostics)
- [go tool trace](https://go.dev/cmd/trace)
- [More powerful Go execution traces](https://go.dev/blog/execution-traces-2024)
- [Go profile-guided optimization](https://go.dev/doc/pgo)

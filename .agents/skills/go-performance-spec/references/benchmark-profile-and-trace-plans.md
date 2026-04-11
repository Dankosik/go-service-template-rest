# Benchmark Profile And Trace Plans

## Behavior Change Thesis
When loaded for symptom "the proof type is known but the spec needs concrete Go benchmark/profile/trace obligations," this file makes the model write executable commands and benchmark hygiene rules instead of likely mistake vague "benchmark it" language, timed setup pollution, or mismatched profile/trace collection.

## When To Load
Load after the measurement category is chosen and the spec needs exact Go proof obligations, fixture labels, thresholds, or B.Loop/b.N/profile/trace command shape.

## Decision Rubric
- Prefer `testing.B.Loop` for new benchmarks when the repo Go version supports it; it excludes setup before the loop and cleanup after the loop from timing.
- Preserve `b.N` style for older Go versions or existing benchmark families; require `ResetTimer`, `StopTimer`, or equivalent timer handling when setup or cleanup would pollute measured work.
- Use `RunParallel` only when the operation models concurrent callers; it does not prove production boundedness.
- Require `-benchmem` when allocations, bytes/op, or GC pressure matter.
- Require `benchstat` for A/B comparison of repeated Go benchmark output.
- Choose CPU, heap, allocs, mutex, block, goroutine profile, or runtime trace according to the symptom.
- Treat PGO command examples as lifecycle obligations only after representative CPU profiles are available.

## Imitate
- Parser hot path: benchmark `small`, `typical`, and `large` fixtures, `-benchmem`, `-count=20`, and pass only if large-case ns/op is not worse than baseline by more than the accepted threshold and allocations/op do not regress. Copy the fixture labels plus threshold.
- Contention hypothesis: require `go test -trace`, `go tool trace -pprof=sync`, and a mutex profile because blocked goroutines may not dominate CPU samples. Copy the trace-derived profile use.
- PGO plan: first release without PGO, collect CPU profiles from busy replicas for the same wall duration, merge with `go tool pprof -proto`, and consume `default.pgo` only if benchmark and canary gates pass. Copy the release staging.

## Reject
- Dataset allocation inside the timed loop when the contract measures query evaluation.
- `RunParallel` used to justify unbounded goroutines in production.
- Microbenchmark CPU profile applied as `default.pgo` for an HTTP service binary.
- Trace collection because it "looks detailed" while the stated hypothesis is CPU-bound compression.

## Agent Traps
- Omitting `-run='^$'` and accidentally running unrelated tests during benchmark proof.
- Writing one command but no baseline/candidate pair.
- Collecting profile data from the wrong package or a benchmark fixture that does not match the accepted workload.
- Treating a command snippet as enough when the spec still lacks a threshold.

## Validation Shape
Use command patterns like these with repository-specific names:

```bash
go test -run='^$' -bench='BenchmarkParser/(small|typical|large)$' -benchmem -count=20 ./internal/parser > old.txt
go test -run='^$' -bench='BenchmarkParser/(small|typical|large)$' -benchmem -count=20 ./internal/parser > new.txt
benchstat old.txt new.txt
go test -run='^$' -bench='BenchmarkFanout$' -trace trace.out ./internal/fanout
go tool trace -pprof=sched trace.out > sched.pprof
go tool pprof -top sched.pprof
go tool pprof -proto profile-a.pprof profile-b.pprof > merged.pprof
go build -pgo=merged.pprof ./cmd/service
```

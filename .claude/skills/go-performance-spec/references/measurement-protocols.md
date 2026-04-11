# Measurement Protocols

## Behavior Change Thesis
When loaded for symptom "the spec needs to decide what proof is sufficient," this file makes the model choose a symptom-matched measurement protocol with baseline, variance, and pass/fail rules instead of likely mistake using a familiar microbenchmark, a single best run, or every diagnostic tool at once.

## When To Load
Load when the proof type is still a decision: benchmark, scenario test, profile, trace, load test, canary telemetry, or production capture.

## Decision Rubric
- Focused benchmark: use for stable in-process functions or adapters where setup can be controlled.
- Scenario benchmark or load test: use when the path crosses serialization, DB/cache, scheduling, network, or realistic data shape.
- CPU profile: use when sampled CPU time should explain the bottleneck.
- Heap or allocation profile: use when allocation rate, live heap, retention, or GC pressure is the risk.
- Mutex/block profile or runtime trace: use when CPU is idle, goroutines block, fan-out is suspicious, or scheduler/queueing timeline matters.
- Canary or production telemetry: use when real traffic mix, tenant skew, deployment shape, DB/cache state, or network behavior dominates.
- Require the same workload labels, environment/runtime class, repeat count, and thresholds for baseline and candidate.

## Imitate
- Allocation regression: require `go test -bench` with `-benchmem -count=20`, `benchstat`, and heap profile comparison only if allocation/op or bytes/op worsens above the accepted threshold. Copy the escalation rule.
- Fan-out endpoint: require runtime trace plus synchronization profile because the hypothesis is goroutine blocking and lock contention, not CPU. Copy the symptom-tool match.
- DB pool saturation: require scenario load plus runtime metrics for pool wait count and wait duration, not a microbenchmark of row scanning. Copy the distinction between local micro-cost and system capacity proof.
- PGO adoption: require representative CPU profiles from production or representative scenario workload; reject microbenchmarks as the whole-binary profile source.

## Reject
- Microbenchmark-only approval for a user-visible p99 request path with DB, cache, network, and JSON.
- Rerunning benchmarks until `benchstat` looks favorable. The count and comparison rule must be fixed in the spec.
- Capturing CPU, heap, mutex, and block profiles together and treating all outputs as precise.
- A runtime trace as primary proof for CPU-bound compression.

## Agent Traps
- Selecting tools because they are impressive rather than because they answer the bottleneck hypothesis.
- Forgetting the baseline command and only specifying the candidate command.
- Treating "run pprof" as reproducible proof without workload, profile type, capture window, and analysis command.
- Ignoring measurement overhead when production profiling or tracing is part of the plan.

## Validation Shape
Use commands like these only after substituting the repository package, benchmark name, workload labels, and threshold:

```bash
go test -run='^$' -bench='BenchmarkTarget$' -benchmem -count=20 ./internal/pkg > old.txt
go test -run='^$' -bench='BenchmarkTarget$' -benchmem -count=20 ./internal/pkg > new.txt
benchstat old.txt new.txt
go test -run='^$' -bench='BenchmarkTarget$' -cpuprofile cpu.pprof -memprofile mem.pprof ./internal/pkg
go tool pprof -top cpu.pprof
go test -run='^$' -bench='BenchmarkTarget$' -trace trace.out ./internal/pkg
go tool trace trace.out
```

For production evidence, specify the safe profiling window, isolation rule, and the go/no-go threshold before capture.

# Measurement Protocols

## When To Load
Load this when the spec needs to choose how performance will be measured, how baseline and target are compared, how variance will be handled, or which proof is sufficient for a latency, throughput, allocation, contention, or capacity claim.

Keep the output as a measurement contract. Do not interpret results unless the task is explicitly in validation.

## Option Comparisons
- Focused benchmark: use for stable in-process functions or adapters where setup can be controlled and `-benchmem` can reveal allocation changes.
- Scenario benchmark: use when the path crosses multiple local components, serialization, cache state, or realistic data shape.
- CPU profile: use when CPU time dominates and the bottleneck should appear as sampled execution.
- Heap or allocation profile: use when memory footprint, allocation rate, GC pressure, or object retention is the risk.
- Mutex or block profile: use when CPU is underutilized or goroutines appear blocked on synchronization.
- Runtime trace: use for scheduler, syscall, network, synchronization, fan-out, queueing, or goroutine lifecycle questions that profiles can hide.
- Load or canary telemetry: use when DB/cache, network, deployment, or production traffic shape dominates and a local benchmark cannot prove the contract.

## Accepted Examples
Accepted example: an allocation regression spec requires `go test -bench` with `-benchmem -count=20`, followed by `benchstat`, and a heap profile only if allocation/op worsens above the accepted threshold.

Accepted example: a fan-out endpoint spec requires a runtime trace and `go tool trace -pprof=sync` because the hypothesis is goroutine blocking and lock contention, not CPU cost.

Accepted example: a DB pool saturation spec requires scenario load plus runtime telemetry for `DB.Stats()` wait count and wait duration, not a microbenchmark of row scanning alone.

Accepted example: a PGO adoption spec requires representative CPU profiles from production or a documented representative benchmark, and explicitly rejects microbenchmarks as the profile source for the whole binary.

## Rejected Examples
Rejected example: using only a microbenchmark to approve a user-visible p99 request latency claim for a path that includes DB, cache, network, and JSON encoding.

Rejected example: rerunning benchmarks until `benchstat` reports significance. Pick the count up front and keep it fixed.

Rejected example: collecting CPU, heap, mutex, and block profiles simultaneously, then treating all outputs as precise.

Rejected example: treating a trace as the primary proof for excessive CPU usage when a CPU profile is the right first tool.

## Pass/Fail Rules
Pass when:
- the protocol names baseline, candidate, environment/runtime class, dataset, command, repeat count, and acceptance threshold
- before/after comparisons use the same workload and stable labels
- `benchstat` or an equivalent statistical comparison is required for repeated Go benchmark output
- the selected diagnostic matches the symptom
- production profiling overhead and sampling rules are bounded when production capture is expected

Fail when:
- the proof path cannot be reproduced
- measurement tools are selected because they are familiar rather than because they match the bottleneck hypothesis
- repeated benchmark counts, environment, or dataset shape differ between baseline and candidate
- variance is ignored or hidden by only reporting a single best run
- system-level claims rely on microbenchmarks alone

## Validation Commands
Use these command patterns in spec obligations:

```bash
go test -run='^$' -bench='BenchmarkTarget$' -benchmem -count=20 ./internal/pkg > old.txt
go test -run='^$' -bench='BenchmarkTarget$' -benchmem -count=20 ./internal/pkg > new.txt
benchstat old.txt new.txt
go test -run='^$' -bench='BenchmarkTarget$' -cpuprofile cpu.pprof -memprofile mem.pprof ./internal/pkg
go tool pprof -top cpu.pprof
go test -run='^$' -bench='BenchmarkTarget$' -trace trace.out ./internal/pkg
go tool trace trace.out
go tool trace -pprof=sync trace.out > sync.pprof
go tool pprof -top sync.pprof
```

When production evidence is required, specify the safe profiling window and isolation rule:

```bash
curl -o cpu.pprof 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30'
go tool pprof -top ./bin/service cpu.pprof
```

## Exa Source Links
- [Go diagnostics](https://go.dev/doc/diagnostics)
- [Go testing package benchmarks](https://pkg.go.dev/testing/)
- [benchstat command](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [go tool trace](https://go.dev/cmd/trace)
- [Go profile-guided optimization](https://go.dev/doc/pgo)

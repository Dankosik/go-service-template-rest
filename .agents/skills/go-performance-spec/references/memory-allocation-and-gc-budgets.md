# Memory Allocation And GC Budgets

## When To Load
Load this when the performance spec must define allocation rate, bytes/op, live heap, peak RSS, GC pause, GC CPU overhead, `GOGC`, `GOMEMLIMIT`, memory leak detection, or container memory guardrails.

Keep this pre-coding. State the memory budget, proof path, and runtime guardrails; do not prescribe object-pool, pointer-layout, or escape-analysis refactors unless the approved design already requires them.

## Option Comparisons
- Allocation budget: use when request or job throughput is sensitive to bytes/op, allocs/op, or GC frequency.
- Live-heap budget: use when long-lived data, cache size, queue backlog, or per-tenant state drives memory.
- Peak-memory budget: use when container limits or bursty payloads can trigger OOM risk.
- GC trade-off budget: use when the spec must choose acceptable CPU versus memory behavior for `GOGC` or `GOMEMLIMIT`.
- Runtime telemetry gate: use when local benchmarks cannot prove production memory behavior because traffic mix and live heap differ.
- Blocked memory contract: choose when no workload size, memory limit, or telemetry source exists.

## Accepted Examples
Accepted example: a batch import path budgets `<= 12 KiB/op` and `<= 40 allocs/op` for `10k-row` chunks, live heap `<= 512 MiB` during steady-state import, and no sustained growth after completion. Acceptance requires `-benchmem`, heap profile comparison, and canary telemetry for `go.memory.used`, allocation rate, and GC pause or scheduler latency.

Accepted example: a cache proposal states an L1 cache memory cap of `128 MiB` per process, expected live entries, TTL, and eviction behavior. The spec rejects raising `GOGC` as the primary fix until cache sizing and hit-ratio evidence exist.

Accepted example: a containerized service sets a memory acceptance envelope using the runtime memory metric equivalent to total runtime memory minus released heap, and requires rollback if canary memory stays above 85% of the container limit for 10 minutes.

## Rejected Examples
Rejected example: "reduce allocations" without naming a path, workload, bytes/op target, or user/system consequence.

Rejected example: tuning `GOGC` lower to reduce peak memory without acknowledging the expected increase in GC CPU and possible latency impact.

Rejected example: approving `GOMEMLIMIT` as a safety net while the live heap can exceed the configured limit under the expected workload.

Rejected example: using a heap profile alone to prove request latency improvement when the acceptance budget is p99 latency under load.

## Pass/Fail Rules
Pass when:
- allocation, live heap, peak memory, and GC-related thresholds are tied to workload shape and runtime class
- local proof includes `-benchmem` or memory profiles when allocation claims matter
- runtime validation names memory, GC, scheduler, and latency signals when production shape matters
- `GOGC` and `GOMEMLIMIT` choices state the CPU/memory trade-off and rollback condition
- memory caps for caches, queues, and per-tenant state are explicit

Fail when:
- memory targets are only "lower memory" or "no leaks"
- GC tuning is selected before measuring allocation rate and live heap shape
- local microbenchmarks are used as the only proof for long-lived heap or container memory behavior
- runtime telemetry cannot distinguish allocation rate, live heap, peak memory, and GC/scheduler impact
- a memory fix shifts risk to latency, DB/cache behavior, or rollout without handoff

## Validation Commands
Use these as specification proof obligations:

```bash
go test -run='^$' -bench='BenchmarkImport/(chunk10k|chunk100k)$' -benchmem -count=20 ./internal/importer > import-mem.txt
benchstat import-mem.txt
go test -run='^$' -bench='BenchmarkImport/chunk100k$' -memprofile mem.pprof ./internal/importer
go tool pprof -top mem.pprof
go test -run='^$' -bench='BenchmarkImport/chunk100k$' -trace trace.out ./internal/importer
go tool trace trace.out
```

For production validation, require repository-specific queries for memory used, allocation rate, GC CPU or pause signals, scheduler latency, request latency, and OOM or restart counts.

## Exa Source Links
- [Go garbage collector guide](https://go.dev/doc/gc-guide)
- [Go diagnostics](https://go.dev/doc/diagnostics)
- [runtime/metrics package](https://pkg.go.dev/runtime/metrics)
- [OpenTelemetry Go runtime metrics semantic conventions](https://opentelemetry.io/docs/specs/semconv/runtime/go-metrics/)
- [Go testing package benchmarks](https://pkg.go.dev/testing/)

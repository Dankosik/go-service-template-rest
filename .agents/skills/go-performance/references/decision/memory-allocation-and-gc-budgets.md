# Memory Allocation And GC Budgets

## Behavior Change Thesis
When loaded for symptom "the risk is allocation rate, live heap, GC, memory limit, or container memory," this file makes the model specify memory and GC envelopes with proof trade-offs instead of likely mistake vague "reduce allocations" goals or first-step `GOGC` tuning.

## When To Load
Load when the performance spec must define allocation rate, bytes/op, live heap, peak RSS, GC pause, GC CPU overhead, `GOGC`, `GOMEMLIMIT`, memory leak detection, or container memory guardrails.

## Decision Rubric
- Use allocation budgets when request or job throughput is sensitive to bytes/op, allocs/op, or GC frequency.
- Use live-heap budgets when long-lived data, cache size, queue backlog, or per-tenant state drives memory.
- Use peak-memory budgets when container limits, bursts, large payloads, or queue buildup can trigger OOM risk.
- Use GC trade-off budgets when the spec must choose acceptable CPU versus memory behavior for `GOGC` or `GOMEMLIMIT`.
- Use runtime telemetry gates when local benchmarks cannot prove production memory behavior because traffic mix and live heap differ.
- Block the contract when workload size, memory limit, or telemetry source is missing.

## Imitate
- Batch import budgets `<= 12 KiB/op`, `<= 40 allocs/op` for `10k-row` chunks, live heap `<= 512 MiB` during steady-state import, and no sustained growth after completion. Copy the separation between allocation/op and live-heap behavior.
- L1 cache proposal states `128 MiB` per process, expected live entries, TTL, and eviction behavior, and rejects raising `GOGC` until cache sizing and hit-ratio evidence exist. Copy the cap-before-tuning rule.
- Containerized service uses a memory envelope tied to runtime memory telemetry and rolls back if canary memory stays above 85% of container limit for 10 minutes. Copy the runtime envelope plus duration.

## Reject
- "Reduce allocations" with no path, workload, bytes/op target, or user/system consequence.
- Lowering `GOGC` to reduce peak memory without stating the expected GC CPU and latency trade-off.
- Approving `GOMEMLIMIT` as a hard OOM guard while expected live heap plus runtime overhead is already close to the limit.
- Using a heap profile alone to prove request latency improvement when the acceptance budget is p99 latency under load.

## Agent Traps
- Treating allocation rate, live heap, peak RSS, and GC pause as interchangeable memory signals.
- Choosing object pools or pointer-layout changes before proving allocation shape and ownership.
- Letting cache, queue, or per-tenant memory grow without a cap because the feature is "only" an optimization.
- Forgetting that reducing memory can increase CPU and latency through GC behavior.

## Validation Shape
Use proof obligations that separate allocation, live heap, and runtime memory behavior:

```bash
go test -run='^$' -bench='BenchmarkImport/(chunk10k|chunk100k)$' -benchmem -count=20 ./internal/importer > import-mem.txt
benchstat import-mem.txt
go test -run='^$' -bench='BenchmarkImport/chunk100k$' -memprofile mem.pprof ./internal/importer
go tool pprof -top mem.pprof
```

For production validation, require repository-specific queries for memory used, allocation rate, GC CPU or pause signals, scheduler latency, request latency, OOM events, and restart counts.

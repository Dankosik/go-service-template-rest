# Concurrency Contention And Capacity

## When To Load
Load this when performance depends on goroutine fan-out, worker pools, queue depth, lock contention, scheduler latency, DB pool capacity, backpressure, saturation, overload behavior, or capacity headroom.

Keep this as a specification contract. Do not prescribe a specific lock, channel, goroutine layout, or implementation refactor unless the approved design already requires it.

## Option Comparisons
- Keep serial: choose when the bottleneck is not parallelizable, dependency limits dominate, or added concurrency would increase tail latency.
- Bounded fan-out: choose when independent subwork can run concurrently and there is a clear per-request or per-tenant cap, cancellation rule, and failure aggregation policy.
- Worker pool: choose when queueing and capacity must be controlled across requests or messages.
- Rate limit or shed: choose when protecting a dependency or preserving a better user experience under overload matters more than accepting all work.
- Queue and async process: choose when request-path latency is too variable and delayed completion is acceptable.
- Capacity increase: choose only with evidence that the bottleneck is resource capacity, not lock contention, retry amplification, or poor query shape.

## Accepted Examples
Accepted example: a search aggregation endpoint allows at most 6 outbound shard calls per request, uses the request deadline, and caps global in-flight shard calls per instance. Pass criteria include `p99 <= 300ms`, no goroutine leak after cancellation tests, and trace evidence that scheduler latency does not dominate.

Accepted example: an import worker pool has queue length `<= 1,000`, backlog age `p95 <= 2m`, worker concurrency tied to DB pool budget, and a shed/defer rule when DB wait duration crosses the threshold. The spec records that increasing workers without increasing DB capacity is rejected.

Accepted example: a suspected lock bottleneck requires mutex and block profiles plus trace-derived synchronization profiles before approving a new concurrency structure.

## Rejected Examples
Rejected example: "spawn one goroutine per row" with no row cap, tenant fairness, cancellation, memory bound, or downstream pool budget.

Rejected example: raising worker count because throughput is low while DB wait duration and connection wait count are already high.

Rejected example: adding a cache and worker queue to hide lock contention without measuring whether the lock is the real bottleneck.

Rejected example: treating `go test -race` as the only performance proof. Race checks are necessary for correctness in concurrency changes, but they do not prove latency or capacity.

## Pass/Fail Rules
Pass when:
- concurrency limits are explicit at the per-request, per-tenant, per-instance, and dependency levels that matter
- queue depth, backlog age, worker count, and rejection or deferral behavior are specified when queueing exists
- cancellation and deadline behavior is part of the performance contract
- the measurement plan includes trace or contention profiles when blocking or scheduling is the hypothesis
- capacity changes include dependency pool and downstream saturation checks

Fail when:
- fan-out is unbounded or only bounded by input size
- queueing has no max depth, overload rule, or observability
- capacity is increased without ruling out contention, retries, or dependency pool saturation
- concurrency changes lack race-aware correctness validation and performance proof obligations
- runtime telemetry cannot detect saturation after rollout

## Validation Commands
Use these as specification proof obligations:

```bash
go test -run='^$' -bench='BenchmarkShardFanout/(serial|bounded)$' -benchmem -count=20 ./internal/search > fanout.txt
benchstat fanout.txt
go test -run='^$' -bench='BenchmarkShardFanout/bounded$' -trace trace.out ./internal/search
go tool trace trace.out
go tool trace -pprof=sync trace.out > sync.pprof
go tool trace -pprof=sched trace.out > sched.pprof
go tool pprof -top sync.pprof
go test -race ./internal/search ./internal/importer
```

For production or staging validation, include the local load command and telemetry query that observe queue depth, backlog age, goroutine count, scheduler latency, DB pool wait, and saturation.

## Exa Source Links
- [Go diagnostics](https://go.dev/doc/diagnostics)
- [go tool trace](https://go.dev/cmd/trace)
- [More powerful Go execution traces](https://go.dev/blog/execution-traces-2024)
- [runtime/metrics package](https://pkg.go.dev/runtime/metrics)
- [OpenTelemetry Go runtime metrics semantic conventions](https://opentelemetry.io/docs/specs/semconv/runtime/go-metrics/)
- [Google SRE: Production Services Best Practices](https://sre.google/sre-book/service-best-practices/)

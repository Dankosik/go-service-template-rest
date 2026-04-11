# Concurrency Contention And Capacity

## Behavior Change Thesis
When loaded for symptom "the performance idea adds fan-out, workers, queues, locks, or capacity changes," this file makes the model choose bounded concurrency and saturation proof instead of likely mistake "more goroutines," "more workers," or "bigger pools" without downstream limits.

## When To Load
Load when performance depends on goroutine fan-out, worker pools, queue depth, lock contention, scheduler latency, DB pool capacity, backpressure, saturation, overload behavior, or capacity headroom.

## Decision Rubric
- Keep serial when the bottleneck is not parallelizable, dependency limits dominate, or parallelism worsens tail latency.
- Use bounded fan-out only with per-request, per-tenant, per-instance, and dependency caps where those dimensions matter.
- Use a worker pool when queueing and capacity must be controlled across requests or messages.
- Tie worker concurrency to the downstream pool or resource budget; increasing workers is not a fix for DB waits.
- Choose queue/defer/shed behavior when accepting all work would violate latency, memory, or dependency capacity budgets.
- Require cancellation, deadlines, failure aggregation, and leak-sensitive validation when concurrency changes are part of the contract.

## Imitate
- Search aggregation allows at most 6 outbound shard calls per request, uses the request deadline, and caps global in-flight shard calls per instance. Copy the separate per-request and per-instance caps.
- Import workers use queue length `<= 1,000`, backlog age `p95 <= 2m`, and worker concurrency tied to DB pool budget. Copy the explicit rejection of adding workers while DB waits are high.
- Suspected lock bottleneck requires mutex/block profiles plus trace-derived synchronization profiles before approving a new concurrency structure. Copy the evidence gate before redesign.

## Reject
- One goroutine per row with no row cap, tenant fairness, cancellation, memory bound, or downstream pool budget.
- Raising worker count while DB wait duration and connection wait count are already high.
- Adding a cache and worker queue to hide lock contention without measuring whether the lock is the bottleneck.
- Treating `go test -race` as the only performance proof; race checks do not prove latency or capacity.

## Agent Traps
- Bounding fan-out only by input size, which still permits pathological tenants or payloads to dominate.
- Describing "backpressure" without a queue max, rejection/defer rule, or telemetry signal.
- Treating scheduler latency, blocking, and lock contention as CPU problems.
- Forgetting shutdown and cancellation in a spec because this is "only" performance planning.

## Validation Shape
Use proof obligations that expose blocking and saturation:

```bash
go test -run='^$' -bench='BenchmarkShardFanout/(serial|bounded)$' -benchmem -count=20 ./internal/search > fanout.txt
benchstat fanout.txt
go test -run='^$' -bench='BenchmarkShardFanout/bounded$' -trace trace.out ./internal/search
go tool trace -pprof=sync trace.out > sync.pprof
go tool trace -pprof=sched trace.out > sched.pprof
go tool pprof -top sync.pprof
go test -race ./internal/search ./internal/importer
```

For staging or production validation, require repository-specific metrics for queue depth, backlog age, goroutine count, scheduler latency, DB pool wait, dependency saturation, and shed/defer counts.

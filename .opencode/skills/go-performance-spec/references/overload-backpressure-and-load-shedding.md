# Overload Backpressure And Load Shedding

## Behavior Change Thesis
When loaded for symptom "the performance envelope depends on overload, shedding, degraded results, queues, retries, or tenant fairness," this file makes the model define capacity-protection semantics and retry limits instead of likely mistake best-effort overload behavior, unbounded queueing, or retries that amplify load.

## When To Load
Load when the spec must define overload behavior, load shedding, degraded responses, queue limits, retry amplification, client-side throttling, per-tenant quotas, or capacity-protection thresholds.

## Decision Rubric
- Serve full result when the path remains inside capacity and latency budgets under peak load.
- Serve degraded result when cheaper, lower-quality, stale-enough, or partial data preserves useful behavior and is honest at the API boundary.
- Queue or defer when delayed completion is acceptable and queue depth, backlog age, memory, and time-to-live are bounded.
- Shed load when accepting all work would violate latency, memory, pool, or dependency capacity budgets.
- Use client throttling or quotas when retry or rejected-request traffic can become its own overload source.
- Fail fast when degraded work is still too expensive or correctness requires no partial result.

## Imitate
- Search endpoint degrades from full corpus to recent-index-only when CPU utilization and request queue age cross thresholds. Copy the disclosure of omitted data and proof that degraded mode is cheaper.
- Importer queue caps per-tenant backlog and returns retryable deferral when backlog age exceeds threshold. Copy the retry budget and jitter requirement.
- Dependency protection returns `429` for tenant quota exhaustion and `503` for global overload, with `Retry-After` only when retry is safe and capacity is expected to recover soon. Copy the differentiated status semantics.

## Reject
- Dropping random requests with no criticality, tenant fairness, idempotency, or client retry guidance.
- Adding retries to hide overload without per-request retry budget and client-level retry ratio guard.
- Queueing overflow work with no max depth, memory budget, or time-to-live.
- Degraded response that is as expensive as the full response or violates freshness/correctness expectations.

## Agent Traps
- Treating overload as reliability-only and leaving it out of the performance spec.
- Adding queues that improve request acceptance while silently increasing memory and completion latency risk.
- Emitting `Retry-After` when clients retrying would worsen overload.
- Designing degradation without an API handoff when client-visible semantics change.

## Validation Shape
Use local proof only as part of a broader overload validation plan:

```bash
go test -run='^$' -bench='BenchmarkSearch/(full|degraded|shed)$' -benchmem -count=20 ./internal/search > overload.txt
benchstat overload.txt
go test -run='^$' -bench='BenchmarkSearch/degraded$' -trace trace.out ./internal/search
go tool trace -pprof=sched trace.out > sched.pprof
go tool pprof -top sched.pprof
```

For staging or canary validation, require request latency, error type, shed count, queue depth, backlog age, retry ratio, CPU, memory, DB/cache pool wait, and tenant distribution.

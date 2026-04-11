# Overload Backpressure And Load Shedding

## When To Load
Load this when the performance spec must define overload behavior, load shedding, degraded responses, queue limits, retry amplification, client-side throttling, per-tenant quotas, or capacity-protection thresholds.

Keep this performance-contract-first. Hand off API status semantics, reliability policy, and delivery rollout when the overload behavior becomes client-visible or operationally risky.

## Option Comparisons
- Serve full result: choose when the path remains within capacity and latency budgets under peak load.
- Serve degraded result: choose when cheaper, lower-quality, or stale-enough data preserves useful behavior and is honest at the API boundary.
- Queue or defer: choose when delayed completion is acceptable and queue depth/backlog age are bounded.
- Shed load: choose when accepting all work would violate latency, memory, or dependency capacity budgets.
- Client-side throttling or quota: choose when retry or rejected-request traffic can itself overload the system.
- Fail fast: choose when degraded work is still too expensive or correctness requires no partial result.

## Accepted Examples
Accepted example: a search endpoint may degrade from full corpus to recent-index-only when CPU utilization and request queue age cross thresholds. The spec defines response disclosure, `p99` latency target while degraded, and the telemetry that proves the degraded path is cheaper.

Accepted example: an importer queue caps per-tenant backlog and returns a retryable deferral when backlog age exceeds the threshold. The spec includes retry budget and jitter to avoid turning small failures into load amplification.

Accepted example: a dependency-protection contract returns `429` for tenant quota exhaustion and `503` for global overload, with `Retry-After` guidance only when retry is safe and capacity is expected to recover soon.

## Rejected Examples
Rejected example: dropping random requests with no criticality, tenant fairness, idempotency, or client retry guidance.

Rejected example: adding more retries to hide overload without a per-request retry budget and client-level retry ratio guard.

Rejected example: queueing all overflow work with no max depth, no memory budget, and no time-to-live.

Rejected example: degraded response that is as expensive as the full response or violates freshness/correctness expectations.

## Pass/Fail Rules
Pass when:
- overload triggers, thresholds, and actions are explicit and tied to resource or SLI signals
- degraded responses state what is omitted, stale, approximate, or delayed
- queue depth, backlog age, and memory/capacity limits exist when queueing is used
- retry behavior includes max attempts, jitter/backoff, and amplification controls
- runtime telemetry can distinguish full, degraded, shed, queued, and dependency-failed modes

Fail when:
- overload behavior is "best effort" without thresholds or client-visible semantics
- retry policy can multiply load during partial outages
- load shedding has no fairness or priority rule
- degradation silently changes correctness or freshness contracts
- validation lacks an overload or capacity-limit scenario

## Validation Commands
Use local proof only as part of a broader overload validation plan:

```bash
go test -run='^$' -bench='BenchmarkSearch/(full|degraded|shed)$' -benchmem -count=20 ./internal/search > overload.txt
benchstat overload.txt
go test -run='^$' -bench='BenchmarkSearch/degraded$' -trace trace.out ./internal/search
go tool trace trace.out
go tool trace -pprof=sched trace.out > sched.pprof
go tool pprof -top sched.pprof
```

For staging or canary validation, require a repository-specific load command plus metrics for request latency, error type, shed count, queue depth, backlog age, retry ratio, CPU, memory, DB/cache pool wait, and tenant distribution.

## Exa Source Links
- [Google SRE: Handling Overload](https://sre.google/sre-book/handling-overload/)
- [Google SRE: Addressing Cascading Failures](https://sre.google/sre-book/addressing-cascading-failures/)
- [Google SRE: Production Services Best Practices](https://sre.google/sre-book/service-best-practices/)
- [Google SRE: Service Level Objectives](https://sre.google/sre-book/service-level-objectives/)
- [OpenTelemetry HTTP metrics semantic conventions](https://opentelemetry.io/docs/specs/semconv/http/http-metrics/)

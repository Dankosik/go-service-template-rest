# Budget Modeling And Hot Path Maps

## Behavior Change Thesis
When loaded for symptom "the performance goal is vague, global, or copied from a dashboard," this file makes the model choose scoped operation budgets, a hot-path map, and a measurable bottleneck hypothesis instead of likely mistake "make it faster" goals, average latency targets, or unreserved component budgets.

## When To Load
Load when a spec needs to turn broad speed intent into explicit latency, throughput, allocation, CPU, contention, or capacity budgets for one or more hot paths.

## Decision Rubric
- Start with the operation class: interactive request, bulk request, async job, queue worker, or dependency callback.
- Budget the behavior users or dependent systems observe before budgeting internals.
- Decompose end-to-end budgets across the path, such as `api -> auth -> domain -> db/cache -> dependency -> encode/write`.
- Keep explicit reserve for jitter, scheduler variance, pool waits, retries, or dependency tails; do not let component budgets sum exactly to the user-visible target unless the spec accepts that risk.
- Split budgets by workload class when interactive, admin, background, and bulk work have different objectives.
- Mark any numeric target without repository evidence as an assumption or blocker.
- Treat a bottleneck as a hypothesis tied to proof, not as a code smell.

## Imitate
- `GET /v1/orders/{id}` is interactive: `p95 <= 80ms`, `p99 <= 180ms`, `<= 8 KiB` response, and no allocation increase above baseline under `150 rps` per instance. Path budget: auth `<= 5ms`, domain `<= 10ms`, primary DB read `<= 45ms`, cache lookup `<= 5ms`, encode/write `<= 10ms`, reserve `15ms`. Copy the component reserve and the explicit DB bottleneck hypothesis.
- Async export separates enqueue latency `p95 <= 250ms`, completion latency `95% <= 5m` for `<= 100k rows`, backlog age `p95 <= 2m`, and queue depth below worker saturation. Copy the split between request-path and lifecycle budgets.
- Cache acceleration has two contracts: warm hit `p95 <= 25ms` and cache-down fallback `p95 <= 120ms` at capped origin concurrency. Copy the fallback budget instead of pretending the fast path is the only path.

## Reject
- "Make checkout fast." It has no operation class, percentile, load, input size, or proof obligation.
- "p99 must stay at 10ms" because the current dashboard shows 10ms, without checking large-tenant or peak-hour shape.
- Component budgets that total `120ms` while the user-visible target is `p95 <= 100ms`.
- Average latency as the primary target for a tail-sensitive interactive path.

## Agent Traps
- Inventing a clean number because the spec "needs" one.
- Treating a current metric as a contract without checking user impact and workload boundary.
- Budgeting only the happy path while cache-down, cold start, retries, or degraded dependency paths dominate the tail.
- Choosing an optimization before the hot-path ownership map says which component owns the suspected bottleneck.

## Validation Shape
Use proof obligations, not optimization instructions:

```bash
go test -run='^$' -bench='BenchmarkOrderRead/(small|large|hot_tenant)$' -benchmem -count=20 ./internal/orders > before.txt
go test -run='^$' -bench='BenchmarkOrderRead/(small|large|hot_tenant)$' -benchmem -count=20 ./internal/orders > after.txt
benchstat before.txt after.txt
```

For scenario or production validation, require the repository-specific load command, trace capture, dashboard query, or canary query. If the command or metric source is unknown during specification, record it as a planning blocker.

# Retry, Overload, And Tail Latency Review

## Behavior Change Thesis
When loaded for symptom "the diff changes retries, fallback, admission, queueing, or deadline behavior in a hot request path," this file makes the model choose a load-amplification and tail-latency finding instead of likely mistake "treat retries and fallbacks only as reliability policy, or suggest more retries for transient errors without checking capacity impact."

## When To Load
Load this when request-path performance depends on retries, fallback to origin, queue growth, missing deadlines, admission control, degraded-mode fan-out, or cascading dependency load.

## Decision Rubric
- Name the amplification loop: retry count times fan-out width, fallback per cache miss/error, queued work per request, or downstream calls after caller cancellation.
- State the trigger condition: timeout, cache outage, partial downstream failure, slow dependency, burst traffic, or overload.
- Treat retries inside fan-out loops and fallback-to-origin-on-every-error as throughput and p99 risks even when success-path benchmarks look fine.
- Require deadlines, cancellation propagation, bounded queues, retry budgets, backoff/jitter, coalescing, or shedding evidence when the changed path can accumulate work.
- Keep the performance finding focused on load, queueing, and tail latency. Hand off full retry/degradation policy design to `go-reliability-review` or `go-reliability-spec` when needed.
- Use `db-cache-and-io-amplification.md` when the issue is normal query/dependency call count rather than failure-mode amplification.

## Imitate
```text
[high] [go-performance-review] internal/profile/handler.go:151
Issue:
Axis: Tail Latency; the changed handler retries each downstream profile shard up to three times inside a 20-shard fan-out, but the request deadline is not passed into the retry loop and the PR evidence measures only the all-success path. During a shard slowdown, one request can now create up to 60 downstream calls and continue work after the caller times out.
Impact:
Under partial outage or burst traffic, the retry loop can multiply dependency load and hold the fan-in open, moving p99 latency and capacity even if the median success path is unchanged.
Suggested fix:
Propagate the request context into the retry loop, cap retries with a per-request budget or deadline-aware stop condition, and validate with a partial-failure workload that records downstream call count and p95/p99 latency.
Reference:
N/A
```

Copy the shape: failure trigger, amplification math, why success-path proof is insufficient, and the smallest bounded-work correction.

## Reject
```text
Issue:
Retries make this more reliable, so performance should be fine.
```

Reject it because reliability success rate and performance under overload can move in opposite directions.

```text
Issue:
Add another retry for transient errors.
```

Reject it when the path is fan-out or hot and no retry budget, deadline, or downstream load proof is provided.

## Agent Traps
- Reviewing only the all-success benchmark while the new work appears only during timeout, miss, or error paths.
- Missing retry multiplication across fan-out width.
- Treating fallback-to-origin as safe because it preserves correctness while it can collapse the origin during cache outage.
- Asking for a broad chaos test when a partial-failure integration benchmark with call-count assertions would prove the merge risk.
- Taking over reliability-policy design instead of writing the performance finding and handing off policy depth.

## Validation Shape
```bash
go test ./internal/profile -run '^TestProfileFanoutRetriesRespectDeadline$' -count=1
go test ./internal/profile -run '^TestProfileFanoutPartialFailureCallCount$' -count=1
go test -run '^$' -bench '^BenchmarkProfileFanout/(success|one-shard-slow|cache-error)$' -benchmem -count=10 ./internal/profile > new.txt
benchstat old.txt new.txt
```

For service proof, record request deadline, fan-out width, retry budget, injected failure rate, downstream call count, queue depth, and p95/p99 latency.

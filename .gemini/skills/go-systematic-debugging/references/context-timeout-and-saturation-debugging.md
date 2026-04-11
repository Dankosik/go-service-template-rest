# Context, Timeout, And Saturation Debugging

## Behavior Change Thesis
When loaded for timeout or saturation symptoms, this file makes the model attribute time to the budget owner, capacity wait, work time, or retry amplification instead of raising timeouts or adding retries first.

## When To Load
Load when the symptom is `context.Canceled`, `context.DeadlineExceeded`, a test timeout, HTTP client latency, DB pool wait, queue wait, retry amplification, shutdown drain, or production saturation.

## Decision Rubric
- Identify who owns the budget: request, handler, DB query, HTTP client, worker, shutdown, or test harness.
- Preserve `context.Canceled`, `context.DeadlineExceeded`, and `context.Cause(ctx)` semantics until the policy-owning layer decides.
- Split elapsed time into queue or pool wait, lock wait, connection setup, work time, retry backoff, response read, and shutdown drain.
- Treat `go test -timeout` as a harness kill switch, not automatically as the application deadline.
- Do not add retries during saturation until you know whether they multiply load.
- Escalate when the safe fix changes timeout budget, retry policy, durability, or user-visible semantics.

## Imitate

```text
request budget: 2s
app validation: 8ms
DB pool wait: 1.7s, WaitCount increased
query execution: 120ms
response encode: not reached
```

Copy the attribution: this points at pool saturation or connection ownership, not a slow query or a need for wider query timeout.

```go
stats := db.Stats()
log.Printf("db stats: open=%d in_use=%d idle=%d wait_count=%d wait_duration=%s max_open=%d",
	stats.OpenConnections, stats.InUse, stats.Idle, stats.WaitCount, stats.WaitDuration, stats.MaxOpenConnections)
```

Use temporary pool stats only when DB pool wait is a live hypothesis; remove or convert to bounded operational telemetry after proof.

## Reject

```go
ctx := context.Background()
```

This drops caller cancellation and hides the owner of the expired budget.

```text
Increased timeout from 2s to 30s and added retries.
```

This can mask pool wait or retry amplification and worsen a saturation incident.

## Agent Traps
- Collapsing caller cancellation, owned deadline expiry, dependency stall, and capacity exhaustion into "timeout."
- Using `context.WithTimeout` without calling the returned cancel function.
- Holding locks or transactions while waiting on network or disk I/O.
- Ignoring response-body read time, row iteration time, or shutdown drain after the first request/DB call returns.
- Treating a retry as free capacity.

## Validation Shape
Capture the exact failing command or incident window, deadline owner, parent context, `ctx.Err()` and `context.Cause(ctx)` when available, retry count, queue depth, worker count, DB pool stats or HTTP timing, and the after-fix evidence that the same boundary no longer exhausts the budget.

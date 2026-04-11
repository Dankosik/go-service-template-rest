# Context, Timeout, And Saturation Debugging

## When To Load
Load this reference when the symptom is `context.Canceled`, `context.DeadlineExceeded`, a test timeout, HTTP client latency, DB pool wait, queue wait, retry amplification, shutdown drain, or production saturation.

Use it before raising timeouts or adding retries.

## Commands
Start with the exact failing scope:

```bash
go test ./path/to/pkg -run '^TestName$' -count=1 -timeout=30s -v
go test ./path/to/pkg -run '^TestName$' -race -count=1 -timeout=30s -v
go test ./path/to/pkg -run '^TestName$' -trace trace.out -count=1 -timeout=30s
go test ./path/to/pkg -run '^TestName$' -blockprofile block.out -blockprofilerate=1 -count=1
go tool trace trace.out
go tool pprof -top block.out
```

For scheduler-level suspicion:

```bash
GODEBUG=schedtrace=1000,scheddetail=1 go test ./path/to/pkg -run '^TestName$' -count=1 -timeout=30s -v
```

For DB pool suspicion, capture `database/sql` stats before and after the operation or at interval:

```go
stats := db.Stats()
log.Printf("db stats: open=%d in_use=%d idle=%d wait_count=%d wait_duration=%s max_open=%d",
	stats.OpenConnections, stats.InUse, stats.Idle, stats.WaitCount, stats.WaitDuration, stats.MaxOpenConnections)
```

For HTTP client timing, add temporary `httptrace.ClientTrace` hooks around DNS, connection acquisition, TLS, request write, and first response byte.

## Evidence To Capture
- owner of the deadline: request, handler, DB query, HTTP client, worker, shutdown, or test harness
- `ctx.Err()` and, when used, `context.Cause(ctx)`
- deadline duration, remaining budget at each boundary, and parent context
- retry count, backoff, queue depth, worker count, and connection pool stats
- HTTP timing: DNS, connect, TLS, got connection, wrote request, first response byte
- DB timing: pool wait, query time, row iteration time, transaction scope, and driver cancellation behavior
- whether the operation did slow real work, waited for capacity, or waited forever on coordination

## Bad Debugging Moves
- replacing inbound request context with `context.Background()` to avoid cancellation
- converting `context.Canceled` and `context.DeadlineExceeded` into vague timeout strings
- raising a timeout before knowing where time was spent
- adding retries that multiply load during saturation
- holding locks or transactions while waiting on network or disk I/O
- treating `go test -timeout` as the application deadline instead of a test harness kill switch

## Good Debugging Moves
- keep context as the first parameter and propagate it across I/O boundaries
- always call the cancel function returned by `WithCancel`, `WithDeadline`, or `WithTimeout`
- decide whether cancellation means caller went away, owned budget expired, dependency stalled, or capacity was exhausted
- split queue wait from work time
- preserve typed context errors until the layer that owns the policy decision
- escalate if the safe fix changes timeout budget, retry policy, durability, or user-visible semantics

## Example Triage
If a handler returns `context deadline exceeded`:

```text
request budget: 2s
app validation: 8ms
DB pool wait: 1.7s, WaitCount increased
query execution: 120ms
response encode: not reached
```

This points at pool saturation or connection ownership, not a slow query. A retry or wider query timeout would likely amplify the incident.

## Source Links
- [context package](https://pkg.go.dev/context)
- [Go blog: context](https://go.dev/blog/context)
- [Go blog: pipelines and cancellation](https://go.dev/blog/pipelines)
- [Canceling in-progress database operations](https://go.dev/doc/database/cancel-operations)
- [database/sql package](https://pkg.go.dev/database/sql)
- [net/http/httptrace package](https://pkg.go.dev/net/http/httptrace)
- [Go diagnostics](https://go.dev/doc/diagnostics)

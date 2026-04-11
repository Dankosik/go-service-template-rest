# Timers, Tickers, And Shutdown Examples

## When To Load
Load this when a review touches `time.After`, `time.Tick`, `time.NewTimer`, `time.NewTicker`, `time.AfterFunc`, `Timer.Stop`, `Timer.Reset`, `Ticker.Stop`, sleeps in tests or loops, retry timing, fake clocks, or shutdown races around time.

## Review Lens
Time is not a synchronization substitute. Review whether a timer or ticker has an owner, whether shutdown unblocks the loop, and whether Stop/Reset/AfterFunc completion semantics are version-aware.

## Bad Review Example
```text
[medium] poller/poller.go:67
time.After in a loop leaks. Use a ticker.
```

Why it fails: Go 1.23 changed GC behavior for unreferenced timers/tickers. The finding should focus on the actual merge risk: timer churn, stale/reset semantics in version-sensitive code, or no cancellation path.

## Good Review Example
```text
[high] [go-concurrency-review] poller/poller.go:67
Issue:
Axis: Timers, Tickers, And Time-Based Coordination; the worker recreates `time.After(interval)` on every loop and the select has no `<-ctx.Done()` or `stop` case. `Stop` closes `p.stop` but this goroutine remains parked until the next timer fires.
Impact:
Shutdown latency is bounded by the poll interval and tests can hang under long intervals; this is merge-risk lifecycle drift, not just allocation style.
Suggested fix:
Create one `time.NewTicker(interval)`, `defer ticker.Stop()`, and select on both `ticker.C` and the stop or context channel. If using `AfterFunc`, coordinate explicitly when `Stop` returns false because it does not wait for the function to complete.
Reference:
`time` package docs for `Timer`, `Ticker`, `Stop`, and `Reset`; validate with `go test ./internal/poller -run TestStopReturnsImmediately -count=100 -timeout=5s`.
```

## Failure Mode
Write a finding when:
- a loop uses `time.Sleep` or `time.After` instead of a real stop signal and can delay shutdown;
- a ticker is created in a long-lived worker without a clear `Stop` on every exit path;
- `AfterFunc.Stop` is treated as if it waits for the function to finish;
- `Timer.Reset` can overlap with another goroutine receiving from the timer or running the previous callback without coordination;
- a test relies on sleep duration rather than a completion signal, fake clock, or `testing/synctest`;
- a review calls `time.After` or `time.Tick` a leak without checking Go version and actual retention behavior.

## Smallest Safe Correction
Prefer corrections like:
- use one owned `Ticker` with `defer ticker.Stop()` in the goroutine that owns the loop;
- include `<-ctx.Done()` or `<-stop>` in the same `select` as timer/ticker receives;
- coordinate `AfterFunc` callbacks with a completion channel or `WaitGroup` when `Stop` returns false;
- use `time.NewTimer` plus version-aware Stop/Reset handling when the timer must be reused;
- replace sleep polling tests with explicit gates, fake clocks, or `testing/synctest` when the project can target a Go version that includes it.

## Validation Evidence
Use validation that proves prompt shutdown and removes timing luck:
```bash
go test ./internal/poller -run TestStopReturnsImmediately -count=100 -timeout=5s
go test -race ./internal/poller -run TestTickerLoopStops -count=100
```

For code using `testing/synctest`, require a self-contained test bubble and avoid external I/O that prevents durable blocking.

## Source Links From Exa
- [time package docs](https://pkg.go.dev/time)
- [testing/synctest package docs](https://pkg.go.dev/testing/synctest)
- [Testing concurrent code with testing/synctest](https://go.dev/blog/synctest)
- [context package docs](https://pkg.go.dev/context)


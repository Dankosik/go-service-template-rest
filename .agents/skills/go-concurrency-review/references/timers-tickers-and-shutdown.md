# Timers, Tickers, And Shutdown

Behavior Change Thesis: When loaded for timer, ticker, sleep, or time-based shutdown symptoms, this file makes the model review ownership and prompt unblock semantics instead of repeating stale `time.After` leak folklore or accepting sleep as synchronization.

## When To Load
Symptom: the diff touches `time.After`, `time.Tick`, `time.NewTimer`, `time.NewTicker`, `time.AfterFunc`, `Timer.Stop`, `Timer.Reset`, `Ticker.Stop`, sleeps in loops or tests, retry timing, fake clocks, or shutdown races around time.

## Decision Rubric
- Time is not a synchronization substitute. Ask which signal unblocks the goroutine when shutdown or cancellation happens.
- Long-lived tickers need a clear owner and `Stop` on every exit path.
- `time.After` in a loop is not automatically a leak on modern Go; focus the finding on timer churn, delayed shutdown, lost reset semantics, or version-sensitive retention when that is the actual merge risk.
- `AfterFunc.Stop` does not wait for an already-running function; require explicit completion coordination when the callback touches shared state or shutdown depends on it.
- `Timer.Reset` needs coordination when another goroutine may receive the old tick or the previous callback may still run.
- Sleep-based tests should usually be replaced with gates, fake clocks, or `testing/synctest` when the project can rely on it.

## Imitate
```text
[high] [go-concurrency-review] poller/poller.go:67
Issue:
Axis: Timers, Tickers, And Time-Based Coordination; the worker recreates `time.After(interval)` on every loop and the select has no `<-ctx.Done()` or `<-stop>` case. `Stop` closes `p.stop`, but this goroutine remains parked until the next timer fires.
Impact:
Shutdown latency is bounded by the poll interval and tests can hang under long intervals; this is merge-risk lifecycle drift, not just allocation style.
Suggested fix:
Create one owned `time.NewTicker(interval)`, `defer ticker.Stop()`, and select on both `ticker.C` and the stop or context channel. If using `AfterFunc`, coordinate explicitly when `Stop` returns false because it does not wait for the function to complete.
Reference:
Validate with `go test ./internal/poller -run TestStopReturnsImmediately -count=100 -timeout=5s`.
```

Copy the shape: it avoids outdated leak claims and anchors the defect in prompt shutdown and ownership.

## Reject
```text
[medium] poller/poller.go:67
time.After in a loop leaks. Use a ticker.
```

Reject this shape: it may be stale for the Go version and misses the concrete liveness defect that matters to review.

```go
time.Sleep(10 * time.Millisecond)
require.True(t, stopped)
```

Reject this as primary proof: scheduler timing does not prove the stop signal was observed or the goroutine exited.

## Agent Traps
- Do not flag `time.After` solely from memory of older Go behavior; tie the finding to the repo's Go version or to a version-independent issue.
- Do not forget `Ticker.Stop` when the ticker is created outside the loop owner.
- Do not treat a false return from `AfterFunc.Stop` as callback completion.
- Do not replace every timer with a ticker; one-shot timers, reused timers, and fake clocks may be the smaller correction.

## Validation Shape
- Prove prompt shutdown with a completion signal and test timeout.
- Use race evidence when timer callbacks touch shared state.
- For fake-clock or `testing/synctest` tests, keep the test bubble self-contained; external I/O or goroutines outside the bubble can make the proof misleading.
- Good commands look like `go test ./internal/poller -run TestStopReturnsImmediately -count=100 -timeout=5s` and `go test -race ./internal/poller -run TestTickerLoopStops -count=100`.

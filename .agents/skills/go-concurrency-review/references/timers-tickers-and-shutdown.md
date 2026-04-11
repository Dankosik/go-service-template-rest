# Timers, Tickers, And Shutdown

Behavior Change Thesis: When loaded for timer, ticker, sleep, or time-based shutdown symptoms, this file makes the model review ownership and prompt unblock semantics instead of repeating stale `time.After` leak folklore or accepting sleep as synchronization.

## When To Load
Symptom: the diff touches `time.After`, `time.Tick`, `time.NewTimer`, `time.NewTicker`, `time.AfterFunc`, `context.AfterFunc`, `Timer.Stop`, `Timer.Reset`, `Ticker.Stop`, sleeps in loops or tests, retry timing, fake clocks, or shutdown races around time.

## Decision Rubric
- Time is not a synchronization substitute. Ask which signal unblocks the goroutine when shutdown or cancellation happens.
- Owned tickers need a clear stop story when ticks must stop after the owner exits; on Go 1.23+ do not frame `Stop` as required only for garbage collection.
- Stopping a ticker does not close `Ticker.C`; a goroutine ranging on or selecting only that channel still needs a separate stop, context, or owner-exit signal.
- `time.After` in a loop is not automatically a leak on modern Go; focus the finding on timer churn, delayed shutdown, lost reset semantics, or version-sensitive retention when that is the actual merge risk.
- For channel timers on Go 1.23+, do not require the old stop-and-drain pattern for stale values after `Stop` or `Reset` returns unless `GODEBUG=asynctimerchan=1` is in effect; still require owner coordination when another goroutine may already be acting on a tick.
- `time.AfterFunc` and `context.AfterFunc` stop paths do not wait for an already-running callback; require explicit completion coordination when the callback touches shared state or shutdown depends on it. For `time.AfterFunc`, `Reset` can allow overlapping callbacks when the docs say it returns false.
- Sleep-based tests should usually be replaced with gates, fake clocks, or `testing/synctest` when the project can rely on it.

## Imitate
```text
[high] [go-concurrency-review] poller/poller.go:67
Issue:
Axis: Timers, Tickers, And Time-Based Coordination; the worker recreates `time.After(interval)` on every loop and the select has no `<-ctx.Done()` or `<-stop>` case. `Stop` closes `p.stop`, but this goroutine remains parked until the next timer fires.
Impact:
Shutdown latency is bounded by the poll interval and tests can hang under long intervals; this is merge-risk lifecycle drift, not just allocation style.
Suggested fix:
Create one owned `time.NewTicker(interval)`, `defer ticker.Stop()`, and select on both `ticker.C` and the stop or context channel. If using `time.AfterFunc` or `context.AfterFunc`, coordinate explicitly when `Stop` returns false because it does not wait for the function to complete.
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
- Do not flag `time.After` solely from memory of older Go behavior; tie the finding to the repo's Go version, timer mode, or a version-independent issue.
- Do not forget `Ticker.Stop` when ticks can continue to drive work after the owner exits, but do not claim an unreferenced Go 1.23+ ticker leaks solely because it was not stopped.
- Do not expect `Ticker.Stop` to make ticker-channel receivers exit; it only stops future ticks.
- Do not treat a false return from `time.AfterFunc` or `context.AfterFunc` stop as callback completion.
- Do not replace every timer with a ticker; one-shot timers, reused timers, and fake clocks may be the smaller correction.

## Validation Shape
- Prove prompt shutdown with a completion signal and test timeout.
- Use race evidence when timer callbacks touch shared state.
- For fake-clock or `testing/synctest` tests, keep the test bubble self-contained; external I/O or goroutines outside the bubble can make the proof misleading.
- Good commands look like `go test ./internal/poller -run TestStopReturnsImmediately -count=100 -timeout=5s` and `go test -race ./internal/poller -run TestTickerLoopStops -count=100`.

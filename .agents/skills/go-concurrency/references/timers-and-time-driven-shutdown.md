# Timers And Time-Driven Shutdown

## Behavior Change Thesis
When loaded for timer, ticker, or sleep symptoms, this file replaces pre-Go-1.23 timer folklore with the semantics this module actually compiles under, so the review reports the shutdown latency or lost-signal defect that is real instead of a leak that no longer exists.

## When To Load
Symptom: the diff touches `time.After`, `time.Tick`, `time.NewTimer`, `time.NewTicker`, `Timer.Reset`, `Ticker.Stop`, `time.AfterFunc`, `context.AfterFunc`, sleeps in loops or tests, retry timing, or shutdown behavior that depends on time.

## Decision Rubric
- This module is `go 1.26.5` with no `godebug` directive, so Go 1.23+ timer semantics are in force. The garbage collector recovers unreferenced timers and tickers whether or not they were stopped: `time.After` in a loop is not a leak, `Tick` is not a leak, and `Stop` is no longer needed to help collection. A finding that says otherwise is repeating advice the standard library retracted.
- What `Stop` still buys is future work, not memory: ticks that would keep driving a loop after its owner is gone, and an `AfterFunc` callback that would still fire. Ask what the next fire would do, not whether `Stop` was called.
- `Ticker.Stop` does not close `Ticker.C`. A goroutine ranging over that channel, or selecting only on it, is never woken by `Stop` — it needs a context or done arm.
- Since Go 1.23 a channel timer needs no drain: after `Stop` returns, a receive on `t.C` is guaranteed to block rather than deliver a stale time, so the old `if !t.Stop() { <-t.C }` dance is obsolete and its `Reset` hazards are gone.
- The stop function of `time.AfterFunc` and `context.AfterFunc` returns false once the callback has started, and does not wait for it. If shutdown or shared state depends on the callback finishing, the code must coordinate that explicitly.
- Time is not a synchronization edge. When a loop waits only on its timer, the shutdown latency floor is the interval — that is the finding, and it is version-independent.
- Prefer `testing/synctest` over sleeps for time-driven behavior; `cmd/service/internal/bootstrap`, `cmd/outbox-relay/internal/bootstrap`, and their neighbors already use `synctest.Test` with `synctest.Wait`. It holds only while the code under test stays inside the bubble — real network, external processes, and goroutines started outside it break the fake clock's guarantee.

## Reject
- "`time.After` in a loop leaks; use a ticker" — stale for this module. If the churn or the missing stop arm is the real problem, report that instead.
- `time.Sleep` followed by an assertion, as proof that a stop was observed — the scheduler decides whether it passes.
- A test that assumes a canceled context beats a very short timer in `select` — when both are ready the choice is random.

## Validation Shape
- Prove prompt shutdown by asserting the stop call returns after the signal, with the test timeout as the diagnostic guard.
- Use `synctest` fake time to prove interval and backoff behavior without waiting for it.
- Add `-race`, or `make test-race`, when a timer callback touches state another goroutine reads.

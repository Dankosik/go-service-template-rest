# Timers And Time-Driven Shutdown

## Load When
Symptom: the diff touches `time.After`, `time.Tick`, `time.NewTimer`, `time.NewTicker`, `Timer.Reset`, `Ticker.Stop`, `time.AfterFunc`, `context.AfterFunc`, sleeps in loops or tests, retry timing, or shutdown behavior that depends on time.

## Decide
- Resolve available APIs through [Go Modern Version](../../go-modern-version/SKILL.md).
  For channel-timer behavior, inspect the executable's main module and effective
  `GODEBUG` settings against the [Go timer contract](https://go.dev/wiki/Go123Timer).
  Under the Go 1.23 timer semantics, unreferenced timers and tickers can be
  collected without `Stop`; do not infer a leak solely from a missing stop call.
- What `Stop` still buys is future work, not memory: ticks that would keep driving a loop after its owner is gone, and an `AfterFunc` callback that would still fire. Ask what the next fire would do, not whether `Stop` was called.
- `Ticker.Stop` does not close `Ticker.C`. A goroutine ranging over that channel, or selecting only on it, is never woken by `Stop` — it needs a context or done arm.
- Under those semantics, `Stop` and `Reset` exclude stale channel values from
  the previous timer configuration. Establish the effective semantics before
  removing legacy drain logic.
- The stop function of `time.AfterFunc` and `context.AfterFunc` returns false once the callback has started, and does not wait for it. If shutdown or shared state depends on the callback finishing, the code must coordinate that explicitly.
- Time is not a synchronization edge. When a loop waits only on its timer, the shutdown latency floor is the interval — that is the finding, and it is version-independent.
- Prefer `testing/synctest` over sleeps when the target module supports it. It
  holds only while the code under test stays inside the bubble — real network,
  external processes, and goroutines started outside it break the fake clock's
  guarantee.

## Reject
- "`time.After` in a loop leaks; use a ticker" without establishing the effective
  timer semantics. If churn or a missing stop arm is the problem, report that.
- `time.Sleep` followed by an assertion, as proof that a stop was observed — the scheduler decides whether it passes.
- A test that assumes a canceled context beats a very short timer in `select` — when both are ready the choice is random.

## Prove
- Prove prompt shutdown by asserting the stop call returns after the signal, with the test timeout as the diagnostic guard.
- Use fake time supported by the target module to prove interval and backoff
  behavior without waiting for it.
- Add `-race`, or `make test-race`, when a timer callback touches state another goroutine reads.

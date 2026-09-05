# Goroutine Lifetime And Blocking Sites

## Load When

Load when goroutines, channels, errgroups, watchers, worker loops, cancellation,
joins, or early returns change.

## Decide

For each goroutine list every blocking site—send/receive, wait, lock, timer,
network/database call—and the exact signal that unblocks it. Cancellation works
only where observed and propagated. Every goroutine has an owner, stop signal,
and join point, or an explicit process-lifetime reason.

Reuse `internal/background.Supervisor` for process work: it owns cancel, panic
containment, join, and readiness failure. Its bare `errgroup.Group` intentionally
does not cancel siblings on one failure, and shutdown reports an over-budget task
without blocking later drain. `errgroup.WithContext` also cancels its context
when `Wait` returns, so post-wait work needs another context.

Only the actor that knows all sends are finished closes a data channel. Close
broadcasts to receivers; it does not ask senders to stop. Review both halves of
abandoned pairs, because an early receiver return can strand senders silently.
A default select branch owns explicit drop/retry/accounting policy.

## Prove

Park the goroutine at each decision-changing blocking site with a handshake,
cancel/stop it, and assert the exported join returns. Use goleak where the
package already owns a gate; add one only when a new package owns background
lifetimes. Test timeout diagnoses a failed join but is not the oracle.

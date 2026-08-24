---
name: go-concurrency
description: "Happens-before in Go. Use when correctness depends on overlapping goroutines, publication of shared state, bounded concurrent work, or stopping and joining goroutines."
metadata:
  invocation: model
  kind: method
---

# Go Concurrency

Judge every change by its **happens-before** story: a claim that two events cannot race is either an edge you can name — channel operation, lock, `WaitGroup`, `Once`, atomic — or it is false.

`shared state -> goroutine lifetime -> happens-before edges -> cancellation -> unblock and join -> bounds -> proof`

Every goroutine has an owner, a signal it observes to stop, a guaranteed unblock for each blocking site, and a join point; a goroutine missing any of these outlives its purpose and leaks work, memory, or writes into freed assumptions. Worker pools and fan-outs carry explicit bounds, and timers and tickers have named stop owners.

Load the [shared specialist contract](../../contracts/specialist-contract.md).
From every changed spawn site to its join or explicit process-lifetime
disposition, build `GoroutineStory{owner, stop, blocking_sites, unblock, join,
bound, happens_before, proof}`. Account for shared state, callers,
cancellation, close ownership, timers, and synchronization identity. Complete
when every story names an edge or a race and every blocking site has an unblock
disposition. The race detector can reject a story; it cannot prove one. Use
existing race, leak, `synctest`, and lint gates for mechanical evidence.

Load the [review selector](references/index.md) for fire-and-forget or blocked
goroutines, publication or lock scope, unbounded fan-out or queues, and
timer-driven coordination.

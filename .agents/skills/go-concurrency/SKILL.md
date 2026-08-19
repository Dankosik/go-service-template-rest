---
name: go-concurrency
description: "Happens-before: Use for goroutines, shared state, channels, locks, atomics, timers, bounds, cancellation, or joins. Own lifecycle; Skip resilience."
---

# Go Concurrency

Judge every change by its **happens-before** story: a claim that two events cannot race is either an edge you can name — channel operation, lock, `WaitGroup`, `Once`, atomic — or it is false.

`shared state -> goroutine lifetime -> happens-before edges -> cancellation -> unblock and join -> bounds -> proof`

Every goroutine has an owner, a signal it observes to stop, a guaranteed unblock for each blocking site, and a join point; a goroutine missing any of these outlives its purpose and leaks work, memory, or writes into freed assumptions. Worker pools and fan-outs carry explicit bounds, and timers and tickers have named stop owners.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct
affected values and activities by tracing goroutines, shared state, callers,
cancellation, unblock, close, join, timers, and synchronization identity. Build
each happens-before story and complete when the shared finding envelope names an
edge or a race for every story. The race detector can reject a story; it cannot
prove one. Use existing race, leak, `synctest`, and lint gates for mechanical
evidence and spend review on the invariant.

Load the [review selector](references/index.md) for fire-and-forget or blocked
goroutines, publication or lock scope, unbounded fan-out or queues, and
timer-driven coordination. Hand lifecycle policy to `go-reliability`, placement
to `go-implementation-ownership`, and state durable beyond this process to
[concurrency control](../../../docs/universal-disciplines/concurrency-control/SKILL.md).

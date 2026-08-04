---
name: go-concurrency
description: "Concurrency: Use for goroutines, shared state, channels, locks, atomics, timers, worker bounds, cancellation, or joins. Own happens-before/lifecycle; Skip resilience policy, replay, or context semantics."
---

# Go Concurrency

Judge every change by its **happens-before** story: a claim that two events cannot race is either an edge you can name — channel operation, lock, `WaitGroup`, `Once`, atomic — or it is false.

`shared state -> goroutine lifetime -> happens-before edges -> cancellation -> unblock and join -> bounds -> proof`

Every goroutine has an owner, a signal it observes to stop, a guaranteed unblock for each blocking site, and a join point; a goroutine missing any of these outlives its purpose and leaks work, memory, or writes into freed assumptions. Worker pools and fan-outs carry explicit bounds, and timers and tickers have named stop owners.

Load the [shared specialist contract](../specialist-contract.md). This skill has one review branch: reconstruct affected values and activities by tracing changed goroutines, shared state, callers, cancellation, unblock, close, join, timer, and sync-identity edges, then build each happens-before story. Complete when the shared finding envelope accounts for every story with a named edge or a named race — the race detector rejects a story, it does not prove one. This repository already proves concurrency with `make test-race`, `go.uber.org/goleak` package gates, and `testing/synctest` bubbles, and mandatory lint already fails `copylocks`, `waitgroup`, and `lostcancel`, so review spends on the invariant rather than on what a gate reports without help. Missing lifecycle policy returns to the named `go-reliability` Decision owner.

Load the [review selector](references/index.md) when the diff contains fire-and-forget or blocked goroutines, shared-state publication or lock scope, unbounded fan-out or queues, or timer-driven coordination. Hand placement to `go-implementation-ownership`. Load [`concurrency-control`](../../../docs/universal-disciplines/concurrency-control/SKILL.md) when the contested state is durable and shared beyond this process — across requests, replicas, or services: it forces the weakest mechanism that closes a named breaking schedule at the isolation level actually in force, with fencing wherever exclusivity can expire, instead of a lock whose scope stops at the process boundary.

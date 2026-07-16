---
name: go-concurrency
description: "Concurrency: Use when changed Go touches goroutines, shared state, channels, locks, atomics, WaitGroups, timers, worker bounds, cancellation unblock, or join protocols. Own happens-before and lifecycle-mechanism review; Skip when service resilience policy, durable replay, or Go context API semantics is primary."
---

# Go Concurrency

Load the [shared specialist contract](../specialist-contract.md). This skill has one review branch: reconstruct affected values and activities by tracing changed goroutines, shared state, callers, cancellation, unblock, close, join, timer, and sync-identity edges, then build each happens-before story. Complete when the shared finding envelope accounts for every story; name any outside boundary or proof blocker with focused proof. Missing lifecycle policy ends the run with a named `go-reliability` Decision handoff; conformance Review resumes separately after acceptance.

Load the [review selector](references/index.md) only when a concrete pressure changes the result; add another reference only for an independent pressure. Hand placement to `go-implementation-ownership`.

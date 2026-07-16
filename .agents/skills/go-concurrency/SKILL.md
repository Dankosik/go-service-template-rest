---
name: go-concurrency
description: "Concurrency: Use when changed Go touches goroutines, shared state, channels, locks, atomics, WaitGroups, timers, worker bounds, cancellation unblock, or join protocols. Own happens-before and lifecycle-mechanism review; Skip when service resilience policy, durable replay, or Go context API semantics is primary."
---

# Go Concurrency

Load the [shared specialist contract](../specialist-contract.md). This skill has one review branch: prove happens-before, ownership, cancellation/unblock, bounded goroutines, close/join, timer, and sync-identity safety. It is complete when every affected mechanism is dispositioned as a finding or no finding with focused proof, and missing lifecycle policy is handed to `go-reliability` rather than invented.

Load the [review selector](references/index.md) only when a concrete pressure changes the result; add another reference only for an independent pressure. Hand placement to `go-implementation-ownership`.

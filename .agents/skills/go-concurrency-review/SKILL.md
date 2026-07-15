---
name: go-concurrency-review
description: "Use when changed Go touches goroutines, shared state, channels, locks, atomics, WaitGroups, timers, worker bounds, cancellation unblock, or join protocols; Own concrete happens-before and lifecycle-mechanism correctness; Skip when the primary defect is service resilience policy, durable replay, or Go context API semantics."
---

# Go Concurrency Review

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Prove happens-before, ownership, cancellation/unblock, bounded goroutines, close/join, timer, and sync-identity safety. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Escalate lifecycle policy to `go-reliability-spec` and placement to `go-implementation-ownership-spec`. Return the owned decision or evidence-backed finding, forced consequence, and focused proof; stop rather than inventing another owner’s policy.

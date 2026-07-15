---
name: go-reliability-spec
description: "Use when timeout and deadline budgets, retries, overload handling, degradation, readiness, startup, drain, shutdown, recovery, or rollout behavior must be decided before coding; Own end-to-end service resilience policy and proof; Skip when the primary decision is concrete synchronization, durable distributed recovery, or implementation."
---

# Go Reliability Spec

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Define budgets, safe retry/error classes, overload/degradation, lifecycle, recovery, rollout, and falsifying proof. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Escalate durable recovery to `go-distributed-spec` and synchronization mechanism to `go-concurrency-review`. Return the owned decision or evidence-backed finding, forced consequence, and focused proof; stop rather than inventing another owner’s policy.

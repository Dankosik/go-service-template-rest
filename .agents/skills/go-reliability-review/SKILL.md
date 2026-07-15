---
name: go-reliability-review
description: "Use when changed Go may violate accepted end-to-end timeout, deadline, retry, overload, degradation, readiness, startup, drain, shutdown, recovery, or rollout behavior; Own service resilience-policy conformance; Skip when the primary defect is concrete synchronization, durable replay, or Go context API semantics."
---

# Go Reliability Review

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Review budget propagation, retries, overload/degradation, readiness/liveness, startup/drain/shutdown, recovery, and rollout. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Escalate changed timeout, retry, overload, lifecycle, or rollout policy to `go-reliability-spec`. Return the owned decision or evidence-backed finding, forced consequence, and focused proof; stop rather than inventing another owner’s policy.

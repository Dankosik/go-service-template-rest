---
name: go-reliability
description: "Reliability: Use when timeouts, retries, overload, degradation, readiness, startup, drain, shutdown, recovery, or rollout needs a decision, or when changed Go must conform to accepted resilience policy. Own service resilience policy and review; Skip when synchronization, durable replay, or Go context API semantics is primary."
---

# Go Reliability

Load the [shared specialist contract](../specialist-contract.md). Keep dependency criticality, end-to-end budgets, retries, overload/degradation, readiness/liveness, startup/drain/shutdown, recovery, and rollout coherent.

## Choose The Branch

- **Decision** — select when resilience policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing pressure. Complete when budgets, failure behavior, recovery, proof, rollout consequences, and blockers are explicit.
- **Review** — select when changed Go must conform to accepted resilience policy. Load the [review selector](references/review/index.md) for the violated contract. Complete when every affected dependency and lifecycle path is dispositioned as a finding or no finding with the smallest correction and focused proof; missing policy stays in the decision branch.

Hand concrete synchronization to `go-concurrency`, durable recovery to `go-distributed`, and context API misuse to `go-idiomatic`.

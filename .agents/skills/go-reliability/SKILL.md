---
name: go-reliability
description: "Reliability: Use when timeouts, retries, overload, degradation, readiness, startup, drain, shutdown, recovery, or rollout needs a decision, or when changed Go must conform to accepted resilience policy. Own service resilience policy and review; Skip when synchronization, durable replay, or Go context API semantics is primary."
---

# Go Reliability

Load the [shared specialist contract](../specialist-contract.md). Reconstruct affected dependencies and lifecycle stages from accepted behavior, changed call paths and configuration, startup/shutdown wiring, and rollout topology. Classify criticality, then trace failure through each stage, binding end-to-end budgets, timeout, retry, overload, degradation, readiness/liveness, recovery, evidence, and rollout behavior.

## Choose The Branch

- **Decision** — select when resilience policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing pressure. Complete when shared Decision dispositions cover every dependency and lifecycle stage with failure, recovery, and rollout consequences explicit.
- **Review** — select when changed Go must conform to accepted resilience policy. Load the [review selector](references/review/index.md) for the violated contract. Trace every affected failure path into the shared finding envelope, naming any outside boundary or proof blocker with the smallest correction and focused proof. Missing policy returns to the named Resilience Decision owner.

Hand concrete synchronization to `go-concurrency`, durable recovery to `go-distributed`, and context API misuse to `go-idiomatic`.

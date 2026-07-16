---
name: go-distributed
description: "Distributed flow: Use when durable cross-service consistency, sagas, messages, replay, ordering, compensation, redrive, or reconciliation needs a decision, or when changed durable behavior needs review. Own distributed recovery policy and conformance; Skip when local transactions, synchronization, service lifecycle, or system topology is primary."
---

# Go Distributed

Load the [shared specialist contract](../specialist-contract.md). Keep process ownership, at-least-once delivery, success boundaries, idempotency, ordering, replay, compensation, redrive, reconciliation, and compatibility coherent.

## Choose The Branch

- **Decision** — select when durable-flow policy is absent or changing. First prove cross-service coordination is needed, then load the [decision selector](references/decision/index.md) for one result-changing pressure. Complete when ownership, recovery, forced consequences, proof, and blockers are explicit.
- **Review** — select when changed durable behavior must conform to accepted policy. Load the [review selector](references/review/index.md) for the changed failure boundary. Complete when every affected durable path is dispositioned as a finding or no finding with the smallest correction and proof; missing policy stays in the decision branch.

Hand business identity to `go-domain-invariant` and local DB/cache mechanics to `go-db-cache`.

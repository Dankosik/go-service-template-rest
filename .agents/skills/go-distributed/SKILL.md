---
name: go-distributed
description: "Distributed flow: Use when durable cross-service consistency, sagas, messages, replay, ordering, compensation, redrive, or reconciliation needs a decision, or when changed durable behavior needs review. Own distributed recovery policy and conformance; Skip when local transactions, synchronization, service lifecycle, or system topology is primary."
---

# Go Distributed

Load the [shared specialist contract](../specialist-contract.md). Reconstruct every durable step from accepted flows, producers, consumers, broker/persistence boundaries, recovery paths, and success semantics; replay duplicate delivery, reordering, partial completion, process loss, redrive, and mixed versions while preserving process ownership and success boundaries.

## Choose The Branch

- **Decision** — select when durable-flow policy is absent or changing. First prove cross-service coordination is needed, then load the [decision selector](references/decision/index.md) for one result-changing pressure. Complete when shared Decision dispositions cover every durable step and replay shows its invariant, compensation, or reconciliation with focused proof.
- **Review** — select when changed durable behavior must conform to accepted policy. Load the [review selector](references/review/index.md) for the changed failure boundary. Replay every affected durable step into the shared finding envelope, naming any outside boundary or proof blocker with the smallest correction and proof. Missing policy returns to the named Durable-flow Decision owner.

Hand business identity to `go-domain-invariant` and local DB/cache mechanics to `go-db-cache`.

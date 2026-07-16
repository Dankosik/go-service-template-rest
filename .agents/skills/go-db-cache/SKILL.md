---
name: go-db-cache
description: "DB/cache: Use when runtime SQL, transactions, cache role, freshness, invalidation, fallback, or observability needs a decision, or when changed DB/cache paths need conformance review. Own data-access and cache policy and review; Skip when schema authority, distributed consistency, endpoint semantics, or broad concurrency is primary."
---

# Go DB Cache

Load the [shared specialist contract](../specialist-contract.md). Keep query and transaction ownership, DB resource lifetime, tenant-complete cache keys, serialization, freshness, invalidation, fallback, and origin protection coherent.

## Choose The Branch

- **Decision** — select when DB/cache policy is absent or changing. Load the [decision selector](references/decision/index.md) only when a pressure can change the result. Complete when the policy, forced consequences, focused proof, and blockers are explicit; a cache without measured need is rejected.
- **Review** — select when changed SQL or cache code must conform to accepted policy. Load the [review selector](references/review/index.md) for the violated contract. Complete when every affected path is dispositioned as a finding or no finding with the smallest safe correction and proof; unresolved policy stays in the decision branch.

Hand authoritative models to `go-data-architecture` and durable recovery to `go-distributed`.

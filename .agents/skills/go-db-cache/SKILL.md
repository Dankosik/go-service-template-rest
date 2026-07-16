---
name: go-db-cache
description: "DB/cache: Use when runtime SQL, transactions, cache role, freshness, invalidation, fallback, or observability needs a decision, or when changed DB/cache paths need conformance review. Own data-access and cache policy and review; Skip when schema authority, distributed consistency, endpoint semantics, or broad concurrency is primary."
---

# Go DB Cache

Load the [shared specialist contract](../specialist-contract.md). Reconstruct affected data paths from changed callers, queries, transactions, cache/config surfaces, fallbacks, and origin access; trace each through DB resource lifetime while preserving tenant identity, serialization, freshness, invalidation, and origin protection.

## Choose The Branch

- **Decision** — select when DB/cache policy is absent or changing. Load the [decision selector](references/decision/index.md) only when a pressure can change the result. Complete when shared Decision dispositions cover every affected data path, forced consequence, and focused proof; a cache without measured need is rejected.
- **Review** — select when changed SQL or cache code must conform to accepted policy. Load the [review selector](references/review/index.md) for the violated contract. Account for every affected path through the shared finding envelope, naming any outside boundary or proof blocker with the smallest safe correction and proof. Missing policy ends this run with a named DB/cache Decision handoff; conformance Review begins separately after acceptance.

Hand authoritative models to `go-data-architecture` and durable recovery to `go-distributed`.

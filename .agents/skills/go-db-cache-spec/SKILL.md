---
name: go-db-cache-spec
description: "Use when runtime SQL access, transaction controls, cache role, staleness, invalidation, fallback, or DB/cache observability must be decided before coding; Own data-access and cache behavior; Skip when the primary decision is authoritative schema architecture, distributed consistency, endpoint semantics, or implementation."
---

# Go Db Cache Spec

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Decide SQL/query and transaction ownership plus cache consistency, invalidation, fallback, and observability; reject cache without measured need. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Escalate authoritative schema policy to `go-data-architecture-spec` and durable recovery to `go-distributed-spec`. Return the owned decision or evidence-backed finding, forced consequence, and focused proof; stop rather than inventing another owner’s policy.

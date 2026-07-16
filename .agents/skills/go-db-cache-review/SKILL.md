---
name: go-db-cache-review
description: "Review changed Go SQL or cache paths for query, transaction, DB-resource, cache-isolation, freshness, serialization, and origin-protection defects. Skip schema architecture, business policy, and broad concurrency or reliability review."
---

# Go Db Cache Review

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Review bind/allowlist safety, transaction boundaries and retry scope, and DB resource lifetime; for cache paths, review tenant-complete keys, serialization, freshness ownership, and origin protection. When a concrete pressure appears, load its reference from [the reference selector](references/index.md); load one further reference only for an independent pressure. Hand off unresolved DB/cache policy to `go-db-cache-spec` and data truth to `go-data-architecture-spec`. Return the owned decision, an evidence-backed finding with its forced consequence and focused proof, or no DB/cache findings.

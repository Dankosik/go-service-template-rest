---
name: go-db-cache
description: "DB/cache: Use for runtime SQL, transactions, freshness, invalidation, or fallback. Own access policy; Skip schema, distributed consistency, and API semantics."
metadata:
  invocation: model
  kind: method
---

# Go DB Cache

Every read and write is an **access path**: transaction boundary, tenant identity,
query shape, and any cache's authority, freshness, invalidation, and fallback.

`caller intent -> transaction -> query -> cache contract -> fallback -> observability -> proof`

Load the [shared specialist contract](../../contracts/specialist-contract.md). Reconstruct
affected paths from callers, queries, transactions, cache/config surfaces,
fallbacks, and DB resource lifetime. What must be atomically true together
defines the transaction. A cache needs measured value, tenant-scoped keys, a
freshness bound, an invalidation owner, and bounded origin fallback.

For a **Decision**, load [transaction and commit outcome](references/decision/transaction-boundary-and-commit-outcome.md)
when deciding atomicity, safe retry, or an unknown commit. Reject a cache without
measured need. For **Review**, load [PostgreSQL access review](references/review/postgres-access-review.md)
when the query seam, rows, errors, or deadline changes. Account for every path
in the shared finding envelope.

This template ships `internal/infra/postgres` and no data-cache client. A
distributed cache therefore adds a dependency and operational surface;
`internal/health` and `internal/infra/oidcjwt` are the in-process precedents.

Hand data authority to `go-data-architecture`, durable recovery to
`go-distributed`, cache semantics to [cache engineering](../../../docs/universal-disciplines/cache-engineering/SKILL.md),
and measured database bottlenecks to [PostgreSQL performance](../../../docs/universal-disciplines/postgres-performance/SKILL.md).

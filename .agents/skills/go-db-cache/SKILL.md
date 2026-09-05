---
name: go-db-cache
description: "Runtime data access. Use when SQL or transaction boundaries, query semantics, cache freshness or invalidation, or bounded fallback determine a request path."
metadata:
  invocation: model
  kind: method
---

# Go DB Cache

Every read and write is an **access path**: transaction boundary, tenant identity,
query shape, and any cache's authority, freshness, invalidation, and fallback.

`caller intent -> transaction -> query -> cache contract -> fallback -> observability -> proof`

Trace the changed caller intent through its terminal result, including tenant
identity, commit outcome, and resource lifetime. What must be atomically true
together defines the transaction. A cache needs measured value, tenant-scoped
keys, a freshness bound, an invalidation owner, and bounded origin fallback.

When comparing multiple affected access paths or handing off a Decision or
Review, record
`AccessPath{caller, transaction, query, tenant, cache_authority, freshness,
invalidation, fallback, commit_outcome, resource_lifetime, proof}` for each path.
A single local path needs a grounded disposition and matching proof. For a
delegated Decision or Review, load the
[shared specialist contract](../../contracts/specialist-contract.md).

For a **Decision**, load [transaction and commit outcome](references/decision/transaction-boundary-and-commit-outcome.md)
when deciding atomicity, safe retry, or an unknown commit. Reject a cache without
measured need. For **Review**, load [PostgreSQL access review](references/review/postgres-access-review.md)
when the query seam, rows, errors, or deadline changes. Complete when every
access path has one atomicity and commit-outcome disposition and no cache can
become an unnamed authority.

This template ships `internal/infra/postgres` and no data-cache client. A
distributed cache therefore adds a dependency and operational surface;
`internal/health` and `internal/infra/oidcjwt` are the in-process precedents.

Use [cache engineering](../../../docs/universal-disciplines/cache-engineering/SKILL.md)
when cache semantics themselves are open and [PostgreSQL
performance](../../../docs/universal-disciplines/postgres-performance/SKILL.md)
for a measured database bottleneck.

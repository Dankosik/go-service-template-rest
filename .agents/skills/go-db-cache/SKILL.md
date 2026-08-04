---
name: go-db-cache
description: "DB/cache: Use for runtime SQL, transactions, cache freshness/invalidation, fallback, or observability. Own access policy; Skip schema authority, distributed consistency, API semantics, or concurrency."
---

# Go DB Cache

Every read and write is an **access path** with a contract: which transaction boundary it commits inside, which tenant identity it carries, and — when a cache sits on it — which freshness it promises and who invalidates it.

`caller intent -> transaction boundary -> query shape -> cache contract -> fallback -> observability -> proof`

The transaction boundary is a business decision: what must be atomically true together defines it, not repository method borders. A cache is a bounded copy that earns existence through measured need, and every cached value names its authority, key scope including tenant, freshness bound, and invalidation path. Fallbacks protect the origin: a failing cache degrades into bounded origin load rather than a stampede.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct affected data paths from changed callers, queries, transactions, cache/config surfaces, fallbacks, and origin access, then trace each through DB resource lifetime.

## Choose The Branch

- **Decision** — select when DB/cache policy is absent or changing. Load [transaction boundary and commit outcome](references/decision/transaction-boundary-and-commit-outcome.md) when the change decides what commits together, whether a failed write may run again, or what an unknown commit outcome means. Complete when shared Decision dispositions cover every affected data path, forced consequence, and focused proof; a cache without measured need is rejected.
- **Review** — select when changed SQL or cache code must conform to accepted policy. Load [postgres access review](references/review/postgres-access-review.md) when a diff changes which seam issues a query, how its rows and errors are handled, or how its deadline is set. Account for every affected path through the shared finding envelope.

This template ships PostgreSQL through `internal/infra/postgres` and no data cache: no cache client is in `go.mod`, so a distributed cache is a new dependency with its own failure and operational surface, not a local change. An in-process cache is cheaper than it looks — `golang.org/x/sync` is already a direct dependency, and `internal/health` and `internal/infra/oidcjwt` are the existing precedents for a cached read with an explicit staleness bound.

Hand authoritative models to `go-data-architecture` and durable recovery to `go-distributed`. Load [`cache-engineering`](../../../docs/universal-disciplines/cache-engineering/SKILL.md) for every cache decision and cache review — key scope, freshness class, layer, fill coordination, invalidation, degraded mode, and proof are its contract, and it defines behavior when a reader's in-flight fill races a writer's invalidation instead of a TTL plus delete-on-write. Load [`postgres-performance`](../../../docs/universal-disciplines/postgres-performance/SKILL.md) when the database bottleneck is asserted rather than measured: it forces baseline-to-delta attribution instead of index guessing.

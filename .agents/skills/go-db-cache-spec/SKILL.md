---
name: go-db-cache-spec
description: "Use when runtime SQL access, transaction controls, cache role, staleness, invalidation, fallback, or DB/cache observability must be decided before coding; Own data-access and cache behavior; Skip when the primary decision is authoritative schema architecture, distributed consistency, endpoint semantics, or implementation."
---

# Go DB Cache Spec

Load the [shared specialist contract](../specialist-contract.md), then apply this repository-access boundary.

## Outcome And Boundary

Turn accepted data, domain, and distributed policy into measurable runtime SQL and cache contracts. Own repository-facing query discipline, transaction boundaries, context and resource safety, retry and commit-uncertainty handling, cache necessity/topology, cache addressing dimensions and cached-payload representation, freshness/TTL/invalidation/stampede behavior, degradation, and mechanism-focused observability and proof.

Do not choose authoritative models, constraints/index policy, schema migration, business acceptance, cross-service recovery topology, endpoint semantics, or package placement. Consume those decisions and record only the access/cache consequences they force.

## DB And Cache Core

1. **Define the origin path before caching it.** Name the operation, read/write owner, consistency class, stable query, explicit columns, bounded round trips, filter/sort shape, and deterministic pagination. Prove N+1, plan, latency, load, or cost pressure rather than using cache to hide an undefined query.
2. **Keep transaction ownership at the use-case boundary.** State all reads, writes, locks, and post-commit effects in the unit. Retry only bounded transient classes by replaying the whole idempotent transaction; never retry an arbitrary statement, and resolve commit uncertainty through a stable idempotency key, authoritative read-back, or reconciliation before repeating effects.
3. **Budget context and resources explicitly.** Propagate caller cancellation, bound pool acquisition and DB/cache work within the end-to-end budget, and normally keep cache timeout shorter than origin timeout. Require rollback on incomplete transactions, rows/body close and terminal error checks, and dedicated-connection cleanup where applicable.
4. **Treat cache as an accelerator by default.** Include no-cache when cache is live; introduce cache only for material evidence and a clear correctness contract. Default ordinary read acceleration to cache-aside, and choose read-through, write-through, write-behind, local, Redis, hybrid, or client-side caching only with its extra consistency and ownership obligations. If accepted policy deliberately makes cache part of observable semantics, name that exception and its stronger failure/decode contract instead of inheriting accelerator defaults.
5. **Make freshness and invalidation concrete.** Assign each operation a staleness class; use deterministic versioned tenant-safe keys containing every response-shaping dimension and a defined value/version format. Require TTL or equivalent bounded freshness, jitter, negative-cache limits, stampede control, and write-linked invalidation/update intent or an explicit harmless-loss fallback; let `go-distributed-spec` own recovery when that linkage crosses a durable boundary.
6. **Specify cache failure as behavior.** Default ordinary read-cache failure and decode failure to bounded miss/fail-open behavior with origin protection; justify fail-closed from accepted user-visible semantics. Define saturation, stale serving, fallback budget, request coalescing/rate limits, and recovery without turning an outage into an origin stampede.
7. **Prove the mechanisms.** Require low-cardinality hit/miss/stale/error/fallback, pool, query, and origin-load signals without raw keys, user/request IDs, or secrets. Test miss/hit, stale/expiry, invalidation race, duplicate/retry and commit uncertainty, cache outage, corrupt value, tenant/key isolation, and origin protection with measurable acceptance bounds.

For Redis client-side caching, additionally require tracking mode, invalidation delivery mode, local TTL/memory bounds, lost-invalidation flush behavior, and redirected-invalidation race handling.

## Reference Files
References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Load lazily for the symptom that matches the active seam; if a reference would not change a decision, do not load it.

| Symptom | Behavior Change | Load |
| --- | --- |
| Slow SQL, N+1, dynamic filters, pagination, generated-query contract, or cache proposed before origin shape is proven | Makes the model require a named, bounded origin query contract and reject cache-as-cover instead of approving Redis around an undefined query path | [sql-access-discipline-and-query-budget.md](references/sql-access-discipline-and-query-budget.md) |
| Write transaction boundary, retry eligibility, idempotency keys, `ON CONFLICT`, or cache invalidation coupled to writes | Makes the model choose whole-use-case retry plus idempotent write and durable invalidation linkage or an explicit harmless-loss fallback instead of statement-level retry or best-effort dual writes | [transaction-retry-and-idempotency-contracts.md](references/transaction-retry-and-idempotency-contracts.md) |
| DB/cache deadline hierarchy, request cancellation, pool saturation, dedicated connection use, or fallback budget | Makes the model budget cache, origin, and pool waits explicitly instead of assuming a handler timeout or larger pool setting is enough | [context-timeout-and-connection-budget.md](references/context-timeout-and-connection-budget.md) |
| Cache requested because a path is slow, or topology is unclear across no-cache, local, distributed, hybrid, or client-side caching | Makes the model compare no-cache and topology tradeoffs with evidence, divergence, memory, key-safety, and client-side invalidation hazards instead of defaulting to Redis | [cache-necessity-and-topology.md](references/cache-necessity-and-topology.md) |
| Freshness window, TTL, jitter, invalidation source, versioned keys, stale-while-revalidate, negative caching, or key transitions | Makes the model assign an operation-level freshness class and invalidation contract instead of treating TTL as correctness proof | [cache-invalidation-staleness-and-ttl.md](references/cache-invalidation-staleness-and-ttl.md) |
| Cache outage, fail-open/fail-closed policy, origin protection, telemetry labels, degraded-mode proof, or test obligations | Makes the model specify containment and low-cardinality proof for degraded cache paths instead of saying "fall back to DB" or testing only hits | [cache-failure-observability-and-testing.md](references/cache-failure-observability-and-testing.md) |

## Return And Stop

Return the runtime path and evidence; SQL/query and transaction contract; context/resource and retry/commit-uncertainty rules; cache necessity and topology; staleness, key/value, TTL, invalidation, jitter, and stampede policy; failure/degradation and origin protection; observability, tests, acceptance bounds, assumptions, forced neighbor consequences, and reopen conditions.

Stop or reject cache without measured need; missing consistency/freshness; incomplete tenant, version, scope, or response-shaping keys; absent invalidation/TTL/stampede policy; undefined timeout/fallback/origin protection; unsafe retry or commit uncertainty; best-effort invalidation dual writes; missing degraded-path proof; or unclassified security-sensitive cache data. Name unresolved data, domain, distributed, API, reliability, security, or rollout policy and do not invent it.

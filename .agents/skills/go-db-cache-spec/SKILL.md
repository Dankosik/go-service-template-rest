---
name: go-db-cache-spec
description: "Design DB-access and cache specifications for Go services. Use when planning or revising SQL access discipline, query and transaction risk controls, cache strategy, staleness and fallback semantics, and DB/cache observability before coding. Skip when the task is a local code fix, primary schema ownership and migration design, endpoint-only API work, CI/container setup, or low-level implementation tuning."
---

# Go DB Cache Spec

## Purpose
Turn runtime DB and cache behavior into explicit, measurable contracts before coding so that performance, correctness, and failure handling are not left to implementation guesswork.

## Scope
Use this skill to define or review DB access discipline, query-shape controls, transaction boundaries, cache strategy, staleness contracts, invalidation rules, fallback behavior, and DB/cache telemetry expectations.

## Boundaries
Do not:
- redesign primary domain ownership or full schema architecture unless DB/cache correctness depends on it
- recommend caching without explicit correctness, staleness, invalidation, and failure behavior
- optimize query code by taste rather than access-pattern evidence
- leave transaction scope, timeout assumptions, or fallback semantics implicit

## Escalate When
Escalate if access patterns are unknown, cacheability depends on unresolved invariants, invalidation cannot be made trustworthy, or DB and cache decisions materially change API-visible consistency or failure behavior.

## Core Defaults
- Keep SQL access query-first and explicit; SQL text plus generated interfaces is the production contract.
- Treat cache as an accelerator, not a source of truth, unless an explicit exception is justified.
- Introduce cache only when there is measured bottleneck evidence and a clear correctness/staleness contract.
- Prefer fail-open read-cache behavior with bounded timeouts and origin-protection controls.
- Prefer compatibility-first evolution (`expand -> migrate/backfill -> contract`) over destructive-first change.

## Expertise

### SQL Access Discipline
- Define query shape per operation class: expected round trips, hot-path query budget, and JOIN or bulk-fetch strategy.
- Use stable query naming and explicit column lists for production business paths.
- Reject `N+1` and chatty repository loops in the spec.
- Allow dynamic identifiers only through explicit allowlists.
- Make pagination deterministic; default hot or deep lists to keyset pagination.

### Transactions, Retries, And Idempotency
- Keep transaction ownership explicit at the use-case boundary.
- Do not hold transactions open across network calls or other cross-service I/O.
- Retries must be explicit and bounded:
  - retry the whole transaction
  - only for transient classes
  - use jittered, bounded attempts
- Retried writes must be idempotent through `ON CONFLICT`, idempotency keys, or an equivalent design.
- Reject hidden distributed-ACID assumptions across service boundaries.

### Context, Timeouts, And Connection Budget
- Propagate context end to end through handler, service, repository, DB, and cache layers.
- Require explicit DB and cache deadlines; no infinite-timeout calls.
- Use cache timeout strictly shorter than origin timeout.
- Make pool-capacity assumptions explicit and validate them with connection-capacity math.
- Require resource-return safety such as `rows.Close`, `rows.Err`, and correct `QueryRow.Scan` handling.

### Cache Necessity And Topology
- Require measured evidence before approving cache: latency, cost, or load reduction must be plausible and material.
- Reject cache when strict consistency is required and no safe bypass exists, when key correctness dimensions cannot be encoded safely, or when ownership/observability is missing.
- Choose topology by constraints:
  - local cache for ultra-low latency with acceptable replica divergence
  - distributed cache for fleet-wide coherence and shared hit ratio
  - hybrid cache only with explicit L1/L2 coherence rules
- Make memory bounds, eviction expectations, timeout budget, and coherence rules explicit.

### Cache Pattern, Consistency, And Invalidation
- Default to cache-aside; require explicit reasons for write-through, write-behind, read-through, or other patterns.
- Define staleness contract per operation class:
  - strong paths bypass cache
  - eventual paths state a bounded staleness window
- Make invalidation source explicit: TTL-only, write-triggered invalidation/update, or event-driven invalidation.
- Require TTL on every entry and use jitter for medium/high-cardinality groups.
- Require stampede controls for hot or expensive keys:
  - request coalescing
  - bounded fallback concurrency
  - backoff on repeated miss-path failures
- If stale-while-revalidate is used, define `fresh_ttl`, `stale_window`, and who may receive stale data.
- If negative caching is used, keep TTL short and separate business negatives from dependency failures.

### Key Safety, Tenant Isolation, And Serialization
- Keys must be deterministic, versioned, tenant-safe, and include every response-shaping dimension that matters.
- Make key length caps, value size caps, and qualifier normalization explicit.
- In pooled caches, tenant dimension is mandatory.
- Secrets do not belong in shared cache; PII belongs there only with explicit controls.
- Make serialization versioning and decode-failure behavior explicit; a decode failure should normally behave like a miss plus observability.
- Never rely on runtime wildcard key scans on production request paths.

### Failure, Degradation, And Origin Protection
- Classify cache dependency behavior per path; fail-open is the default for read acceleration.
- Define timeout hierarchy, retry budget, fallback mode, and containment controls for DB/cache paths.
- Protect origin systems during cache outage with coalescing, bounded fallback concurrency, and degraded responses where allowed.
- Require a fast bypass or disable switch for rollback-safe cache deactivation.
- Make degraded-mode activation and deactivation signals observable.

### Interfaces With API, Data, And Distributed Design
- If cache changes API-visible consistency, freshness, or idempotency, make those semantics explicit.
- Coordinate schema and compatibility windows with data-ownership decisions; cache-key version transitions must survive rollout.
- When invalidation or rebuild depends on async signals, require atomic outbox-equivalent publication, consumer idempotency, and replay-safe handling.
- Reject destructive-first schema assumptions that quietly invalidate cache correctness mid-rollout.

### Observability, Security, And Testing
- Require DB latency/error/pool saturation visibility and cache hit/miss/error/bypass/stale/fallback visibility.
- Keep telemetry low-cardinality; raw keys, user IDs, request IDs, and similar identifiers do not belong in metric labels.
- Require strict boundary validation, parameterized SQL, bounded miss-path concurrency, and no secret leakage in telemetry.
- Define test obligations for hit, miss, error, bypass, stale, negative-cache, timeout, fallback, and stampede-control behavior.
- Require verification in cache-available and cache-degraded modes.

## Decision Quality Bar
For every major DB/cache recommendation, include:
- the runtime problem and evidence
- at least two viable options
- the selected option and at least one explicit rejection reason
- consistency/staleness semantics
- failure policy and origin-protection behavior
- observability and test obligations
- measurable acceptance boundaries
- assumptions, blockers, and reopen conditions

## Deliverable Shape
When writing the DB/cache spec or review, cover:
- SQL access risk profile
- cache necessity decision
- topology and pattern choice
- staleness and consistency contract
- key, tenant, and version safety
- invalidation, TTL, jitter, and stampede controls
- failure and degradation policy
- observability and verification obligations

## Escalate Or Reject
- cache introduced without measured bottleneck evidence
- missing or ambiguous staleness/consistency contract
- key design missing tenant, scope, or version safety
- no explicit invalidation, TTL, jitter, or stampede strategy
- undefined timeout hierarchy, fallback, or origin protection
- missing DB retry/idempotency constraints
- async invalidation that relies on dual writes instead of atomic linkage
- observability or test obligations missing for changed DB/cache paths
- security-sensitive cache surface changed without classification and isolation controls

---
name: go-db-cache-spec
description: "Design DB-access and cache specifications for Go services. Use when planning or revising SQL access discipline, query and transaction risk controls, cache strategy, staleness and fallback semantics, and DB/cache observability before coding. Skip when the task is a local code fix, primary schema ownership and migration design, endpoint-only API work, CI/container setup, or low-level implementation tuning."
---

# Go DB Cache Spec

## Purpose
Turn runtime DB and cache behavior into explicit, measurable contracts before coding so performance, correctness, and failure handling are not left to implementation guesswork.

## Specialist Stance
- Treat DB access and cache behavior as runtime correctness contracts, not implementation tuning.
- Decide query shape, transaction ownership, timeout budget, cache role, staleness, invalidation, fallback, and observability together.
- Keep caches as accelerators unless the approved behavior deliberately makes them part of observable semantics.
- Hand off primary schema architecture, API contract shape, delivery rollout, and local code review when the work leaves this seam.

## Scope
Use this skill to specify SQL access discipline, query-shape controls, transaction boundaries, retry and idempotency constraints, cache necessity, cache topology, staleness contracts, invalidation rules, fallback behavior, and DB/cache telemetry expectations.

Do not use this skill to own primary data modeling, schema migration strategy, endpoint contract design, CI/container configuration, or line-level implementation review. Escalate those seams to the matching architecture, API, data, reliability, security, or review skill.

## Operating Loop
1. Frame the runtime path: operation class, read/write ownership, consistency requirement, hot-path evidence, and failure mode.
2. Load only the relevant reference files from the selector below. Use them for examples and official-source reminders instead of expanding SKILL.md.
3. Compare viable options, including the no-cache option, and reject options that cannot meet correctness, staleness, failure, or proof obligations.
4. Write section-ready spec content with selected and rejected choices, explicit assumptions, acceptance checks, and downstream handoffs.
5. Stop at the specification boundary. Do not drift into implementation code or schema-architecture ownership.

## Reference Files
Load reference files lazily. Pick the smallest set that matches the active DB/cache seam.

| Need | Load |
| --- | --- |
| Query shape, query budgets, N+1 risk, pagination, generated SQL contracts, SQL telemetry grouping | `references/sql-access-discipline-and-query-budget.md` |
| Transaction scope, isolation, retries, `ON CONFLICT`, idempotency, cache invalidation linked to writes | `references/transaction-retry-and-idempotency-contracts.md` |
| Context propagation, DB/cache deadlines, pool capacity, connection return safety, timeout hierarchy | `references/context-timeout-and-connection-budget.md` |
| Cache approval, no-cache decision, local vs distributed vs hybrid topology, client-side caching | `references/cache-necessity-and-topology.md` |
| Staleness windows, TTL, jitter, invalidation source, key versioning, stale-while-revalidate, negative caching | `references/cache-invalidation-staleness-and-ttl.md` |
| Cache outage policy, origin protection, observability, low-cardinality telemetry, degraded-mode tests | `references/cache-failure-observability-and-testing.md` |

## Core Defaults
- Keep SQL access query-first and explicit: stable query names, explicit column lists, bounded round trips, and deterministic pagination for list paths.
- Introduce cache only when latency, load, or cost evidence makes the benefit material and the correctness contract is clear.
- Default to cache-aside for read acceleration. Choose write-through, write-behind, read-through, local cache, Redis, or hybrid only when the spec states the extra contract.
- Default read-cache failures to fail-open with bounded timeouts and origin-protection controls. If fail-closed is proposed, explain the user-visible semantics that require it.
- Keep transaction ownership at the use-case boundary. Retry only bounded, transient classes, and retry the whole transaction with idempotent write semantics.
- Require explicit DB and cache deadlines. Cache timeout should normally be shorter than origin timeout.
- Require every cache entry to have a TTL or equivalent bounded freshness mechanism; use jitter for large synchronized groups.
- Require deterministic, versioned, tenant-safe keys that include every response-shaping dimension.
- Treat decode failures as cache misses plus observability unless the cache is the source of observable semantics.
- Keep telemetry low-cardinality: no raw cache keys, user IDs, request IDs, or secrets in metric labels.

## Decision Quality Bar
For every major DB/cache recommendation, include:
- the runtime problem and evidence
- at least two viable options, including no-cache when cache is on the table
- the selected option and at least one explicit rejection reason
- consistency and staleness semantics
- timeout, fallback, and origin-protection behavior
- observability and test obligations
- measurable acceptance boundaries
- assumptions, blockers, and reopen conditions

## Deliverable Shape
When writing the DB/cache spec, cover:
- SQL access risk profile
- cache necessity decision
- topology and pattern choice
- staleness and consistency contract
- key, tenant, and version safety
- invalidation, TTL, jitter, and stampede controls
- failure and degradation policy
- observability and verification obligations
- downstream handoffs caused by API-visible freshness, schema ownership, async invalidation, or rollout risk

## Escalate Or Reject
- cache introduced without measured bottleneck evidence
- missing or ambiguous staleness or consistency contract
- key design missing tenant, scope, version, or response-shaping dimensions
- no explicit invalidation, TTL, jitter, or stampede strategy
- undefined timeout hierarchy, fallback, or origin protection
- missing DB retry and idempotency constraints
- async invalidation relying on untracked best-effort dual writes
- observability or test obligations missing for changed DB/cache paths
- security-sensitive cache surface changed without classification and isolation controls

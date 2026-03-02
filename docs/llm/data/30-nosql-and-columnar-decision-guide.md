# NoSQL and columnar datastore decision guide for LLMs

## Load policy
- Load: Optional
- Use when:
  - Choosing datastore class for a new service or major workload change
  - Deciding between SQL OLTP, NoSQL (document/key-value/wide-column/time-series), and analytical/columnar DB
  - Designing partitioning, consistency model, retention, or hot-partition mitigation
  - Reviewing datastore choice for workload fit and operational readiness
- Do not load when: The task is a small local code change with fixed datastore choice and no data-architecture decision

## Purpose
- This document defines operational defaults for datastore selection.
- Goal: prevent architecture-by-hype decisions and force workload-fit-first choices.
- Treat this as an LLM contract: start from defaults, document deviations, and reject vague “scale/perf” claims without evidence.

## Baseline assumptions
- Default system-of-record for service business data: SQL OLTP (PostgreSQL-compatible).
- Default architecture: one service owns one datastore boundary.
- Default write path: transactional command path stays on OLTP.
- NoSQL or columnar DB is an explicit exception, not the default.
- If workload shape, consistency, or ops maturity is missing:
  - keep SQL OLTP,
  - optionally add cache for proven read bottlenecks,
  - postpone new datastore adoption.

## Required inputs before choosing datastore class
Resolve these first. If missing, apply defaults and state assumptions.

- Dominant access patterns:
  - point lookups, aggregate reads, range scans, time-window queries, ad-hoc analytics
- Read/write profile:
  - QPS, write amplification risk, expected data growth
- Consistency and correctness requirements:
  - cross-entity invariants, read-your-writes, conflict handling, acceptable staleness
- Data lifecycle:
  - retention window, delete policy, archival/downsampling requirements
- Query flexibility:
  - fixed/predefined queries vs ad-hoc exploratory queries
- Operational maturity:
  - on-call expertise, backup/restore drills, observability for partitions/compaction/replication
- Cacheability signals:
  - deterministic cache key, expected key reuse, invalidation strategy, security/privacy boundaries

## Primary decision matrix

| Workload signal | Preferred class | Why | Non-negotiable constraints |
|---|---|---|---|
| Transactional CRUD, relational invariants, frequent updates | SQL OLTP | ACID, joins, mature transactional semantics | Keep service-owned boundary; avoid cross-service DB coupling |
| Aggregate-oriented object reads/writes (entity graph read together) | Document store | Store and read aggregates together | Bound document growth; avoid unbounded arrays; define schema versioning |
| Deterministic key lookups, sessions/idempotency/state by key | Key-value store | Very low-latency key access, simple horizontal scaling | Query-by-key only by default; no keyspace walks in hot path |
| Predictable high-throughput writes with partition-key + range reads | Wide-column store | Scale by partitions for known query patterns | Table-per-query design; no filtering outside key path; control partition size |
| Metrics/events with time-window queries and retention/downsampling | Time-series store | Time-first ingestion/query model and lifecycle controls | Control cardinality of tags/labels; retention policy is mandatory |
| Large-scale scans, aggregations, dashboards/BI, event analytics | Columnar/analytical DB | Column pruning + compression + OLAP execution model | Keep out of critical OLTP write path; use async ingestion model |

## When to choose NoSQL instead of SQL OLTP

### Document store
Choose when:
- Most reads/writes are per aggregate/document.
- Data is naturally hierarchical or polymorphic.
- Join-heavy ad-hoc query needs are limited.

Operational defaults:
- Model from access patterns, not from normalized ERD first.
- Embed data read together; reference only where growth/cardinality demands it.
- Require schema version field + backward-compatible readers.
- Enforce max document growth policy (especially arrays).

Do not choose when:
- Core workload needs frequent multi-entity joins and changing ad-hoc query shapes.
- Strict cross-document transactional invariants dominate.

### Key-value store
Choose when:
- Main access path is strict key -> value lookup.
- Use cases are sessions, idempotency keys, short-lived materialized responses, rate/lock tokens.
- Latency target is dominated by repeated point reads.

Operational defaults:
- Key format MUST include all correctness dimensions (`tenant`, `scope`, `version`, etc.).
- TTL is lifecycle hygiene only; never correctness-critical deadline enforcement.
- Enforce keyspace iteration policy:
  - no blocking full keyspace operations in production paths.

Do not choose when:
- Primary product queries need secondary filters/sorting over many attributes.
- Team expects OLTP-like relational querying without maintaining additional projections.

### Wide-column store
Choose when:
- Query set is known and stable.
- Workload is partition-key-centric with high write throughput.
- Horizontal partition scaling is required beyond comfortable OLTP limits.

Operational defaults:
- Table-per-query is mandatory.
- Partition key must be chosen from measured access distribution.
- Bucketing/sharding strategy is mandatory to bound partition size.
- Reject query patterns that require filtering outside partition/clustering keys.

Do not choose when:
- Query patterns are evolving rapidly or largely unknown.
- Product requires flexible ad-hoc filtering across many dimensions.

### Time-series store
Choose when:
- Data is time-indexed events/measurements.
- Product needs time-range queries, rollups, and retention controls.
- Ingestion is append-heavy.

Operational defaults:
- Schema is access-pattern-first:
  - define dimensions/tags, measures/fields, and query windows before schema.
- Cardinality budget is mandatory for tags/labels.
- Retention + downsampling policy is mandatory before production rollout.

Do not choose when:
- Workload is mostly transactional entity updates.
- Team cannot operate retention/compaction/downsampling lifecycle.

## When columnar/analytical DB is justified (and how it differs from OLTP)

Use columnar/analytical DB when all are true:
- Dominant workload is scan/filter/group/aggregate over large history.
- Data freshness can be near-real-time or batch-latency, not strict immediate consistency.
- Team accepts append-first ingestion and asynchronous correction flows.

Do not use as primary OLTP store when:
- Service requires frequent row-level updates/deletes with strict low-latency transactional guarantees.
- Request path requires single-row transactional semantics as first-class behavior.

OLTP vs columnar default differences:
- Access unit:
  - OLTP: row/transaction
  - Columnar: column segments and scan ranges
- Write model:
  - OLTP: frequent updates/deletes
  - Columnar: append-heavy with background merge/compaction/mutation workflows
- Consistency:
  - OLTP: strong transactional semantics
  - Columnar: often eventual in ingestion/compaction path
- Serving role:
  - OLTP: source of truth for command path
  - Columnar: analytical/read model fed asynchronously

## Access-pattern-first modeling and partitioning defaults
Default rule: datastore model starts from explicit query catalog.

Required artifact before schema/key design:
- Access pattern list with:
  - request shape (filters, sort, pagination/time window),
  - expected cardinality/selectivity,
  - latency/SLO target,
  - consistency requirement per operation.

Partitioning defaults:
- Partition key MUST be evaluated for:
  - distribution (avoid concentration on few keys),
  - growth (bounded partition size over time),
  - read/write locality for dominant queries.
- Add synthetic bucketing/shard suffix when natural key causes concentration.
- For time-partitioned systems, align partition granularity with retention and query windows.

Hot partition defaults:
- Must define detection signals before production:
  - top-N key/partition traffic share,
  - per-partition latency and throttling,
  - write/read skew.
- Must define mitigation playbook:
  - re-keying or bucketing strategy,
  - pre-splitting/high-cardinality key components,
  - client-side backoff and workload shaping.

## Consistency trade-offs defaults
- Consistency model must be explicit per critical operation, not global hand-waving.
- For eventually consistent reads:
  - define acceptable staleness window,
  - define conflict resolution behavior.
- For conditional writes/CAS/optimistic locking:
  - use where correctness depends on write ordering.
- For multi-region replication:
  - document conflict model and “last-writer-wins” implications if applicable.
- Never claim strict consistency on paths that traverse async ingestion or background compaction.

## Retention and lifecycle defaults
- Retention policy is required for NoSQL and columnar stores before launch.
- Retention policy must specify:
  - duration per data class,
  - deletion mechanism (TTL/partition drop/rules),
  - archival target and restore requirements.
- TTL semantics must be documented precisely:
  - if TTL deletion is asynchronous/best-effort, application logic must not depend on exact expiration timestamp.
- For time-series and analytical workloads:
  - define downsampling/rollup windows and loss of granularity explicitly.

## Cache-first guardrail (before new datastore)
Default rule: do not introduce NoSQL/columnar only to hide read latency without workload proof.

Before adopting a new datastore for read performance, verify:
- Repeated read patterns and hot keys are measured.
- Deterministic cache key can be built safely (tenant/auth/version aware).
- Invalidation/TTL semantics are defined and observable.
- Security/privacy boundaries permit caching.

If these hold, default to trying cache-aside first for read bottlenecks.
If cacheability is poor or query shape fundamentally mismatches OLTP, then evaluate NoSQL/columnar class.

## Decision rules (if/then)
Use these rules in order:

1. If strict transactional invariants across multiple entities are required, keep SQL OLTP.
2. If workload is unknown or query surface is still rapidly changing, keep SQL OLTP and postpone NoSQL specialization.
3. If dominant read/write is by aggregate document and joins are limited, use document store.
4. If dominant access is deterministic key lookup with simple value semantics, use key-value store.
5. If workload is partition-key + range driven at high scale with stable query set, use wide-column store.
6. If workload is time-window metrics/events with retention/downsampling requirements, use time-series store.
7. If workload is large analytical scans/aggregations and async ingestion is acceptable, add columnar/analytical DB as read model.
8. If the only pain is repeated expensive reads and cacheability is high, add cache before adding new primary datastore.
9. If team cannot run backup/restore/on-call/observability for the new engine, reject datastore migration.

## Anti-patterns to reject
- Choosing NoSQL/columnar because of trend, not measured workload.
- Replacing OLTP source of truth with columnar DB for transactional command path.
- NoSQL adoption without explicit access-pattern catalog.
- Designing partition keys without skew and growth analysis.
- Accepting hot partitions as “autoscaling issue” instead of data-model issue.
- Using TTL as exact business-time enforcement.
- Treating key-value store as flexible query engine.
- Wide-column queries that rely on filtering outside key model.
- Time-series tags/labels with unbounded cardinality (user/session/request IDs as default labels).
- Multi-datastore architecture without ownership boundaries and reconciliation plan.
- Adding datastore complexity without operational readiness (monitoring, backup, restore, runbooks, capacity controls).

## MUST / SHOULD / NEVER

### MUST
- MUST keep SQL OLTP as default unless workload-fit criteria explicitly justify alternatives.
- MUST document access patterns before selecting NoSQL or columnar class.
- MUST define partitioning, hot-partition detection, and mitigation before production.
- MUST define consistency and conflict behavior for critical operations.
- MUST define retention and deletion semantics explicitly.
- MUST validate operational readiness (observability, backup/restore, on-call runbooks) before go-live.

### SHOULD
- SHOULD prefer a single primary system-of-record per bounded context.
- SHOULD add cache before datastore migration when bottleneck is read amplification and cacheability is strong.
- SHOULD keep analytical DB as asynchronous read model, not command path dependency.
- SHOULD use additive, backward-compatible schema/key evolution plans.
- SHOULD benchmark representative access patterns before committing to new datastore class.

### NEVER
- NEVER select NoSQL/columnar without explicit workload evidence.
- NEVER assume OLTP-like joins/transactions in stores that do not provide them by default.
- NEVER use exact-time business logic that depends on best-effort TTL deletion.
- NEVER allow unbounded scans/keyspace walks in hot production paths.
- NEVER ignore partition skew/hot keys as purely infrastructure concern.
- NEVER ship multi-datastore design without clear ownership and reconciliation boundaries.

## Review checklist
Before approving datastore-choice changes, verify:

- Workload fit evidence:
  - Access patterns are documented and mapped to chosen datastore primitives.
  - Read/write profile and latency goals are explicit.
- Modeling and partitioning:
  - Key/schema design matches dominant access patterns.
  - Partition strategy includes skew mitigation and bounded growth.
  - Hot-partition detection metrics and runbook exist.
- Consistency and correctness:
  - Required guarantees are explicit per operation.
  - Conflict handling/idempotency is defined for distributed writes.
  - Staleness window is documented where eventual consistency applies.
- Retention and lifecycle:
  - Retention, expiration, archival, and purge semantics are explicit.
  - Time-series/analytical rollup and granularity loss are documented.
- Operational readiness:
  - Backup and restore are tested.
  - Capacity and cost controls are defined.
  - Alerts and dashboards exist for engine-specific failure modes.
- Migration/risk control:
  - Rollout is staged with rollback strategy.
  - Dual-write/read-model sync risks are addressed if introducing second datastore.

## What good output looks like
- Datastore choice is justified by measured workload, not by generic scalability claims.
- Access patterns, partitioning, consistency, and retention are explicit and testable.
- Hot-partition and lifecycle risks are handled before incidents, not after.
- NoSQL/columnar adoption is operationally supportable by the team.
- LLM-generated proposals remain conservative, reviewable, and rollback-safe.

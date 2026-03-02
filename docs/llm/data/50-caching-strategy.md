# Caching strategy instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Deciding whether a service should add or change caching
  - Designing cache topology (`local`, `distributed`, or `hybrid`)
  - Defining cache patterns (`cache-aside`, `read-through`, `write-through`, `stale-while-revalidate`)
  - Defining TTL, invalidation, stampede protection, and fallback behavior
  - Reviewing cache correctness, observability, reliability, and test coverage
- Do not load when: Task is a local refactor with no cache behavior, no read-path bottleneck, and no caching boundary change

## Purpose
- This document treats caching as a controlled architecture tool, not an automatic optimization.
- Goal: improve latency and cost without breaking correctness, tenant isolation, or operational stability.
- Treat this as an LLM contract: apply defaults first, document deviations, and reject cache designs that are not observable or rollback-safe.

## Baseline assumptions
- Cache is an accelerator, not source of truth (unless explicitly approved by ADR).
- Default read pattern: `cache-aside` with bounded staleness.
- Default write consistency: source-of-truth write first, then cache update/invalidation.
- Default failure policy for read caches: `fail-open` with bounded timeouts.
- Default security posture: never cache secrets; cache PII only with explicit approval and isolation controls.
- If consistency, staleness, or data classification is missing:
  - assume bounded eventual consistency is acceptable for read-only views,
  - assume strict consistency paths must bypass cache,
  - and state assumptions explicitly.

## Required inputs before adding or changing cache
Resolve these first. If missing, apply defaults and state assumptions.

- Endpoint/job SLO target (`p95`, `p99`) and current bottleneck evidence
- Read/write profile (hot keys, request repetition, expected hit ratio)
- Data volatility and allowed staleness window
- Correctness requirements (read-your-writes, linearizability, audit-critical behavior)
- Tenant model and authorization scope that affect response shape
- Key design inputs (all dimensions that change result)
- Invalidation source (TTL-only, write-triggered invalidation, event-driven invalidation)
- Failure policy (cache down/timeout/network partition)
- Capacity constraints (local heap/GC budget, shared cache memory budget, eviction policy)
- Rollout and rollback plan (feature flag, canary, bypass switch)

## Decision framework: when cache is needed vs not needed
Default rule: do not add cache until bottleneck is measured.

### Cache is justified when
- Read path is measurably hot and repetitive (high key reuse, not random one-off reads).
- Primary store or upstream dependency is proven bottleneck for latency or cost.
- Deterministic key can be constructed from all correctness dimensions.
- Business accepts bounded staleness for that response class.
- Team can operate cache with metrics, alerts, and failure runbook.

### Cache is not justified when
- SLO is already met without cache.
- Data changes frequently and stale reads are unacceptable.
- Strict read-after-write or strict authorization visibility is required.
- Key cannot safely encode all inputs (tenant/scope/version/locale/feature flags).
- Team cannot define invalidation, fallback, and observability before rollout.

### Cache is forbidden by default when
- Response may leak secrets or cross-tenant data without strict isolation.
- Cache would become hard dependency with no safe fallback to origin.
- Design relies on exact expiration moment for correctness.
- No objective measurement plan exists (`hit/miss`, latency, evictions, fallback rate).

### Precompute vs cache
- Use cache when: response is expensive but reusable and staleness window is acceptable.
- Use precompute/materialized read model when: query set is predictable and stable low latency is needed without TTL edge effects.
- Default: long-lived acceleration should be modeled as read model/materialization with explicit ownership, not indefinite cache layering.

## Topology selection: local vs distributed vs hybrid
Default rule: choose topology by consistency and scale constraints, not by preference.

### Local cache (`in-memory`)
Use when:
- Single instance or small replica count.
- Data is read-mostly and divergence between instances is acceptable.
- Ultra-low latency is required and network RTT to shared cache is significant.

Mandatory controls:
- Hard size/cost limit.
- TTL for all cache entries.
- Eviction policy and memory/GC observability.
- No assumption that local cache values are globally consistent.

### Distributed cache (`Redis`/`Memcached`-like)
Use when:
- Many service replicas need shared cache state.
- Need fleet-wide hit ratio and centralized memory policy.
- Want to offload origin under horizontal scale.

Mandatory controls:
- Explicit timeout budget and fallback on cache failures.
- Explicit memory limits and eviction policy.
- Serialization format/version discipline.
- Batch/pipeline behavior for high-throughput key access.

### Hybrid (`L1 local + L2 distributed`)
Use when:
- Need very low tail latency and shared cache coherence across replicas.
- Read volume is high enough to justify two-layer complexity.

Mandatory controls:
- Versioned keys and deterministic invalidation path.
- Independent TTL policy for L1 and L2.
- Guardrails against stale amplification (L1 serving old values after L2 refresh).

## Pattern defaults and consistency trade-offs
Default rule: `cache-aside` first. Other patterns require explicit justification.

### Cache-aside (default)
Flow:
1. Try cache read.
2. On miss, load from origin.
3. Store with TTL (+ jitter) and return.

Rules:
- Stampede protection is mandatory for expensive/hot keys.
- Do not write cache before origin read/write is confirmed.
- Miss path must be observable by reason (`cold`, `expired`, `evicted`, etc.).

### Read-through
- Allowed as implementation style when wrapper/library owns miss loading.
- In application-level Go services, emulate with cache-aside behavior to keep control explicit.

### Write-through
- Allowed for bounded data classes where immediate post-write reads dominate.
- Default write order: persist to source of truth, then update/invalidate cache.
- If cache update fails after source write, do not fail the business write by default; enqueue/trigger repair and record metric.

### Write-behind / write-back
- Not a default for this template.
- Allowed only with explicit durability model, retry semantics, and replay-safe recovery plan.

### Stale-while-revalidate (SWR)
Use only when stale response is contractually acceptable.

Mandatory model:
- `fresh_ttl`: period where value is returned as fresh.
- `stale_window`: extra bounded period where stale can be served while async refresh runs.

Rules:
- Never serve stale for strict-consistency or security-critical data.
- Track stale responses separately (`stale_hit` outcome).
- Refresh path must be bounded by timeout and concurrency controls.

### Stampede protection
Mandatory for hot/expensive keys.

Required controls:
- In-process request coalescing per key (`singleflight` or equivalent).
- Randomized TTL jitter to avoid synchronized expiry.
- Optional distributed lock for cross-instance coalescing when needed.
- Bounded concurrent origin loads and backoff on repeated failures.

### TTL and jitter defaults
- Every cache entry MUST have TTL unless ADR says data is not cache.
- Baseline TTL ranges (starting defaults, tune by measurements):
  - entity/profile reads: `30s-300s`
  - list/search summaries: `10s-120s`
  - negative cache (`not found`): `5s-30s`
- Jitter default: `+-10%` of TTL for medium/high-cardinality key groups.
- Never assume exact expiry timestamp for correctness.

### Negative caching
- Allowed for stable negative results (`not found`, empty set with stable semantics).
- Must use short TTL.
- Must distinguish negative result from dependency failure.
- Never cache transient upstream failures as negative business state.

## Key design, tenant safety, serialization, and memory discipline

### Key design defaults
Default schema:
`{svc}:{env}:{dataset}:v{keyver}:tenant:{tenantID}:{entity}:{id}:{qualifiers...}`

Rules:
- Include all dimensions that affect response: tenant, auth scope, locale, feature variant, schema/key version.
- Normalize qualifiers (stable ordering, normalized casing) before key creation.
- Use `v{keyver}` in key for schema/key-semantics changes.
- Guardrails:
  - key length `<= 256 bytes`
  - serialized value size `<= 1 MiB` before compression
- Hash long/unbounded segments (for example, search filters blob) instead of embedding raw payload.
- Runtime scan policy: never use `KEYS`; use bounded `SCAN` only for operational workflows.

### Tenant and security safety
- Tenant identifier in key is mandatory for pooled multi-tenant caches.
- If authorization scope changes response, include scope dimension in key.
- Shared cache must not store secrets or highly sensitive PII by default.
- Per-user/private cache responses must never reuse generic shared keys.

### Serialization defaults
- Default: JSON for interoperability and debugging.
- Alternative binary formats are allowed only when measured CPU/latency benefit is proven.
- Store payload schema version (`valver`) in payload or key.
- On deserialization error:
  - count as cache corruption/schema mismatch metric,
  - treat as miss,
  - remove invalid entry asynchronously or inline when safe.

### Compression defaults
- Do not compress by default for small payloads.
- Consider compression only above measured threshold (starting point `4-16 KiB`).
- Encoded payload must be self-describing (algorithm + version marker).
- Enforce decompressed-size limit to avoid decompression bombs.

### Memory discipline defaults
- Local cache:
  - must be bounded (entry count or cost in bytes),
  - must expose memory/eviction metrics,
  - must be reviewed against heap/GC budget.
- Distributed cache:
  - explicit maxmemory and eviction policy are mandatory,
  - monitor eviction and expiry rates,
  - avoid unbounded key cardinality.
- For very high cardinality of tiny objects, consider packing strategy (for example hash packing) only with documented TTL/invalidation trade-offs.

## Failure behavior and fallback when cache is degraded
Default rule: cache failures should not take down read path.

### Default policy (`fail-open` for read cache)
- Cache timeout must be shorter than origin timeout.
- On cache timeout/error, treat as miss and continue with origin path.
- Use circuit-breaker/disable-window for cache dependency on repeated failures.
- Retries to cache should be minimal and jittered; never unbounded.

### Overload and outage controls
- Protect origin during cache outage:
  - coalescing on miss path,
  - concurrency limits for origin fallback,
  - optional degraded response path where allowed.
- Support fast bypass switch (feature flag/config) to disable cache logic without redeploy.

### Fail-closed exceptions
- Allowed only for explicitly approved domains (for example, lock/token semantics where cache is state authority).
- Requires separate ADR with availability and correctness impact analysis.

## Mandatory metrics, alerts, and dashboards
Default rule: cache without observability is not merge-ready.

### Application-level metrics (mandatory)
- `cache_requests_total{cache,op,outcome}`
  - `outcome` bounded to: `hit|miss|error|bypass|stale_hit|negative_hit`
- `cache_misses_total{cache,reason}`
  - `reason` bounded to: `cold|expired|evicted|invalidated|not_found|bypass|dependency_error|serialization_error`
- `cache_op_duration_seconds{cache,op}` histogram
- `cache_errors_total{cache,class}` with bounded classes (`timeout`, `conn`, `protocol`, `auth`, `other`)
- `cache_fallback_total{cache,to,reason}`
- `cache_stale_served_total{cache,reason}` when SWR or serve-stale is enabled
- Optional but recommended:
  - `cache_inflight_loads{cache}` gauge
  - `cache_refresh_total{cache,outcome}`

Rules:
- Label cardinality must stay bounded.
- Never use raw key, user ID, email, or request ID as metric labels.

### Backend cache metrics (mandatory in monitoring)
- Redis-like:
  - keyspace hits/misses
  - expired keys
  - evicted keys
  - memory used and fragmentation indicators
- Memcached-like:
  - get hits/misses
  - evictions
  - memory usage

### Alerting minimum
- Sustained hit-ratio drop vs baseline.
- Eviction spike and memory pressure.
- Cache error-rate increase and fallback surge.
- Stale-served ratio above service contract threshold.

## Mandatory tests for cache correctness and reliability
Default rule: cache changes require behavior tests, not only line coverage.

### Unit tests (table-driven)
Must cover:
- hit path (origin not called)
- miss path (origin called once, cache populated)
- expired/evicted behavior
- negative caching semantics
- deserialization error path
- cache timeout/error fallback path
- SWR window behavior if enabled

### Concurrency tests
Must cover:
- stampede suppression for same key under parallel requests
- bounded origin calls under concurrent misses
- race detector clean run for concurrent cache wrapper code

### Integration tests
Must run in two modes:
- cache available (real backend)
- cache degraded/unavailable (timeouts or connection errors)

Must verify:
- fail-open behavior
- bypass switch behavior
- no correctness regression in origin-backed path

### Load and failure tests
Required profiles:
- warm cache
- cold cache/new node
- cache outage/high error rate

Must measure:
- endpoint `p95/p99`
- primary store load increase under cache degradation
- miss reason distribution
- eviction/expiration dynamics

### Optional consistency sampling check
- For a low sample rate, compare cached read vs origin read to detect stale/corrupt drift.
- Must be feature-flagged and budgeted to avoid self-inflicted load.

## Decision rules (if/then)
Use these in order:

1. If strict consistency is required for the operation, bypass cache by default.
2. If no measured read bottleneck exists, do not add cache.
3. If key reuse is low or key space is highly random, do not add cache.
4. If data is read-heavy and staleness is acceptable, use cache-aside.
5. If immediate post-write reads dominate and data class is bounded, consider write-through + cache-aside fallback.
6. If many replicas need shared cache state, use distributed cache.
7. If single instance or divergence is acceptable and latency is critical, use local cache.
8. If both ultra-low latency and shared-state benefits are required, use hybrid with explicit invalidation rules.
9. If hot-key stampede risk exists, require coalescing + TTL jitter before rollout.
10. If cache outage can overload origin, require fallback guardrails and bypass switch before rollout.
11. If cache stores tenant-sensitive data, require tenant-safe keys and security review before rollout.
12. If required observability or tests are missing, reject merge.

## Anti-patterns to reject
- Adding cache without bottleneck evidence.
- Treating cache as source of truth without explicit architecture decision.
- Missing TTL or synchronized TTL for large key groups.
- No stampede protection on hot keys.
- Caching upstream errors as valid negative business state.
- Key design without tenant/scope/version dimensions.
- Unbounded key cardinality from raw query parameters.
- Runtime `KEYS` scans on production traffic paths.
- Shared cache as mandatory dependency with no fallback.
- High-cardinality metric labels (`key`, `user_id`, `request_id`).
- Ignoring local cache heap/GC impact.
- Ignoring distributed cache eviction/memory pressure behavior.

## MUST / SHOULD / NEVER

### MUST
- MUST justify cache with measured latency/cost bottleneck.
- MUST define and document staleness contract per cached data class.
- MUST use explicit key schema with tenant and version safety.
- MUST set TTL for cache entries and apply jitter for large key groups.
- MUST implement stampede protection for hot/expensive keys.
- MUST define fail-open/fallback behavior with bounded timeouts.
- MUST instrument cache outcomes, miss reasons, latency, and errors.
- MUST provide unit + concurrency + integration coverage for cache behavior.

### SHOULD
- SHOULD start with cache-aside before more complex patterns.
- SHOULD use distributed cache when fleet-wide coherence is needed.
- SHOULD keep local cache bounded and minimal in responsibility.
- SHOULD use short TTL negative caching only for stable negative results.
- SHOULD use feature flags for cache rollout and bypass.
- SHOULD document eviction policy and memory budget in service config.

### NEVER
- NEVER add cache as default without evidence.
- NEVER rely on exact expiration timing for business correctness.
- NEVER cache secrets in shared cache.
- NEVER use runtime `KEYS` in production request path.
- NEVER treat cache connectivity failure as automatic request failure for read acceleration paths.
- NEVER ship cache changes without metrics and fallback runbook.

## Review checklist
Before approving cache-related changes, verify:

- Decision quality:
  - Bottleneck evidence exists.
  - Chosen topology (`local`/`distributed`/`hybrid`) is justified.
  - Staleness and consistency contract is explicit.
- Pattern correctness:
  - Pattern choice is explicit (`cache-aside`, `write-through`, `SWR`, etc.).
  - Stampede protection is present where needed.
  - TTL/jitter policy is configured and tested.
- Key and data safety:
  - Key schema includes tenant/scope/version dimensions.
  - Key/value size guardrails are enforced.
  - Serialization versioning and decode-failure handling are defined.
- Resilience:
  - Cache timeout budget is bounded and shorter than origin.
  - Fail-open/fallback behavior is explicit.
  - Bypass switch/circuit behavior is available.
- Observability:
  - Required application metrics exist with bounded labels.
  - Backend cache metrics are wired into dashboards.
  - Alerts for hit ratio, errors, evictions, fallback are defined.
- Testing:
  - Unit tests cover hit/miss/error/stale/negative paths.
  - Concurrency tests verify coalescing and race safety.
  - Integration tests validate both cache-up and cache-down modes.
  - Load/failure profile results are available for changed behavior.

## What good output looks like
- Cache is introduced only for measured bottlenecks.
- Consistency and staleness semantics are explicit per operation.
- Key design is deterministic, tenant-safe, and versioned.
- Stampede, TTL jitter, and fallback are built in, not optional.
- Metrics and tests make correctness and degradation visible.
- LLM-generated cache proposals are conservative, reviewable, and rollback-safe.

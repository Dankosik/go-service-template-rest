# Keep Redis and Valkey out until measured adopter evidence earns them

status: ready

Problem: A template-level Redis or Valkey option would create a supported
runtime, operations, and correctness contract before any named workload has
shown that a cache is needed. The safe current outcome is to preserve the
template's cache-free behavior and make the evidence that may reopen it
explicit.

## Scope and non-goals

This specification covers whether the current template exposes an optional
Redis or Valkey cache capability. It accepts no cache-related behavior delta:
the template adds no dependency, profile, selector, configuration, runtime
client, probe, readiness rule, telemetry surface, or generic cache abstraction.

It does not select a cached value, product, version, client, topology, provider,
operator, deployment route, or implementation owner. It does not define cache
keys, source revisions, freshness, TTL, invalidation, serialization, negative
entries, coalescing, tenant isolation, degraded behavior, or semantic
telemetry. Those decisions require a named feature and remain outside this
template-level contract.

Technical Design is not triggered because there is no runtime, integration,
data, package, or lifecycle behavior to realize. Test Design is not triggered
because the accepted outcome adds no behavior and its absence and authority
invariants have direct repository-level falsifiers.

## Behavior and contract delta

### Current disposition

- Cache-related behavior is deliberately unchanged. Selecting or initializing
  this template offers no Redis or Valkey capability and introduces no
  cache-specific operational obligation.
- Cache absence cannot affect startup, liveness, readiness, request handling,
  shutdown, or telemetry because no cache dependency exists.
- PostgreSQL remains authoritative for persisted feature state. Any future cache
  is a derived, disposable observation and cannot become a durable write path or
  second source of truth under this capability.
- A feature that independently earns caching owns the complete cached-value
  contract: authority and source revision; complete key and tenant or policy
  scope; freshness, TTL, invalidation, and read-after-write behavior;
  serialization and mixed-version handling; negative entries; fill and
  coalescing behavior; degraded response and origin protection; privacy; and
  semantic telemetry.
- Deployment remains the owner of any future product and topology selection,
  network and TLS trust, ACLs and credentials, server memory and eviction
  policy, operator or provider contract, failover behavior, and recovery owner.

### Decision-flip test

The cache capability remains rejected because the required evidence is
conjunctive and none of its decisive elements currently exists:

| Required evidence | Current result |
| --- | --- |
| A named feature and cached value | Absent. No accepted adopter or value is named. |
| A representative cache-disabled baseline | Absent. No request distribution, fleet size, origin headroom, working-set/cardinality estimate, value-size distribution, or cache-outage load case was supplied. |
| An accepted latency or origin-capacity target missed by the baseline | Absent. No numeric or qualitative production objective rejects the current path. |
| Evidence that PostgreSQL/query/computation optimization, request coalescing, and a feature-owned local adapter cannot satisfy the target | Absent. No named workload has exhausted those smaller alternatives. |
| An exact distributed-cache contract | Absent. No product, supported version, client, topology, operator/provider, TLS/auth route, command subset, or recovery owner is accepted. |

The current repository check at `40e6d212799ae8677b675339929c559246536181`
plus the dirty worktree found no Redis or Valkey runtime dependency, profile,
selector, bootstrap path, or named adopter evidence. The unrelated dirty
dependency and bootstrap changes are not cache evidence.

## Invariants and edge cases

- A Redis or Valkey server response never outranks PostgreSQL authority.
- Similar client calls across features do not establish shared cache semantics.
- Redis, Valkey, standalone, Sentinel, Cluster, and managed-provider offerings
  are not interchangeable compatibility targets.
- TTL, invalidation, Pub/Sub, transactions, replication acknowledgements,
  distributed locks, or client-side caching cannot by themselves establish
  freshness, atomicity with PostgreSQL, or safe failover.
- A cache probe can never gate liveness. Readiness criticality, fail-open,
  fallback, stale serving, shedding, and origin protection are feature-specific
  behavioral decisions and have no template default.
- Cache keys, values, tenant identities, subjects, and raw server errors cannot
  become generic transport telemetry attributes. Hit, miss, stale,
  negative-hit, coalescing, invalidation-lag, fallback, and origin-load signals
  remain feature-owned semantics.

## Decisions, constraints, and authorities

The following generic abstractions remain rejected:

- `Cache[K,V]`, `Get/Set/Delete`, or a pluggable backend interface;
- generic `GetOrLoad`, repository decorators, or read-through/write-through
  wrappers;
- global TTL, jitter, stale-on-error, negative-cache, or fail-open defaults;
- generic key, namespace, tenant-prefix, or serializer registries;
- automatic invalidation buses or Pub/Sub-as-correctness abstractions;
- Redis transaction wrappers implying PostgreSQL/cache atomicity;
- distributed locks or distributed `singleflight` packages;
- automatic client-side caching;
- one configuration accepting standalone, Sentinel, Cluster, Redis, Valkey,
  and managed providers interchangeably;
- readiness gating merely because a dependency is configured;
- cache hit rate as the primary success or safety measure; and
- cache use for authoritative records, durable work, sessions, locks, or
  write-behind under this capability.

These are non-goals, not deferred implementation choices. Reopening any one of
them requires the evidence described below; implementation similarity alone is
insufficient.

## Success criteria and proof expectations

This specification passes while all of the following remain true:

1. The template dependency manifests, profile selectors and generation oracles,
   configuration surfaces, bootstrap lifecycle, and health/readiness wiring
   contain no Redis or Valkey capability.
2. No repository-wide cache API or feature-semantic default exists.
3. PostgreSQL remains the persisted-state authority, and cached-value semantics
   remain with a named feature rather than template infrastructure.
4. Cache-related dependency absence leaves selected and unselected template
   behavior unchanged.

Proof is bounded to current repository inspection of those surfaces and the
accepted research synthesis. It establishes absence and ownership constraints;
it does not establish performance, provider compatibility, production topology,
or a future cache implementation's correctness.

## Risks, assumptions, and reopen conditions

- No representative workload, latency objective, origin QPS or capacity,
  deployment size, tenant model, freshness budget, provider entitlement, or
  live topology was supplied. No benchmark or live-provider mutation was
  authorized or run.
- Dependency absence is current-tree evidence. Refresh it if `go.mod`,
  configuration, bootstrap, initializer/profile, or CI generation surfaces
  change after this specification.
- Official client, server, and provider documentation establishes mechanisms,
  not production fit. External production accounts establish failure families,
  not transferable performance results.
- Redis and Valkey versions, client defaults, licensing, and managed-provider
  support are drift-prone. Refresh them only after a product/provider is
  selected, before Technical Design approval, and before release.
- Research saturated the relevant candidate families, not every vendor or
  library. A new vendor within an already-rejected family does not reopen this
  disposition unless it changes an accepted decision-flip condition.

Reopen Specification only when a named feature supplies the complete
cache-disabled baseline and accepted target above, proves that the smaller
alternatives cannot satisfy it, and names the exact product/topology/operator
contract plus the complete feature-owned cache semantics. A shared
lifecycle-only pack additionally requires either one required template profile
with a named adopter and support owner whose production proof earns the support
burden, or at least two real features with the same exact product, topology,
credentials, lifecycle, resource bounds, and transport telemetry.

If that evidence arrives, the maximum starting hypothesis is one product- and
topology-specific client-lifecycle pack with explicit endpoints, TLS/auth and
secret sources; bounded pools, buffers, timeouts, and retries; partial-bootstrap
cleanup and close; sanitized transport telemetry; a bounded probe mechanism;
and native client exposure to a feature-owned adapter. This is a ceiling for a
future specification, not accepted current behavior. It owns no cached-value
semantics and no automatic readiness criticality.

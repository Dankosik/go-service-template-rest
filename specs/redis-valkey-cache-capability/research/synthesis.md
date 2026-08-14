# Optional Redis or Valkey Cache Capability Research Synthesis

Status: Research complete; ready for Specification

Independent review: PASS on 2026-08-12 with no material findings

Evidence snapshot: 2026-08-12

Repository baseline: `40e6d212799ae8677b675339929c559246536181` plus the current dirty worktree

## Research boundary

This note researches whether the template should add an optional Redis or
Valkey capability and, if so, what can safely be reused. It does not specify a
behavioral contract, choose package or configuration design, create tasks, add
a profile, select a provider, or change runtime code.

PostgreSQL remains the authoritative store. A cache is a derived and disposable
copy. The research question is therefore narrower than “which cache API should
the template expose”: it asks whether a shared runtime dependency is earned and
which mechanics, if any, can move below feature ownership without silently
choosing correctness, privacy, or degraded-service policy.

## Executive synthesis

1. **No shared cache pack is earned by the current evidence.** The repository
   has no named cached value, measured origin bottleneck, latency or capacity
   target, freshness budget, tenant-key contract, invalidation trigger, or
   operated Redis/Valkey topology. Adding a client or profile now would be
   speculative infrastructure.
2. **No generic cache framework is safe.** Keys, source revision, TTL,
   staleness, negative entries, serialization, invalidation, read-after-write,
   fallback, coalescing identity, and semantic telemetry are one feature-owned
   contract. A `Cache[K,V]`, `GetOrLoad`, generic key builder, or backend-neutral
   adapter would hide rather than solve that contract.
3. **The default remains no cache; optimize PostgreSQL first.** Measure the
   named query or computation, remove avoidable calls and rows, inspect plans,
   indexes and pool behavior, and retest. A materialized view is another
   persisted projection with explicit refresh semantics, not a free cache.
4. **Process-local caching is a separate candidate, not a smaller Redis.** It
   avoids a network dependency but creates per-process divergence, deploy-time
   cold fill, fleet-proportional origin load, and explicit process-memory and
   eviction obligations. Direct feature-owned state or a mature bounded helper
   is enough when a workload earns it.
5. **Redis and Valkey provide useful mechanisms, not cache correctness.** Their
   replication, eviction, failover, cluster routing, transactions, client-side
   invalidation, and managed-provider behavior do not make PostgreSQL and the
   cache atomic or strongly consistent.
6. **A future lifecycle-only client pack is the largest safe reusable
   boundary.** If named adopters and operations evidence flip the decision, the
   pack may own one explicit product/topology client, validated TLS and
   credentials, bounded connection and command resources, bootstrap cleanup and
   close, sanitized transport telemetry, and a probe mechanism. It must expose
   the native client to feature-owned adapters and own no value semantics.
7. **“Redis or Valkey” is not one durable compatibility promise.** Valkey began
   from Redis OSS 7.2.4, while both products and managed providers now evolve
   independently. A future profile must pin the product, client, topology,
   version range, and command subset it certifies.

The Research disposition is therefore **`constraint_only`**: retain the
repository extension seams, add no dependency or profile now, and carry a
precise lifecycle-only ceiling into Specification if later evidence reopens the
capability.

## Open-item map

| Question | Research method | Disposition | Downstream owner |
| --- | --- | --- | --- |
| Is a cache needed? | Current-state baseline, PostgreSQL alternatives, production counter-evidence | No current workload earns one. Reopen only from representative measurements and an accepted service objective. | Specification, then Performance and Data Design |
| Which cache family fits? | Candidate-family sweep across no cache, database, local, distributed, managed and feature-owned alternatives | No-cache remains preferred; the other families are conditional substitutes, not interchangeable backends. | Specification |
| What can be shared? | Repository ownership and lifecycle analysis plus client source review | At most product-specific client lifecycle and transport mechanics; not value semantics. | Specification, then Go Ownership and System Design |
| How is correctness preserved? | PostgreSQL authority, invalidation-race, replication and transaction evidence | The feature must bind every entry to authority revision and define staleness, mutation order, fencing and reconciliation. | Specification, Data Design and Test Design |
| How does failure degrade? | Local/external cache outage, cold-fill, eviction, failover and origin-overload evidence | No generic fail-open, stale-on-error or fallback rule is safe. | Specification and Reliability Design |
| How are tenants and secrets protected? | Repository secret policy, Redis/Valkey security docs and cache-key privacy analysis | Feature owns complete isolation dimensions; deployment owns network/ACL boundary; client pack may only consume explicit secret sources. | Specification, Security and Delivery Design |
| What proves production readiness? | Client, server and provider contract comparison | Deterministic adapter tests plus exact product/version/topology/provider failure proof; local compatibility is not provider certification. | Test Design and Delivery Design |

## Current repository authority

### PostgreSQL remains truth

The current architecture places feature behavior and ports in
`internal/<feature>`, concrete integrations under `internal/infra/*`, and
construction and lifecycle in bootstrap
([Repository Architecture](../../../docs/repo-architecture.md)). PostgreSQL is
the optional persisted-data owner; a new Redis/Valkey neighbor would be a
derived-copy mechanism, never a second authority or a durable write path.

The repository has no Redis, Valkey, Memcached, or in-memory value-cache runtime
dependency. The current worktree already includes
[`golang.org/x/sync/singleflight`](https://pkg.go.dev/golang.org/x/sync/singleflight)
through [`go.mod`](../../../go.mod), but `singleflight` only suppresses
overlapping calls. It stores no value, supplies no freshness, and creates no
publication fence.

PostgreSQL already provides the first optimization route: identify workload via
[`pg_stat_statements`](https://www.postgresql.org/docs/current/pgstatstatements.html),
inspect the real plan with
[`EXPLAIN`](https://www.postgresql.org/docs/current/using-explain.html), and fix
query shape, round trips and indexes before copying state. PostgreSQL
[materialized views](https://www.postgresql.org/docs/current/rules-materializedviews.html)
remain an option for a named projection, but their refresh and visibility are a
data contract, not a transparent optimization.

### Health and readiness are not cache policy

[`internal/health`](../../../internal/health/service.go) has a process-local
cached readiness verdict. It fails closed before the first evaluation and after
the configured freshness window; a bootstrap-owned refresher performs the
actual probes
([startup readiness](../../../cmd/service/internal/bootstrap/startup_readiness.go)).
That cache is owned by health semantics and is evidence for the existing probe
and lifecycle pattern, not a reusable value-cache abstraction.

A future Redis/Valkey client may expose a bounded capability probe. Whether its
failure gates service readiness depends on the consuming feature's accepted
degraded behavior and proven origin capacity. It must never gate liveness. A
cache that can be bypassed safely may be observable without making the whole
service unready; a cache required for correct service cannot be called
“optional” during failure.

### Configuration, secrets and outbound trust

Configuration is typed, rejects unknown keys, keeps non-secret defaults in YAML,
and routes secrets through explicit environment or file sources
([Configuration Source Policy](../../../docs/configuration-source-policy.md)).
Any future client configuration must therefore make endpoint, product,
topology, TLS trust, username and credential source explicit. It must not search
ambient files or silently fall through credential modes.

The endpoint set is a bounded outbound dependency. Deployment owns DNS,
network egress, private connectivity, certificates, ACLs and server-side memory
policy. Bootstrap may consume admitted endpoints; it must not accept arbitrary
feature-supplied cache hosts. Redis recommends a trusted network, ACLs and TLS
rather than public exposure
([Redis security](https://redis.io/docs/latest/operate/oss_and_stack/management/security/),
[TLS](https://redis.io/docs/latest/operate/oss_and_stack/management/security/encryption/),
[ACLs](https://redis.io/docs/latest/operate/oss_and_stack/management/security/acl/)).

### Bootstrap, telemetry and template profiles

Bootstrap already owns optional dependency construction, partial-initialization
cleanup, probe registration, bounded shutdown and close ordering
([startup dependencies](../../../cmd/service/internal/bootstrap/startup_dependencies.go),
[run](../../../cmd/service/internal/bootstrap/run.go),
[shutdown](../../../cmd/service/internal/bootstrap/shutdown.go)). A future
client would follow that lifecycle with one long-lived client per dependency;
bootstrap would not choose keys, TTL, invalidation or fallback.

Transport telemetry can safely record low-cardinality product, topology,
operation class, latency, error class, retry count, pool occupancy and
connection state. Keys, values, tenant IDs, subjects and raw server errors are
not attributes. Hit, miss, stale, negative-hit, coalesced-fill, invalidation-lag,
fallback and origin-load signals are semantic and remain with the feature
adapter.

The initializer and CI have no cache selector or cache-profile oracle
([initializer](../../../scripts/init-module.sh),
[profile checks](../../../scripts/ci/template-init-check.sh),
[CI](../../../.github/workflows/ci.yml)). Adding a selector would create a
supported product/version/topology contract and generated-file removal proof,
not merely add `go.mod` lines. No such support burden is justified today.

## Candidate map

| Candidate | Relationship | When it survives | Cost or rejection condition |
| --- | --- | --- | --- |
| No cache | Baseline substitute | Origin already meets the accepted latency and capacity objective | No current evidence rejects it |
| PostgreSQL query/index/pool optimization | Prerequisite or substitute | The repeated cost is database work and the authority can serve it directly | Requires a representative plan and workload; does not help non-database computation |
| Request memoization or `singleflight` | Narrow complement | Duplicate work occurs within one request or overlapping calls | Per-process only; no retained value, freshness or ordering |
| Process-local value cache | Substitute | Per-instance divergence, deploy cold fill and bounded process memory are acceptable | Fleet-proportional misses, local eviction and no cross-instance invalidation |
| Self-operated Redis or Valkey | Distributed substitute | Cross-instance reuse is measured and repays the dependency, failure and operations cost | Network partition, async replication, eviction, failover ambiguity, cluster operations |
| Managed Redis or Valkey | Distributed substitute with delegated server operations | The provider's exact engine, topology, availability, TLS and failover contract is accepted | Provider removes patching, not application correctness, key policy or outage-origin load |
| Managed HTTP/CDN/API cache | Protocol-edge substitute | Reuse is an HTTP response property with complete `Cache-Control`, `Vary`, auth and purge semantics | Different key/freshness model; provider overrides can leak or stale responses |
| Provider KV/cache service | Provider-specific substitute | Its consistency and locality contract matches the feature | Often eventually consistent or negatively cached; not Redis-compatible semantics |
| Feature-owned adapter over one mechanism | Required ownership shape | Any mutable cache is accepted | Some mechanics may repeat until identical real contracts prove reuse |
| Repository-wide cache framework | Rejected abstraction | Reopen only after multiple features prove the same authority, isolation, freshness and failure contract | Backend-neutral shape hides the material differences |

The family search is saturated at the decision level. Additional Redis hosts,
managed vendors, local libraries or CDN products instantiate one of these
families; they do not create a new ownership model.

## Cache behavior belongs to the cached value

Every cached value needs one explicit contract:

| Concern | Required decision | Why infrastructure cannot infer it |
| --- | --- | --- |
| Authority | PostgreSQL query/revision or other canonical source | A successful cache write does not make the copy true |
| Key | Tenant, authorization/policy, locale, representation, query and schema dimensions | Omission can return another tenant's or policy's data |
| Freshness | TTL, maximum accepted staleness, read-your-writes and bypass rules | Time is not an ordering proof and tolerance is business-specific |
| Fill | Who loads, with what deadline, and whether calls coalesce | Leader cancellation and shared errors affect other callers |
| Mutation | Commit order, invalidate/update step and ambiguous-result handling | PostgreSQL and Redis/Valkey have no shared atomic transaction |
| Concurrency | Authority revision or generation fence | An old fill can arrive after a newer commit and republish stale data |
| Negative entry | Which authoritative absence, TTL and create invalidation | Errors and authorization denial are not “not found” |
| Serialization | Schema/version, mixed-deploy readers, rollback and poison handling | Cache formats outlive one process and can cause mass refill |
| Degradation | Error, origin fallback, bounded stale value, shed or partial response | Fallback may brown out the authority or violate correctness |
| Proof | Race, partition, eviction, cold start, mixed version and cache-disabled load | Hit rate alone says nothing about correctness or safety |

### Cache-aside, read-through and write-through

**Cache-aside** is the default when application code owns the authority and the
complete cache contract. The feature reads the copy, loads from PostgreSQL on
miss, and publishes only if its revision/generation is still current. Mutation
commits PostgreSQL first, then invalidates or updates the derived copy. A failed
cache operation cannot roll back committed truth.

**Read-through** is justified only when one feature-owned adapter can centralize
fills without hiding caller-specific authorization, deadlines or error policy.
A generic `GetOrLoad` chooses too much: key completeness, loader lifetime,
shared-error behavior, negative entries and stale fallback.

**Write-through** is not a generic consistency solution. Redis transactions
serialize operations inside Redis but provide no rollback and no atomic commit
with PostgreSQL
([Redis transactions](https://redis.io/docs/latest/develop/using-commands/transactions/)).
The authority must commit first; any synchronous cache update remains a derived
side effect with explicit ambiguous/failure behavior.

**Write-behind** is rejected for this capability. It would make cache state part
of the durable write path and introduce loss, replay, reconciliation and data
authority obligations outside the intended boundary.

### TTL, versioned keys and invalidation races

TTL bounds how long an otherwise untouched entry may remain; it does not order
an older fill against a newer commit. Deleting on mutation also fails when an
in-flight old read publishes after the delete. Meta's production account shows
this stale-fill family and the need for version-aware conflict handling
([Cache made consistent](https://engineering.fb.com/2022/06/08/core-infra/cache-made-consistent/)).
Valkey documents the same client-side invalidation race and requires flushing
local state after losing the invalidation connection
([Valkey client-side caching](https://valkey.io/topics/client-side-caching/)).

Versioned namespaces prevent new code from decoding old formats, but they do
not fence data revision. A mixed deployment needs compatible readers or an
explicit bypass; deleting the whole namespace converts compatibility risk into
a cold-fill surge. Authority revision/generation belongs in the feature's
publication rule, with TTL as a backstop and reconciliation when missed
invalidation matters.

Redis Pub/Sub is at-most-once: a disconnected consumer loses invalidations
permanently
([Redis Pub/Sub delivery](https://redis.io/docs/latest/develop/pubsub/#delivery-semantics)).
It may reduce staleness but cannot be the only freshness proof. A durable log or
outbox can improve delivery, yet still needs idempotence, ordering, lag bounds
and reconciliation; those are feature/domain decisions, not cache-client work.

### Negative caching, stampedes and request coalescing

Only an authoritative “not found” may become a negative entry. Authorization
denial, timeout, decode error and dependency failure must not. The negative key
must include the same tenant and policy dimensions as a positive value, use its
own short freshness bound, and be invalidated by creation. Otherwise negative
caching makes recovery or newly created data invisible.

AWS's production guidance documents local-cache divergence, external-cache
outage fallback, cold starts, negative entries, poison formats and origin
stampedes
([Caching challenges and strategies](https://aws.amazon.com/builders-library/caching-challenges-and-strategies/)).
Jittered expiry can spread ordinary refreshes but does not protect a full cache
outage. Origin protection needs a bounded concurrency and load budget, not an
unlimited “fall back to PostgreSQL” rule.

`singleflight.Group` is available directly for same-process overlapping fills.
It is not a cache or distributed lock. The feature must provide the complete
coalescing key, give the shared fill a lifetime independent of any one waiter,
decide whether a canceled caller stops waiting, and define how shared errors
behave. A repository wrapper would add no safe default.

### Process-local memory and mature helpers

The standard-library starting point for a small fixed workload is a typed map
and ordinary locking; Go recommends that most code prefer this to specialized
[`sync.Map`](https://pkg.go.dev/sync#Map) so invariants remain visible.
Unbounded maps are not production caches. A real local cache needs an explicit
item/cost ceiling, admission and eviction behavior, oversized-entry policy,
expiry cleanup, shutdown behavior and process-memory headroom.

Mature helpers remain feature-level choices rather than template dependencies:

| Helper | Useful mechanism | Why it is not a shared policy |
| --- | --- | --- |
| [`ristretto/v2`](https://pkg.go.dev/github.com/dgraph-io/ristretto/v2) | Bounded cost and TinyLFU-style admission | Sets may be rejected/admitted asynchronously; cost and correctness expectations are workload-specific |
| [`otter/v2`](https://pkg.go.dev/github.com/maypok86/otter/v2) | Size/weight limits, TTL, loading, refresh and stats | Loader, refresh and propagation options bundle cancellation and freshness choices |
| [`theine-go`](https://pkg.go.dev/github.com/Yiling-J/theine-go) | Bounded admission, TTL, loading and persistence options | Persistence/loading modes create format and lifecycle policy |
| [`fastcache`](https://pkg.go.dev/github.com/VictoriaMetrics/fastcache) | Bounded high-throughput byte storage | Expiration and herd protection are deliberately caller-owned |

Select one only after a representative local-cache workload measures that the
stdlib shape is insufficient. No current workload reaches that rung.

## Redis and Valkey operational contract

### Product and version compatibility

Valkey forked from Redis OSS 7.2.4 and documents compatibility with Redis OSS
7.2 and earlier
([Valkey history](https://valkey.io/topics/history/),
[migration guidance](https://valkey.io/topics/migration/)). Current Redis and
Valkey releases have since added independent commands and behavior. Redis 8 is
available under AGPLv3 as well as Redis's source-available licenses
([Redis licensing](https://redis.io/legal/licenses/)); Valkey is BSD-licensed
under the Linux Foundation. Licensing and provider availability are deployment
selection inputs, not reasons to pretend current engines are interchangeable.

A compatibility claim must identify:

- server product and allowed versions;
- standalone, Sentinel or cluster topology;
- command subset and scripting/module use;
- client version and protocol mode;
- managed-provider restrictions and failover behavior;
- TLS, ACL and credential-rotation path;
- exact conformance proof.

### Replication, failover and cluster topology

Redis replication is asynchronous; `WAIT` improves acknowledgement evidence but
does not make the system strongly consistent or prevent acknowledged write loss
during failover
([Redis replication](https://redis.io/docs/latest/operate/oss_and_stack/management/replication/)).
Valkey Cluster likewise does not provide strong consistency, uses asynchronous
replication and can lose acknowledged writes
([Valkey Cluster tutorial](https://valkey.io/topics/cluster-tutorial/),
[cluster specification](https://valkey.io/topics/cluster-spec/)). Reads routed to
replicas may be stale.

Standalone, Sentinel and Cluster are different client contracts. Sentinel
requires discovery and failover handling; its notifications are hints, not a
durable event stream
([Redis Sentinel client specification](https://redis.io/docs/latest/develop/reference/sentinel-clients/)).
Cluster requires slot routing and `MOVED`/`ASK` handling and constrains
multi-key operations to compatible hash slots. An endpoint list cannot safely
hide those differences.

A command timeout after a write is ambiguous: the server may have applied the
command even though the caller did not observe the reply. Automatic retry is
safe only when the feature proves the operation is idempotent or otherwise
fenced. Cache correctness cannot depend on a remote write acknowledgement.

### Server and client memory

Redis and Valkey need explicit server `maxmemory` and eviction policy. LRU and
LFU are approximations; `noeviction` rejects writes, while eviction can remove
data or version markers at any time
([Redis eviction](https://redis.io/docs/latest/develop/reference/eviction/),
[Valkey eviction](https://valkey.io/topics/lru-cache/)). Replication and client
buffers can sit outside the dataset limit, and large commands can temporarily
overshoot it. Redis separately documents client-buffer pressure and a
`maxmemory-clients` control
([Redis client handling](https://redis.io/docs/latest/develop/reference/clients/)).

The application client also consumes memory through pools, read/write buffers,
queued commands, pipelines and decoded values. Those bounds must be included in
the service process budget and tested at the chosen client defaults; “managed”
does not remove them.

## Go client assessment

The source review establishes viable clients, not a selection:

| Client | Current evidence | Unsafe implicit policy to make explicit | Research disposition |
| --- | --- | --- | --- |
| [`redis/go-redis/v9`](https://github.com/redis/go-redis) | Official Redis Go client; standalone, Sentinel and Cluster; pooling; context; OpenTelemetry integration | Current [`Options`](https://github.com/redis/go-redis/blob/master/options.go) include independent dial/read/write timeouts, retries/backoff and a pool sized from `GOMAXPROCS`; unbounded active connections are possible unless configured. Experimental client-side caching has narrow protocol/topology constraints. | Strong Redis candidate after exact-version source review; do not wrap its command surface or enable experimental caching generically. |
| [`valkey-io/valkey-go`](https://github.com/valkey-io/valkey-go) | Native Valkey Go client; standalone, Sentinel and Cluster; automatic pipelining; client-side caching; OpenTelemetry integration | Read-only retries, pipelining/cancellation behavior, per-connection buffers, blocking pool and client-cache memory are material runtime policy. Close waits for pending work. | Strong Valkey candidate after measured pool/buffer/retry bounds; expose natively to a feature adapter. |
| [`valkey-io/valkey-glide`](https://github.com/valkey-io/valkey-glide) | Valkey-supported multi-language client family with Go support, listed in the [Valkey client matrix](https://valkey.io/clients/) | Broader runtime/build footprint and product support matrix need repository-specific lifecycle and supply-chain proof. | Live alternative, not locally preferred without a requirement its architecture uniquely satisfies. |

As of the evidence snapshot, go-redis releases show v9.22.0 and valkey-go's Go
module is at v1.0.76
([go-redis releases](https://github.com/redis/go-redis/releases),
[valkey-go module](https://pkg.go.dev/github.com/valkey-io/valkey-go)). These are
refresh-sensitive observations. Client defaults have changed between releases;
the accepted version's source and changelog must be re-read before design and
again before release.

The lifecycle-only pack must not normalize away native errors or commands. A
lowest-common-denominator wrapper would lose product capabilities while still
failing to provide cross-product consistency. Feature adapters should depend on
their own narrow port when testability or domain ownership requires one; the
infrastructure package should use the chosen client directly.

## Managed and provider caching

Managed Redis/Valkey delegates provisioning, patching and some failover work; it
does not delegate key design, cache correctness, serialization rollout, origin
overload, tenant isolation or request deadlines. Provider engine versions and
topologies differ. For example, ElastiCache currently documents distinct Redis
OSS and Valkey version support, cluster-mode choices, provider parameters and
TLS/auth rotation behavior
([engine versions](https://docs.aws.amazon.com/AmazonElastiCache/latest/dg/engine-versions.html),
[cluster modes](https://docs.aws.amazon.com/AmazonElastiCache/latest/dg/Replication.Redis-RedisCluster.html),
[configuration](https://docs.aws.amazon.com/AmazonElastiCache/latest/dg/RedisConfiguration.html),
[in-transit encryption](https://docs.aws.amazon.com/AmazonElastiCache/latest/dg/in-transit-encryption.html)).
Exact-provider certification is therefore non-substitutable.

HTTP/CDN caching is a different protocol-owned candidate. RFC 9111 makes
freshness, `Vary`, authenticated response storage and stale serving explicit
([HTTP Caching](https://www.rfc-editor.org/rfc/rfc9111.html)). The API/feature
must emit a complete cache contract and test the provider's effective key;
generic Redis mechanics cannot help. Provider KV products can differ further:
Cloudflare Workers KV documents eventually consistent reads and negatively
cached misses
([KV consistency](https://developers.cloudflare.com/kv/concepts/how-kv-works/)).

Provider-side caching is preferable only when its native protocol owns the
cached representation and the feature can prove every authorization, tenant,
variant and purge dimension. It is not a template-wide substitute for a
feature-owned adapter.

## Privacy, tenant isolation and serialization

A cache key is a security boundary. It must include every dimension that can
change the authorized representation: tenant, subject or policy generation,
locale, visibility, query, projection and schema version as applicable. A
database number or textual prefix is not sufficient tenant isolation by
itself. Service authorization remains mandatory; network ACLs and key-pattern
permissions are independent defense in depth.

Keys can leak through traces, metrics, slow logs, administrative tools and
provider consoles. Values can contain personal or regulated data and can remain
in memory, replicas, snapshots or backups after PostgreSQL changes. Each
feature must decide whether the value may be cached at all, which fields are
permitted, encryption and retention requirements, and what deletion evidence is
possible. The generic pack logs neither key nor value.

Serialized cache entries require an explicit envelope or otherwise provable
format version, bounded size, strict decode limits and a mixed-deploy/rollback
rule. An incompatible entry should be treated according to the feature's
contract, usually bypassed and replaced under a generation fence. A broad flush
is not a safe migration default because it creates a synchronized origin surge.
AWS's production guidance treats external-cache format changes with the same
caution as persistent data because poison entries can survive deployments.

## Smallest conditionally safe reusable boundary

No new implementation is justified now. If the decision-flip conditions are
met, the maximum safe shared pack is:

| Pack may own | Pack must not own |
| --- | --- |
| One explicit server product, client and topology contract | A backend-neutral `Cache` interface or Redis/Valkey interchangeability claim |
| Typed endpoint, TLS, ACL identity and explicit secret-source configuration | Feature keys, namespaces, tenant/auth dimensions or serializer |
| One long-lived client, dial/pool/buffer bounds, partial-init cleanup and close | TTL, freshness, negative caching, stale serving or read-your-writes |
| Dial and command budget ceilings nested under request/startup/shutdown budgets | Cache-aside/read-through/write-through/write-behind policy |
| Conservative retry defaults that do not retry ambiguous writes without feature opt-in | Invalidation, revision/generation fences, Pub/Sub correctness or reconciliation |
| Native command/error/capability exposure to feature-owned adapters | Distributed locks, generic coalescing, leases or a transaction illusion |
| Sanitized low-cardinality transport and pool telemetry | Keys, values, tenant attributes or semantic hit/miss/stale/fallback metrics |
| A bounded probe mechanism and last observation | Automatic readiness criticality, liveness gating or generic fail-open behavior |

This boundary follows existing repository ownership: concrete driver under
`internal/infra/<product>`, typed configuration under `internal/config`, and
construction/close in bootstrap. A feature-specific adapter and any consumer
port remain with `internal/<feature>`. These are downstream placement
constraints, not a design selection.

## Rejected generic abstractions

- `Cache[K,V]`, `Get/Set/Delete`, or pluggable backend interface;
- generic `GetOrLoad`, repository decorator, read-through or write-through
  wrapper;
- global TTL, jitter, stale-on-error, negative-cache or fail-open defaults;
- generic key, namespace, tenant-prefix or serializer registry;
- automatic invalidation bus or Pub/Sub-as-correctness abstraction;
- a Redis transaction wrapper implying PostgreSQL/cache atomicity;
- distributed lock or distributed `singleflight` package;
- automatic client-side caching;
- one configuration accepting standalone, Sentinel, Cluster, Redis, Valkey and
  managed providers interchangeably;
- readiness gating merely because the dependency was configured;
- cache hit rate as the primary success or safety measure;
- cache use for authoritative records, durable work, sessions, locks or
  write-behind under this capability.

These abstractions are rejected because their apparent reuse is the policy the
feature must own. Similar `GET` and `SET` calls do not establish shared
authority, isolation, freshness, failure or rollback semantics.

## Decision-flip conditions and downstream implications

### Reopen no-cache

Reopen the decision only when a named feature supplies a representative
cache-disabled baseline showing repeated removable work and an accepted latency
or origin-capacity target that PostgreSQL/query/computation optimization cannot
meet. The evidence must include request distribution, fleet size, origin
headroom, working-set/cardinality estimate, value sizes and a cache-outage load
case.

Choose process-local caching only when per-process divergence, deploy cold fill,
memory ceiling and fleet-amplified origin load are explicitly acceptable.

Choose Redis/Valkey only when cross-instance reuse is necessary and the feature
can state and test its authority revision, complete key, TTL/staleness,
mutation/invalidation fence, negative-entry rule, serialization rollout,
eviction behavior, outage fallback/shed bound and privacy classification. A
named product, version, topology, operator/provider, TLS/auth route and recovery
owner must exist.

Choose managed edge caching only when the API owner can prove the complete
effective HTTP cache key and freshness contract, including authorization and
tenant variants and provider overrides.

### Reopen shared lifecycle code

A lifecycle-only pack becomes plausible when either:

1. a required template profile has a named adopter and support owner whose
   production proof justifies carrying the dependency; or
2. at least two real features independently need the same exact product,
   topology, credentials, lifecycle, resource bounds and transport telemetry.

Expand beyond lifecycle only if multiple real features also prove identical
authority, key/isolation dimensions, freshness invariant, failure semantics and
verification. Similar client calls are insufficient.

### Downstream obligations if reopened

- **Specification:** name the cached value and authority, performance outcome,
  feature/deployment ownership, product/topology support and complete cache
  semantics before accepting a profile.
- **System/Data Design:** preserve PostgreSQL authority; define generation
  fencing, mutation order, reconciliation, mixed-version formats and rollback.
- **Security:** threat-model endpoint trust, ACLs, credential rotation, key/value
  exposure, tenant collision and administrative access.
- **Reliability/Performance:** budget pool/buffers/timeouts/retries, origin
  fallback, cold fill, eviction, failover, cluster reshard and memory headroom
  from representative measurements.
- **Observability:** separate transport/pool health from feature hit/miss/stale,
  invalidation lag, fallback and origin-load signals; retain bounded labels.
- **Test Design:** deterministic cache-disabled, hit/miss, stale-fill race,
  mutation, negative, mixed-version, eviction, partition, failover, cancellation,
  shutdown and tenant-isolation proofs; exact-product/provider certification is
  separate from local tests.
- **Delivery:** add a profile only with selected/absent generation oracles,
  dependency removal proof, secret wiring, network policy, server memory/eviction
  configuration, rollback and support ownership.

## Evidence limits, freshness and saturation

- No representative workload, latency objective, origin QPS/capacity, deployment
  size, tenant model, freshness budget, provider entitlement or live topology
  was supplied. No benchmark or live provider mutation was authorized or run.
- The repository baseline is the current dirty worktree at HEAD `40e6d212`;
  dependency absence is current-tree evidence and must be refreshed if `go.mod`,
  config, bootstrap or profiles change.
- Official client/server/provider documentation establishes mechanisms, not
  local production fit. AWS, Meta and Facebook production evidence establishes
  failure families, not transferable performance results. Facebook's lease
  design is useful counter-evidence against TTL-only correctness, not a template
  implementation recommendation
  ([Scaling Memcache at Facebook](https://www.usenix.org/system/files/conference/nsdi13/nsdi13-final170.pdf)).
- Redis/Valkey versions, client defaults, licensing and managed-provider support
  are drift-prone. Refresh them when a product/provider is selected, before
  Technical Design approval, and before release.
- Search stopped after queries across local/distributed caching, cache-aside and
  inline strategies, stampede/coalescing, invalidation fencing, eviction,
  privacy/tenant isolation, client libraries, cluster/failover, managed caches
  and production incidents yielded only representatives of the mapped families.

Research is ready to inform Specification, but the current no-pack disposition
is itself a constraint: Specification must not invent a cached workload merely
to justify infrastructure.

## Standalone prompt for Specification

```text
Continue with the Specification macro phase only for the optional Redis or
Valkey cache capability. Research is ready in
specs/redis-valkey-cache-capability/research/synthesis.md.

Start from the researched disposition that the current template should add no
cache dependency, profile, or generic cache abstraction because no named
workload or production objective earns one. Treat PostgreSQL as authority and
feature code as owner of keys, source revisions, freshness, TTL, invalidation,
serialization, negative entries, coalescing, tenant isolation, degraded
behavior, and semantic telemetry.

The leading conditional hypothesis is that, only if accepted adopter evidence
meets the research decision-flip conditions, the largest safe reusable
capability is a product- and topology-specific client-lifecycle pack: explicit
TLS/auth/endpoints, bounded pools/buffers/timeouts/retries, bootstrap cleanup and
close, sanitized transport telemetry, a probe mechanism, and native client
exposure to a feature-owned adapter. It must not define a Cache interface,
generic key or codec, TTL/fallback defaults, invalidation bus, distributed lock,
client-side cache, or automatic readiness criticality.

First, reconstruct the accepted behavioral outcome and test the falsifier: a
named cached value, representative cache-disabled baseline, accepted latency or
origin-capacity target, and exact product/topology/operator contract now exist
and cannot be satisfied by PostgreSQL optimization, request coalescing, or a
feature-owned local adapter. If that evidence is still absent, preserve the
no-pack disposition and do not invent a capability contract. If it is present,
write the smallest specification that fixes authority, ownership, supported
product/topology, failure semantics, privacy, lifecycle, observable outcomes,
and verification boundaries while leaving implementation and package design
open.

Read the workflow router and Specification phase owner, then the research
synthesis. Preserve every rejected generic abstraction and evidence limit.
Stop at the Specification macro-phase boundary: do not write Technical Design,
Test Design, tasks, or code. Finish with the standalone prompt for the next
authorized macro phase.
```

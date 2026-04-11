# Cache Necessity And Topology

## When To Load
Load this when the spec must decide whether cache is justified and which topology fits: no cache, local in-process cache, Redis or another distributed cache, hybrid L1/L2, or Redis client-side caching. Use it before approving a cache solely because a path is slow.

Keep the skill on runtime cache contracts. If the right answer is a new read model, denormalized projection, or source-of-truth change, record a data-architecture handoff.

## Viable Options
- No cache: fix query shape, index evidence, pagination, batching, or response contract first.
- Local in-process cache: choose for ultra-low latency and low coordination needs where per-instance divergence is acceptable and bounded.
- Distributed cache: choose for shared hit ratio, fleet-wide warmup, and cross-instance reuse when network and availability costs are acceptable.
- Hybrid L1/L2: choose only with explicit L1 and L2 TTLs, invalidation/coherence rules, and memory caps.
- Redis client-side caching: choose when local-cache latency matters and invalidation tracking, connection health, flush-on-disconnect, and server memory tradeoffs are acceptable.
- Cache-aside: default read acceleration pattern when origin remains authoritative.

## Selected And Rejected Examples
Selected example: a public catalog content path has measured launch traffic, repeated reads, and content that changes a few times per day. A distributed Redis cache is viable for marketing copy keyed by tenant, locale, catalog version, and price-rule dimensions, while stock or admin-sensitive fields stay origin-backed or split into a stronger path.

Selected example: a small immutable lookup table can use local cache with a process memory cap and a short TTL if per-instance divergence is harmless and rollout invalidation is not needed for correctness.

Selected example: Redis client-side caching can be viable for very hot keys if the spec includes invalidation tracking mode, local max TTL, memory cap, and a flush-on-lost-invalidation-connection rule.

Rejected example: caching an invoice list after one p95 screenshot with no query plan, no hit-rate hypothesis, and no row-count distribution. The no-cache option should remain selected until evidence exists.

Rejected example: whole-response caching for a response that mixes tenant-specific price rules, locale, stock counts, and admin-only fields without encoding every response-shaping dimension.

Rejected example: hybrid L1/L2 cache without a coherence rule. Hybrid adds another failure mode and should not be selected by default.

## Staleness And Failure Semantics
- Local cache implies instance divergence; the spec must define the maximum allowed divergence or reject local cache for that path.
- Distributed cache implies dependency and network failure behavior; read-acceleration paths usually fall back to origin with bounded concurrency.
- Client-side caching must flush local state when invalidation connectivity is lost; otherwise stale data can outlive the server-side contract.
- If strict consistency is required and no safe bypass exists, reject cache for that operation class.

## Acceptance Checks
- The spec states measured or required benefit: latency, DB load, cost, or availability target.
- The no-cache option is compared and rejected only with evidence.
- Chosen topology includes memory bounds, key shape, TTL/freshness class, invalidation source, and outage behavior.
- Every response-shaping dimension appears in the key or is explicitly excluded by a correctness argument.
- Local and hybrid caches include per-instance divergence and invalidation/coherence checks.
- Redis client-side caching includes tracking mode, local TTL cap, memory cap, and lost-invalidation behavior.

## Exa Source Links
- [Redis client-side caching reference](https://redis.io/docs/latest/develop/reference/client-side-caching/)
- [Redis keys and values](https://redis.io/docs/latest/develop/using-commands/keyspace)
- [Redis `EXPIRE`](https://redis.io/docs/latest/commands/expire)
- [Redis client observability](https://redis.io/docs/latest/develop/clients/observability/)

# Cache Necessity And Topology

## Behavior Change Thesis
When loaded for a "let's add cache" proposal or unclear cache topology, this file makes the model compare no-cache, local, distributed, hybrid, and client-side choices against evidence and consistency constraints instead of likely mistake `default to Redis because the path is slow`.

## When To Load
Load this when the spec must decide whether cache is justified and which topology fits: no cache, local in-process cache, Redis or another distributed cache, hybrid L1/L2, or Redis client-side caching.

Keep the skill on runtime cache contracts. If the right answer is a new read model, denormalized projection, or source-of-truth change, record a data-architecture handoff.

## Decision Rubric
- Keep no-cache selected until the spec has material benefit evidence: latency target, DB load, cost, availability pressure, hit-rate hypothesis, and origin-query evidence.
- Choose local in-process cache only when per-instance divergence is harmless and bounded by TTL, rollout behavior, and memory cap.
- Choose distributed cache when shared hit ratio, fleet-wide warmup, or cross-instance reuse justifies network and dependency costs.
- Choose hybrid L1/L2 only with separate TTLs, coherence/invalidation rules, memory caps, and lost-invalidation behavior.
- Choose client-side caching only with tracking mode, local max TTL, memory cap, connection-health behavior, and flush-on-disconnect rules.
- Default to cache-aside when origin remains authoritative; require explicit reasons for other patterns.

## Imitate
- Public catalog content: measured repeated reads, content changes a few times per day, Redis viable for marketing copy keyed by tenant, locale, catalog version, and price-rule dimensions. Copy the habit of keeping stock and admin-sensitive fields origin-backed or stronger.
- Small immutable lookup table: local cache with memory cap and short TTL because per-instance divergence is harmless. Copy only when rollout invalidation is not correctness-critical.
- Very hot keys with Redis client-side caching: tracking mode, local TTL cap, memory cap, and flush on lost invalidation connection. Copy the habit of naming the extra coherence contract.

## Reject
- Invoice-list cache after one p95 screenshot with no query plan, hit-rate hypothesis, or row-count distribution. Reject because no-cache still has not lost.
- Whole-response cache that mixes tenant-specific price rules, locale, stock counts, and admin-only fields without encoding every response-shaping dimension. Reject because key correctness is unproven.
- Hybrid L1/L2 cache without a coherence rule. Reject because hybrid adds a second stale-data mode.

## Agent Traps
- Do not approve cache solely because the path is slow; first ask whether query shape, pagination, batching, response contract, or data architecture is the right seam.
- Do not treat local cache as a cheaper Redis; instance divergence is a product-visible behavior when freshness matters.
- Do not make cache the source of truth by accident through fail-closed semantics or stale serve.

## Validation Shape
- Local cache implies instance divergence; the spec must define the maximum allowed divergence or reject local cache for that path.
- Distributed cache implies dependency and network failure behavior; read-acceleration paths usually fall back to origin with bounded concurrency.
- Client-side caching must flush local state when invalidation connectivity is lost; otherwise stale data can outlive the server-side contract.
- If strict consistency is required and no safe bypass exists, reject cache for that operation class.
- The spec states measured or required benefit: latency, DB load, cost, or availability target.
- The no-cache option is compared and rejected only with evidence.
- Chosen topology includes memory bounds, key shape, TTL/freshness class, invalidation source, and outage behavior.
- Every response-shaping dimension appears in the key or is explicitly excluded by a correctness argument.
- Local and hybrid caches include per-instance divergence and invalidation/coherence checks.
- Redis client-side caching includes tracking mode, local TTL cap, memory cap, and lost-invalidation behavior.

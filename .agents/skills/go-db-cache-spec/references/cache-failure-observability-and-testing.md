# Cache Failure Observability And Testing

## Behavior Change Thesis
When loaded for cache outage, degraded mode, origin protection, telemetry, or proof obligations, this file makes the model specify bounded fallback and low-cardinality evidence instead of likely mistake `say "fall back to DB" and test only cache hits`.

## When To Load
Load this when the spec must define cache outage behavior, degraded mode, origin protection, DB/cache telemetry, low-cardinality metric labels, trace attributes, or test obligations.

Keep this focused on runtime DB/cache contracts. If the work is a line-level implementation review, use the relevant review skill instead.

## Decision Rubric
- Default read-acceleration cache errors to fail-open within the remaining request budget, with origin protection.
- Use fail-closed only when cache is part of observable semantics, such as an approved rate-limit or session contract; name the user-visible failure.
- Allow stale serve only for named operation classes inside a declared stale window.
- Include a fast bypass or runtime-disable path when rollout or outage risk is hard to reverse.
- Protect the origin with request coalescing, fallback concurrency caps, backoff, or degraded responses during cache outage.
- Keep telemetry low-cardinality: grouped cache operation, outcome, freshness class, dependency status, and stable query/cache groups, never raw keys or identifiers.

## Imitate
- Catalog read Redis timeout: record `timeout`, treat as miss, fall back to PostgreSQL behind request coalescing and a fallback cap; stale response allowed only for the public catalog class when origin capacity is exhausted. Copy the bounded fail-open shape.
- Decode failure: increment grouped decode-failure metric, evict or ignore the bad entry, and fall back to origin. Copy the habit of not poisoning the response or retrying Redis indefinitely.
- Telemetry labels: use `cache_group`, `operation`, `outcome`, and `freshness_class`. Copy the stable grouping, not the exact label names.

## Reject
- Metrics labeled by raw cache key, user ID, request ID, SQL literal, or unbounded tenant ID. Reject because observability becomes cardinality and privacy risk.
- Redis outage sending every request to PostgreSQL concurrently with no coalescing or fallback cap. Reject because fail-open becomes an origin outage amplifier.
- Test plan that covers only cache hits. Reject because the contract is mostly defined by miss, error, timeout, bypass, stale, negative-cache, and stampede-control behavior.

## Agent Traps
- Do not use "fall back to DB" without bounding concurrency and remaining deadline.
- Do not turn fail-closed into the default just because cache errors are easier to observe; it changes user-visible semantics.
- Do not report cache metrics without origin outcome when both cache and origin are attempted.

## Validation Shape
- Fail-open does not mean unbounded origin load; it requires containment.
- Stale serve is allowed only for the named operation classes and only inside the stated window.
- Cache telemetry failures must not hide origin failures; record both cache outcome and origin outcome when both are attempted.
- Raw keys and sensitive values do not belong in metric labels or logs. Use grouped key classes or stable operation names.
- Redis client-side caching must flush local entries on lost invalidation connectivity to avoid stale data escaping the contract.
- The spec defines cache hit, miss, timeout, error, decode failure, bypass, stale, and origin failure behavior for every affected path.
- Origin protection exists for hot or expensive keys: coalescing, fallback cap, retry/backoff, or degraded response.
- Telemetry includes DB latency/error/pool pressure and cache hit/miss/error/timeout/bypass/stale/fallback signals.
- Metrics and spans use low-cardinality labels and redact or avoid raw SQL literals, raw cache keys, user IDs, and secrets.
- Tests cover cache-available and cache-degraded modes, including timeout, stale-window boundary, invalidation lag, negative caching, and stampede control where applicable.
- Rollout includes a quick bypass or disable mechanism when cache behavior is risky or hard to reverse.

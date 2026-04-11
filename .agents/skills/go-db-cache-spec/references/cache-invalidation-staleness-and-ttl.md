# Cache Invalidation Staleness And TTL

## Behavior Change Thesis
When loaded for cache freshness, TTL, invalidation, stale-while-revalidate, negative caching, or key transition ambiguity, this file makes the model choose an operation-level freshness and invalidation contract instead of likely mistake `treat TTL as proof of correctness or rely on broad key scans`.

## When To Load
Load this when the spec must define freshness classes, TTL, jitter, write-triggered invalidation, event-driven invalidation, stale-while-revalidate, negative caching, key versioning, or cache-key transitions.

Stay out of primary schema ownership. If the invalidation answer requires a durable outbox table, projection rebuild, or cross-service event contract, write the DB/cache requirement and hand off the detailed data/distributed design.

## Decision Rubric
- Name the freshness class per operation: strong, bounded stale, stale-while-revalidate, or best-effort.
- Use TTL-only only when bounded staleness is product-acceptable and exact invalidation is not required.
- Use write-triggered invalidation only when the writer owns all affected keys or can target them deterministically.
- Use durable event-driven invalidation with idempotent consumers, replay, lag observability, and lagged-reader behavior when missed events would violate the contract.
- Allow ephemeral invalidation signals only with bounded-stale semantics, loss detection or resync behavior, max TTL, and reader behavior while signals are delayed or lost.
- Use versioned namespaces for rollout or broad invalidation when wildcard key scans would otherwise be tempting; include cleanup so old versions do not leak memory.
- Use stale-while-revalidate only with fresh TTL, stale window, refresh owner, coalescing, eligible consumers, and refresh-failure behavior.
- Use negative caching for expensive repeated business misses, or a separate short-lived degraded error cache when protecting an unavailable origin; never mix transient dependency failures with business negatives.

## Imitate
- Public profile read: cache-aside with `fresh_ttl=30s`, tenant/user/version key dimensions, and origin fallback; admin permission reads bypass cache or require a fresh permission version. Copy the split between public bounded-stale and admin strong paths.
- Catalog copy: admin writer triggers targeted invalidation plus TTL fallback, with exact key dimensions or versioned namespace. Copy the habit of avoiding production wildcard scans.
- Public catalog stale-while-revalidate: define fresh TTL, stale window, request coalescing, stale-eligible consumers, and refresh-failure behavior. Copy only when stale serving is product-acceptable.

## Reject
- Redis expired keyspace notifications as the exact moment a user-visible freshness window ends. Reject because expiry events are not exact wall-clock freshness proof.
- TTL-only for permissions or API key rotation when admin changes must reflect immediately. Reject because bounded stale violates the operation class.
- Negative caching dependency failures as if the resource truly does not exist. Reject because transient origin failure is not a business negative.

## Agent Traps
- Do not use TTL as a substitute for invalidation when the contract requires immediate or strong behavior.
- Do not treat Pub/Sub or keyspace notifications as durable invalidation without TTL, resync, or bounded-stale acceptance.
- Do not use Redis Cluster keyspace notifications without naming per-node subscription coverage or a resync path.
- Do not invalidate with `KEYS` or broad runtime wildcard scans from production request paths; if `SCAN` is used off-request for maintenance, require idempotent effects and tolerate duplicates or keys changing during iteration.
- Do not forget tenant, authorization, locale, version, price rule, and other response-shaping dimensions in the key.

## Validation Shape
- State the freshness class per operation: strong, bounded stale, stale-while-revalidate, or best-effort.
- TTL is a maximum cache age, not proof that invalidation happened at a specific wall-clock moment.
- Event-driven invalidation must define lag/loss behavior, duplicate handling, and either replay/durability or ephemeral-signal containment.
- Versioned keys should include rollout and cleanup behavior so old versions do not become unbounded memory leaks.
- Decode failure normally behaves like a miss plus an error counter; do not serve undecodable data.
- Every cacheable operation has a named staleness window or explicit strong-consistency bypass.
- Every entry has TTL or equivalent bounded freshness; large synchronized groups use jitter.
- Invalidation source is named: TTL-only, write-triggered, event-driven, versioned namespace, or manual purge.
- Key shape includes tenant, scope, version, locale, authorization, price rule, and other response-shaping dimensions as applicable.
- No production request path depends on `KEYS` or broad runtime wildcard scans.
- Any accepted `SCAN` maintenance path is bounded, idempotent, and not treated as exact live key inventory.
- Stale-while-revalidate has fresh TTL, stale window, refresh owner, concurrency bound, and failure behavior.
- Negative caching has a short TTL and does not cache dependency errors as business negatives.

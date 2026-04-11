# Cache Invalidation Staleness And TTL

## When To Load
Load this when the spec must define freshness classes, TTL, jitter, write-triggered invalidation, event-driven invalidation, stale-while-revalidate, negative caching, key versioning, or cache-key transitions. Use it before coding any cache path whose correctness depends on invalidation timing.

Stay out of primary schema ownership. If the invalidation answer requires a durable outbox table, projection rebuild, or cross-service event contract, write the DB/cache requirement and hand off the detailed data/distributed design.

## Viable Options
- TTL-only: viable when bounded staleness is acceptable and write volume or ownership makes exact invalidation unnecessary.
- Write-triggered delete/update: viable when the writer owns all affected keys and can target them deterministically.
- Event-driven invalidation: viable when a durable publication mechanism, idempotent consumer, replay, and lag observability are in scope.
- Versioned key namespace: viable during rollout or when broad invalidation by prefix would otherwise require unsafe key scans.
- Stale-while-revalidate: viable when serving stale data is product-acceptable and the spec names fresh TTL, stale window, refresh owner, and concurrency guard.
- Negative caching: viable for expensive repeated misses, with a short TTL and separate handling for business negatives vs dependency failures.

## Selected And Rejected Examples
Selected example: public profile reads may be stale for up to 30 seconds, so use cache-aside with `fresh_ttl=30s`, tenant/user/version key dimensions, and origin fallback on miss. Admin permission reads bypass cache or require a fresh permission version.

Selected example: catalog copy can use write-triggered invalidation from the admin writer plus TTL fallback. The spec names exact key dimensions and either targeted keys or a versioned namespace; it does not depend on production wildcard key scans.

Selected example: stale-while-revalidate for expensive public catalog pages defines `fresh_ttl`, `stale_window`, request coalescing, who can receive stale responses, and what happens when refresh fails.

Rejected example: relying on Redis expired keyspace notifications as the exact moment a user-visible freshness window ends. Redis documents expired events as generated when the key is deleted, not exactly when TTL reaches zero.

Rejected example: TTL-only for permissions or API key rotation when the admin UI must reflect changes immediately.

Rejected example: negative caching dependency failures as if the user or resource truly does not exist. Cache business negatives separately from transient origin failures.

## Staleness And Failure Semantics
- State the freshness class per operation: strong, bounded stale, stale-while-revalidate, or best-effort.
- TTL is a maximum cache age, not proof that invalidation happened at a specific wall-clock moment.
- Event-driven invalidation must define lag behavior, replay, duplicate handling, and what readers do while invalidation is delayed.
- Versioned keys should include rollout and cleanup behavior so old versions do not become unbounded memory leaks.
- Decode failure normally behaves like a miss plus an error counter; do not serve undecodable data.

## Acceptance Checks
- Every cacheable operation has a named staleness window or explicit strong-consistency bypass.
- Every entry has TTL or equivalent bounded freshness; large synchronized groups use jitter.
- Invalidation source is named: TTL-only, write-triggered, event-driven, versioned namespace, or manual purge.
- Key shape includes tenant, scope, version, locale, authorization, price rule, and other response-shaping dimensions as applicable.
- No production request path depends on `KEYS` or broad runtime wildcard scans.
- Stale-while-revalidate has fresh TTL, stale window, refresh owner, concurrency bound, and failure behavior.
- Negative caching has a short TTL and does not cache dependency errors as business negatives.

## Exa Source Links
- [Redis `EXPIRE`](https://redis.io/docs/latest/commands/expire)
- [Redis keys and values, including key expiration and `SCAN`/`KEYS`](https://redis.io/docs/latest/develop/using-commands/keyspace)
- [Redis keyspace notifications](https://redis.io/docs/latest/develop/pubsub/keyspace-notifications/)
- [Redis client-side caching reference](https://redis.io/docs/latest/develop/reference/client-side-caching/)

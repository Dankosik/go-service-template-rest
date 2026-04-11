# Cache Failure Observability And Testing

## When To Load
Load this when the spec must define cache outage behavior, degraded mode, origin protection, DB/cache telemetry, low-cardinality metric labels, trace attributes, or test obligations. Use it before coding paths where cache availability, telemetry, or proof quality affects correctness.

Keep this focused on runtime DB/cache contracts. If the work is a line-level implementation review, use the relevant review skill instead.

## Viable Options
- Fail-open read acceleration: on cache miss, timeout, error, or decode failure, fall back to origin within budget and record the cache outcome.
- Fail-closed cache dependency: use only when the cache is part of the observable semantics, such as a separately approved rate-limit/session contract, and name the user-visible failure.
- Bounded stale serve: allow stale data during cache/origin trouble only for operation classes with a declared stale window.
- Fast bypass switch: include a runtime-disable or config path when cache rollout or outage needs quick deactivation.
- Origin protection: use request coalescing, bounded fallback concurrency, backoff, or degraded responses to avoid stampedes during cache outage.
- Low-cardinality telemetry: capture DB/cache operation, result class, freshness class, and dependency status without raw keys or identifiers.

## Selected And Rejected Examples
Selected example: Redis timeout on a catalog read is recorded as `timeout`, treated like a miss, and falls back to PostgreSQL behind request coalescing and a fallback concurrency cap. If origin capacity is exhausted, the spec can allow a bounded stale response only for the public catalog class.

Selected example: a cache decode failure increments a low-cardinality decode-failure metric, evicts or ignores the bad entry, and falls back to origin. It does not poison the response or retry Redis indefinitely.

Selected example: observability uses stable labels such as `cache_group`, `operation`, `outcome`, and `freshness_class`; tracing follows database and Redis semantic conventions where available.

Rejected example: metrics labeled by raw cache key, user ID, request ID, SQL literal, or tenant ID when tenant count is unbounded.

Rejected example: Redis outage causes every request to hit PostgreSQL concurrently with no coalescing or fallback cap.

Rejected example: a test plan that covers only cache hits. Cache specs need miss, error, timeout, bypass, stale, negative-cache, and stampede-control cases where those behaviors exist.

## Staleness And Failure Semantics
- Fail-open does not mean unbounded origin load; it requires containment.
- Stale serve is allowed only for the named operation classes and only inside the stated window.
- Cache telemetry failures must not hide origin failures; record both cache outcome and origin outcome when both are attempted.
- Raw keys and sensitive values do not belong in metric labels or logs. Use grouped key classes or stable operation names.
- Redis client-side caching must flush local entries on lost invalidation connectivity to avoid stale data escaping the contract.

## Acceptance Checks
- The spec defines cache hit, miss, timeout, error, decode failure, bypass, stale, and origin failure behavior for every affected path.
- Origin protection exists for hot or expensive keys: coalescing, fallback cap, retry/backoff, or degraded response.
- Telemetry includes DB latency/error/pool pressure and cache hit/miss/error/timeout/bypass/stale/fallback signals.
- Metrics and spans use low-cardinality labels and redact or avoid raw SQL literals, raw cache keys, user IDs, and secrets.
- Tests cover cache-available and cache-degraded modes, including timeout, stale-window boundary, invalidation lag, negative caching, and stampede control where applicable.
- Rollout includes a quick bypass or disable mechanism when cache behavior is risky or hard to reverse.

## Exa Source Links
- [Redis client observability](https://redis.io/docs/latest/develop/clients/observability/)
- [OpenTelemetry database client span semantic conventions](https://opentelemetry.io/docs/specs/semconv/database/database-spans/)
- [OpenTelemetry Redis semantic conventions](https://opentelemetry.io/docs/specs/semconv/db/redis/)
- [Redis client-side caching reference](https://redis.io/docs/latest/develop/reference/client-side-caching/)
- [Go: Managing connections](https://go.dev/doc/database/manage-connections)

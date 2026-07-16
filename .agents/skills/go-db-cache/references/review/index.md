# Reference Selector

Choose the reference whose examples change the local finding.

| Symptom | Load | Distinction preserved |
| --- | --- | --- |
| Dynamic SQL, binding, query loops, one-row/result handling, or cursor cleanup is primary. | [sql-query-and-resource-safety-review.md](sql-query-and-resource-safety-review.md) | Bind/allowlist/batch/close locally; do not switch drivers or redesign schema. |
| Transaction scope, split writes, retries, isolation, commit handling, or cache work around commit changed. | [transaction-boundary-review.md](transaction-boundary-review.md) | Restore atomic DB scope and post-commit cache order; do not invent distributed policy. |
| Caller context, timeout, cancel, statement/connection ownership, rows, or transaction cleanup is primary. | [context-timeout-and-rows-cleanup.md](context-timeout-and-rows-cleanup.md) | Preserve caller-derived cancellation and explicit cleanup; do not invent budgets. |
| Key dimensions, deterministic material, tenant/auth scope, payload version, or corrupt decode changed. | [cache-key-isolation-and-serialization.md](cache-key-isolation-and-serialization.md) | Restore isolation and safe serialization rather than hashing incomplete keys. |
| Write invalidation, TTL, negative caching, stale serving, cache-aside freshness, or overwrite changed. | [invalidation-ttl-and-staleness-review.md](invalidation-ttl-and-staleness-review.md) | Name exact freshness ownership; avoid TTL handwaving, wildcard deletes, and cached transient failures. |
| Hot misses, cache outage, `singleflight`, locks, stale fallback, or origin load changed. | [stampede-fallback-and-origin-protection.md](stampede-fallback-and-origin-protection.md) | Require bounded fallback and correctly scoped protection without overstating local coordination. |

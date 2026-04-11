# Stampede Fallback And Origin Protection

## Behavior Change Thesis
When loaded for hot miss, cache outage, `singleflight`, or Redis lock changes, this file makes the model choose bounded origin protection and correctly scoped coalescing/locking instead of likely mistakes such as treating every cache error as a miss, collapsing the wrong key dimensions, or pretending process-local coalescing is distributed protection.

## When To Load
Load this reference when a Go diff changes cache miss handling, fallback on cache outage, Redis lock use, `singleflight`, stale serving, local/in-process cache refill, or origin DB protection.

Keep findings local: identify the changed miss/fallback path that can multiply origin load or serve the wrong contract. Escalate new stale-while-revalidate policy, retry budgets, overload behavior, or distributed-lock correctness requirements.

## Decision Rubric
- Many concurrent misses for the same key can all call the DB/origin.
- `singleflight` key is broader or narrower than the cache key and can collapse unrelated requests or fail to collapse identical ones.
- Redis lock is acquired without `NX` plus expiry, or released with an unsafe plain `DEL` after the lock can expire.
- Lock TTL can expire before the protected work finishes without any accepted stale/duplicate-work policy.
- Cache outage falls back directly to DB for every request with no bound, coalescing, or stale option.
- A fallback serves stale data without an explicit stale window or product/API contract.
- Cache set failure after origin fetch causes repeated expensive recomputation without logging or suppression.

## Bad Example: Uncoalesced Cache Miss

```go
func (s *Store) CachedReport(ctx context.Context, reportID string) (Report, error) {
	key := "report:" + reportID
	if b, err := s.cache.Get(ctx, key).Bytes(); err == nil {
		return decodeReport(b)
	}

	report, err := s.repo.Report(ctx, reportID)
	if err != nil {
		return Report{}, err
	}
	b, err := json.Marshal(report)
	if err != nil {
		return Report{}, err
	}
	_ = s.cache.Set(ctx, key, b, time.Minute).Err()
	return report, nil
}
```

Review finding shape:

```text
[medium] [go-db-cache-review] store/report_cache.go:64
Issue: The changed hot report cache-aside path lets every concurrent miss query the origin for the same key.
Impact: When a popular key expires or Redis is cold, identical requests can stampede the DB and exhaust the pool.
Suggested fix: Coalesce the miss path with the repository's existing singleflight/lock helper using the same dimensions as the cache key, and keep fallback behavior bounded.
Reference: Redis stampede guidance and Go singleflight duplicate-call suppression.
```

## Good Example: Local Coalescing With Matching Key

```go
func (s *Store) CachedReport(ctx context.Context, reportID string) (Report, error) {
	key := "report:" + reportID
	if b, err := s.cache.Get(ctx, key).Bytes(); err == nil {
		return decodeReport(b)
	}

	v, err, _ := s.reportGroup.Do(key, func() (any, error) {
		if b, err := s.cache.Get(ctx, key).Bytes(); err == nil {
			return decodeReport(b)
		}

		report, err := s.repo.Report(ctx, reportID)
		if err != nil {
			return Report{}, err
		}
		b, err := json.Marshal(report)
		if err != nil {
			return Report{}, err
		}
		if err := s.cache.Set(ctx, key, b, time.Minute).Err(); err != nil {
			return Report{}, err
		}
		return report, nil
	})
	if err != nil {
		return Report{}, err
	}
	return v.(Report), nil
}
```

This is a local-process protection. If the stampede is cross-process and correctness matters, escalate to distributed coordination or reliability design instead of pretending `singleflight` is enough.

## Bad Example: Unsafe Redis Lock Release

```go
func (s *Store) refresh(ctx context.Context, key string) error {
	ok, err := s.redis.SetNX(ctx, "lock:"+key, "1", 30*time.Second).Result()
	if err != nil || !ok {
		return err
	}
	defer s.redis.Del(ctx, "lock:"+key)

	return s.rebuild(ctx, key)
}
```

If `rebuild` runs longer than the lock TTL, another worker can acquire the lock; the deferred `DEL` can then remove the other worker's lock.

## Good Example: Tokened Lock Or Existing Helper

```go
func (s *Store) refresh(ctx context.Context, key string) error {
	lockKey := "lock:" + key
	token := newLockToken()
	ok, err := s.redis.SetNX(ctx, lockKey, token, 30*time.Second).Result()
	if err != nil || !ok {
		return err
	}
	defer releaseLockIfTokenMatches(ctx, s.redis, lockKey, token)

	return s.rebuild(ctx, key)
}
```

Prefer the repository's existing lock helper. If the correct fix needs fencing tokens, multi-node Redlock analysis, or long-running lease extension, escalate to reliability/distributed design.

## Bad Example: Unbounded Cache-Outage Fallback

```go
func (s *Store) FeatureConfig(ctx context.Context, tenantID string) (Config, error) {
	key := "feature-config:" + tenantID
	b, err := s.cache.Get(ctx, key).Bytes()
	if err == nil {
		return decodeConfig(b)
	}
	return s.repo.FeatureConfig(ctx, tenantID)
}
```

This treats every Redis error as a miss. During a cache outage, all requests hit the DB.

## Good Example: Distinguish Miss From Cache Failure

```go
func (s *Store) FeatureConfig(ctx context.Context, tenantID string) (Config, error) {
	key := "feature-config:" + tenantID
	b, err := s.cache.Get(ctx, key).Bytes()
	if err == nil {
		return decodeConfig(b)
	}
	if err != redis.Nil {
		return Config{}, fmt.Errorf("read feature config cache: %w", err)
	}
	return s.loadFeatureConfigMiss(ctx, key, tenantID)
}
```

If product requirements say Redis failure should fall back to DB, require a bound such as coalescing, rate limiting, stale data with a named window, or an accepted reliability-risk handoff.

## Agent Traps
- Do not treat `singleflight` as cross-process protection. It only coalesces work inside the current process unless the repository wraps it with distributed behavior.
- Do not use a coalescing key that is missing tenant, version, locale, auth, or feature dimensions from the cache key.
- Do not release Redis locks with a plain `DEL` when the lock can expire and be reacquired by another owner; require token-checked release or an existing safe helper.
- Do not treat every Redis error as `redis.Nil`; cache failure and cache miss have different origin-protection consequences.
- Do not invent distributed locking, stale-while-revalidate, or retry budgets in a review finding when the local code only needs same-process coalescing or a bounded fallback handoff.

## Smallest Safe Fix
- Add or reuse `singleflight` for same-process hot miss coalescing when the repository already accepts it.
- Use the cache key, including tenant/version dimensions, as the coalescing key.
- Recheck cache inside the coalesced function to avoid duplicate origin fetch after another goroutine fills it.
- For Redis locks, require `SET NX` with an expiry and token-checked release.
- Separate cache miss from cache failure; do not treat every cache error as unlimited origin fallback.
- Escalate when the right fix requires stale serving, distributed locks, fencing tokens, retry budgets, or overload policy.

## Validation Shape
- Add a concurrency test that launches many identical misses and asserts one origin call.
- Add a test that different tenant/version keys do not share a `singleflight` result.
- Add a fake Redis failure test and assert fallback is bounded or returns the intended error.
- Add a lock-release test showing an expired/reacquired lock is not deleted by the old holder.
- Run race tests for shared cache wrapper state.

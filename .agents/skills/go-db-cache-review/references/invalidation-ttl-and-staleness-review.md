# Invalidation TTL And Staleness Review

## When To Load
Load this reference when a Go diff changes cache write-through/cache-aside logic, `Set` TTLs, `Del` invalidation, negative caching, client-side cache invalidation, cache freshness comments, or stale fallback behavior.

Keep findings local: ask for the smallest correction that restores the existing freshness contract. Escalate API-visible stale windows, new read-your-writes expectations, outbox/event invalidation, or broader consistency policy.

## Review Smell Patterns
- A write path updates the database but does not invalidate or update the exact cache keys it can make stale.
- Invalidation happens before the DB transaction commits.
- `SET` overwrites a Redis key without preserving or replacing its TTL.
- TTL is used as the only freshness mechanism for data that must be invalidated on writes.
- Negative cache stores transient dependency failures as "not found."
- Cache-aside write path ignores marshal or cache set errors without matching the existing fallback contract.
- Client-side cache is introduced without invalidation or max TTL protection.
- Wildcard key deletion is used on a request path.

## Bad Example: DB Write Leaves Cache Stale

```go
func (s *Store) RenameProject(ctx context.Context, tenantID, projectID, name string) error {
	_, err := s.db.ExecContext(ctx, `
		update projects set name = $1
		where tenant_id = $2 and id = $3`, name, tenantID, projectID)
	return err
}
```

Review finding shape:

```text
[high] [go-db-cache-review] store/projects.go:73
Issue: The changed project write updates the source of truth but does not invalidate the tenant-scoped project cache key used by the read path.
Impact: Readers can keep seeing the old project name until TTL expiry, which violates the existing write-driven freshness pattern in this package.
Suggested fix: After a successful commit/update, delete or refresh the exact project cache key using the same key builder as reads.
Reference: Redis cache-aside requires the application to manage freshness; Redis TTL removes keys only after the configured lifetime.
```

## Good Example: Exact Key Invalidation After The DB Write

```go
func (s *Store) RenameProject(ctx context.Context, tenantID, projectID, name string) error {
	if _, err := s.db.ExecContext(ctx, `
		update projects set name = $1
		where tenant_id = $2 and id = $3`, name, tenantID, projectID); err != nil {
		return err
	}

	key := projectKey(tenantID, projectID)
	if err := s.cache.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("invalidate project cache: %w", err)
	}
	return nil
}
```

If the write is inside a transaction, move invalidation after `Commit`. If the current code treats cache invalidation as best-effort, preserve that policy and make the failure observable if local patterns do so.

## Bad Example: Overwrite Drops TTL

```go
func (s *Cache) TouchProject(ctx context.Context, key string, value []byte) error {
	return s.redis.Set(ctx, key, value, 0).Err()
}
```

Redis `SET` overwrites the old value and discards previous TTL unless the command/client uses a TTL-preserving option. This can turn a bounded cache entry into a persistent one.

## Good Example: Explicit TTL On Write

```go
func (s *Cache) TouchProject(ctx context.Context, key string, value []byte) error {
	return s.redis.Set(ctx, key, value, 10*time.Minute).Err()
}
```

Use a repository constant or freshness owner instead of inventing a TTL in review.

## Bad Example: Negative Cache Stores Transient Error

```go
func (s *Store) Product(ctx context.Context, id string) (Product, error) {
	product, err := s.repo.Product(ctx, id)
	if err != nil {
		_ = s.cache.Set(ctx, "product:"+id+":missing", "1", time.Hour).Err()
		return Product{}, err
	}
	return product, nil
}
```

## Good Example: Negative Cache Only For Authoritative Miss

```go
func (s *Store) Product(ctx context.Context, id string) (Product, error) {
	product, err := s.repo.Product(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		_ = s.cache.Set(ctx, "product:"+id+":missing", "1", time.Minute).Err()
		return Product{}, ErrNotFound
	}
	if err != nil {
		return Product{}, err
	}
	return product, nil
}
```

The local finding is that transient source errors must not be converted into cached business truth. If the API contract for negative caching is unclear, escalate.

## Smallest Safe Fix
- Delete or refresh exact keys after successful writes when the current package already owns those keys.
- Move invalidation after successful commit for transactional writes.
- Add or preserve TTL on every cache write unless the contract explicitly says persistent.
- Limit negative caching to authoritative misses and use a short, explicit TTL.
- Replace wildcard request-path invalidation with exact key builders, tags/sets already present in the codebase, or a design escalation.
- When stale serving is intentional, require a named stale window and validation path rather than silent fallback.

## Validation Ideas
- Add read-after-write tests that prime cache, update DB, then verify the next read does not return stale cached data.
- Add a TTL assertion using Redis `TTL`/client equivalent after cache writes.
- Add tests distinguishing `sql.ErrNoRows` from transient DB/cache errors in negative-cache paths.
- Add a transaction test proving cache invalidation does not run if the transaction rolls back.

## Source Links From Exa
- Redis `EXPIRE` command and TTL behavior: https://redis.io/docs/latest/commands/expire/
- Redis `SET` command and expiration options: https://redis.io/docs/latest/commands/set/
- Redis keys and key expiration overview: https://redis.io/docs/latest/develop/using-commands/keyspace/
- Redis cache-aside query caching tutorial: https://redis.io/learn/howtos/solutions/microservices/caching
- Redis client-side caching invalidation reference: https://redis.io/docs/latest/develop/reference/client-side-caching/

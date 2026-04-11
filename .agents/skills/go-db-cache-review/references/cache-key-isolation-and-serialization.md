# Cache Key Isolation And Serialization

## When To Load
Load this reference when a Go diff changes cache key construction, cached value serialization, decode behavior, tenant/auth/locale/feature scoping, cache schema versions, or query-response caching.

Keep findings local: identify the missing key dimension or unsafe decode path in the changed code. Escalate security policy, API-visible consistency, or data ownership changes instead of inventing broad cache contracts here.

## Review Smell Patterns
- Cache key omits tenant, organization, auth scope, locale, feature flag, role, or response version when those inputs change the value.
- Cache key uses raw user-provided strings without a stable delimiter, escaping, or hashing strategy.
- Different resources share a prefix and can collide, such as `user:` for both profile and permissions.
- Key includes a JSON-serialized map or option object without a stable canonicalization policy.
- Cached payload has no schema/version marker while decode code assumes a current struct.
- Decode errors are treated as cache misses without invalidation or logging, causing repeated corrupt reads.
- Negative cache stores dependency failures as "not found."

## Bad Example: Cross-Tenant Key Collision

```go
func (s *Store) CachedProfile(ctx context.Context, tenantID, userID string) (Profile, error) {
	key := "profile:" + userID
	if b, err := s.cache.Get(ctx, key).Bytes(); err == nil {
		var profile Profile
		if json.Unmarshal(b, &profile) == nil {
			return profile, nil
		}
	}

	profile, err := s.repo.Profile(ctx, tenantID, userID)
	if err != nil {
		return Profile{}, err
	}
	b, _ := json.Marshal(profile)
	_ = s.cache.Set(ctx, key, b, 10*time.Minute).Err()
	return profile, nil
}
```

Review finding shape:

```text
[critical] [go-db-cache-review] store/profile_cache.go:21
Issue: The changed profile cache key uses only userID even though the lookup is tenant-scoped.
Impact: Two tenants with the same user ID can read each other's cached profile, which is an isolation breach.
Suggested fix: Include tenantID and a cache schema/version segment in the key, and invalidate corrupt entries instead of silently re-reading them.
Reference: Redis keys are a single namespace and require collision avoidance by schema.
```

## Good Example: Explicit Dimensions And Decode Handling

```go
const profileCacheVersion = "v2"

func profileKey(tenantID, userID string) string {
	return "tenant:" + tenantID + ":profile:" + profileCacheVersion + ":user:" + userID
}

func (s *Store) CachedProfile(ctx context.Context, tenantID, userID string) (Profile, error) {
	key := profileKey(tenantID, userID)
	if b, err := s.cache.Get(ctx, key).Bytes(); err == nil {
		var profile Profile
		if err := json.Unmarshal(b, &profile); err == nil {
			return profile, nil
		}
		_ = s.cache.Del(ctx, key).Err()
	}

	profile, err := s.repo.Profile(ctx, tenantID, userID)
	if err != nil {
		return Profile{}, err
	}
	b, err := json.Marshal(profile)
	if err != nil {
		return Profile{}, err
	}
	_ = s.cache.Set(ctx, key, b, 10*time.Minute).Err()
	return profile, nil
}
```

The local fix is the key and decode behavior. If adding tenant isolation affects authorization semantics, hand off to security review/spec.

## Bad Example: Ambiguous Query Cache Key

```go
func productSearchKey(filter ProductFilter) string {
	return "products:" + filter.Query + ":" + filter.Sort
}
```

This key can collide when fields contain the delimiter, and it may omit page, locale, visibility, or feature dimensions.

## Good Example: Canonical Fields Plus Digest

```go
func productSearchKey(filter ProductFilter) (string, error) {
	keyInput := struct {
		Query      string `json:"query"`
		Sort       string `json:"sort"`
		Page       int    `json:"page"`
		Locale     string `json:"locale"`
		Visibility string `json:"visibility"`
		Version    string `json:"version"`
	}{
		Query:      filter.Query,
		Sort:       filter.Sort,
		Page:       filter.Page,
		Locale:     filter.Locale,
		Visibility: filter.Visibility,
		Version:    "v4",
	}
	b, err := json.Marshal(keyInput)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return "products:search:" + hex.EncodeToString(sum[:]), nil
}
```

Use the repository's existing canonicalization helper if one exists. The review finding should demand deterministic, complete key material, not a particular hash algorithm.

## Smallest Safe Fix
- Add missing key dimensions required by the current response contract.
- Add a version segment when cached value shape changed.
- Use a centralized key builder in the same package when duplicate ad hoc keys are spreading.
- Escape or digest user-controlled variable-length key parts.
- Delete or bypass corrupt entries after decode errors; do not serve partial zero values.
- Escalate tenant or sensitive-data exposure to `go-security-review` in addition to the local cache finding.

## Validation Ideas
- Add table tests proving distinct tenants, locales, pages, roles, or feature flags produce distinct keys.
- Add a corrupt-cache-entry test and assert the code refetches or invalidates instead of serving a zero-value response.
- Add a cache-version migration test when payload shape changes.
- Add a test that delimiters in user input cannot collide with another key.

## Source Links From Exa
- Redis keys and values, key namespace conventions: https://redis.io/docs/latest/develop/using-commands/keyspace/
- Redis cache-aside query caching tutorial: https://redis.io/learn/howtos/solutions/microservices/caching
- Redis client-side caching invalidation reference: https://redis.io/docs/latest/develop/reference/client-side-caching/
- Redis `SET` command and TTL overwrite behavior: https://redis.io/docs/latest/commands/set/

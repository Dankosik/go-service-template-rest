**Implemented Test Scope**
- Proposed unit tests for the `AccountSummary` read service covering cache hit/miss behavior, tenant-safe keying, `summary_version`-safe keying, TTL jitter, short negative caching for stable `not found`, fail-open degradation on Redis failures, corrupt-entry handling, and same-key miss coalescing.
- Proposed integration tests for the mixed Postgres plus Redis read path covering shared-cache tenant isolation, Redis outage fail-open behavior, and safe reads while `summary_version` backfill is incomplete.
- Proposed integration tests for the backfill job covering resumable progress and idempotent reruns across the `expand -> backfill -> contract` rollout.

**Scenario Coverage**
- `[unit]` `TestAccountSummaryReaderCacheHitReturnsCachedSummaryWithoutOriginRead`: seed a positive cache entry for one `(tenant, account)` pair and assert the cached summary is returned unchanged and Postgres is not called.
- `[unit]` `TestAccountSummaryReaderCacheMissLoadsFromOriginAndStoresTenantScopedEntryWithJitteredTTL`: start with a miss, load from Postgres, assert one cache write, and assert TTL stays within `108s..132s` for the 2 minute base TTL.
- `[unit]` `TestAccountSummaryReaderStableNotFoundWritesNegativeCacheForTenSeconds`: origin returns stable `not found`; assert a negative cache entry is written with `10s` TTL and a second read does not re-hit Postgres before expiry.
- `[unit]` `TestAccountSummaryReaderTransientOriginErrorDoesNotNegativeCache`: origin returns a transient failure; assert the error is returned and no negative cache entry is created.
- `[unit]` `TestAccountSummaryReaderCacheGetFailureFailsOpenToPostgres`: Redis read failure is treated as a miss and the authoritative Postgres result is returned.
- `[unit]` `TestAccountSummaryReaderCacheSetFailureReturnsOriginResult`: Redis write failure after a successful origin read does not fail the read path or mutate the returned summary.
- `[unit]` `TestAccountSummaryReaderCorruptOrWrongVersionCacheEntryIsDiscarded`: malformed or version-mismatched cached payload is treated as a miss, reloaded from Postgres, and replaced instead of being served.
- `[unit]` `TestAccountSummaryReaderCacheKeyIncludesTenantAndSummaryVersion`: the same logical account selector under two tenants and two summary versions produces distinct keys so cross-tenant and stale-version reuse cannot occur.
- `[unit]` `TestAccountSummaryReaderConcurrentMissesCoalesceSingleOriginLoadPerKey`: multiple goroutines reading the same cold `(tenant, account, summaryVersion)` key trigger exactly one origin load and all receive the same result.
- `[unit]` `TestAccountSummaryReaderConcurrentMissesDoNotCoalesceAcrossTenants`: concurrent cold reads for the same account ID in two tenants do not share singleflight state or cached values.
- `[unit]` `TestAccountSummaryReaderIncompleteSummaryVersionUsesSafeFallbackAndSkipsFinalCacheWrite`: rows with missing or not-yet-current `summary_version` use the safe fallback read path and do not populate the final versioned cache entry.
- `[unit]` `TestAccountSummaryReaderCurrentSummaryVersionPrefersVersionedSummary`: rows already backfilled to the current `summary_version` bypass fallback and are cached only under the versioned key.
- `[integration]` `TestAccountSummaryReadPathMissThenHitAcrossPostgresAndRedis`: first read populates Redis from Postgres and the second read returns the same tenant-local summary from cache.
- `[integration]` `TestAccountSummaryReadPathRedisOutageFailsOpen`: with Redis unavailable, reads still succeed from Postgres and do not surface cache-layer failures to callers.
- `[integration]` `TestAccountSummaryReadPathPartialBackfillDoesNotServeIncompleteOrStaleCorruptData`: a fixture with both backfilled and not-yet-backfilled rows returns the correct summary per row and never upgrades incomplete data into a final cached value.
- `[integration]` `TestAccountSummaryReadPathTenantIsolationWithSharedRedis`: two tenants with the same account identifier use the same Redis instance without cross-tenant cache bleed.
- `[integration]` `TestAccountSummaryBackfillResumesAfterPartialBatchFailure`: interrupt the backfill after a committed batch, rerun, and assert remaining rows complete without corrupting already-processed rows.
- `[integration]` `TestAccountSummaryBackfillRerunIsIdempotent`: run the backfill twice against already-updated rows and assert stable summary contents, stable `summary_version`, and no extra updates on the second run.

**Key Test Files**
- Proposed [internal/app/accountsummary/reader_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/app/accountsummary/reader_test.go) for cache-aside read-path unit coverage.
- Proposed [internal/app/accountsummary/key_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/app/accountsummary/key_test.go) for tenant and `summary_version` cache-key isolation checks if key construction is split out; otherwise fold these cases into `reader_test.go`.
- Proposed [test/account_summary_read_path_integration_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/test/account_summary_read_path_integration_test.go) for Postgres plus Redis integration coverage of miss/hit, fail-open, mixed-version safety, and tenant isolation.
- Proposed [test/account_summary_backfill_integration_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/test/account_summary_backfill_integration_test.go) for resumable and idempotent backfill behavior against real Postgres state.
- These are proposed placements because the current repository does not yet contain an `AccountSummary` implementation surface.

**Validation Commands**
- `make test`
- `make test-race`
- `REQUIRE_DOCKER=1 make test-integration`
- `MIGRATION_DSN='<postgres-dsn>' make migration-validate`
- If local DSN rehearsal is not wired, use `make docker-migration-validate` instead of the `MIGRATION_DSN=...` form.

**Observed Result**
- No repository files were edited.
- No validation commands were run.
- This deliverable is the requested exact test-addition plan, not implemented test code.

**Design Escalations**
- The prompt does not define whether safe fallback results during incomplete backfill may be cached under any transitional key; these tests assume they must not populate the final `summary_version` cache entry.
- The prompt does not define the durable resume marker shape for the backfill job; the resumability tests should bind to the real checkpoint contract once chosen.
- The prompt does not state whether cache payload versioning lives only in the key, only in the value, or both; the corrupt/wrong-version cache test assumes mismatches are treated as misses and rewritten.

**Residual Risks**
- If `AccountSummary` visibility also varies by auth scope, locale, or feature flags, tenant-only keying is insufficient and more key-dimension tests are required.
- Contract-phase safety still depends on migration verification proving 100% backfill completion before fallback removal; these tests only prove application behavior before contract.
- If Redis expiry behavior is only unit-tested with fakes and not exercised in integration with a real Redis instance, serialization and TTL edge behavior can still drift.

**Implemented Test Scope**
- Proposed only, with no repository edits: unit tests for the `AccountSummary` reader covering cache hit/miss behavior, fail-open Redis behavior, negative-cache safety, tenant isolation, TTL jitter, and same-key miss coalescing.
- Proposed only, with no repository edits: unit tests for mixed-version read behavior covering incomplete `summary_version` fallback and canonical reads once backfill is complete.
- Proposed only, with no repository edits: integration tests over Postgres plus Redis covering miss-to-hit flow, shared-cache tenant isolation, Redis outage fail-open behavior, partial-backfill safety, and rerun/idempotent backfill behavior.

**Scenario Coverage**
- `[unit]` `TestAccountSummaryReaderCacheHitReturnsCachedSummaryWithoutOriginRead`: seed a tenant-scoped positive cache entry and assert the summary is returned unchanged with no Postgres read.
- `[unit]` `TestAccountSummaryReaderCacheMissLoadsOriginAndStoresJitteredTTL`: start from a miss, load from Postgres once, and assert the cache write uses a TTL inside the `108s..132s` window for the 2 minute base TTL.
- `[unit]` `TestAccountSummaryReaderStableNotFoundWritesNegativeCacheForTenSeconds`: return a stable `not found` from origin, assert a `10s` negative cache entry is written, and assert a second read avoids Postgres before expiry.
- `[unit]` `TestAccountSummaryReaderTransientOriginErrorDoesNotWriteNegativeCache`: return a transient origin failure and assert the error is surfaced without creating a negative cache entry.
- `[unit]` `TestAccountSummaryReaderCacheGetFailureFailsOpenToOrigin`: make Redis read fail and assert the authoritative Postgres result is still returned.
- `[unit]` `TestAccountSummaryReaderCacheSetFailureReturnsOriginResult`: make Redis write fail after a successful origin read and assert the read still succeeds with the origin result.
- `[unit,race]` `TestAccountSummaryReaderConcurrentSameTenantMissesCoalesceSingleOriginLoad`: use a gated origin fake and assert N concurrent misses for the same tenant/account trigger exactly one origin load.
- `[unit,race]` `TestAccountSummaryReaderConcurrentSameAccountDifferentTenantsDoNotShareLoadOrCache`: run concurrent misses for the same account ID under two tenants and assert they do not coalesce or reuse each other’s cached value.
- `[unit]` `TestAccountSummaryReaderIncompleteSummaryVersionUsesSafeFallbackAndSkipsStableCacheWrite`: return a row with missing or incomplete `summary_version`, assert the safe fallback path is used, and assert no steady-state cache entry is written from that fallback result.
- `[unit]` `TestAccountSummaryReaderCurrentSummaryVersionReturnsCanonicalSummaryAndCachesIt`: return a row at the current `summary_version`, assert the canonical summary path is used, and assert the result is cached normally.
- `[integration]` `TestAccountSummaryReadPathMissThenHitAcrossPostgresAndRedis`: first read populates Redis from Postgres and the second read returns the same tenant-local summary from cache.
- `[integration]` `TestAccountSummaryReadPathRedisOutageFailsOpen`: bring Redis down or force connection failure and assert reads still succeed from Postgres without surfacing cache errors.
- `[integration]` `TestAccountSummaryReadPathSharedRedisPreservesTenantIsolation`: use one Redis instance with two tenants sharing the same logical account ID and assert no cross-tenant cache bleed.
- `[integration]` `TestAccountSummaryReadPathFallbackReadDoesNotPersistAfterBackfillCompletes`: read once while `summary_version` is incomplete, complete the backfill for that row, read again, and assert the canonical summary is returned instead of the earlier fallback result.
- `[integration]` `TestAccountSummaryBackfillRerunAfterPartialProgressCompletesRemainingRows`: stop the backfill after a committed partial run, rerun it through the normal resume path, and assert remaining rows complete without corrupting already-processed rows.
- `[integration]` `TestAccountSummaryBackfillSecondFullRunIsIdempotent`: run the backfill twice after completion and assert stable row contents, stable `summary_version`, and no extra writes on the second run.

**Key Test Files**
- Proposed [internal/app/accountsummary/reader_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/app/accountsummary/reader_test.go) for reader/cache unit coverage.
- Proposed [internal/app/accountsummary/backfill_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/app/accountsummary/backfill_test.go) for backfill-orchestration unit coverage if resume logic lives in the app layer.
- Proposed [test/account_summary_cache_migration_integration_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/test/account_summary_cache_migration_integration_test.go) for Postgres plus Redis integration coverage.

**Validation Commands**
- `go test ./internal/app/accountsummary -run '^TestAccountSummaryReader' -count=1`
- `go test ./internal/app/accountsummary -run '^TestAccountSummaryBackfill' -count=1`
- `go test -race ./internal/app/accountsummary -run 'TestAccountSummaryReaderConcurrent(SameTenantMissesCoalesceSingleOriginLoad|SameAccountDifferentTenantsDoNotShareLoadOrCache)$' -count=1`
- `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestAccountSummary(ReadPath|Backfill)' -count=1`
- `make migration-validate`
- `make check`

**Observed Result**
- No tests were implemented or run because the task explicitly requested a no-edit, deliverable-only response.

**Design Escalations**
- The prompt requires safe fallback when `summary_version` is incomplete but does not define whether fallback results may be cached under any transitional key. The test set above assumes fallback data must not populate the steady-state cache entry.
- Negative caching is allowed only for stable `not found`, but the origin signal that distinguishes stable absence from transient failure is not specified. These tests need an explicit typed result or sentinel error from the source-of-truth path.
- Backfill must be resumable and idempotent, but the progress contract is not specified. The integration tests above prove rerun semantics without locking a particular checkpoint or watermark design.

**Residual Risks**
- This set proves per-process miss coalescing, not fleet-wide origin protection during a shared Redis outage across multiple service replicas.
- These tests reduce rollout risk, but contract-phase safety still depends on production verification that backfill is complete before fallback logic is removed.
- If `AccountSummary` visibility also varies by auth scope, locale, or feature flags, tenant-only isolation coverage is insufficient and the cache-dimension test set must expand.

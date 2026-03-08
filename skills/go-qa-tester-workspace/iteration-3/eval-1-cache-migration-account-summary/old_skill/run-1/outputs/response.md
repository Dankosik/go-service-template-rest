Implemented Test Scope
- No repository files were edited. This is the smallest sufficient test plan I would add for the `AccountSummary` cache-aside read path and the `summary_version` expand/backfill/contract rollout.
- Add unit tests for cache hit/miss behavior, TTL jitter, stable negative caching, fail-open Redis degradation, tenant isolation, and same-key miss coalescing.
- Add integration tests for mixed-version reads, safe fallback while backfill is incomplete, shared-Redis tenant isolation, and resumable plus idempotent backfill behavior.

Scenario Coverage
- `[unit]` `TestAccountSummaryReadCacheHitReturnsCachedSummary`: preload a cached summary for one `(tenant, account)` pair; assert the cached value is returned and Postgres is not called.
- `[unit]` `TestAccountSummaryReadCacheMissLoadsFromPostgresAndCachesWithJitteredTTL`: start with a miss; assert one Postgres read, one cache write, and a positive-cache TTL within `108s..132s`.
- `[unit]` `TestAccountSummaryReadStableNotFoundNegativeCachesForTenSeconds`: when Postgres returns an authoritative stable `not found`, write a negative cache entry with `10s` TTL and assert a second read inside that window does not hit Postgres.
- `[unit]` `TestAccountSummaryReadIncompleteBackfillDoesNotNegativeCache`: when `summary_version` is missing or incomplete during backfill, assert the read falls back safely and is not treated as a stable `not found`.
- `[unit]` `TestAccountSummaryReadRedisGetFailureFailsOpenToPostgres`: make Redis `GET` fail and assert the authoritative Postgres summary is still returned.
- `[unit]` `TestAccountSummaryReadRedisSetFailureReturnsOriginResult`: make Redis `SET` fail after a successful Postgres read and assert the read still succeeds with the Postgres result.
- `[unit,race]` `TestAccountSummaryReadConcurrentMissesCoalescePerTenantAccount`: issue parallel cold reads for the same tenant/account and assert exactly one Postgres load occurs.
- `[unit,race]` `TestAccountSummaryReadConcurrentMissesDoNotCoalesceAcrossTenants`: issue parallel cold reads for the same logical account ID in different tenants and assert they do not share coalescing state or cached values.
- `[integration]` `TestAccountSummaryReadMissThenHitAcrossPostgresAndRedis`: with real Postgres and Redis, assert the first read populates cache and the second read for the same tenant/account returns the same summary from Redis.
- `[integration]` `TestAccountSummaryReadRedisOutageFailsOpen`: with Redis unavailable, assert reads still succeed from Postgres and do not surface cache-layer failures.
- `[integration]` `TestAccountSummaryMixedVersionReadDoesNotServeStaleData`: seed a mix of fully backfilled rows and rows with incomplete `summary_version`; assert complete rows return their current summary, incomplete rows take the safe fallback path, and no stale-corrupt summary is served.
- `[integration]` `TestAccountSummaryReadDoesNotLeakAcrossTenantsWithSharedRedis`: use one Redis instance with two tenants sharing the same logical account ID and assert reads never return the other tenant’s summary.
- `[integration]` `TestAccountSummaryBackfillResumeAfterInterruptionCompletesRemainingRowsSafely`: interrupt a backfill run mid-progress, rerun it, and assert remaining rows complete without breaking already-correct rows.
- `[integration]` `TestAccountSummaryBackfillRerunIsIdempotent`: run the backfill twice and assert summary contents plus `summary_version` state stay stable on the second run.

Key Test Files
- Proposed [internal/app/accountsummary/service_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/app/accountsummary/service_test.go) for read-path unit coverage.
- Proposed [internal/app/accountsummary/backfill_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/app/accountsummary/backfill_test.go) for backfill unit coverage.
- Proposed [test/account_summary_read_path_integration_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/test/account_summary_read_path_integration_test.go) for Postgres plus Redis read-path integration coverage.
- Proposed [test/account_summary_backfill_integration_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/test/account_summary_backfill_integration_test.go) for resumable and idempotent backfill integration coverage.

Validation Commands
- `make test`
- `make test-race`
- `REQUIRE_DOCKER=1 make test-integration`
- `make migration-validate`
- `make check-full`

Observed Result
- No repository files were edited.
- No validation commands were executed because the task requested only the test deliverable, not implementation.
- The repository does not currently expose an `AccountSummary` package or test surface, so the file paths above are proposed placements consistent with the existing `internal/app/*` and `test/*_integration_test.go` layout.

Design Escalations
- Clarify whether fallback results read while `summary_version` is incomplete may be cached at all, or must bypass Redis until backfill completes.
- Clarify what qualifies as a "stable not found" during expand/backfill so tests can distinguish true absence from transitional incompleteness.
- Clarify freshness behavior when a row moves from fallback-readable to fully backfilled, because that determines the strongest post-backfill stale-data assertion.

Residual Risks
- If tenant binding happens in the HTTP or auth layer, these tests do not yet prove that request identity is threaded correctly into the read inputs.
- If `AccountSummary` is assembled from multiple underlying tables or async rollups, one more integration scenario may be needed to prove fallback never publishes a partially assembled summary as authoritative.
- Until cache behavior during incomplete backfill is explicitly decided, the suite can prove safe observable reads but not the exact allowed cache population rules during transition.

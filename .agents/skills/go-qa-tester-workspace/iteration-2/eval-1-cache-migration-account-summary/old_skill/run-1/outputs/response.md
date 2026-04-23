**Implemented Test Scope**
- No repository files were edited. This deliverable defines the exact tests I would add for the `AccountSummary` cache-aside read path and the `summary_version` backfill rollout.
- Add unit tests around the reader to prove cache semantics, tenant isolation, fail-open degradation, version-safe fallback, and same-key miss coalescing.
- Add integration tests over real Postgres and Redis plus the backfill runner to prove mixed-version safety, shared-cache isolation, resumable execution, and idempotent reruns.

**Scenario Coverage**
- `[unit]` `TestAccountSummaryReader_CacheHitReturnsCachedSummary`: preload a cached summary for `(tenantA, account42, currentVersion)` and assert the reader returns it without calling Postgres.
- `[unit]` `TestAccountSummaryReader_CacheMissStoresPositiveEntryWith120sTTLJitter`: start from a miss, load from Postgres, write a tenant-scoped positive cache entry, and assert TTL stays within `108s..132s`.
- `[unit]` `TestAccountSummaryReader_StableNotFoundWritesNegativeCacheFor10Seconds`: when Postgres returns authoritative stable `not found`, write a negative cache entry with `10s` TTL and assert a second read inside that window does not hit Postgres.
- `[unit]` `TestAccountSummaryReader_TransientOriginErrorDoesNotNegativeCache`: when Postgres returns a transient error, return the error and assert no negative cache entry is written.
- `[unit]` `TestAccountSummaryReader_CacheReadFailureFailsOpenToPostgres`: make Redis `GET` fail and assert the reader still returns the authoritative Postgres summary.
- `[unit]` `TestAccountSummaryReader_CacheWriteFailureReturnsOriginResult`: make Redis `SET` fail after a successful Postgres read and assert the read still succeeds with the Postgres result.
- `[unit]` `TestAccountSummaryReader_IncompleteSummaryVersionFallsBackSafelyWithoutPoisoningFinalCache`: simulate a row with incomplete or missing `summary_version`, assert the safe fallback path is used, and assert that incomplete data is not promoted into the final positive cache entry.
- `[unit,race]` `TestAccountSummaryReader_ConcurrentColdReadsCoalescePerTenantScopedKey`: fire parallel reads for the same cold `(tenant, account)` key and assert exactly one Postgres load occurs.
- `[unit,race]` `TestAccountSummaryReader_ConcurrentColdReadsDoNotCoalesceAcrossTenants`: fire parallel cold reads for the same account ID under different tenants and assert they use separate coalescing groups and separate origin loads.
- `[integration]` `TestAccountSummaryReadPath_PostgresMissThenRedisHit`: with real Postgres and Redis, assert the first read populates cache and the second read for the same tenant/account is served from Redis.
- `[integration]` `TestAccountSummaryReadPath_RedisOutageFailsOpen`: make Redis unavailable and assert reads still succeed from Postgres without surfacing cache-layer failure to callers.
- `[integration]` `TestAccountSummaryReadPath_MixedBackfillStateDoesNotServeStaleCorruptData`: seed a mix of fully backfilled and incomplete rows, then assert incomplete rows use fallback, complete rows use versioned data, and no stale-corrupt summary is served.
- `[integration]` `TestAccountSummaryReadPath_SharedRedisDoesNotLeakAcrossTenantsWithSameAccountID`: use one Redis instance with two tenants sharing the same logical account ID and assert cached values never cross tenant boundaries.
- `[integration]` `TestAccountSummaryBackfill_ResumeAfterPartialBatchFailure`: fail the backfill after a partial batch, rerun it, and assert remaining rows complete without duplicate work, skipped rows, or changed completed rows.
- `[integration]` `TestAccountSummaryBackfill_RerunIsIdempotent`: run the backfill twice against already-processed rows and assert stable summaries plus stable `summary_version` state.

**Key Test Files**
- Proposed [internal/app/accountsummary/reader_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/app/accountsummary/reader_test.go): unit tests for cache hit/miss, TTL, negative-cache rules, fail-open behavior, fallback semantics, and singleflight coalescing.
- Proposed [internal/app/accountsummary/backfill_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/app/accountsummary/backfill_test.go): unit tests for resumable and idempotent backfill behavior if the runner lives in the application layer.
- Proposed [test/account_summary_read_path_integration_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/test/account_summary_read_path_integration_test.go): integration tests for real Postgres + Redis read-path behavior and tenant isolation.
- Proposed [test/account_summary_backfill_integration_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/test/account_summary_backfill_integration_test.go): integration tests for partial-backfill safety, resume, and rerun idempotence.

**Validation Commands**
- `go test ./internal/app/accountsummary -run 'TestAccountSummaryReader_' -count=1`
- `go test ./internal/app/accountsummary -run 'TestAccountSummaryBackfill_' -count=1`
- `go test -race ./internal/app/accountsummary -run 'TestAccountSummaryReader_(ConcurrentColdReadsCoalescePerTenantScopedKey|ConcurrentColdReadsDoNotCoalesceAcrossTenants)$' -count=1`
- `REQUIRE_DOCKER=1 go test -tags=integration ./test -run 'TestAccountSummary(ReadPath|Backfill)' -count=1`
- `go test ./... -count=1`

**Observed Result**
- Not run. This was a read-only deliverable; no test files were added and no validation commands were executed.

**Design Escalations**
- The repository does not currently contain an `AccountSummary` implementation surface, so the file paths above are proposed placements aligned with the existing unit-under-`internal` and integration-under-`test` split documented in [test/README.md](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/test/README.md#L1).
- The task requires positive-cache TTL `120s +/-10%`, but the current repo default is `redis.fresh_ttl=60s` in [internal/config/defaults.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/config/defaults.go#L27). The tests above assume the feature injects or configures `120s` explicitly.
- `Stable not found` needs an explicit origin signal distinct from transient database failure and backfill-incomplete absence; otherwise the negative-cache test cannot be made precise.
- The prompt does not define whether fallback results during partial backfill may be cached at all. The scenarios above assume incomplete `summary_version` rows must not be written into the final positive cache.

**Residual Risks**
- These tests prove single-process miss coalescing, not fleet-wide origin protection across multiple service instances during a broad Redis outage.
- Without an HTTP or auth-context surface for `AccountSummary`, this set does not prove tenant binding from request identity into repository/cache lookup inputs.
- Contract-phase safety still depends on rollout evidence that backfill reached completion before removing the fallback read path.

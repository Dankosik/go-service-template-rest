Implemented Test Scope
- Proposed unit tests for the `AccountSummary` read service covering cache hit/miss behavior, tenant-safe keying, TTL jitter, short negative caching for stable `not found`, fail-open degradation on Redis failures, corrupt-entry handling, and same-key miss coalescing.
- Proposed unit tests for backfill orchestration covering safe fallback when `summary_version` is incomplete, skip behavior for already-versioned rows, resumable checkpoint handling, and rerun idempotence.
- Proposed integration tests using real Postgres + Redis + migrations to prove miss-to-hit behavior, Redis outage fail-open reads, partial backfill mixed-version safety, tenant isolation, and resumable/idempotent backfill execution.

Scenario Coverage
- `[unit]` `TestAccountSummaryReader_CacheHitReturnsCachedSummaryWithoutOriginRead`: seed a positive cache entry for `tenantA/account42`; assert the cached summary is returned and Postgres is not called.
- `[unit]` `TestAccountSummaryReader_CacheMissLoadsFromOriginAndStoresJitteredTTL`: start with a cache miss; assert one Postgres read, one cache write, and TTL within `108s..132s` for the 2 minute base TTL.
- `[unit]` `TestAccountSummaryReader_StableNotFoundWritesNegativeCacheFor10Seconds`: origin returns stable `not found`; assert a negative cache entry is written for `10s` and the second read avoids Postgres.
- `[unit]` `TestAccountSummaryReader_OriginFailureDoesNotNegativeCache`: origin returns a transient error; assert the error is returned and no negative cache entry is written.
- `[unit]` `TestAccountSummaryReader_CacheGetFailureFailsOpenToPostgres`: Redis get timeout/error is treated as a miss; assert the Postgres result is returned instead of a cache-layer failure.
- `[unit]` `TestAccountSummaryReader_CacheSetFailureReturnsOriginResult`: Redis set failure after a successful Postgres read does not fail the read path.
- `[unit]` `TestAccountSummaryReader_CorruptOrWrongVersionCacheEntryIsDiscarded`: malformed or version-mismatched cached payload is treated as a miss, reloaded from Postgres, and replaced rather than served.
- `[unit]` `TestAccountSummaryReader_CacheKeyIncludesTenantAndSummaryVersion`: the same logical account selector under two tenants and two summary versions produces distinct keys, preventing cross-tenant or stale-version reuse.
- `[unit,race]` `TestAccountSummaryReader_ConcurrentMissesCoalesceSingleOriginLoadPerKey`: N goroutines requesting the same cold tenant/account key all receive the same result and trigger exactly one Postgres load.
- `[unit,race]` `TestAccountSummaryReader_ConcurrentMissesDoNotCoalesceAcrossTenants`: concurrent cold reads for `tenantA/account42` and `tenantB/account42` do not share coalescing state or cached values.
- `[unit]` `TestAccountSummaryReader_BackfillFallbackWhenSummaryVersionIncomplete`: rows with missing or pre-current `summary_version` use the safe fallback read path and are not interpreted as fully backfilled summaries.
- `[unit]` `TestAccountSummaryReader_PrefersVersionedSummaryOnceBackfilled`: rows with the current `summary_version` bypass fallback and are cached only under the versioned key.
- `[integration]` `TestAccountSummaryReadPath_MissThenHitAcrossPostgresAndRedis`: first read populates Redis from Postgres; second read returns the same tenant-local summary from cache.
- `[integration]` `TestAccountSummaryReadPath_RedisOutageFailsOpen`: with Redis unavailable, reads still succeed from Postgres and do not leak cache errors to callers.
- `[integration]` `TestAccountSummaryReadPath_PartialBackfillMixedVersionSafety`: a fixture with backfilled and not-yet-backfilled rows returns the correct summary per tenant without stale-corrupt mixing.
- `[integration]` `TestAccountSummaryReadPath_TenantIsolationWithSameLogicalAccountID`: two tenants with the same account identifier value never reuse each other’s cache entry or fallback result.
- `[integration]` `TestAccountSummaryBackfill_ResumesFromCheckpointAfterPartialBatchFailure`: fail the backfill mid-run, persist the checkpoint, rerun, and assert remaining rows complete without duplicates or skips.
- `[integration]` `TestAccountSummaryBackfill_RerunIsIdempotent`: run the backfill twice against already-updated rows and assert no data drift, no extra version bumps, and stable row counts.

Key Test Files
- `internal/app/accountsummary/reader_test.go`
- `internal/app/accountsummary/backfill_job_test.go`
- `test/account_summary_read_path_integration_test.go`
- `test/account_summary_backfill_integration_test.go`

Validation Commands
- `go test ./internal/app/accountsummary -count=1`
- `go test -race ./internal/app/accountsummary -run 'TestAccountSummaryReader_(ConcurrentMissesCoalesceSingleOriginLoadPerKey|ConcurrentMissesDoNotCoalesceAcrossTenants)$' -count=1`
- `REQUIRE_DOCKER=1 go test -tags=integration ./test -run 'TestAccountSummary(ReadPath|Backfill)' -count=1`
- `make migration-validate`
- `make check-full`

Observed Result
- Not run in this task. This was a no-edit test-design deliverable, so the commands above are the validation set I would execute after adding the tests.

Design Escalations
- The prompt does not define whether fallback results during backfill may be cached; these tests assume fallback data is never promoted into the final `summary_version` cache key.
- The prompt does not define whether miss coalescing is only in-process or cross-instance; the listed tests prove in-process same-key collapse. A distributed-lock requirement would need one more integration test.

Residual Risks
- These tests prove correctness and isolation, but not fleet-wide origin protection during a full Redis outage; if that matters, add a multi-instance load test.
- Contract-phase safety after removing fallback logic still depends on production verification queries showing `summary_version` backfill completion before old-path removal.

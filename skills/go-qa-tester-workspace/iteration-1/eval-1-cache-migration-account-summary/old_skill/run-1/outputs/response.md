**Implemented Test Scope**
- Add unit tests for the cache-aside `AccountSummary` reader to prove tenant-safe keying, positive and negative TTL rules, fail-open cache behavior, and same-key miss coalescing.
- Add integration tests for mixed-version Postgres reads and shared-Redis reads so partial backfill and cache isolation are proven against real dependencies.
- Add integration tests for backfill reruns and partial-progress resume behavior so the `summary_version` rollout is safe across `expand -> backfill -> contract`.

**Scenario Coverage**
- `Unit` `TestAccountSummaryReaderCacheHitReturnsCachedSummary`: a preloaded cache entry for `(tenant, account)` is returned unchanged and Postgres is not called.
- `Unit` `TestAccountSummaryReaderCacheMissStoresTenantScopedEntryWithTTLJitterBounds`: a cache miss loads from Postgres, writes a key that includes tenant scope, and uses a TTL in the inclusive `108s..132s` range.
- `Unit` `TestAccountSummaryReaderStableNotFoundStoresNegativeCacheForTenSeconds`: a stable source-of-truth not-found result writes a negative cache entry with `10s` TTL and a second read does not re-hit Postgres before expiry.
- `Unit` `TestAccountSummaryReaderIncompleteSummaryVersionDoesNotNegativeCache`: a backfill-incomplete row does not create a negative cache entry and instead returns the safe Postgres fallback result.
- `Unit` `TestAccountSummaryReaderCacheGetFailureFallsBackToPostgres`: a Redis read failure still returns the authoritative Postgres summary.
- `Unit` `TestAccountSummaryReaderCacheSetFailureReturnsOriginValue`: a Redis write failure does not fail the read and does not change the returned Postgres result.
- `Unit` `TestAccountSummaryReaderConcurrentMissesCoalescePerTenantScopedKey`: concurrent reads for the same `(tenant, account)` perform one origin load and all callers receive the same summary.
- `Unit` `TestAccountSummaryReaderConcurrentMissesDoNotCoalesceAcrossTenants`: the same account ID in two tenants uses different cache and singleflight keys and produces separate origin loads.
- `Integration` `TestAccountSummaryReadUsesLegacyFallbackWhileSummaryVersionIsIncomplete`: with `summary_version` null or incomplete after `expand`, the read path returns the safe legacy-derived summary instead of empty or partially backfilled data.
- `Integration` `TestAccountSummaryReadPrefersVersionedSummaryAfterBackfill`: once backfill completes for a row, the read path returns the versioned summary and caches that value.
- `Integration` `TestAccountSummaryReadDoesNotLeakAcrossTenantsWithSharedRedis`: two tenants with the same account identifier populate and read through the same Redis instance without cross-tenant cache bleed.
- `Integration` `TestAccountSummaryBackfillRerunIsIdempotent`: rerunning the backfill over already-processed rows leaves `summary_version` and summary contents unchanged.
- `Integration` `TestAccountSummaryBackfillResumesAfterPartialProgress`: after an interrupted backfill batch, the next run completes the remaining rows without reprocessing completed rows into a different result.

**Key Test Files**
- Proposed `internal/app/accountsummary/reader_test.go` for cache hit and miss, TTL, negative-cache gating, fail-open cache errors, and same-key miss coalescing.
- Proposed `test/account_summary_read_integration_test.go` for mixed-version Postgres reads and shared-Redis tenant isolation.
- Proposed `test/account_summary_backfill_integration_test.go` for idempotent and resumable backfill behavior against real Postgres state.

**Validation Commands**
- `make test`
- `make test-race`
- `REQUIRE_DOCKER=1 make test-integration`
- `make migration-validate`

**Observed Result**
- No repository files were edited and no new tests were executed, because this task explicitly requested a no-edit deliverable.
- The current repository does not yet contain an `AccountSummary` implementation surface, so the file paths above are proposed placements aligned with the existing `internal/app/*` and `test/*` split.
- The validation commands above are the exact commands I would run after adding the proposed tests; no readiness claim is made from this proposal-only pass.

**Design Escalations**
- The read path needs an explicit distinction between `stable not found`, `backfill incomplete`, and origin failure. Without that, negative-cache tests cannot be made precise and safe.
- The rollout needs one explicit rule for cache compatibility across backfill completion: either pre-backfill and post-backfill summaries are semantically identical, or cache key or value versioning must be defined to prevent serving a stale pre-backfill representation after backfill finishes.
- The backfill job needs a clear resume contract, such as idempotent UPSERT semantics plus a stable scan order or persisted progress cursor, so the resumable and idempotent tests assert one concrete behavior instead of a guessed one.

**Residual Risks**
- Unit tests alone would not prove Redis TTL wiring, serialization, or real shared-cache isolation, which is why the integration layer remains necessary.
- `make migration-validate` proves the SQL sequence can run, but it does not by itself prove the application read path handles mixed-version data correctly; that remains the job of the integration tests above.
- If `AccountSummary` is derived from multiple tables or asynchronous rollups, one more integration scenario may be needed to prove the fallback path never caches a partially assembled summary as a valid positive result.

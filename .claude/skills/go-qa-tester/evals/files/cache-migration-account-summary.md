We are changing the `AccountSummary` read path:

- Reads come from Postgres as source of truth.
- We are adding a Redis cache-aside layer.
- Cache rules:
  - tenant-scoped keys
  - TTL is 2 minutes with `+-10%` jitter
  - negative cache is allowed only for stable `not found` results for 10 seconds
  - cache failures must be fail-open for reads
  - same-key concurrent misses should coalesce so origin is not hammered
- We are also adding a new `summary_version` column and rolling it out with `expand -> backfill -> contract`.
- During backfill, reads must fall back safely when `summary_version` data is incomplete.
- Backfill job must be resumable and idempotent.
- The team wants strong confidence that a cache outage or partial backfill will not return cross-tenant or stale-corrupt data.

Do not edit repository files. I only want the exact tests you would add, which layer each test belongs to, and which validation commands you would run.

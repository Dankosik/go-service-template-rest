# SQL Access Discipline And Query Budget

## When To Load
Load this when the planned change needs SQL access contracts before coding: query shape, round-trip budget, N+1 risk, dynamic filters, pagination, generated SQL interfaces, or SQL telemetry grouping. Use it before approving a cache that might only be hiding an undefined or inefficient origin query.

Stay in the runtime DB/cache seam. If the answer depends on primary schema ownership, table decomposition, retention, or migration rollout, hand off to data architecture instead of expanding this file into schema design.

## Viable Options
- Origin-only SQL contract: keep the path uncached, name the query or query class, set max round trips per request, require explicit columns, and prove the query with plan and latency evidence.
- Bulk-fetch or join contract: replace per-row repository loops with one query or one bounded set of queries that preserves ownership and deterministic ordering.
- Split query classes: define separate hot-path queries for materially different filters or consistency classes instead of one generic query builder with unbounded shape.
- Keyset pagination: use deterministic cursor predicates for deep or hot list paths when offset pagination would create unstable or expensive scans.
- Cache-after-origin-contract: allow cache only after the origin query budget, hit-rate hypothesis, freshness class, and failure behavior are explicit.

## Selected And Rejected Examples
Selected example: for `GET /accounts/{id}/invoices?status=open`, specify `ListOpenInvoicesByAccount` as one named query class with explicit selected columns, deterministic order, a page-size cap, and a query budget such as one page query plus an optional separate count only when the API contract requires it. Redis approval remains blocked until query plan, p95/p99 latency, row-count distribution, and expected hit rate exist.

Selected example: for catalog reads, split marketing/catalog copy from stock-sensitive fields. Cacheable copy can have an eventual freshness class, while stock or admin-sensitive reads stay origin-backed or use a separate contract with stronger consistency.

Rejected example: adding a whole-response Redis cache to hide an N+1 path where the service loads a parent row and then loops through child queries. The spec should reject the cache until the origin access pattern is bounded.

Rejected example: request-controlled `ORDER BY` or dynamic table names assembled from input. If dynamic identifiers are unavoidable, the spec must require an allowlist and a bounded set of query shapes.

Rejected example: production request paths that discover cache keys with Redis wildcard scans. Use deterministic key construction, maintained indexes, versioned namespaces, or explicit invalidation targets instead.

## Staleness And Failure Semantics
- The SQL origin remains the source of truth unless a separate approved decision changes observable semantics.
- A cache miss, decode failure, or cache timeout must not change SQL correctness; it should fall through to the origin path within the request's remaining budget.
- Query-budget failures should be visible as origin latency/error failures, not hidden behind indefinite cache retries.
- If the response mixes strong and eventual fields, split the runtime contract rather than assigning one freshness class to the whole response.

## Acceptance Checks
- Every production path names its query or query class, expected result cardinality, max round trips, max page size, and deterministic ordering.
- Hot list paths document offset vs keyset decision and the acceptance evidence needed for the chosen form.
- No N+1 service or repository loop is accepted without an explicit bounded-query exception.
- Dynamic identifiers are either absent or allowlisted.
- Go row resources and errors are accounted for in the spec obligations: `Rows.Close`, `Rows.Err`, `QueryRow.Scan`, and context-aware calls where the implementation will exist.
- SQL telemetry requirements use low-cardinality summaries or stable query names, not raw high-cardinality parameter values.

## Exa Source Links
- [Go `database/sql` package](https://pkg.go.dev/database/sql)
- [Go: Canceling in-progress operations](https://go.dev/doc/database/cancel-operations)
- [Go: Managing connections](https://go.dev/doc/database/manage-connections)
- [OpenTelemetry database client span semantic conventions](https://opentelemetry.io/docs/specs/semconv/database/database-spans/)
- [Redis keys and values, including `SCAN` vs `KEYS`](https://redis.io/docs/latest/develop/using-commands/keyspace)

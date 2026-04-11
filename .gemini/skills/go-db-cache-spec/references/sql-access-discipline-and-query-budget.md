# SQL Access Discipline And Query Budget

## Behavior Change Thesis
When loaded for slow SQL, N+1, dynamic filtering, pagination, or cache-before-origin ambiguity, this file makes the model choose a named, bounded origin query contract instead of likely mistake `approve a cache or generic query builder around an undefined SQL path`.

## When To Load
Load this when the planned change needs a query-shape decision before coding, especially when cache is being proposed because an origin path is slow or unmeasured.

Stay in the runtime DB/cache seam. If the answer depends on primary schema ownership, table decomposition, retention, or migration rollout, hand off to data architecture instead of expanding this file into schema design.

## Decision Rubric
- Keep the origin path legible before cache: name the query or query class, expected cardinality, selected columns, max round trips, page-size cap, ordering, and proof evidence.
- Reject cache as the first fix for N+1 or unbounded repository loops; choose a join, bulk fetch, or bounded query set first.
- Split materially different filters, consistency classes, or hot paths into separate query classes instead of one unbounded query builder.
- Use keyset pagination for hot or deep lists when offset would create unstable or expensive scans; keep offset only when bounded and accepted by evidence.
- Allow dynamic SQL identifiers only through an allowlist and a bounded set of shapes. Parameterized values alone do not make dynamic identifiers safe.
- Keep SQL telemetry tied to stable query names or grouped operation classes, not raw literals or high-cardinality parameter values.

## Imitate
- `ListOpenInvoicesByAccount`: one named query class, explicit selected columns, deterministic order, page-size cap, and one page query plus optional count only when the API contract requires it. Copy the habit of blocking Redis until plan, latency, row-count distribution, and hit-rate evidence exist.
- Catalog read split: cache marketing copy under an eventual freshness class while stock and admin-sensitive fields stay origin-backed or use a stronger path. Copy the habit of separating fields with different consistency classes.

## Reject
- Whole-response Redis cache around a path that loads one parent row and loops through child queries. Reject because it hides an unbounded origin contract.
- Request-controlled `ORDER BY` or table names assembled from input. Reject unless the spec defines allowlisted identifiers and bounded shapes.
- Production request path using Redis wildcard scans to discover keys. Reject in favor of deterministic keys, maintained indexes, versioned namespaces, or explicit invalidation targets.

## Agent Traps
- Do not say "add an index" as the DB/cache spec outcome when the primary schema decision belongs to data architecture; record the required evidence and handoff.
- Do not let "generated SQL" hide unbounded query shape; generated interfaces still need query names, cardinality, and pagination contracts.
- Do not assign one freshness class to a response that mixes strong and eventual fields; split the runtime contract.

## Validation Shape
- The SQL origin remains the source of truth unless a separate approved decision changes observable semantics.
- A cache miss, decode failure, or cache timeout must not change SQL correctness; it should fall through to the origin path within the request's remaining budget.
- Query-budget failures should be visible as origin latency/error failures, not hidden behind indefinite cache retries.
- Every production path names its query or query class, expected result cardinality, max round trips, max page size, and deterministic ordering.
- Hot list paths document offset vs keyset decision and the acceptance evidence needed for the chosen form.
- No N+1 service or repository loop is accepted without an explicit bounded-query exception.
- Dynamic identifiers are either absent or allowlisted.
- Go row resources and errors are accounted for in the spec obligations: `Rows.Close`, `Rows.Err`, `QueryRow.Scan`, and context-aware calls where the implementation will exist.
- SQL telemetry requirements use low-cardinality summaries or stable query names, not raw high-cardinality parameter values.

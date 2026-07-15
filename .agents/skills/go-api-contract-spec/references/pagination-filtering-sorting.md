# Pagination, Filtering, And Sorting

## When To Load
Load this when designing list endpoints, cursor or offset pagination, filter or sort syntax, sparse fields, `total_count`, collection links, or multi-item result semantics.

## Decision Rubric
- Add pagination up front for growable collections. Adding it later can break clients that assumed one response was complete.
- Prefer cursor pagination for mutable or large collections. Offset is exception-only when drift is acceptable.
- Make cursors opaque, URL-safe, short-lived when needed, and continuation-only. They must not encode authorization or expose offsets, timestamps, shard IDs, or query plans.
- Use this skill's default `page_size=50`, max `200`, unless the API already has a different policy.
- Define the end signal: absent or empty `next_cursor`, `Link: rel="next"` absence, or another explicit contract.
- Sorting must be deterministic. If the visible sort field is not unique, add a stable tie-breaker to the contract even if clients cannot choose it.
- Filters and sort fields are allow-list based. Unknown filters, unknown sort fields, and wrong types should fail unless a legacy compatibility rule says otherwise.
- Say whether pages are snapshot-like or live. For live pagination, disclose duplicate or skip risk and the recovery rule.
- Treat `total_count` as its own contract: exact, approximate, delayed, capped, or omitted.

## Imitate
```http
GET /v1/orders?page_size=50&sort=-created_at HTTP/1.1

HTTP/1.1 200 OK
Content-Type: application/json

{
  "data": [
    {
      "id": "ord_124",
      "created_at": "2026-03-10T18:31:00Z"
    }
  ],
  "pagination": {
    "next_cursor": "cur_01J8V7Q9Y0T8F",
    "page_size": 50
  }
}
```

Good: the response gives a continuation token. The contract still needs to say that `created_at` ties are broken by a stable internal key such as `id`.

```http
GET /v1/orders?status=paid&created_after=2026-01-01T00:00:00Z&page_size=25 HTTP/1.1
```

Good if `status` is a documented enum, `created_after` is RFC 3339 UTC, and unknown filters return a validation problem.

## Reject
```http
GET /v1/orders?page=5&sort=name HTTP/1.1
```

Bad for a mutable collection when `name` is not unique and page numbers can skip or duplicate rows.

```json
{ "items": [], "has_more": true }
```

Bad: clients do not know how to obtain the next page.

## Agent Traps
- Base64-encoding a database offset does not make a cursor contract opaque enough.
- Re-check authorization on every paginated request; a cursor is not an access token.
- Empty `data` with a `next_cursor` is surprising. Allow it only with an explicit live/degraded pagination rule.
- Ignoring unknown filters can leak more data than the caller meant to request.
- Sparse fields are a security surface. Use an allow list; never expose internal or sensitive fields by name.
- Changing default sort order, page size, or `total_count` meaning inside a major version is a compatibility change.

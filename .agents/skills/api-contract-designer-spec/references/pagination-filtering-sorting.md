# Pagination, Filtering, And Sorting

## When To Read This
Read this when designing collection endpoints, list responses, cursor or offset pagination, filter/query syntax, sort syntax, sparse field selection, `total_count`, collection links, or multi-item result semantics. Keep the output at the contract level; do not design SQL indexes, query planners, or cache internals here.

## Compact Principles
- Add pagination from the start for any collection that can grow. Adding it later can break clients that assumed a complete collection in one response.
- Prefer cursor pagination for mutable or large collections. Use offset only when the product need and data shape make drift acceptable.
- Make cursors opaque, URL-safe, and limited to pagination continuation. They must not grant authorization and must not expose database offsets, timestamps, shard IDs, or query plans.
- Define `page_size` default and maximum. This skill's repo default is `50` and max `200` unless the API's established policy says otherwise.
- Allow the server to return fewer items than requested. A short page does not necessarily mean end of collection unless the contract says it does.
- Use a clear end-of-collection signal, such as absent/empty `next_cursor` or no `Link: rel="next"`.
- Sorting must be deterministic. If the exposed sort field is not unique, append a stable tie-breaker in the contract or define the server's stable order.
- Filtering and sorting are allow-list based. Unknown filters, unknown sort fields, and wrong filter types should fail consistently.
- Keep filter and sort parameters stable across pages. If the client changes them while using a cursor, define whether the cursor is rejected or the new request starts a new result set.
- State whether list results are snapshot-like or live. For live pagination, disclose duplicate/skip risk under concurrent writes and the recovery rule.
- Treat `total_count` as a separate contract. If exposed, say whether it is exact, approximate, delayed, capped, or omitted by design.
- For bulk or multi-item operations, define all-or-nothing vs per-item result shapes instead of hiding partial failure behind a single success flag.

## Good Examples
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

Contract notes: `sort=-created_at` is stable only if the contract also uses an internal tie-breaker such as `id`, even if that tie-breaker is not exposed in the query syntax.

```http
GET /v1/orders?status=paid&created_after=2026-01-01T00:00:00Z&page_size=25 HTTP/1.1
```

Good contract rule: `status` accepts only documented enum values; `created_after` is RFC 3339 UTC; unknown filters fail with a validation problem.

## Bad Examples
```http
GET /v1/orders?page=5&sort=name HTTP/1.1
```

Why it can be bad: page numbers over a mutable collection can skip or duplicate results, and `name` may not be a deterministic order without a tie-breaker.

```json
{
  "items": [],
  "has_more": true
}
```

Why it is incomplete: clients do not know how to obtain the next page.

## Edge Cases That Often Fool Agents
- Base64-encoding a database offset does not make a cursor opaque enough if clients can infer or depend on its structure.
- A cursor should not encode authorization. Re-check authorization for every paginated request.
- Empty `data` with a `next_cursor` can be valid for live or degraded pagination only if documented; many clients otherwise treat an empty page as terminal.
- Unknown filters should not be ignored silently unless the API has an explicit legacy compatibility rule. Ignoring filters can leak more data than the caller meant to request.
- Changing default sort order or page size in the same major version is a semantic compatibility change.
- `total_count` can be expensive or stale. Omission is often better than implying exactness.
- Sparse field selection can become a security issue if internal or sensitive fields are selectable by name. Use an allow list.
- Link headers are useful for navigational links, but response-body cursors are often easier for JSON clients. Choose one primary contract and document it consistently.

## Source Links Gathered Through Exa
- Google AIP-158, Pagination: https://google.aip.dev/158
- Microsoft REST API Guidelines, Collections and Pagination: https://github.com/microsoft/api-guidelines/blob/master/Guidelines.md
- RFC 8288, Web Linking: https://www.rfc-editor.org/rfc/rfc8288.html
- OpenAPI Specification v3.1.1, parameters and schema modeling: https://spec.openapis.org/oas/v3.1.1.html

# Vendor Payout REST Contract (`v1`)

## Key Contract Choices
- Selected: `POST /v1/payouts` is always asynchronous and returns `202 Accepted` plus a status resource at `GET /v1/payout-operations/{operation_id}`.
- Rejected: blocking on the PSP and returning `201 Created` or `200 OK`. With 5-90 second PSP latency and common network timeouts, that would make retries ambiguous and increase duplicate-payout risk.
- Selected: `GET /v1/payouts/{payout_id}` is the authoritative read for one payout after the initiation operation succeeds.
- Selected: `GET /v1/payouts` is a payout-history projection and is explicitly eventual; it must disclose freshness with `as_of`.
- Rejected: using the history projection as the only read path. That would remove a clean read-after-write path for newly created payouts.

## Resource And Endpoint Matrix

### Resource Model
- `payout_initiation_operation`
  - Tracks the outcome of `POST /v1/payouts`.
  - Status enum: `pending`, `running`, `succeeded`, `failed`.
  - `succeeded` means the request was processed into a definitive payout resource or a definitive duplicate replay result. It does **not** mean funds have settled.
- `payout`
  - Represents the business payout after the initiation operation succeeds.
  - Status enum: `processing`, `paid`, `failed`, `canceled`.
  - `processing` means accepted into the payout flow but not yet terminal.

### Endpoint Matrix
| Endpoint | Semantics | Consistency | Success statuses | Retry class |
| --- | --- | --- | --- | --- |
| `POST /v1/payouts` | Submit a payout initiation request; creates or reuses a `payout_initiation_operation` | Strong for operation creation | `202 Accepted` | Retry-safe by contract with required `Idempotency-Key` |
| `GET /v1/payout-operations/{operation_id}` | Poll initiation progress and discover the resulting payout resource | Strong | `200 OK`, `304 Not Modified` | Retry-safe by protocol |
| `GET /v1/payouts/{payout_id}` | Read one payout | Strong | `200 OK`, `304 Not Modified` | Retry-safe by protocol |
| `GET /v1/payouts` | Read payout history | Eventual | `200 OK` | Retry-safe by protocol |

### Status And Error Matrix

#### `POST /v1/payouts`
- `202 Accepted`
  - Operation accepted for processing.
  - Response headers:
    - `Location: /v1/payout-operations/{operation_id}`
    - `Retry-After: 3` (advisory poll interval)
    - `X-Request-Id: {request_id}`
- `400 Bad Request`
  - Malformed JSON, unknown top-level fields, invalid JSON types, trailing tokens.
- `401 Unauthorized` / `403 Forbidden`
  - Caller is not authenticated or cannot initiate payouts in the current tenant/account.
- `404 Not Found`
  - Referenced `vendor_id` or `destination_id` does not exist in caller scope.
- `409 Conflict`
  - Same `Idempotency-Key` reused with a different request payload.
  - Other synchronous state conflicts, such as vendor or destination not eligible for payout.
- `422 Unprocessable Entity`
  - Semantically invalid request that can be rejected synchronously, such as non-positive amount or unsupported currency.
- `428 Precondition Required`
  - Missing required `Idempotency-Key`.
- `429 Too Many Requests`
  - Caller exceeded a rate limit; response includes `Retry-After`.
- `500 Internal Server Error`
  - Unexpected server failure.
- `503 Service Unavailable`
  - Temporary service unavailability before acceptance.

#### `GET /v1/payout-operations/{operation_id}`
- `200 OK`
- `304 Not Modified`
  - Returned when `If-None-Match` matches the current operation representation.
- `404 Not Found`
  - Unknown operation in caller scope.
- `410 Gone`
  - Operation resource expired after retention; clients should follow the saved payout link if they already have it.
- `401 Unauthorized` / `403 Forbidden`
- `429 Too Many Requests`
- `500 Internal Server Error`
- `503 Service Unavailable`

#### `GET /v1/payouts/{payout_id}`
- `200 OK`
- `304 Not Modified`
  - Returned when `If-None-Match` matches the current payout representation.
- `404 Not Found`
- `401 Unauthorized` / `403 Forbidden`
- `429 Too Many Requests`
- `500 Internal Server Error`
- `503 Service Unavailable`

#### `GET /v1/payouts`
- `200 OK`
- `400 Bad Request`
  - Unknown filter, invalid cursor, invalid `page_size`, or invalid sort field.
- `401 Unauthorized` / `403 Forbidden`
- `429 Too Many Requests`
- `500 Internal Server Error`
- `503 Service Unavailable`

## Request, Response, And Error Model

### `POST /v1/payouts` Request
`Content-Type: application/json`

Required header:
- `Idempotency-Key: <opaque-string>`

Body:

```json
{
  "vendor_id": "ven_123",
  "destination_id": "dst_456",
  "amount": {
    "value": "1250.00",
    "currency": "USD"
  },
  "client_reference": "withdrawal-2026-03-07-001",
  "metadata": {
    "batch_id": "batch-18"
  }
}
```

Field contract:
- `vendor_id`: opaque vendor identifier.
- `destination_id`: opaque payout destination identifier already registered for that vendor.
- `amount.value`: decimal string, never a JSON number.
- `amount.currency`: ISO 4217 uppercase currency code.
- `client_reference`: optional caller-supplied reference for reconciliation.
- `metadata`: optional JSON object for caller-defined bounded metadata.

### `202 Accepted` Response Body

```json
{
  "id": "pop_01JV6M9R7Q8Y4X5Z6A7B8C9D0E",
  "type": "payout_initiation_operation",
  "status": "pending",
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-07T12:00:00Z",
  "_links": {
    "self": "/v1/payout-operations/pop_01JV6M9R7Q8Y4X5Z6A7B8C9D0E"
  }
}
```

Notes:
- The response body is the operation resource representation.
- `Location` always points to the same operation URL that appears in `_links.self`.
- The response does not claim payout settlement, only accepted async processing.

### `GET /v1/payout-operations/{operation_id}` Response

In progress:

```json
{
  "id": "pop_01JV6M9R7Q8Y4X5Z6A7B8C9D0E",
  "type": "payout_initiation_operation",
  "status": "running",
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-07T12:00:04Z",
  "_links": {
    "self": "/v1/payout-operations/pop_01JV6M9R7Q8Y4X5Z6A7B8C9D0E"
  }
}
```

Succeeded:

```json
{
  "id": "pop_01JV6M9R7Q8Y4X5Z6A7B8C9D0E",
  "type": "payout_initiation_operation",
  "status": "succeeded",
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-07T12:00:07Z",
  "result": {
    "payout_id": "pout_01JV6M9VQ3HTP1M0J4S1F2G3H4",
    "href": "/v1/payouts/pout_01JV6M9VQ3HTP1M0J4S1F2G3H4"
  },
  "_links": {
    "self": "/v1/payout-operations/pop_01JV6M9R7Q8Y4X5Z6A7B8C9D0E",
    "payout": "/v1/payouts/pout_01JV6M9VQ3HTP1M0J4S1F2G3H4"
  }
}
```

Failed:

```json
{
  "id": "pop_01JV6M9R7Q8Y4X5Z6A7B8C9D0E",
  "type": "payout_initiation_operation",
  "status": "failed",
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-07T12:00:05Z",
  "error": {
    "type": "https://api.example.com/problems/payout-initiation-failed",
    "title": "Payout initiation failed",
    "status": 503,
    "detail": "The payout could not be initiated.",
    "code": "payout_initiation_failed"
  },
  "_links": {
    "self": "/v1/payout-operations/pop_01JV6M9R7Q8Y4X5Z6A7B8C9D0E"
  }
}
```

Operation semantics:
- `pending` or `running`: client should continue polling the operation resource.
- `succeeded`: payout resource exists and should be used for later settlement tracking.
- `failed`: the `POST` command did not complete into a definitive payout resource. Replaying the same `Idempotency-Key` returns the same failed operation; starting a new business attempt requires a new idempotency key.

### `GET /v1/payouts/{payout_id}` Response

```json
{
  "id": "pout_01JV6M9VQ3HTP1M0J4S1F2G3H4",
  "vendor_id": "ven_123",
  "destination_id": "dst_456",
  "amount": {
    "value": "1250.00",
    "currency": "USD"
  },
  "client_reference": "withdrawal-2026-03-07-001",
  "status": "processing",
  "created_at": "2026-03-07T12:00:07Z",
  "updated_at": "2026-03-07T12:00:07Z",
  "settled_at": null,
  "failure": null,
  "_links": {
    "self": "/v1/payouts/pout_01JV6M9VQ3HTP1M0J4S1F2G3H4"
  }
}
```

Field contract:
- `status`
  - `processing`: payout exists and is not terminal.
  - `paid`: terminal success.
  - `failed`: terminal failure; `failure` is populated.
  - `canceled`: terminal cancellation if the business later supports cancelation before settlement.
- `failure`
  - Object with stable machine-readable `code` and sanitized `detail`.
- `settled_at`
  - Timestamp only for terminal settlement outcomes.

### `GET /v1/payouts` Response

Query parameters:
- `cursor`
- `page_size` with default `50` and max `200`
- `vendor_id`
- `status`
- `created_after`
- `created_before`
- `sort`
  - Default: `-created_at`
  - Stable tie-breaker: `id`

Body:

```json
{
  "items": [
    {
      "id": "pout_01JV6M9VQ3HTP1M0J4S1F2G3H4",
      "vendor_id": "ven_123",
      "destination_id": "dst_456",
      "amount": {
        "value": "1250.00",
        "currency": "USD"
      },
      "status": "paid",
      "created_at": "2026-03-07T12:00:07Z",
      "updated_at": "2026-03-07T12:05:12Z",
      "settled_at": "2026-03-07T12:05:12Z"
    }
  ],
  "next_cursor": "eyJjcmVhdGVkX2F0IjoiMjAyNi0wMy0wN1QxMjowNToxMloiLCJpZCI6InBvdXRfMDEifQ==",
  "as_of": "2026-03-07T12:05:15Z",
  "sort": "-created_at"
}
```

History-read semantics:
- `as_of` is the freshness marker for the projection.
- The collection does **not** provide read-after-write guarantees.
- A payout returned by `GET /v1/payouts/{id}` may be absent from `GET /v1/payouts` until the projection catches up.

### Error Model
Errors use `application/problem+json`.

Base shape:

```json
{
  "type": "https://api.example.com/problems/idempotency-key-reuse-mismatch",
  "title": "Idempotency key reuse conflict",
  "status": 409,
  "detail": "The supplied Idempotency-Key was already used with a different request payload.",
  "instance": "/v1/payouts",
  "code": "idempotency_key_reuse_mismatch",
  "request_id": "req_01JV6M9S7Q8Y4X5Z6A7B8C9D0E",
  "errors": [
    {
      "field": "Idempotency-Key",
      "message": "Key already bound to a different request payload."
    }
  ]
}
```

Stable problem codes for this surface:
- `invalid_request`
- `idempotency_key_required`
- `idempotency_key_reuse_mismatch`
- `vendor_not_found`
- `destination_not_found`
- `payout_not_eligible`
- `rate_limited`
- `operation_not_found`
- `operation_gone`
- `payout_not_found`
- `internal_error`

## Boundary And Cross-Cutting Policies
- Media types
  - Success responses use `application/json`.
  - Error responses use `application/problem+json`.
- Strict JSON
  - Unknown fields are rejected.
  - Trailing tokens are rejected.
  - Malformed JSON is rejected with `400`.
- Retry and idempotency
  - `POST /v1/payouts` requires `Idempotency-Key`.
  - Idempotency scope is: authenticated tenant/account + `POST` + `/v1/payouts`.
  - Idempotency retention is at least `24h` from the first accepted request.
  - Same key + same semantically equivalent payload returns the same operation resource and the same `Location`.
  - Same key + different payload returns `409 Conflict`.
- Retry guidance for clients
  - Safe to retry `POST /v1/payouts` with the same `Idempotency-Key` after a client timeout, connection reset, or `5xx`/`429` response.
  - Same-key retries never create a second payout; they only return the original accepted or failed operation outcome.
  - `GET` endpoints are safe to retry without extra contract rules.
  - Clients must never retry a payout initiation with a new idempotency key unless they intend to create a new payout.
- Preconditions
  - No `If-Match` write precondition is required in `v1`.
  - The only mandatory write precondition is `Idempotency-Key`; missing header returns `428 Precondition Required`.
- Polling efficiency
  - `GET /v1/payout-operations/{operation_id}` and `GET /v1/payouts/{payout_id}` return `ETag`.
  - Clients may send `If-None-Match` and receive `304 Not Modified`.
- Correlation
  - Responses include `X-Request-Id`.
  - Problem details include `request_id`.
- Rate limits
  - `429` responses include `Retry-After`.
  - Rate limiting does not invalidate an existing idempotency key.
- Operation retention
  - Operation resources are retained for at least the idempotency window.
  - After retention expiry, `GET /v1/payout-operations/{operation_id}` may return `410 Gone`.

## Consistency And Async Notes
- `POST /v1/payouts` is explicitly async because the PSP can take 5-90 seconds.
- Operation terminality and payout terminality are different:
  - A terminal operation (`succeeded` or `failed`) ends the `POST` processing lifecycle.
  - A terminal payout (`paid`, `failed`, `canceled`) ends the business payout lifecycle.
- `operation.status = succeeded` means the client can now switch from polling the operation to polling the payout resource.
- `payout.status = processing` does not imply PSP acceptance is final; it only means the payout exists and settlement is still pending.
- `GET /v1/payouts/{id}` is the authoritative follow-up read after operation success.
- `GET /v1/payouts` is an eventual history view and must be treated as stale unless `as_of` is fresh enough for the client’s use case.

## Compatibility Notes
- Adding this new payout surface in `v1` is an `additive` change.
- The following are `behavior-change` or `breaking` changes inside `v1` and should not be made silently:
  - changing `POST /v1/payouts` from async `202` to sync `201`/`200`
  - changing idempotency from required to optional, or changing same-key replay semantics
  - changing `GET /v1/payouts` from eventual to strong, or from strong item reads to eventual item reads
  - reclassifying error mappings such as `428` for missing `Idempotency-Key` or `409` for same-key/different-payload
- Safe additive growth in `v1`:
  - adding optional response fields
  - adding new problem `code` values
  - adding new filter fields only if unknown-filter failure behavior remains unchanged and the new filters are documented
- Clients should ignore unknown object fields. Clients should **not** assume new enum values can appear without a versioning decision; enum expansion should be treated as at least a `behavior-change`.

## Open Questions And Risks
- Should failed PSP handoff attempts still create a durable payout resource for audit/history, or should some failures remain operation-only failures? This contract currently treats `operation.failed` as “no definitive payout resource exists.”
- Can the service commit to a bounded freshness target for the history projection, or should `as_of` be the only freshness disclosure in `v1`?
- Is `destination_id` always caller-supplied, or can the service derive it from vendor configuration? That choice changes request-body strictness.
- Is `24h` idempotency retention sufficient for payout initiation in this domain, or do business/settlement timelines require a longer duplicate-suppression window?
- Does `canceled` belong in the initial `payout.status` enum, or should cancellation be omitted until a real cancel flow exists?

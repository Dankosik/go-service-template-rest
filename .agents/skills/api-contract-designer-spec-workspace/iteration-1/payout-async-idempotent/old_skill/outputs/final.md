# Vendor Payout Initiation REST Contract

## Resource And Endpoint Matrix

### Selected shape
- Selected: `POST /v1/payouts` is an async start endpoint that returns a `payout-operation` resource.
- Rejected: blocking `POST /v1/payouts` that waits for the PSP and returns only `201 Created`. This was rejected because 5-90 second PSP latency plus common network timeouts would make client retries ambiguous and duplicate-prone.
- Selected: `GET /v1/payouts/{payout_id}` is the authoritative single-resource read, while `GET /v1/payouts` is the eventual-consistency history read.
- Rejected: making every read path eventual. This was rejected because an immediate follow-up lookup after `POST` would otherwise be unable to distinguish "not found" from "not projected yet".

### Resource model
- `payout`: the business resource representing one vendor payout from acceptance through terminal settlement outcome.
- `payout-operation`: the operation resource representing one accepted `POST /v1/payouts` request. It is authoritative for command-processing progress and idempotent replay recovery, not for long-term payout history.
- `destination`: referenced by `destination_id`. Destination-management endpoints are out of scope here; the contract assumes `destination_id` refers to an existing verified payout destination.

### Endpoint matrix

| Endpoint | Consistency | Purpose | Success statuses | Retry classification | Notes |
| --- | --- | --- | --- | --- | --- |
| `POST /v1/payouts` | strong acceptance | Start payout initiation | `202 Accepted` for a new accepted request, `200 OK` for idempotent replay | retry-safe by contract only with `Idempotency-Key` | Returns a `payout-operation` representation and `Location` to the operation resource. |
| `GET /v1/payout-operations/{operation_id}` | strong | Poll the initiation operation | `200 OK`, `304 Not Modified` | retry-safe by protocol | Operation reaches terminal state before payout settlement necessarily does. |
| `GET /v1/payouts/{payout_id}` | strong | Read one payout’s latest authoritative state | `200 OK`, `304 Not Modified` | retry-safe by protocol | Recommended follow-up read after operation success. |
| `GET /v1/payouts` | eventual | Read payout history/projection | `200 OK` | retry-safe by protocol | No read-after-write guarantee. Includes freshness disclosure. |

### Status semantics

#### `payout-operation.status`
- `pending`: request accepted and durably recorded, work not yet started.
- `running`: initiation workflow is in progress.
- `succeeded`: the API-side initiation workflow reached a definitive result. This does not imply the payout has settled successfully; clients must read `payout.status`.
- `failed`: the initiation workflow reached a terminal failure for the accepted payout. Because v1 allocates `payout_id` at acceptance time, the operation still keeps a `resource` link and the payout resource will typically be in terminal `failed` state.

#### `payout.status`
- `accepted`: payout resource exists and the request was accepted, but PSP handoff is not yet complete.
- `processing`: payout is with the PSP or downstream workflow and final settlement is still pending.
- `paid`: terminal success.
- `failed`: terminal failure.

## Request, Response, And Error Model

### `POST /v1/payouts`

#### Request
- `Content-Type: application/json` is required.
- `Idempotency-Key` header is required.
- `traceparent` is optional and passed through for correlation.
- Caller identity determines payout ownership. No `vendor_id` header or body field is accepted in v1.

Example request body:

```json
{
  "amount": {
    "value": "125.00",
    "currency": "USD"
  },
  "destination_id": "dst_01HV6K4Q9D4TB7Z7D5Q0Q8M9PE",
  "client_reference": "weekly-payout-2026-03-07",
  "description": "Weekly vendor payout",
  "metadata": {
    "batch_id": "w10-2026"
  }
}
```

Field rules:
- `amount.value`: string decimal, positive, and valid for the currency’s supported minor-unit precision.
- `amount.currency`: uppercase ISO 4217 code.
- `destination_id`: required; must reference a verified payout destination visible to the authenticated caller.
- `client_reference`: optional opaque caller value, max `128` chars, echoed in reads; uniqueness is not guaranteed by this contract.
- `description`: optional human-readable text, max `140` chars.
- `metadata`: optional object, max `20` key/value pairs, keys max `50` chars, values max `200` chars.
- Unknown fields are rejected.
- Request body size limit: `16 KiB`.

#### Success response

New accepted request:
- Status: `202 Accepted`
- Headers:
  - `Location: /v1/payout-operations/{operation_id}`
  - `Retry-After` may be returned while the operation is non-terminal
  - `Request-Id`

Idempotent replay of the same request:
- Status: `200 OK`
- Headers:
  - `Location: /v1/payout-operations/{operation_id}`
  - `Request-Id`

Response body for both:

```json
{
  "id": "op_01HV6K8W4QX7ZX0RAA7M5J19TH",
  "type": "payout-operation",
  "status": "running",
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-07T12:00:03Z",
  "resource": {
    "type": "payout",
    "id": "po_01HV6K8W98J6CYM18R6K30T8P8",
    "href": "/v1/payouts/po_01HV6K8W98J6CYM18R6K30T8P8"
  },
  "result": null,
  "error": null,
  "links": {
    "self": "/v1/payout-operations/op_01HV6K8W4QX7ZX0RAA7M5J19TH",
    "payout": "/v1/payouts/po_01HV6K8W98J6CYM18R6K30T8P8"
  }
}
```

Rules:
- `resource.id` is allocated at acceptance time so the client gets a stable payout identifier immediately.
- `status=succeeded` means the initiation workflow is done; the client must still inspect `payout.status`.
- If `status=failed`, `error` contains structured failure details and `result` is `null`.

#### Error status matrix for `POST /v1/payouts`

| Status | When used | Retry guidance |
| --- | --- | --- |
| `400 Bad Request` | malformed JSON, trailing tokens, unknown fields, unsupported query/body shape | do not retry unchanged |
| `401 Unauthorized` | missing or invalid authentication | retry only after re-authentication |
| `403 Forbidden` | caller is authenticated but cannot initiate payouts against the resolved account/destination | do not retry unchanged |
| `409 Conflict` | idempotency key reused with a different canonical request, or request conflicts with current payout state/business invariants | do not retry unchanged |
| `415 Unsupported Media Type` | body is not `application/json` | do not retry unchanged |
| `422 Unprocessable Content` | semantically invalid input after decoding, such as unsupported currency/destination combination | do not retry unchanged |
| `428 Precondition Required` | missing `Idempotency-Key` | retry with the header |
| `429 Too Many Requests` | rate limit exceeded | retry after `Retry-After` using the same `Idempotency-Key` and identical body |
| `503 Service Unavailable` | the service cannot durably accept the request, such as idempotency store or intake path unavailable | retry with the same `Idempotency-Key` and identical body |
| `500 Internal Server Error` | unexpected server failure before a deterministic contract outcome can be returned | retry with the same `Idempotency-Key` and identical body |

### `GET /v1/payout-operations/{operation_id}`

- Returns the current operation representation.
- Success: `200 OK`
- Conditional read: supports `ETag` and `If-None-Match`; unchanged resource returns `304 Not Modified`.
- Not found: `404 Not Found`

Terminal success example:

```json
{
  "id": "op_01HV6K8W4QX7ZX0RAA7M5J19TH",
  "type": "payout-operation",
  "status": "succeeded",
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-07T12:00:19Z",
  "resource": {
    "type": "payout",
    "id": "po_01HV6K8W98J6CYM18R6K30T8P8",
    "href": "/v1/payouts/po_01HV6K8W98J6CYM18R6K30T8P8"
  },
  "result": {
    "accepted": true,
    "payout_status": "processing"
  },
  "error": null
}
```

Terminal failure example:

```json
{
  "id": "op_01HV6K8W4QX7ZX0RAA7M5J19TH",
  "type": "payout-operation",
  "status": "failed",
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-07T12:00:21Z",
  "resource": {
    "type": "payout",
    "id": "po_01HV6K8W98J6CYM18R6K30T8P8",
    "href": "/v1/payouts/po_01HV6K8W98J6CYM18R6K30T8P8"
  },
  "result": null,
  "error": {
    "type": "urn:problem-type:payout:temporarily-unavailable",
    "title": "Payout initiation failed",
    "status": 503,
    "detail": "The payout could not be handed off for processing.",
    "code": "temporarily_unavailable",
    "request_id": "req_01HV6K8YPM2E4X25D7C3X7M5V0"
  }
}
```

### `GET /v1/payouts/{payout_id}`

- Returns the authoritative current payout state.
- Success: `200 OK`
- Conditional read: supports `ETag` and `If-None-Match`; unchanged resource returns `304 Not Modified`.
- Not found: `404 Not Found`

Example:

```json
{
  "id": "po_01HV6K8W98J6CYM18R6K30T8P8",
  "type": "payout",
  "status": "processing",
  "amount": {
    "value": "125.00",
    "currency": "USD"
  },
  "destination_id": "dst_01HV6K4Q9D4TB7Z7D5Q0Q8M9PE",
  "client_reference": "weekly-payout-2026-03-07",
  "description": "Weekly vendor payout",
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-07T12:00:19Z",
  "settled_at": null,
  "failure": null,
  "links": {
    "self": "/v1/payouts/po_01HV6K8W98J6CYM18R6K30T8P8"
  }
}
```

Terminal failure uses:

```json
"failure": {
  "code": "psp_rejected",
  "message": "The payout was rejected by the payment provider.",
  "retryable": false
}
```

`failure.message` must be sanitized and stable enough for clients; raw PSP text or infrastructure detail must not be exposed.

### `GET /v1/payouts`

Query parameters:
- `cursor`: opaque cursor for pagination.
- `page_size`: default `50`, max `200`.
- `status`: repeatable or comma-separated whitelist of payout statuses.
- `created_from`, `created_to`: RFC 3339 timestamps.
- `destination_id`
- `client_reference`
- `sort`: only `-created_at` in v1, with `id` as the server-side tie-breaker.
- `fields`: optional sparse-field whitelist. Allowed values: `id,status,amount,created_at,updated_at,client_reference,destination_id,settled_at`.

Unknown filters, invalid sort fields, or unsupported sparse fields fail with `400 Bad Request`.

Example response:

```json
{
  "items": [
    {
      "id": "po_01HV6K8W98J6CYM18R6K30T8P8",
      "type": "payout",
      "status": "paid",
      "amount": {
        "value": "125.00",
        "currency": "USD"
      },
      "client_reference": "weekly-payout-2026-03-07",
      "destination_id": "dst_01HV6K4Q9D4TB7Z7D5Q0Q8M9PE",
      "created_at": "2026-03-07T12:00:00Z",
      "updated_at": "2026-03-07T12:01:10Z",
      "settled_at": "2026-03-07T12:01:10Z"
    }
  ],
  "next_cursor": "g2wAAAABaANkAA...",
  "freshness": {
    "consistency": "eventual",
    "as_of": "2026-03-07T12:01:12Z",
    "may_exclude_recent_writes": true
  }
}
```

## Boundary And Cross-Cutting Policies

- Success payload media type is `application/json`. Error payload media type is `application/problem+json`.
- Problem Details fields are always `type`, `title`, `status`, `detail`; stable extensions are `code`, `request_id`, and optional `errors`.
- Problem `type` values are stable URIs, for example:
  - `urn:problem-type:payout:idempotency-key-required`
  - `urn:problem-type:payout:idempotency-key-conflict`
  - `urn:problem-type:payout:invalid-request`
  - `urn:problem-type:payout:temporarily-unavailable`
- Field-level validation errors use `errors[]` items with `field`, `code`, and `message`.
- Boundary evaluation order is:
  1. authenticate and derive caller-owned payout scope
  2. enforce transport and size limits
  3. strict JSON decode
  4. normalize and validate input
  5. evaluate idempotency
  6. apply business acceptance rules
  7. persist acceptance and start async work
- Idempotency rules:
  - `Idempotency-Key` is mandatory for `POST /v1/payouts`.
  - Scope is `(authenticated payout owner, HTTP method, route template)`.
  - Same key plus same canonical request returns the same logical operation and the same `payout_id` for at least `24h`.
  - Same key plus different canonical request returns `409 Conflict`.
  - Requests rejected before idempotency evaluation, such as malformed JSON or missing auth, are not guaranteed to create an idempotency record.
- Retry rules:
  - If the client sees a timeout, connection reset, `429`, `500`, or `503` after sending `POST /v1/payouts`, it must retry with the same `Idempotency-Key` and byte-equivalent semantics.
  - The client must not change the body while reusing the same key.
  - `GET` endpoints are retry-safe without special headers.
- Conditional requests:
  - `GET /v1/payout-operations/{operation_id}` and `GET /v1/payouts/{payout_id}` emit `ETag` and support `If-None-Match`.
  - No `If-Match` precondition is required in v1 because there are no client-driven update endpoints for these resources.
- Rate limiting:
  - `429 Too Many Requests` returns `Retry-After`.
  - Replayed `POST` retries after `429` must keep the original idempotency key.
- Correlation:
  - The API accepts `traceparent` if provided.
  - The API returns `Request-Id` on both success and error responses.
  - `request_id` is echoed inside problem details.

## Consistency And Async Notes

- `POST /v1/payouts` acknowledges acceptance of the request, not payout settlement.
- `payout-operation` is the authoritative resource for command progress. It exists so clients can safely recover from ambiguous outcomes caused by timeouts or dropped responses.
- `payout` is the authoritative business resource for payout state. Its lifecycle can continue after the operation reaches `succeeded`.
- `GET /v1/payouts` is explicitly eventual. It must not be used as an acceptance confirmation path.
- History responses disclose freshness via `freshness.consistency=eventual` and `freshness.as_of`.
- No read-after-write guarantee is made for payout history in v1.
- Clients that need definitive near-real-time state should use:
  1. `POST /v1/payouts`
  2. `GET /v1/payout-operations/{operation_id}` until terminal
  3. `GET /v1/payouts/{payout_id}` for current payout state

## Compatibility Notes

- This v1 contract is selected as an async contract. Changing `POST /v1/payouts` from `202/200 + payout-operation` to a blocking `201` flow is a behavior change and should be treated as breaking for practical client compatibility.
- Additive changes:
  - adding optional response fields
  - adding new filter parameters
  - adding new problem `code` values
  - adding new non-terminal payout statuses if clients are required to treat status enums as open
- Behavior changes:
  - changing idempotency retention below `24h`
  - changing replay behavior from `200 OK` to some other mapping
  - changing the moment when `payout-operation.status` becomes `succeeded`
  - changing `GET /v1/payouts` consistency from eventual to strong or vice versa
- Breaking changes:
  - making `Idempotency-Key` optional or renaming it
  - changing `amount.value` from string decimal to JSON number
  - removing defined status values or changing their meaning
  - removing `freshness` from the history endpoint
- Clients must treat unknown object fields and future problem `code` values as ignorable, and unknown payout statuses as non-terminal unless explicitly documented otherwise.

## Open Questions And Risks

- Is the caller always acting for exactly one vendor/account from authentication context, or must the API support platform actors creating payouts on behalf of many vendors? If the latter is required, the ownership selector must be explicit and authorized in the contract.
- Is `destination_id` always required, or is there a single default settlement destination per vendor? The current contract assumes an explicit destination reference.
- Does the system truly support an authoritative `GET /v1/payouts/{payout_id}` path, or are all payout reads projection-backed? If all reads are eventual, the contract should keep item lookup eventual too and make the operation resource the only strong immediate read surface.
- Should `client_reference` be uniqueness-constrained per vendor for longer-term business deduplication? The current contract only echoes it and relies on `Idempotency-Key` for retry safety.
- Are webhooks or callbacks needed in addition to polling for terminal payout outcomes? They are out of scope in this contract.

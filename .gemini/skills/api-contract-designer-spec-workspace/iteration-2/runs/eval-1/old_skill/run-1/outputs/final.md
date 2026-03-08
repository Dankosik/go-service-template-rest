# Vendor Payout REST Contract

## Contract Decision

Selected design:
- `POST /v1/payouts` is an asynchronous start endpoint.
- The server creates two client-visible resources:
  - `payout`: the business resource that represents the payout lifecycle.
  - `payout-initiation operation`: the control resource that tracks the long-running initiation step.

Rejected options:
- Synchronous `201 Created` from `POST /v1/payouts`: rejected because PSP latency is commonly `5-90s`, client network timeouts are common, and a synchronous contract would blur accept-vs-complete semantics.
- `POST /v1/payouts` returning only a payout resource without an operation resource: rejected because payout-history reads are eventually consistent, so clients need a strong polling target after timeout/retry ambiguity.

Compatibility class of this design:
- New API surface in `v1`: additive.
- Changing this contract later from async to fake-sync, or from strong operation polling to projection-only reads: behavior change.

## Resource And Endpoint Matrix

### Resources

`payout`
- Opaque identifier: `po_<opaque>`
- Represents the payout from request acceptance through final settlement.
- Lifecycle status enum:
  - `initiating`: request durably accepted; initiation with the PSP is not yet terminal.
  - `processing`: initiation completed and payout is waiting for final settlement.
  - `paid`: terminal success.
  - `failed`: terminal failure.
  - `canceled`: terminal cancellation before settlement.

`payout-initiation operation`
- Opaque identifier: `op_<opaque>`
- Represents only the initiation workflow started by `POST /v1/payouts`.
- Operation status enum:
  - `pending`
  - `running`
  - `succeeded`
  - `failed`
  - `canceled`
- `succeeded` means the initiation step finished and the payout moved into its business lifecycle.
- `succeeded` does not mean the payout is settled.

### Endpoint Matrix

| Endpoint | Purpose | Consistency | Success | Error Semantics | Retry Classification |
| --- | --- | --- | --- | --- | --- |
| `POST /v1/payouts` | Start payout initiation | strong acceptance | `202 Accepted` | `400`, `401`, `403`, `409`, `415`, `422`, `428`, `429`, `503` | retry-safe by contract only when `Idempotency-Key` is present |
| `GET /v1/payout-operations/{operation_id}` | Poll initiation status | strong | `200 OK` | `401`, `403`, `404`, `410`, `429` | safe + idempotent |
| `GET /v1/payouts/{payout_id}` | Read one payout | eventual projection | `200 OK`, `304 Not Modified` | `401`, `403`, `404`, `429` | safe + idempotent |
| `GET /v1/payouts` | Read payout history | eventual projection | `200 OK` | `400`, `401`, `403`, `429` | safe + idempotent |

### `POST /v1/payouts`

Required request headers:
- `Authorization: Bearer <token>`
- `Content-Type: application/json`
- `Idempotency-Key: <1..128 ASCII chars>`

Optional request headers:
- `traceparent`
- `X-Request-Id`

Success response headers:
- `Location: /v1/payout-operations/{operation_id}`
- `Retry-After: 5` while the operation is non-terminal
- `X-Request-Id: <request-id>`

Request body:

```json
{
  "vendor_id": "ven_01JPK7V3YK0VZ0SC7AV2Y3QZ9N",
  "destination_account_id": "dst_01JPK80R4MW8B0P9P6M9W3N6C7",
  "amount": {
    "currency": "USD",
    "value": "1250.00"
  },
  "client_reference": "march-affiliate-2026-00017",
  "memo": "March affiliate payout"
}
```

Field contract:
- `vendor_id`: required opaque vendor identifier scoped to the authenticated tenant.
- `destination_account_id`: required opaque identifier for a previously registered payout destination.
- `amount.currency`: required ISO 4217 uppercase code.
- `amount.value`: required positive decimal string, max 18 digits total, max 2 fraction digits.
- `client_reference`: optional client-supplied business reference, max `128` chars, echoed in reads.
- `memo`: optional human-readable text, max `140` chars.

Successful `202 Accepted` response:

```json
{
  "operation": {
    "id": "op_01JPK85TXSAC6K3Q1SZM4XEPK4",
    "kind": "payout-initiation",
    "status": "pending",
    "created_at": "2026-03-07T12:00:01Z",
    "updated_at": "2026-03-07T12:00:01Z",
    "payout_id": "po_01JPK85TXS8FS7X2Q1R2F7ZJ05",
    "links": {
      "self": "/v1/payout-operations/op_01JPK85TXSAC6K3Q1SZM4XEPK4",
      "payout": "/v1/payouts/po_01JPK85TXS8FS7X2Q1R2F7ZJ05"
    }
  },
  "payout": {
    "id": "po_01JPK85TXS8FS7X2Q1R2F7ZJ05",
    "status": "initiating",
    "vendor_id": "ven_01JPK7V3YK0VZ0SC7AV2Y3QZ9N",
    "destination_account_id": "dst_01JPK80R4MW8B0P9P6M9W3N6C7",
    "amount": {
      "currency": "USD",
      "value": "1250.00"
    },
    "client_reference": "march-affiliate-2026-00017",
    "created_at": "2026-03-07T12:00:01Z",
    "updated_at": "2026-03-07T12:00:01Z",
    "status_updated_at": "2026-03-07T12:00:01Z",
    "links": {
      "self": "/v1/payouts/po_01JPK85TXS8FS7X2Q1R2F7ZJ05",
      "operation": "/v1/payout-operations/op_01JPK85TXSAC6K3Q1SZM4XEPK4"
    }
  }
}
```

`POST /v1/payouts` status mapping:
- `202 Accepted`: request durably accepted; operation and payout identifiers allocated.
- `400 Bad Request`: malformed JSON, duplicate JSON keys, trailing tokens, or invalid query/header syntax.
- `401 Unauthorized`: missing or invalid authentication.
- `403 Forbidden`: authenticated principal cannot initiate payouts for this tenant.
- `409 Conflict`: same `Idempotency-Key` reused with a different effective request payload.
- `415 Unsupported Media Type`: request body is not `application/json`.
- `422 Unprocessable Content`: semantic validation failed after strict decode.
- `428 Precondition Required`: `Idempotency-Key` missing.
- `429 Too Many Requests`: rate-limited; `Retry-After` included.
- `503 Service Unavailable`: server could not durably accept the request or persist idempotency state; safe to retry with the same `Idempotency-Key`.

### `GET /v1/payout-operations/{operation_id}`

Response body:

```json
{
  "id": "op_01JPK85TXSAC6K3Q1SZM4XEPK4",
  "kind": "payout-initiation",
  "status": "running",
  "created_at": "2026-03-07T12:00:01Z",
  "updated_at": "2026-03-07T12:00:10Z",
  "completed_at": null,
  "payout_id": "po_01JPK85TXS8FS7X2Q1R2F7ZJ05",
  "result": null,
  "error": null,
  "links": {
    "self": "/v1/payout-operations/op_01JPK85TXSAC6K3Q1SZM4XEPK4",
    "payout": "/v1/payouts/po_01JPK85TXS8FS7X2Q1R2F7ZJ05"
  }
}
```

Terminal success example:

```json
{
  "id": "op_01JPK85TXSAC6K3Q1SZM4XEPK4",
  "kind": "payout-initiation",
  "status": "succeeded",
  "created_at": "2026-03-07T12:00:01Z",
  "updated_at": "2026-03-07T12:00:19Z",
  "completed_at": "2026-03-07T12:00:19Z",
  "payout_id": "po_01JPK85TXS8FS7X2Q1R2F7ZJ05",
  "result": {
    "payout_status": "processing"
  },
  "error": null,
  "links": {
    "self": "/v1/payout-operations/op_01JPK85TXSAC6K3Q1SZM4XEPK4",
    "payout": "/v1/payouts/po_01JPK85TXS8FS7X2Q1R2F7ZJ05"
  }
}
```

Terminal failure example:

```json
{
  "id": "op_01JPK85TXSAC6K3Q1SZM4XEPK4",
  "kind": "payout-initiation",
  "status": "failed",
  "created_at": "2026-03-07T12:00:01Z",
  "updated_at": "2026-03-07T12:00:22Z",
  "completed_at": "2026-03-07T12:00:22Z",
  "payout_id": "po_01JPK85TXS8FS7X2Q1R2F7ZJ05",
  "result": null,
  "error": {
    "type": "https://api.example.com/problems/psp-initiation-failed",
    "title": "Payout initiation failed",
    "detail": "The payout could not be initiated with the payment service provider.",
    "code": "psp_initiation_failed",
    "retryable": false,
    "request_id": "req_01JPK866Q3R6A7G4VQ2K8YKS0D"
  },
  "links": {
    "self": "/v1/payout-operations/op_01JPK85TXSAC6K3Q1SZM4XEPK4",
    "payout": "/v1/payouts/po_01JPK85TXS8FS7X2Q1R2F7ZJ05"
  }
}
```

Operation read semantics:
- `200 OK`: operation exists; body always includes the current operation status.
- `Retry-After` should be returned while status is `pending` or `running`.
- `404 Not Found`: unknown operation ID or operation outside tenant scope.
- `410 Gone`: operation existed but is past retention; clients must use the payout resource if they already have `payout_id`.

### `GET /v1/payouts/{payout_id}`

Response body:

```json
{
  "id": "po_01JPK85TXS8FS7X2Q1R2F7ZJ05",
  "status": "processing",
  "vendor_id": "ven_01JPK7V3YK0VZ0SC7AV2Y3QZ9N",
  "destination_account_id": "dst_01JPK80R4MW8B0P9P6M9W3N6C7",
  "amount": {
    "currency": "USD",
    "value": "1250.00"
  },
  "client_reference": "march-affiliate-2026-00017",
  "created_at": "2026-03-07T12:00:01Z",
  "updated_at": "2026-03-07T12:00:19Z",
  "status_updated_at": "2026-03-07T12:00:19Z",
  "view_as_of": "2026-03-07T12:00:22Z",
  "consistency": "eventual",
  "failure": null,
  "links": {
    "self": "/v1/payouts/po_01JPK85TXS8FS7X2Q1R2F7ZJ05",
    "operation": "/v1/payout-operations/op_01JPK85TXSAC6K3Q1SZM4XEPK4"
  }
}
```

Read semantics:
- `200 OK`: payout exists in the read model.
- `304 Not Modified`: when `If-None-Match` matches the current representation `ETag`.
- `404 Not Found`: payout is unknown, outside tenant scope, or not yet visible in the eventually consistent projection.
- For the `not yet visible` case, the response body uses Problem Details with code `payout_not_yet_visible` and includes `retry_after_seconds`.

### `GET /v1/payouts`

Query parameters:
- `cursor`: opaque cursor for forward pagination.
- `page_size`: default `50`, max `200`.
- `vendor_id`: optional exact-match filter.
- `status`: optional exact-match filter using payout status values.
- `client_reference`: optional exact-match filter.
- `created_from`: optional inclusive RFC 3339 timestamp filter.
- `created_to`: optional exclusive RFC 3339 timestamp filter.
- `sort`: optional, supported values `created_at` and `-created_at`. Stable tie-breaker is always `id`.

Unknown filters or unsupported sort fields fail with `400 Bad Request`.

Response body:

```json
{
  "items": [
    {
      "id": "po_01JPK85TXS8FS7X2Q1R2F7ZJ05",
      "status": "processing",
      "vendor_id": "ven_01JPK7V3YK0VZ0SC7AV2Y3QZ9N",
      "amount": {
        "currency": "USD",
        "value": "1250.00"
      },
      "client_reference": "march-affiliate-2026-00017",
      "created_at": "2026-03-07T12:00:01Z",
      "updated_at": "2026-03-07T12:00:19Z",
      "status_updated_at": "2026-03-07T12:00:19Z",
      "failure": null,
      "links": {
        "self": "/v1/payouts/po_01JPK85TXS8FS7X2Q1R2F7ZJ05"
      }
    }
  ],
  "next_cursor": "eyJjcmVhdGVkX2F0IjoiMjAyNi0wMy0wN1QxMjowMDowMVoiLCJpZCI6InBvXzAxSlBLOD...",
  "view_as_of": "2026-03-07T12:00:22Z",
  "consistency": "eventual"
}
```

## Request, Response, And Error Model

### Common Response Rules

- Success payloads are `application/json`.
- Error payloads are `application/problem+json`.
- Embedded `operation.error` objects reuse the stable Problem Details fields but omit HTTP `status` because they live inside a successful `200 OK` operation representation.
- All resource IDs are opaque and stable for the life of the resource.
- Timestamps use RFC 3339 UTC.
- Unknown response fields must be ignored by clients.

### Problem Details Shape

```json
{
  "type": "https://api.example.com/problems/validation",
  "title": "Request validation failed",
  "status": 422,
  "detail": "One or more fields are invalid.",
  "code": "validation_failed",
  "request_id": "req_01JPK866Q3R6A7G4VQ2K8YKS0D",
  "errors": [
    {
      "field": "amount.value",
      "message": "must be greater than zero"
    }
  ]
}
```

Stable error codes for this surface:
- `validation_failed`
- `idempotency_key_required`
- `idempotency_key_reused`
- `rate_limited`
- `service_unavailable`
- `payout_not_found`
- `payout_not_yet_visible`
- `operation_not_found`
- `psp_initiation_failed`
- `unsupported_media_type`

Error mapping rules:
- Use `400` for malformed transport-level input.
- Use `422` for syntactically valid JSON that fails field or business validation.
- Use `409` for idempotency-key payload mismatch.
- Use `428` when a required idempotency precondition is absent.

## Boundary And Cross-Cutting Policies

### Idempotency And Retry Rules

- `POST /v1/payouts` requires `Idempotency-Key`.
- Idempotency key scope is the tuple:
  - authenticated tenant
  - HTTP method
  - route template
  - effective request payload
- Minimum dedup retention: `24h`.
- Reusing the same key with the same effective payload returns the same `operation.id` and `payout.id`.
- A replayed accepted request still returns `202 Accepted`; the `operation.status` in the body reflects the current state at replay time.
- Reusing the same key with a different effective payload returns `409 Conflict`.
- If the client times out or loses the response, it must retry with the same `Idempotency-Key` until it receives a response or can resolve the operation state.
- `GET` endpoints are safe and can be retried without special handling.

### Preconditions And Concurrency

- No `If-Match` or optimistic-write preconditions are defined for this surface because only create and read endpoints are in scope.
- `GET /v1/payouts/{payout_id}` may return `ETag`.
- Clients may use `If-None-Match` on `GET /v1/payouts/{payout_id}` and receive `304 Not Modified`.

### Validation And Decode Rules

- Request bodies must be strict JSON objects.
- Unknown fields are rejected.
- Duplicate keys are rejected.
- Trailing tokens are rejected.
- `vendor_id`, `destination_account_id`, and `amount` are required.
- `amount.value` must be positive.
- Tenant context comes from validated authentication, not caller-supplied tenant headers.

### Rate Limit And Correlation

- `429 Too Many Requests` includes `Retry-After`.
- Every response includes `X-Request-Id`.
- `traceparent` is accepted and propagated when supplied.

## Consistency And Async Notes

- Acceptance path:
  - `POST /v1/payouts` is strong only for durable acceptance of the request and issuance of `operation.id` and `payout.id`.
- Initiation status path:
  - `GET /v1/payout-operations/{operation_id}` is the strong polling endpoint.
  - Clients that need immediate post-create certainty must poll the operation resource, not the payout read model.
- Payout read path:
  - `GET /v1/payouts/{payout_id}` and `GET /v1/payouts` are eventual.
  - Both disclose `view_as_of`.
  - No read-after-write guarantee is made for payout reads.
- Meaning of state progression:
  - `operation.status=succeeded` means initiation completed.
  - Final settlement is conveyed only on `payout.status`.
  - Typical progression is `initiating -> processing -> paid|failed|canceled`.
- Visibility lag:
  - A payout may exist and have a valid `payout_id` before it becomes visible in `GET /v1/payouts/{payout_id}` or `GET /v1/payouts`.
  - That lag is disclosed by the `payout_not_yet_visible` error code and the `view_as_of` freshness field.

## Compatibility Notes

Additive changes:
- adding optional response fields
- adding new optional request fields
- adding new optional filters
- adding new Problem Details extension members

Behavior-changing changes:
- changing idempotency retention or replay semantics
- changing any endpoint from strong to eventual or eventual to strong
- changing `202 Accepted` initiation semantics
- changing `404 payout_not_yet_visible` handling for projection lag

Breaking changes:
- making `Idempotency-Key` optional or changing it to a different required header
- removing fields from success or error payloads
- changing field types or path shapes
- adding new required request fields
- changing closed status enums without versioning

Client compatibility expectation:
- clients must ignore unknown object fields
- clients must not treat payout-history ordering as more precise than the documented sort + eventual-consistency contract
- clients must distinguish operation success from payout settlement success

## Open Questions And Risks

- Should the API expose a webhook or callback contract for final settlement, or is polling-only acceptable for `processing` payouts?
- Is `destination_account_id` always a pre-registered object, or must the create contract support inline destination details?
- Are payout status values above sufficient, or do business workflows require separate states such as `submitted`, `rejected`, or `reversed`?
- What is the exact operation-resource retention window after terminal state, and is `410 Gone` acceptable after expiry?
- Should `client_reference` be unique per tenant or per vendor, or is it purely informational?

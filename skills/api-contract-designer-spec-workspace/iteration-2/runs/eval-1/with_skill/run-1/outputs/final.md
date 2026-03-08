# Vendor Payout REST Contract

## Contract Framing And Assumptions

Client-visible problem: authenticated tenant-scoped clients need to initiate vendor payouts over an unreliable network, recover safely from `POST /v1/payouts` timeouts, and observe both PSP-submission progress and final settlement outcome without relying on synchronous completion.

Audience and trust boundary:
- The caller is an authenticated API client operating within one tenant or merchant account.
- Tenant scope comes from validated identity, not from caller-supplied tenant headers.
- `vendor_id`, `destination_id`, `payout_id`, and `operation_id` are opaque identifiers scoped to the caller's tenant.

Nontrivial contract choices:

| Decision | Option A | Option B | Selected | Why |
| --- | --- | --- | --- | --- |
| Initial write acknowledgement | `201 Created` on `/v1/payouts/{id}` with a payout resource only | `202 Accepted` from `POST /v1/payouts` plus a pollable operation resource and an immediately addressable payout resource | Option B | The PSP can take 5-90 seconds and final settlement is later than submission. `202` keeps acceptance separate from completion and gives clients an honest timeout-recovery path. |
| Read-consistency split | All payout reads are eventual from the projection | Single-payout read is strong; payout-history list is eventual with explicit freshness disclosure | Option B | Clients need a dependable read path after network timeouts. Keeping only the history list eventual preserves a projection-backed history view without hiding read-after-write gaps. |
| Retry protection for create | Best-effort retries with no required idempotency contract | `Idempotency-Key` required on `POST /v1/payouts` with defined replay semantics | Option B | Network timeouts are common and payout initiation is retry-unsafe without contract-level deduplication. |

Rejected paths:
- No action-style endpoint such as `/v1/payouts:initiate`; `POST /v1/payouts` already expresses creation/start semantics.
- No fake synchronous success. The initial create does not use `200 OK` or `201 Created` for the first successful acceptance because neither PSP submission nor final settlement is complete at response time.
- No webhook or callback surface in this contract. This version stays poll-based only.

## Resource And Endpoint Matrix

### Resource model

- `payout`: the business resource representing a vendor payout and its lifecycle.
- `payout_operation`: the asynchronous operation resource representing the PSP-submission work started by `POST /v1/payouts`.

### Endpoint matrix

| Endpoint | Purpose | Consistency | Retry class | Success statuses | Error statuses |
| --- | --- | --- | --- | --- | --- |
| `POST /v1/payouts` | Create a payout resource and start asynchronous PSP submission | Strong acceptance | Retry-safe by contract with required `Idempotency-Key` | `202 Accepted`, `200 OK` on terminal same-key replay | `400`, `401`, `403`, `409`, `413`, `415`, `422`, `428`, `429`, `503` |
| `GET /v1/payout-operations/{operation_id}` | Poll submission progress for a previously accepted payout start | Strong | Retry-safe by protocol | `200 OK`, `304 Not Modified` | `401`, `403`, `404`, `429` |
| `GET /v1/payouts/{payout_id}` | Read the canonical current state of one payout | Strong | Retry-safe by protocol | `200 OK`, `304 Not Modified` | `401`, `403`, `404`, `429` |
| `GET /v1/payouts` | Read payout history from an eventually consistent projection | Eventual | Retry-safe by protocol | `200 OK` | `400`, `401`, `403`, `429` |

### Status semantics

`payout.status`:
- `pending_submission`: the payout was accepted and the PSP-submission operation has not completed successfully yet.
- `submitted`: the PSP-submission operation succeeded; final settlement is still pending.
- `settled`: terminal success.
- `failed`: terminal failure. The failure may have happened during submission or after submission during settlement.

`payout_operation.status`:
- `pending`: accepted but submission work has not started yet.
- `running`: submission to the PSP is in progress.
- `succeeded`: PSP handoff completed successfully. This does not imply final settlement.
- `failed`: the asynchronous submission attempt failed before a successful PSP handoff.

Operation-resource semantics:
- The first successful `POST /v1/payouts` allocates a `payout_id` immediately and creates exactly one `payout_operation` for that initiation request.
- The operation resource tracks only the submission phase. It is not the business outcome resource.
- If the operation reaches `failed`, the payout resource transitions to `failed` with `failure.stage=submission`.
- If the operation reaches `succeeded`, the payout resource transitions to `submitted`, and clients must continue polling the payout resource for final settlement.

## Request, Response, And Error Model

### `POST /v1/payouts`

Required headers:
- `Content-Type: application/json`
- `Accept: application/json`
- `Idempotency-Key: <opaque-token>` on every create attempt

Request body:

```json
{
  "vendor_id": "vnd_01HTY7QW8M3G1B2N",
  "destination_id": "dst_01HTY7S4Q5G9H6R3",
  "amount": {
    "currency": "USD",
    "value": "1250.00"
  },
  "client_reference": "invoice-8841",
  "metadata": {
    "invoice_id": "inv_8841"
  }
}
```

Field contract:
- `vendor_id`: required opaque vendor identifier.
- `destination_id`: required opaque payout-destination identifier already associated with that vendor inside the caller's tenant.
- `amount.currency`: required payout currency code supported by the platform.
- `amount.value`: required positive decimal string. Binary floating-point numbers are not accepted.
- `client_reference`: optional caller-supplied business reference. It is not the idempotency key and is not used for deduplication by itself.
- `metadata`: optional string-to-string map for caller annotations.

Initial accepted response:

```http
HTTP/1.1 202 Accepted
Location: /v1/payout-operations/pop_01HTY8D3GB58Q8D4
Retry-After: 3
ETag: "pop_01HTY8D3GB58Q8D4:v1"
```

```json
{
  "operation": {
    "id": "pop_01HTY8D3GB58Q8D4",
    "type": "payout_submission",
    "status": "running",
    "created_at": "2026-03-07T12:34:56Z",
    "updated_at": "2026-03-07T12:34:56Z",
    "result": {
      "payout_id": "po_01HTY8CZP6QTRM6F",
      "payout_uri": "/v1/payouts/po_01HTY8CZP6QTRM6F"
    },
    "failure": null,
    "links": {
      "self": "/v1/payout-operations/pop_01HTY8D3GB58Q8D4",
      "payout": "/v1/payouts/po_01HTY8CZP6QTRM6F"
    }
  },
  "payout": {
    "id": "po_01HTY8CZP6QTRM6F",
    "status": "pending_submission",
    "vendor_id": "vnd_01HTY7QW8M3G1B2N",
    "destination_id": "dst_01HTY7S4Q5G9H6R3",
    "amount": {
      "currency": "USD",
      "value": "1250.00"
    },
    "client_reference": "invoice-8841",
    "created_at": "2026-03-07T12:34:56Z",
    "updated_at": "2026-03-07T12:34:56Z",
    "submitted_at": null,
    "settled_at": null,
    "failure": null,
    "links": {
      "self": "/v1/payouts/po_01HTY8CZP6QTRM6F",
      "operation": "/v1/payout-operations/pop_01HTY8D3GB58Q8D4"
    }
  }
}
```

Idempotent terminal replay:
- If the caller retries with the same `Idempotency-Key` and the same normalized request after the submission operation or payout has already reached a terminal state, the server returns `200 OK` with the same logical `operation` and `payout` resources rather than creating a second payout.

### `GET /v1/payout-operations/{operation_id}`

Response shape:

```json
{
  "id": "pop_01HTY8D3GB58Q8D4",
  "type": "payout_submission",
  "status": "succeeded",
  "created_at": "2026-03-07T12:34:56Z",
  "updated_at": "2026-03-07T12:35:41Z",
  "result": {
    "payout_id": "po_01HTY8CZP6QTRM6F",
    "payout_uri": "/v1/payouts/po_01HTY8CZP6QTRM6F"
  },
  "failure": null,
  "links": {
    "self": "/v1/payout-operations/pop_01HTY8D3GB58Q8D4",
    "payout": "/v1/payouts/po_01HTY8CZP6QTRM6F"
  }
}
```

Failure shape on terminal submission failure:

```json
{
  "id": "pop_01HTY8D3GB58Q8D4",
  "type": "payout_submission",
  "status": "failed",
  "created_at": "2026-03-07T12:34:56Z",
  "updated_at": "2026-03-07T12:35:41Z",
  "result": {
    "payout_id": "po_01HTY8CZP6QTRM6F",
    "payout_uri": "/v1/payouts/po_01HTY8CZP6QTRM6F"
  },
  "failure": {
    "stage": "submission",
    "code": "psp_submission_failed",
    "detail": "The payout could not be handed off to the PSP.",
    "retryable": false
  },
  "links": {
    "self": "/v1/payout-operations/pop_01HTY8D3GB58Q8D4",
    "payout": "/v1/payouts/po_01HTY8CZP6QTRM6F"
  }
}
```

### `GET /v1/payouts/{payout_id}`

Response shape:

```json
{
  "id": "po_01HTY8CZP6QTRM6F",
  "status": "submitted",
  "vendor_id": "vnd_01HTY7QW8M3G1B2N",
  "destination_id": "dst_01HTY7S4Q5G9H6R3",
  "amount": {
    "currency": "USD",
    "value": "1250.00"
  },
  "client_reference": "invoice-8841",
  "created_at": "2026-03-07T12:34:56Z",
  "updated_at": "2026-03-07T12:35:41Z",
  "submitted_at": "2026-03-07T12:35:41Z",
  "settled_at": null,
  "failure": null,
  "links": {
    "self": "/v1/payouts/po_01HTY8CZP6QTRM6F",
    "operation": "/v1/payout-operations/pop_01HTY8D3GB58Q8D4"
  }
}
```

Failure object shape on `payout.status=failed`:

```json
{
  "stage": "submission",
  "code": "psp_submission_failed",
  "detail": "The payout could not be handed off to the PSP."
}
```

or

```json
{
  "stage": "settlement",
  "code": "psp_settlement_failed",
  "detail": "The payout was submitted but did not settle successfully."
}
```

### `GET /v1/payouts`

Query contract:
- `cursor`: opaque cursor for forward pagination.
- `page_size`: default `50`, max `200`.
- `vendor_id`: optional exact-match filter.
- `status`: optional exact-match filter using the payout status enum.
- `created_at_gte`, `created_at_lte`: optional timestamp filters.
- `sort`: whitelist-only. Default `-created_at`; the server applies `id` as a stable tie-breaker.

Unknown filters, unsupported sort fields, or invalid filter value types fail with `400 Bad Request`.

List response shape:

```json
{
  "data": [
    {
      "id": "po_01HTY8CZP6QTRM6F",
      "status": "settled",
      "vendor_id": "vnd_01HTY7QW8M3G1B2N",
      "destination_id": "dst_01HTY7S4Q5G9H6R3",
      "amount": {
        "currency": "USD",
        "value": "1250.00"
      },
      "client_reference": "invoice-8841",
      "created_at": "2026-03-07T12:34:56Z",
      "updated_at": "2026-03-07T12:36:12Z",
      "submitted_at": "2026-03-07T12:35:41Z",
      "settled_at": "2026-03-07T12:36:12Z",
      "failure": null,
      "links": {
        "self": "/v1/payouts/po_01HTY8CZP6QTRM6F"
      }
    }
  ],
  "page": {
    "next_cursor": "eyJjcmVhdGVkX2F0IjoiMjAyNi0wMy0wN1QxMjozNDo1NloiLCJpZCI6InBvXzAxSFRZLi4u"
  },
  "consistency": {
    "model": "eventual",
    "as_of": "2026-03-07T12:36:10Z",
    "may_omit_recent_writes": true
  }
}
```

### Error model

All non-2xx responses use `application/problem+json`.

Problem shape:

```json
{
  "type": "https://api.example.com/problems/idempotency-key-payload-mismatch",
  "title": "Idempotency key reuse with different request payload",
  "status": 409,
  "detail": "Idempotency-Key 4db7d4a2-2d38-46d7-a7ab-57c7365b28da was already used for another payout request.",
  "instance": "/v1/payouts",
  "code": "idempotency_key_payload_mismatch",
  "request_id": "req_01HTY8K4X4P1DHEH",
  "errors": [
    {
      "field": "Idempotency-Key",
      "reason": "payload_mismatch"
    }
  ]
}
```

HTTP error mapping:
- `400 Bad Request`: malformed JSON, trailing tokens, unknown fields, invalid query parameter types, or invalid idempotency-key syntax.
- `401 Unauthorized`: missing or invalid authentication.
- `403 Forbidden`: authenticated caller lacks payout-initiation permission or cannot access the referenced payout or operation in this tenant.
- `404 Not Found`: payout or operation identifier is unknown in the caller's tenant scope.
- `409 Conflict`: same `Idempotency-Key` reused with a different normalized request payload.
- `413 Payload Too Large`: request body exceeds the API request-size limit.
- `415 Unsupported Media Type`: request body is not JSON.
- `422 Unprocessable Content`: well-formed request violates semantic rules such as unknown vendor ownership, invalid destination for the vendor, unsupported currency, or invalid amount rules.
- `428 Precondition Required`: missing `Idempotency-Key` on `POST /v1/payouts`.
- `429 Too Many Requests`: caller exceeded API rate limits. `Retry-After` may be present.
- `503 Service Unavailable`: the service cannot safely accept a payout before it can durably record the idempotent acknowledgement or reach the dependency state required for safe acceptance.

Boundary contract:
- JSON decoding is strict. Unknown fields, malformed payloads, and trailing tokens fail consistently.
- Success responses never embed an error payload.
- Error payloads must not expose PSP credentials, raw SQL, stack traces, or infrastructure topology.
- If the service emits a `request_id` response header elsewhere in the API, the same value should be mirrored in problem responses as `request_id`.

## Retry, Idempotency, And Concurrency Rules

- `POST /v1/payouts` is retry-safe by contract only when the caller sends the same `Idempotency-Key` and the same normalized request payload.
- Idempotency scope is the caller tenant plus `POST /v1/payouts`.
- Payload equivalence is evaluated after JSON parsing and contract-level normalization. Insignificant whitespace and object-member order do not change equivalence.
- Same key plus same payload:
  - If the original request is still non-terminal, return `202 Accepted` with the same `operation.id`, the same `payout.id`, and the same `Location`.
  - If the original request is terminal, return `200 OK` with the current `operation` and `payout` representations for that same logical payout.
- Same key plus different payload returns `409 Conflict` with `code=idempotency_key_payload_mismatch`.
- A caller that times out waiting for `POST /v1/payouts` must replay the same request with the same `Idempotency-Key` until it receives a response.
- Idempotency retention is guaranteed for at least `24h` after the first accepted request and never shorter than the time until the associated payout reaches a terminal state.
- A new business attempt after a terminal payout requires a new `Idempotency-Key`.

Preconditions and conditional requests:
- `Idempotency-Key` is a required precondition on create and maps to `428 Precondition Required` when absent.
- No client mutation endpoint other than create is in scope, so `If-Match` is not required in this contract.
- `GET /v1/payout-operations/{operation_id}` and `GET /v1/payouts/{payout_id}` expose `ETag`.
- Clients may send `If-None-Match` on those reads and receive `304 Not Modified`.

## Async, Freshness, And Webhook Notes

Async behavior:
- `POST /v1/payouts` acknowledges durable acceptance only. It does not imply PSP handoff success and does not imply settlement.
- `Retry-After` on the initial `202 Accepted` response is advisory polling guidance for the operation resource.
- Clients poll `GET /v1/payout-operations/{operation_id}` until the submission phase becomes terminal.
- If the operation reaches `succeeded`, clients then poll `GET /v1/payouts/{payout_id}` until `payout.status` becomes `settled` or `failed`.

Freshness disclosure:
- `GET /v1/payouts/{payout_id}` is the canonical timeout-recovery and current-state read path.
- `GET /v1/payouts` is explicitly projection-backed history and may lag recent writes or status changes.
- The list response exposes `consistency.model=eventual` and `consistency.as_of` so callers can reason about staleness.
- The list endpoint does not promise read-after-write visibility for newly accepted or recently updated payouts.

Webhook notes:
- No client-facing webhook or callback contract is part of this scope.
- If a future version adds payout-status webhooks, that surface must define signature verification, replay protection, duplicate handling, sender timeout expectations, and at-least-once delivery semantics instead of inheriting this polling contract implicitly.

## Compatibility, Artifact Updates, And Handoffs

Compatibility classification:
- Adding `POST /v1/payouts`, `GET /v1/payouts/{payout_id}`, `GET /v1/payouts`, and `GET /v1/payout-operations/{operation_id}` in `v1` is additive.
- Changing the initial create semantics from `202 Accepted` to `201 Created` or `200 OK` later would be a behavior change.
- Changing `GET /v1/payouts` from eventual to strong consistency, or changing `GET /v1/payouts/{payout_id}` from strong to eventual consistency, would be a behavior change.
- Changing `Idempotency-Key` from required to optional, or changing same-key replay semantics, would be a behavior change.
- Removing the operation resource, removing a documented status value, or changing the problem-details shape would be breaking.
- Adding optional response fields is additive. Adding new enum values is safe only if clients are expected to tolerate unknown values.

Artifact updates and handoffs:
- This contract belongs in the API contract artifact and OpenAPI, not in handler code or storage design notes.
- If the system cannot provide a strong single-payout read path, reopen the contract alongside data or cache design before freezing it.
- If payout settlement semantics require compensation, reconciliation, or other externally visible distributed-state rules, reopen with distributed-architecture design before freezing the contract.
- If payout authorization or tenant-isolation rules become materially more complex than this API surface assumes, reopen with security design before freezing the contract.

## Open Questions, Risks, And Reopen Conditions

Open questions:
- Can `GET /v1/payouts/{payout_id}` be served strongly enough for timeout recovery, or must it also be projection-backed?
- What exact semantic validation rules apply to supported currencies, amount precision, and vendor-destination eligibility?
- How long are `payout_operation` resources retained after terminal completion, and should expiry surface as `404 Not Found` or `410 Gone`?
- Is a lookup surface by `client_reference` needed, or is idempotent replay plus payout identifiers sufficient?

Risks:
- If the service cannot actually provide strong single-payout reads, clients may assume a read-after-write guarantee the system does not have.
- If idempotency retention is shorter than the real retry window for callers, duplicate payouts become possible.
- If clients treat `payout_operation.status=succeeded` as final business success, they may stop polling before final settlement completes.

Reopen conditions:
- Any move to synchronous payout creation.
- Any decision to make payout-history reads strongly consistent or single-payout reads eventually consistent.
- Any addition of client cancellation, bulk payout creation, or payout-status webhooks.

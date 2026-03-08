# Vendor Payout REST Contract

## Contract Framing And Assumptions

Client-visible problem: authenticated tenant-scoped clients need to initiate vendor payouts over an unreliable network, recover safely from POST timeouts, and observe both submission progress and final payout outcome without relying on synchronous completion.

Audience and trust boundary:
- Caller is an authenticated API client acting within one tenant or merchant account.
- Tenant context comes from validated identity, not caller-supplied headers.
- `vendor_id`, `destination_id`, `payout_id`, and `operation_id` are opaque identifiers scoped to the caller's tenant.

Nontrivial contract choices:

| Decision | Option A | Option B | Selected | Why |
| --- | --- | --- | --- | --- |
| Write acknowledgement model | `201 Created` on `/v1/payouts/{id}` with `status=processing`, no separate operation resource | `202 Accepted` from `POST /v1/payouts` plus a pollable operation resource and a payout resource | Option B | Keeps acceptance, PSP submission progress, and final settlement distinct. This is more honest for 5-90 second PSP latency and safer for timeout recovery. |
| Read consistency model | All payout reads served from the eventual projection | Item read is strong, history/list read is eventual with explicit freshness disclosure | Option B | Clients need a reliable read path after POST timeouts. Making only history eventual preserves fast projection-backed history without hiding read-after-write gaps. |
| Retry protection | Best-effort retries without a required key | `Idempotency-Key` required on `POST /v1/payouts` | Option B | Network timeouts are common and payout initiation is retry-unsafe without a contract-level dedup key. |

Rejected paths:
- No action-style endpoint such as `/v1/payouts:initiate`; the collection `POST` already expresses creation/start semantics.
- No fake synchronous success. `201` or `200` is not used for the initial create because payout submission and settlement are not complete at response time.

## Resource And Endpoint Matrix

### Resource model

- `payout`: the client-visible business resource representing a vendor payout and its lifecycle.
- `payout_operation`: a separate operation resource representing the asynchronous PSP submission started by `POST /v1/payouts`.

### Endpoint matrix

| Endpoint | Purpose | Consistency | Retry class | Success statuses | Error statuses |
| --- | --- | --- | --- | --- | --- |
| `POST /v1/payouts` | Create a payout resource and start asynchronous submission to the PSP | Strong acceptance | Retry-safe by contract with required `Idempotency-Key` | `202 Accepted` | `400`, `401`, `403`, `409`, `413`, `415`, `422`, `428`, `429`, `503` |
| `GET /v1/payout-operations/{operation_id}` | Poll PSP-submission progress for a previously accepted payout start | Strong | Retry-safe by protocol | `200 OK`, `304 Not Modified` | `401`, `403`, `404`, `429` |
| `GET /v1/payouts/{payout_id}` | Read the canonical current state of one payout | Strong | Retry-safe by protocol | `200 OK`, `304 Not Modified` | `401`, `403`, `404`, `429` |
| `GET /v1/payouts` | Read payout history from an eventually consistent projection | Eventual | Retry-safe by protocol | `200 OK` | `400`, `401`, `403`, `429` |

### Status semantics

`payout.status`:
- `processing`: payout was accepted and submission work is in progress.
- `submitted`: PSP accepted the payout or the service completed PSP handoff; settlement is still pending.
- `settled`: terminal success.
- `failed`: terminal failure.

`payout_operation.status`:
- `pending`: accepted but not yet started.
- `running`: submission to the PSP is in progress.
- `succeeded`: PSP handoff completed successfully. This does not imply settlement.
- `failed`: the asynchronous submission attempt failed before successful PSP handoff.

Operation-resource semantics:
- The initial `POST /v1/payouts` creates a payout resource immediately in `processing`.
- The operation resource tracks only the asynchronous submission phase that can take 5-90 seconds.
- After the operation reaches `succeeded`, clients must poll `GET /v1/payouts/{payout_id}` for final business outcome.
- Final settlement is represented only on the payout resource, not on the operation resource.

## Request, Response, And Error Model

### `POST /v1/payouts`

Required headers:
- `Content-Type: application/json`
- `Accept: application/json`
- `Idempotency-Key: <opaque-token>` required for every create attempt

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
- `destination_id`: required opaque payout-destination identifier.
- `amount.currency`: required ISO-like currency code string already supported by the API.
- `amount.value`: required positive decimal string.
- `client_reference`: optional caller-supplied business reference. It is not the idempotency key.
- `metadata`: optional string map for caller annotations.

Accepted response:

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
    "payout": {
      "id": "po_01HTY8CZP6QTRM6F",
      "uri": "/v1/payouts/po_01HTY8CZP6QTRM6F"
    },
    "failure": null,
    "links": {
      "self": "/v1/payout-operations/pop_01HTY8D3GB58Q8D4",
      "payout": "/v1/payouts/po_01HTY8CZP6QTRM6F"
    }
  },
  "payout": {
    "id": "po_01HTY8CZP6QTRM6F",
    "status": "processing",
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

### `GET /v1/payout-operations/{operation_id}`

Response shape:

```json
{
  "id": "pop_01HTY8D3GB58Q8D4",
  "type": "payout_submission",
  "status": "succeeded",
  "created_at": "2026-03-07T12:34:56Z",
  "updated_at": "2026-03-07T12:35:41Z",
  "payout": {
    "id": "po_01HTY8CZP6QTRM6F",
    "uri": "/v1/payouts/po_01HTY8CZP6QTRM6F"
  },
  "failure": null,
  "links": {
    "self": "/v1/payout-operations/pop_01HTY8D3GB58Q8D4",
    "payout": "/v1/payouts/po_01HTY8CZP6QTRM6F"
  }
}
```

Failure shape on a terminal operation failure:

```json
{
  "id": "pop_01HTY8D3GB58Q8D4",
  "type": "payout_submission",
  "status": "failed",
  "created_at": "2026-03-07T12:34:56Z",
  "updated_at": "2026-03-07T12:35:41Z",
  "payout": {
    "id": "po_01HTY8CZP6QTRM6F",
    "uri": "/v1/payouts/po_01HTY8CZP6QTRM6F"
  },
  "failure": {
    "code": "psp_submission_failed",
    "detail": "The payout could not be submitted to the PSP.",
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

### `GET /v1/payouts`

Query contract:
- `cursor`: opaque cursor for forward pagination.
- `page_size`: default `50`, max `200`.
- `vendor_id`: optional exact-match filter.
- `status`: optional exact-match filter using the payout status enum.
- `created_at_gte`, `created_at_lte`: optional timestamp filters.
- `sort`: whitelist-only. Default is `-created_at`; server applies `id` as a stable tie-breaker.

Unknown filters, unsupported sort fields, or invalid filter types fail with `400 Bad Request`.

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
    "next_cursor": "eyJjcmVhdGVkX2F0IjoiMjAyNi0wMy0wN1QxMjozNDo1NloiLCJpZCI6InBvXzAxSFRZ..."
  },
  "consistency": {
    "model": "eventual",
    "as_of": "2026-03-07T12:36:10Z",
    "stale_possible": true
  }
}
```

### Error model

All non-2xx errors use `application/problem+json`.

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
- `400 Bad Request`: malformed JSON, trailing tokens, unknown fields, invalid query parameter types, invalid header syntax.
- `401 Unauthorized`: missing or invalid authentication.
- `403 Forbidden`: authenticated caller lacks payout-initiation permission.
- `404 Not Found`: payout or operation does not exist in the caller's tenant scope.
- `409 Conflict`: same `Idempotency-Key` reused with a different effective request payload.
- `413 Payload Too Large`: body exceeds service request limit.
- `415 Unsupported Media Type`: non-JSON body.
- `422 Unprocessable Content`: well-formed request violates semantic rules such as unknown vendor ownership, invalid destination for the vendor, or unsupported currency/amount combination.
- `428 Precondition Required`: missing `Idempotency-Key` on `POST /v1/payouts`.
- `429 Too Many Requests`: caller exceeded API rate limits. `Retry-After` may be present.
- `503 Service Unavailable`: service cannot safely accept payout initiation before durable acknowledgement because an internal dependency or the PSP is unavailable.

Boundary contract:
- JSON decoding is strict: unknown fields and malformed bodies fail.
- Success responses never embed an error payload.
- Error payloads must not expose PSP credentials, raw SQL, stack traces, or infrastructure topology.

## Retry, Idempotency, And Concurrency Rules

- `POST /v1/payouts` is retry-safe by contract only when the caller sends the same `Idempotency-Key` and the same effective request payload.
- Idempotency scope is the caller tenant plus `POST /v1/payouts`.
- The server must treat the following as the effective request payload for conflict detection: body plus any client-visible fields that affect the payout resource, excluding transport-only headers.
- Same key plus same payload:
  - If the original request is still in `operation.status=pending|running`, return `202 Accepted` with the same `operation.id` and `payout.id`.
  - If the original initiation operation is terminal, return `200 OK` with the current operation and payout representations.
- Same key plus different payload returns `409 Conflict` with `code=idempotency_key_payload_mismatch`.
- A caller that times out waiting for `POST /v1/payouts` must replay the same request with the same `Idempotency-Key` until it receives a response.
- Idempotency retention is guaranteed for at least `24h` after the first `202 Accepted`, and never less than the lifetime of a non-terminal operation created by that request.
- A new payout attempt after a terminal `failed` payout requires a new `Idempotency-Key`.

Preconditions and conditional requests:
- No client mutation endpoint other than create is in scope, so `If-Match` is not required in this contract.
- `GET /v1/payout-operations/{operation_id}` and `GET /v1/payouts/{payout_id}` expose `ETag`.
- Clients may send `If-None-Match` on those reads and receive `304 Not Modified`.

## Async, Freshness, And Webhook Notes

Async behavior:
- `POST /v1/payouts` acknowledges request acceptance, not PSP completion and not final settlement.
- `Retry-After` on the initial `202` response is advisory poll guidance for the operation resource.
- Clients poll `GET /v1/payout-operations/{operation_id}` until the submission phase is terminal, then poll `GET /v1/payouts/{payout_id}` for final settlement outcome.

Freshness disclosure:
- `GET /v1/payouts/{payout_id}` is the canonical read path and is intended for timeout recovery and current-state checks.
- `GET /v1/payouts` is a projection-backed history endpoint with `consistency.model=eventual`.
- The list response exposes `consistency.as_of` so clients can determine the projection watermark seen by the server.
- The list endpoint does not promise read-after-write visibility for newly created or recently updated payouts.

Webhook notes:
- No client-facing webhook or callback contract is part of this scope.
- If a future version adds payout-status webhooks, that surface must assume at-least-once delivery, duplicates, replay protection, and possible reordering rather than inheriting this polling contract implicitly.

## Compatibility, Artifact Updates, And Handoffs

Compatibility classification:
- Adding `POST /v1/payouts`, `GET /v1/payouts/{payout_id}`, `GET /v1/payouts`, and `GET /v1/payout-operations/{operation_id}` in `v1` is additive.
- Changing initial create semantics from `202 Accepted` to `201 Created` or `200 OK` later would be a behavior change.
- Changing `GET /v1/payouts` between `eventual` and `strong` consistency inside `v1` would be a behavior change.
- Changing `Idempotency-Key` from required to optional, or changing same-key replay semantics, would be a behavior change.
- Removing documented status enum values, problem `code` values, or the operation resource would be breaking.
- Adding optional response fields is additive. Adding new enum values requires clients to ignore unknown values or else it becomes a behavior change for strict clients.

Artifact updates implied by this contract:
- This content belongs in the API contract artifact and OpenAPI, not in handler code or storage design docs.
- The operation-resource and freshness fields are part of the client-visible wire contract and must not be left as implementation details.

Adjacent handoffs:
- If settlement completion depends on cross-service compensation or externally visible reconciliation semantics, reopen with distributed-architecture design before freezing the contract.
- If payout authorization, tenant isolation, or destination exposure rules change materially, reopen with security design before freezing the contract.

## Open Questions, Risks, And Reopen Conditions

Open questions:
- Does `GET /v1/payouts/{payout_id}` have access to a strong canonical store, or must it also be projection-backed? If it cannot be strong, the timeout-recovery story must change and the contract should disclose eventual consistency there as well.
- What exact semantic validation rules apply to `amount`, supported currencies, destination types, and vendor eligibility?
- How long are `payout_operation` resources retained after terminal completion, and should the API return `404` or `410` after retention expiry?
- Should clients need a dedicated lookup by `client_reference` in addition to idempotent replay?
- Should terminal payout failure be represented only as `failed`, or is there a client-visible need to distinguish `rejected`, `canceled`, and `failed_after_submission`?

Risks:
- If the service cannot actually serve strong reads for `GET /v1/payouts/{payout_id}`, clients may falsely assume faster recovery than the system can provide.
- If idempotency retention is shorter than caller retry windows, duplicate payouts become possible.
- If the operation resource is treated as final business outcome instead of submission progress only, clients may stop polling before settlement completes.

Reopen conditions:
- Any move to synchronous payout creation.
- Any decision to make list history strongly consistent or item reads eventually consistent.
- Any addition of cancellation, bulk payouts, client callbacks, or payout updates.

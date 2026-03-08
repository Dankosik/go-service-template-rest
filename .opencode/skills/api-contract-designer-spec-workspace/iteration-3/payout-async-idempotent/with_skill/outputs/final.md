# Vendor Payout REST Contract

## Contract Framing And Assumptions

Client-visible problem:
- Authenticated tenant-scoped clients need to initiate vendor payouts through `POST /v1/payouts`.
- Network timeouts are common, so callers need a contract-safe retry path that does not create duplicate payouts.
- PSP submission can take `5-90s`, and final settlement is later and asynchronous.
- Payout-history reads come from an eventually consistent projection, so freshness must be visible at the API boundary.

Audience and trust boundary:
- Caller is an authenticated API client acting within one tenant or merchant account.
- Tenant scope comes from validated identity, not caller-supplied tenant headers.
- `vendor_id`, `destination_id`, `payout_id`, and `operation_id` are opaque identifiers within the caller's tenant scope.

Assumptions used to make the contract explicit:
- The service can provide a canonical single-payout read path that is stronger than the history projection.
- No client-facing webhook or callback surface is required in this version.
- No client-visible payout cancellation or amendment endpoint is in scope for `v1`.

Nontrivial contract choices:

| Decision | Option A | Option B | Selected | Why |
| --- | --- | --- | --- | --- |
| Initial create acknowledgement | `201 Created` with only a payout resource | `202 Accepted` with a payout resource plus a pollable operation resource | Option B | PSP submission is not complete at response time. `202` keeps acceptance separate from submission completion and final settlement. |
| Read consistency split | All payout reads are eventual from the history projection | `GET /v1/payouts/{payout_id}` is canonical current state; `GET /v1/payouts` remains eventual history | Option B | Timeout recovery needs a dependable read path. Keeping only the history list eventual preserves honest freshness disclosure. |
| Retry protection | Best-effort retries without a mandatory idempotency contract | `Idempotency-Key` required on `POST /v1/payouts` with replay and mismatch rules | Option B | Common network timeouts make payout initiation retry-unsafe without contract-level deduplication. |
| Same-key replay success status | Always return `202 Accepted` for same-key replays | Return `202 Accepted` while work is non-terminal, `200 OK` once the stored outcome is terminal | Option B | Terminal replay should expose the current stored outcome instead of pretending new work was just accepted. |

Rejected paths:
- No action-style endpoint such as `/v1/payouts:initiate`; collection `POST` already expresses create/start semantics.
- No fake synchronous success. Initial acceptance is not modeled as `200 OK` or `201 Created`.
- No callback URL in the request body. Polling is the only client-visible completion model in this contract.

## Resource And Endpoint Matrix

### Resource model

- `payout`: the business resource representing a vendor payout and its lifecycle.
- `payout_operation`: the control-plane resource representing the asynchronous PSP-submission work triggered by `POST /v1/payouts`.

Operation-resource semantics:
- The first successful acceptance allocates both `payout_id` and `operation_id`.
- The operation resource tracks only the submission phase.
- Final settlement is represented only on the payout resource.
- `payout_operation.status=canceled` does not exist in `v1`; there is no client cancellation surface in this contract.

### Endpoint matrix

| Endpoint | Purpose | Consistency | Retry class | Success statuses | Error statuses |
| --- | --- | --- | --- | --- | --- |
| `POST /v1/payouts` | Create a payout resource and start asynchronous PSP submission | Strong acceptance | Retry-safe by contract with required `Idempotency-Key` | `202 Accepted`, `200 OK` on terminal same-key replay | `400`, `401`, `403`, `409`, `413`, `415`, `422`, `428`, `429`, `503` |
| `GET /v1/payout-operations/{operation_id}` | Poll submission progress for a previously accepted payout start | Strong | Retry-safe by protocol | `200 OK`, `304 Not Modified` | `401`, `403`, `404`, `429` |
| `GET /v1/payouts/{payout_id}` | Read the canonical current state of one payout | Strong | Retry-safe by protocol | `200 OK`, `304 Not Modified` | `401`, `403`, `404`, `429` |
| `GET /v1/payouts` | Read payout history from an eventually consistent projection | Eventual | Retry-safe by protocol | `200 OK` | `400`, `401`, `403`, `429` |

### Status semantics

`payout.status`:
- `pending_submission`: payout was accepted and PSP submission has not completed successfully yet.
- `submitted`: PSP handoff succeeded; final settlement is still pending.
- `settled`: terminal success.
- `failed`: terminal failure.

`payout` state transitions:
- `pending_submission -> submitted`
- `pending_submission -> failed`
- `submitted -> settled`
- `submitted -> failed`

`payout_operation.status`:
- `pending`: accepted but work has not started yet.
- `running`: submission to the PSP is in progress.
- `succeeded`: PSP handoff completed successfully. This does not imply settlement.
- `failed`: submission ended without a successful PSP handoff.

Failure-stage semantics:
- If the operation fails before PSP handoff, `payout.status=failed` and `failure.stage=submission`.
- If PSP handoff succeeds but later settlement fails, `payout.status=failed` and `failure.stage=settlement`.

## Request, Response, And Error Model

### `POST /v1/payouts`

Required headers:
- `Content-Type: application/json`
- `Idempotency-Key: <opaque-token>`

Optional headers:
- `Accept: application/json`

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
- `destination_id`: required opaque payout-destination identifier already valid for that vendor in the caller's tenant.
- `amount.currency`: required supported currency code.
- `amount.value`: required positive decimal string. JSON numbers are not accepted.
- `client_reference`: optional caller business reference. It is not the deduplication key.
- `metadata`: optional string-to-string map for caller annotations.

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

Replay behavior:
- Same key plus same normalized payload while the stored operation is non-terminal returns `202 Accepted` with the same logical `operation` and `payout`.
- Same key plus same normalized payload after the stored operation and payout are terminal returns `200 OK` with the current stored `operation` and `payout`.
- Same key plus different normalized payload returns `409 Conflict`.

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

Terminal submission failure shape:

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
    "detail": "The payout could not be handed off to the PSP."
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

Failure object shape when `payout.status=failed`:

```json
{
  "stage": "settlement",
  "code": "psp_settlement_failed",
  "detail": "The payout was submitted but did not settle successfully."
}
```

### `GET /v1/payouts`

Query contract:
- `cursor`: opaque forward-pagination cursor.
- `page_size`: default `50`, maximum `200`.
- `vendor_id`: optional exact-match filter.
- `status`: optional exact-match filter using the `payout.status` enum.
- `created_at_gte`, `created_at_lte`: optional timestamp filters.
- `sort`: whitelist-only. Default `-created_at`; server applies `id` as a stable tie-breaker.

Unknown filters, invalid filter types, or unsupported sort fields fail with `400 Bad Request`.

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
- `400 Bad Request`: malformed JSON, trailing tokens, unknown fields, invalid idempotency-key syntax, invalid query parameter types, or unsupported filters/sorts.
- `401 Unauthorized`: missing or invalid authentication.
- `403 Forbidden`: caller lacks payout-initiation permission or cannot access the referenced payout or operation in this tenant.
- `404 Not Found`: unknown `payout_id` or `operation_id` within the caller's tenant.
- `409 Conflict`: same `Idempotency-Key` reused with a different normalized request payload.
- `413 Payload Too Large`: request body exceeds the API size limit.
- `415 Unsupported Media Type`: request body is not JSON.
- `422 Unprocessable Content`: request is well-formed but violates semantic rules such as unknown vendor ownership, invalid destination for the vendor, unsupported currency, or invalid amount rules.
- `428 Precondition Required`: missing `Idempotency-Key` on `POST /v1/payouts`.
- `429 Too Many Requests`: caller exceeded API rate limits. `Retry-After` may be present.
- `503 Service Unavailable`: the service cannot safely accept the payout before it can durably record the idempotent acknowledgement or reach the dependency state required for safe acceptance.

Boundary behavior:
- JSON decoding is strict. Unknown fields, malformed payloads, and trailing tokens fail consistently.
- Success responses never embed error payloads.
- Problem payloads do not expose PSP credentials, raw PSP responses containing secrets, stack traces, SQL text, or infrastructure topology.
- The API should return a stable `request_id` value so clients can correlate problem responses with server-side diagnostics.

## Retry, Idempotency, And Concurrency Rules

- `POST /v1/payouts` is retry-safe by contract only when the caller reuses the same `Idempotency-Key` with the same normalized request payload.
- Idempotency scope is `(tenant, method, route)`, specifically the caller tenant plus `POST /v1/payouts`.
- Payload equivalence is evaluated after JSON parsing and contract-level normalization. Whitespace and object-member order do not change equivalence.
- A caller that times out waiting for `POST /v1/payouts` must retry the same request with the same `Idempotency-Key` until it receives a definitive response.
- Idempotency retention is guaranteed for at least `24h` after first acceptance and must not expire earlier than the associated payout reaches a terminal state.
- A new business attempt after a terminal payout requires a new `Idempotency-Key`.

Key-consumption rules:
- `202 Accepted`, `200 OK` replay, and `409 Conflict` mismatch are key-aware outcomes.
- `400`, `401`, `403`, `413`, `415`, `422`, and `428` do not create a payout and do not reserve the key for a successful future attempt.
- `429` and `503` mean the service did not durably accept a payout for that request; the client may retry with the same key.

Preconditions and conditional requests:
- `Idempotency-Key` is a required create precondition and maps to `428 Precondition Required` when absent.
- No `If-Match` write precondition is defined in `v1` because no client-visible mutation endpoint other than create is in scope.
- `GET /v1/payout-operations/{operation_id}` and `GET /v1/payouts/{payout_id}` expose `ETag`.
- Clients may send `If-None-Match` on those reads and receive `304 Not Modified`.

## Async, Freshness, And Webhook Notes

Async behavior:
- `POST /v1/payouts` means the payout request was durably accepted, not that PSP submission succeeded and not that funds settled.
- `Location` points to the operation resource because the submission phase is the in-flight control-plane lifecycle.
- `Retry-After` on the initial `202 Accepted` response is advisory polling guidance.
- Clients poll `GET /v1/payout-operations/{operation_id}` until the submission phase reaches `succeeded` or `failed`.
- If the operation reaches `succeeded`, clients then poll `GET /v1/payouts/{payout_id}` until `payout.status` becomes `settled` or `failed`.

Freshness disclosure:
- `GET /v1/payouts/{payout_id}` is the canonical timeout-recovery and current-state read path.
- `GET /v1/payouts` is explicitly history/projection-backed and does not promise read-after-write visibility.
- The list response includes `consistency.model=eventual` and `consistency.as_of`.
- A newly accepted or recently updated payout may be absent or stale in the list response until projection catch-up completes.

Webhook notes:
- No webhook or callback contract is part of this `v1` surface.
- If a future version adds payout-status webhooks while list reads remain eventual, that surface must define at-least-once delivery, deduplication, replay protection, sender timeout expectations, and a monotonic reconciliation aid such as event version or event timestamp.

## Compatibility, Artifact Updates, And Handoffs

Compatibility classification:
- Adding `POST /v1/payouts`, `GET /v1/payouts/{payout_id}`, `GET /v1/payouts`, and `GET /v1/payout-operations/{operation_id}` to `v1` is additive.
- Changing initial acceptance from `202 Accepted` to synchronous `201 Created` or `200 OK` is a behavior change.
- Changing `Idempotency-Key` from required to optional, or changing same-key replay semantics, is a behavior change.
- Changing `GET /v1/payouts` from eventual to strong consistency or changing `GET /v1/payouts/{payout_id}` from strong to eventual consistency is a behavior change.
- Removing the operation resource, removing documented status values, or changing the stable problem-details profile is breaking.
- Adding optional response fields is additive. Adding new enum values is safe only if client guidance explicitly requires tolerance for unknown enum values.

Artifact updates and handoffs:
- This deliverable belongs in the API contract/OpenAPI surface, not in handler, router, or storage design artifacts.
- If strong single-payout reads are not actually achievable, reopen the contract together with data/cache design before freezing the API.
- If final settlement semantics require externally visible compensation, reconciliation, or multi-service callback behavior, reopen with distributed-architecture design.
- If payout authorization or tenant-isolation behavior becomes materially more complex than assumed here, reopen with security design.

## Open Questions, Risks, And Reopen Conditions

Open questions:
- Can `GET /v1/payouts/{payout_id}` be served strongly enough for timeout recovery, or must it also be projection-backed?
- How long should `payout_operation` resources remain readable after terminal completion, and should expiry surface as `404 Not Found` or `410 Gone`?
- Are there business duplicate-prevention rules beyond `Idempotency-Key`, such as a uniqueness rule around `client_reference`?
- What are the exact amount precision, scale, and supported-currency rules that belong in `422` validation?
- Is a lookup surface by `client_reference` needed, or are idempotent replay plus payout identifiers sufficient?

Risks:
- If clients interpret `payout_operation.status=succeeded` as final business success, they may stop polling before settlement completes.
- If idempotency retention is shorter than real client retry behavior, duplicate payouts become possible.
- If the eventual-history freshness fields are absent or inaccurate, clients may misread stale projection data as payout failure or loss.

Reopen conditions:
- A requirement to make payout initiation synchronous.
- A requirement to add payout cancellation, amendment, or bulk initiation.
- A decision to add payout-status webhooks or callbacks.
- A decision that single-payout reads are also eventually consistent.

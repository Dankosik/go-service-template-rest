# Partner Bulk Catalog Import API Contract

## Contract Framing And Assumptions

- Audience: authenticated partner API clients and the partner UI. Partner identity comes from auth context, not a caller-supplied partner ID in the path or body.
- Client-visible goal: accept a large CSV upload, require malware scanning before any business processing, expose an asynchronous import job, support either polling or signed completion webhooks, and surface partial row failures honestly.
- Consistency model:
  - `POST` acceptance is strong for request validation and job creation.
  - `GET` job and item-result reads are eventual because they are served from a cache-backed view.
  - Clients must not assume read-after-write on job status.
- Selected contract decisions:
  - Use an upload-session plus opaque signed upload URL, then create an import job referencing the uploaded file.
    Rejected alternative: a direct multipart `POST /v1/catalog-imports` because 50 MB uploads plus mandatory scanning create poor retry behavior, larger gateway pressure, and an unclear acceptance boundary.
  - Use a job resource at `/v1/catalog-imports/{import_id}` plus a paginated `/item-results` subresource for row outcomes.
    Rejected alternative: embedding every row failure directly in the job resource because response size becomes unbounded.
  - Support optional completion webhooks by referencing a pre-registered partner webhook endpoint ID.
    Rejected alternative: arbitrary callback URLs per request because it weakens SSRF controls, complicates secret bootstrap, and makes ownership validation harder.

## Resource And Endpoint Matrix

| Surface | Method and path | Purpose | Success statuses | Retry class | Consistency |
| --- | --- | --- | --- | --- | --- |
| Upload session | `POST /v1/catalog-imports/upload-sessions` | Reserve an upload slot, validate media and size, return signed upload instructions | `201 Created` | Retry-safe by contract with `Idempotency-Key` | Strong |
| Binary upload | `PUT {upload.url}` | Upload CSV bytes to the opaque signed URL from the session response | Storage-provider `2xx` | Retry-safe if client repeats the same bytes before URL expiry | Strong for the object upload itself |
| Import job create | `POST /v1/catalog-imports` | Start malware scan plus async import processing for an uploaded file | `202 Accepted` | Retry-safe by contract with `Idempotency-Key` | Strong for acceptance |
| Import job read | `GET /v1/catalog-imports/{import_id}` | Read import status, progress, terminal outcome, freshness, and links | `200 OK`, `304 Not Modified` | Safe and idempotent | Eventual |
| Item results | `GET /v1/catalog-imports/{import_id}/item-results` | Read paginated row-level outcomes for a completed import | `200 OK`, `304 Not Modified` | Safe and idempotent | Eventual |

No synchronous import endpoint is defined in `v1`.

### Upload / Initiate Flow

1. Client creates an upload session.
2. Service returns a signed single-use upload URL and the exact media and size constraints for that session.
3. Client uploads the CSV bytes to the signed URL before `upload.expires_at`.
4. Client starts the import job by referencing `upload_session_id`.
5. Service acknowledges with `202 Accepted`, then performs malware scan and processing asynchronously.
6. Client either polls the job resource or receives a signed terminal webhook if `notification.mode=webhook`.

## Request, Response, And Error Model

### 1. `POST /v1/catalog-imports/upload-sessions`

Required headers:

- `Idempotency-Key`
- `Content-Type: application/json`

Request body:

```json
{
  "file_name": "catalog-2026-03-07.csv",
  "content_type": "text/csv",
  "size_bytes": 48012012,
  "sha256": "2b6f4d7c7f9f2c..."
}
```

Contract rules:

- `content_type` whitelist:
  - `text/csv`
  - `application/vnd.ms-excel`
- `size_bytes` must be `> 0` and `<= 50000000`.
- `sha256` is optional but, when present, must be a lowercase hex SHA-256 of the uploaded bytes.
- Compressed archives such as `application/zip` are not accepted in `v1`.

Success response:

```json
{
  "id": "up_01JNWQ4W3R8B5P7M9M6T2Y4X6A",
  "status": "pending_upload",
  "file_name": "catalog-2026-03-07.csv",
  "content_type": "text/csv",
  "size_bytes": 48012012,
  "max_size_bytes": 50000000,
  "accepted_media_types": [
    "text/csv",
    "application/vnd.ms-excel"
  ],
  "scan_required": true,
  "upload": {
    "method": "PUT",
    "url": "https://uploads.example.com/opaque-signed-url",
    "headers": {
      "Content-Type": "text/csv",
      "Content-Length": "48012012"
    },
    "expires_at": "2026-03-07T12:15:00Z"
  },
  "expires_at": "2026-03-07T12:30:00Z"
}
```

Status semantics:

- `201 Created` with `Location: /v1/catalog-imports/upload-sessions/{id}`
- The API contract does not require a public `GET` upload-session endpoint in `v1`; clients progress by uploading and then creating the import job.

Immediate error mapping:

| Status | When | Problem `code` |
| --- | --- | --- |
| `400 Bad Request` | malformed JSON, unknown fields, trailing tokens | `invalid_request` |
| `409 Conflict` | same `Idempotency-Key` reused with a different payload | `idempotency_key_reuse_mismatch` |
| `413 Payload Too Large` | declared `size_bytes` exceeds `50000000` | `file_too_large` |
| `415 Unsupported Media Type` | `content_type` is outside the whitelist | `unsupported_media_type` |
| `422 Unprocessable Content` | `size_bytes` is zero, `sha256` format is invalid, or values are otherwise structurally valid but contract-invalid | `invalid_upload_metadata` |
| `429 Too Many Requests` | partner exceeded session-creation quota | `rate_limit_exceeded` |
| `503 Service Unavailable` | upload-url or scanning dependencies unavailable before session creation | `dependency_unavailable` |

### 2. `POST /v1/catalog-imports`

Required headers:

- `Idempotency-Key`
- `Content-Type: application/json`

Request body:

```json
{
  "upload_session_id": "up_01JNWQ4W3R8B5P7M9M6T2Y4X6A",
  "notification": {
    "mode": "webhook",
    "endpoint_id": "wh_01JNWQ7W5X6S8C4D9P3V1N2B7K"
  }
}
```

`notification` contract:

- `mode` enum: `poll`, `webhook`
- If `mode=poll`, `endpoint_id` must be omitted.
- If `mode=webhook`, `endpoint_id` is required and must identify a webhook endpoint already registered to the authenticated partner.

Acceptance rules:

- The referenced upload session must exist, belong to the same partner, not be expired, and have a completed object upload.
- Malware scanning starts only after the job is accepted.
- CSV parsing and row validation happen asynchronously after the scan passes.
- Uploading the file does not itself create the job; the explicit `POST /v1/catalog-imports` boundary is the business acceptance point.

Success response:

```json
{
  "id": "imp_01JNWQ9QX74ZK4V2Y6A8R3M5HT",
  "status": "pending",
  "phase": "scanning",
  "created_at": "2026-03-07T12:05:10Z",
  "updated_at": "2026-03-07T12:05:10Z",
  "upload_session_id": "up_01JNWQ4W3R8B5P7M9M6T2Y4X6A",
  "notification": {
    "mode": "webhook",
    "endpoint_id": "wh_01JNWQ7W5X6S8C4D9P3V1N2B7K",
    "event": "catalog.import.completed"
  },
  "summary": {
    "rows_total": null,
    "rows_processed": 0,
    "rows_succeeded": 0,
    "rows_failed": 0
  },
  "links": {
    "self": "/v1/catalog-imports/imp_01JNWQ9QX74ZK4V2Y6A8R3M5HT",
    "item_results": "/v1/catalog-imports/imp_01JNWQ9QX74ZK4V2Y6A8R3M5HT/item-results?outcome=failed"
  },
  "view_freshness": {
    "consistency": "eventual",
    "as_of": "2026-03-07T12:05:10Z",
    "source_updated_at": "2026-03-07T12:05:10Z",
    "max_expected_lag_seconds": 30
  }
}
```

Status semantics:

- `202 Accepted`
- `Location: /v1/catalog-imports/{import_id}`
- `Retry-After: 5`

Immediate error mapping:

| Status | When | Problem `code` |
| --- | --- | --- |
| `400 Bad Request` | malformed JSON, unknown fields, trailing tokens | `invalid_request` |
| `404 Not Found` | `upload_session_id` or `endpoint_id` does not exist for this partner | `resource_not_found` |
| `409 Conflict` | same `Idempotency-Key` reused with a different payload | `idempotency_key_reuse_mismatch` |
| `415 Unsupported Media Type` | request body is not JSON | `unsupported_media_type` |
| `422 Unprocessable Content` | upload session expired, upload not completed, upload hash mismatch, notification object invalid | `invalid_import_request` |
| `429 Too Many Requests` | partner exceeded job-creation quota or concurrent import cap | `rate_limit_exceeded` |
| `503 Service Unavailable` | import cannot be accepted because required dependencies are unavailable | `dependency_unavailable` |

### 3. `GET /v1/catalog-imports/{import_id}`

Response body shape:

```json
{
  "id": "imp_01JNWQ9QX74ZK4V2Y6A8R3M5HT",
  "status": "partially_failed",
  "phase": "complete",
  "created_at": "2026-03-07T12:05:10Z",
  "updated_at": "2026-03-07T12:08:42Z",
  "completed_at": "2026-03-07T12:08:41Z",
  "upload_session_id": "up_01JNWQ4W3R8B5P7M9M6T2Y4X6A",
  "notification": {
    "mode": "webhook",
    "endpoint_id": "wh_01JNWQ7W5X6S8C4D9P3V1N2B7K",
    "event": "catalog.import.completed",
    "last_delivery_status": "delivered"
  },
  "summary": {
    "rows_total": 1200,
    "rows_processed": 1200,
    "rows_succeeded": 1180,
    "rows_failed": 20
  },
  "failure": null,
  "links": {
    "self": "/v1/catalog-imports/imp_01JNWQ9QX74ZK4V2Y6A8R3M5HT",
    "item_results": "/v1/catalog-imports/imp_01JNWQ9QX74ZK4V2Y6A8R3M5HT/item-results?outcome=failed"
  },
  "view_freshness": {
    "consistency": "eventual",
    "as_of": "2026-03-07T12:08:45Z",
    "source_updated_at": "2026-03-07T12:08:41Z",
    "max_expected_lag_seconds": 30
  }
}
```

`status` enum:

- Non-terminal: `pending`, `running`
- Terminal: `succeeded`, `partially_failed`, `failed`, `canceled`

`phase` enum:

- `scanning`
- `queued`
- `processing`
- `complete`

Terminal failure shape:

```json
{
  "failure": {
    "code": "malware_detected",
    "message": "The uploaded file failed mandatory malware scanning.",
    "retryable": false
  }
}
```

Terminal-failure codes defined in `v1`:

- `malware_detected`
- `invalid_csv_format`
- `schema_validation_failed`
- `dependency_failure`
- `partner_configuration_invalid`

Read semantics:

- `ETag` is returned on every successful `GET`.
- `If-None-Match` is supported and returns `304 Not Modified`.
- While the job is non-terminal, the server may include `Retry-After`.
- `GET` may lag the worker state and may lag the webhook event for the same job.

### 4. `GET /v1/catalog-imports/{import_id}/item-results`

Query parameters:

- `outcome`: `failed`, `succeeded`, `all` (default: `failed`)
- `page_size`: default `50`, max `200`
- `cursor`: opaque cursor

Rules:

- Sorting is fixed to `row_number ASC` with a stable tie-breaker on `partner_item_key`.
- Unknown query parameters fail with `400 Bad Request`.
- This endpoint is only guaranteed to return finalized row outcomes after the job reaches a terminal state.
- Before terminal completion, the endpoint returns `409 Conflict` with `code=results_not_finalized`.

Response body:

```json
{
  "data": [
    {
      "row_number": 17,
      "partner_item_key": "SKU-123",
      "outcome": "failed",
      "code": "missing_required_field",
      "message": "price is required",
      "field_errors": [
        {
          "field": "price",
          "code": "required",
          "message": "price is required"
        }
      ]
    }
  ],
  "page": {
    "next_cursor": "eyJyb3dfbnVtYmVyIjoxOH0",
    "page_size": 50
  },
  "summary": {
    "rows_total": 1200,
    "rows_succeeded": 1180,
    "rows_failed": 20
  },
  "view_freshness": {
    "consistency": "eventual",
    "as_of": "2026-03-07T12:08:45Z",
    "source_updated_at": "2026-03-07T12:08:41Z",
    "max_expected_lag_seconds": 30
  }
}
```

Partial-failure representation:

- A partially successful import is never represented as a plain success boolean.
- The job resource exposes aggregate counts through `summary`.
- The item-results subresource exposes row-level failures and successes in a bounded, paginated form.

### Problem Details Profile

All non-`2xx` API responses use `application/problem+json` with:

- `type`
- `title`
- `status`
- `detail`
- `instance` when available

Stable extensions:

- `code`
- `request_id`
- `errors` for field-level issues

Boundary choice for `400` vs `422`:

- `400` for malformed syntax, unknown fields, invalid cursor encoding, and other parse-level issues.
- `422` for well-formed requests that violate business-visible contract rules.

## Retry, Idempotency, And Concurrency Rules

- `POST /v1/catalog-imports/upload-sessions` requires `Idempotency-Key`.
- `POST /v1/catalog-imports` requires `Idempotency-Key`.
- Idempotency key scope is `{partner, route, method}` with a deduplication TTL of `24h`.
- Same idempotency key with the same effective payload returns an equivalent response.
- Same idempotency key with a different payload returns `409 Conflict`.
- `GET` endpoints are safe and idempotent.
- Clients should honor `Retry-After` on `202`, on non-terminal job reads when present, and on `429`.
- Polling faster than the advised interval may be throttled with `429 Too Many Requests`.
- Partner-level concurrent import limits are surfaced as `429 Too Many Requests` with `code=rate_limit_exceeded` and a `Retry-After` header; `v1` does not use a separate status for quota vs concurrency throttling.
- No optimistic write concurrency contract is needed in `v1` because the public mutating surface is create-only.

## Async, Freshness, And Webhook Notes

### Job Lifecycle

| `status` | `phase` | Meaning |
| --- | --- | --- |
| `pending` | `scanning` | Job accepted; malware scan not finished |
| `pending` | `queued` | Scan passed; waiting for processing capacity |
| `running` | `processing` | Rows are being parsed and applied |
| `succeeded` | `complete` | All rows processed successfully |
| `partially_failed` | `complete` | Some rows succeeded and some failed |
| `failed` | `complete` | No useful business result is available |
| `canceled` | `complete` | Job was intentionally stopped; no further row processing occurs |

Freshness disclosure:

- `GET /v1/catalog-imports/{id}` and `/item-results` are cache-backed view reads.
- Each response includes `view_freshness.consistency="eventual"`.
- `view_freshness.as_of` is when the returned view was produced.
- `view_freshness.source_updated_at` is the latest authoritative job transition known to the view.
- `view_freshness.max_expected_lag_seconds=30` is the client-visible freshness budget for `v1`.
- A webhook may report the terminal transition before the polling endpoint reflects that transition.

### Completion Webhook Contract

Trigger:

- Sent only for terminal states on jobs created with `notification.mode=webhook`.
- Event name: `catalog.import.completed`

Delivery semantics:

- At least once
- Duplicates possible
- Reordering is possible relative to polling reads and other partner events
- Sender timeout: `5s`
- Retry schedule after non-`2xx` or timeout: `30s`, `2m`, `10m`, `1h`, `6h`
- A `2xx` response acknowledges the event and stops retries

Headers:

- `Content-Type: application/json`
- `X-Webhook-Id: evt_...`
- `X-Webhook-Event: catalog.import.completed`
- `X-Webhook-Timestamp: 2026-03-07T12:08:42Z`
- `X-Webhook-Signature: v1=<hex-hmac-sha256(secret, timestamp + "." + raw_body)>`

Verification rules:

- Signature is computed over the exact raw request body and timestamp string.
- Replay window is `5m`; older timestamps must be rejected.
- Clients deduplicate on `X-Webhook-Id`.

Payload:

```json
{
  "id": "evt_01JNWQBXZ4N4A6J3Q8C1Y7H5SR",
  "type": "catalog.import.completed",
  "created_at": "2026-03-07T12:08:42Z",
  "data": {
    "import_id": "imp_01JNWQ9QX74ZK4V2Y6A8R3M5HT",
    "status": "partially_failed",
    "summary": {
      "rows_total": 1200,
      "rows_succeeded": 1180,
      "rows_failed": 20
    },
    "failure": null,
    "links": {
      "self": "/v1/catalog-imports/imp_01JNWQ9QX74ZK4V2Y6A8R3M5HT",
      "item_results": "/v1/catalog-imports/imp_01JNWQ9QX74ZK4V2Y6A8R3M5HT/item-results?outcome=failed"
    }
  }
}
```

## Compatibility, Artifact Updates, And Handoffs

- Compatibility class for this proposal: additive new `v1` surface.
- Additive changes that are safe in `v1`:
  - new optional fields in responses
  - new terminal `failure.code` values
  - new webhook event fields
- Behavior changes that need explicit review even if wire-compatible:
  - changing `view_freshness.max_expected_lag_seconds`
  - changing the webhook retry schedule or signature construction
  - changing the meaning of `partially_failed` vs `failed`
  - changing throttling guidance or `Retry-After` behavior
- Breaking changes for `v1`:
  - removing `partially_failed` from the status enum
  - changing required request fields
  - reducing accepted media types or size limits
  - changing the webhook header names or signature version without overlap

Artifact alignment:

- OpenAPI should model the five surfaces above and treat the job resource as the source of truth for import lifecycle semantics.
- The problem-details profile, webhook payload schema, and freshness fields should be shared components.

Adjacent handoffs if scope expands:

- Security handoff if webhook registration, secret rotation, or partner authorization rules need deeper definition.
- Domain-invariant handoff if row identity, upsert vs replace semantics, or duplicate-row policy are not yet settled.
- Data/cache handoff if the `30s` freshness budget or result-retention strategy cannot be supported.

## Open Questions, Risks, And Reopen Conditions

- Open question: what exact CSV schema and header contract does the file follow? This contract assumes the schema exists elsewhere and only covers transport plus lifecycle semantics.
- Open question: do partners need a public webhook-registration API in the same release, or is `endpoint_id` backed by existing partner configuration?
- Open question: what retention period applies to uploaded files, job resources, and item-result pages?
- Open question: what partner-visible quotas should be published for session creation, job creation, and polling?
- Open question: is `canceled` a real client-visible outcome in `v1`, or should it be omitted until a cancel operation exists?
- Risk: if the cache-backed view can lag by more than the promised `30s`, the contract needs either a weaker freshness promise or a stronger read path.
- Risk: if partners require arbitrary callback URLs instead of registered endpoints, the webhook contract must reopen for trust-boundary review.
- Reopen this contract before coding if synchronous acceptance semantics, direct multipart upload, live streaming row errors, or stronger read-after-write guarantees are required.

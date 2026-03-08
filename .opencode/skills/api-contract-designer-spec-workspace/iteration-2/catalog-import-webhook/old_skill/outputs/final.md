# Partner Bulk Catalog Import API Contract

## Contract Summary

- Client-visible goal: let an authenticated partner submit a CSV catalog import up to `50 MiB`, require malware scanning before any business processing, expose one pollable job resource for the full lifecycle, and optionally send a signed terminal webhook.
- API style: REST over HTTP under `/v1`, JSON control-plane payloads, `application/problem+json` for errors.
- Auth context: partner identity comes from authenticated credentials; partner scope is not caller-controlled through headers or path parameters.
- Out of scope for this contract: CSV column schema, webhook endpoint registration API, internal worker topology, storage implementation, and rollout mechanics.

## Decision Summary

| Topic | Viable options | Selected contract | Rejected option | Why | Compatibility class |
| --- | --- | --- | --- | --- | --- |
| Upload transport | Direct multipart to API, or create-job then one-time upload session | Create the job first, return a single-use upload target, then require an explicit upload-completion call | Large direct multipart to the JSON API | Keeps a stable `job_id` from the start, handles `50 MiB` better, and keeps malware scanning and async processing honest | New upload-session fields are additive; changing back to direct multipart would be breaking |
| Completion consumption | Poll only, webhook only, or poll plus optional webhook | Polling is always available; `completion.mode=webhook` adds a signed terminal webhook | Webhook-only completion | Partners need a recovery path when webhook delivery fails or when the read model lags | Adding webhook support is additive; removing polling would be breaking |
| Partial item failures | Fail the whole import on any bad row, or finish the job with row-level failure reporting | `status=succeeded` means processing finished; `result.outcome` distinguishes `complete_success`, `partial_success`, and `no_rows_accepted` | Treat any row failure as job failure | Clients need row-level truth without conflating business-row rejections with process failure | Adding new optional failure detail fields is additive; changing meaning of `status=succeeded` would be breaking |
| Freshness disclosure | Pretend `GET` is authoritative, or disclose eventual view semantics | `GET` is explicitly eventual and returns `version` plus `consistency.as_of`; webhooks may arrive before `GET` catches up | Implicit read-after-write semantics | The UI reads from a cache-backed view and must not be given a false strong-read guarantee | Tightening freshness claims is behavior-changing; removing freshness fields would be breaking |
| Webhook addressing | Per-job arbitrary callback URL, or reference a pre-registered endpoint | `completion.mode=webhook` requires a partner-owned `webhook_endpoint_id` | Arbitrary callback URL in each job request | Keeps trust boundaries stable and avoids SSRF-style surprises in the client contract | Adding arbitrary URLs later would be behavior-changing |

## Resource And Endpoint Matrix

| Endpoint / resource | Consistency | Purpose | Success status | Retry / idempotency | Notes |
| --- | --- | --- | --- | --- | --- |
| `POST /v1/catalog-import-jobs` | Strong acceptance | Create one import job and return an upload session | `201 Created` for a new job, `200 OK` for idempotent replay | Retry-safe by contract only with required `Idempotency-Key` | Returns `Location` to the job resource and a single-use upload target |
| `POST /v1/catalog-import-jobs/{job_id}/upload-completions` | Strong acceptance | Confirm that the CSV upload finished and start scan / queueing | `202 Accepted` on first acceptance, `200 OK` on equivalent replay | Retry-safe by contract for the same `upload_id` on the same job | Returns current job representation and poll guidance |
| `GET /v1/catalog-import-jobs/{job_id}` | Eventual | Poll the lifecycle and terminal result of one job | `200 OK`, `304 Not Modified` with `If-None-Match` | Retry-safe by protocol | Backed by a lagging view; exposes `version` and `consistency.as_of` |
| `GET /v1/catalog-import-jobs/{job_id}/item-failures` | Eventual once terminal | Page through row-level failures for a completed import | `200 OK` | Retry-safe by protocol | Only authoritative after `status=succeeded`; cursor pagination only |
| Outbound webhook `POST` to the partner endpoint | N/A | Notify the partner that a webhook-enabled job reached a terminal state | Partner returns any `2xx` within `5s` | Sender retries at least once on timeout / non-`2xx` | Signed, at-least-once, duplicate-tolerant |

### End-to-end flow

1. Partner creates a job with source-file metadata and a completion preference.
2. API returns `job_id`, `Location`, and a time-limited single-use upload target.
3. Partner uploads the raw CSV bytes to the upload target using the returned method and required headers.
4. Partner calls `POST /v1/catalog-import-jobs/{job_id}/upload-completions` with the returned `upload_id`.
5. Service validates the upload, runs malware scanning, and only then queues business processing.
6. Partner polls `GET /v1/catalog-import-jobs/{job_id}` or waits for the signed terminal webhook if `completion.mode=webhook`.
7. If the job finished with row-level rejections, partner reads `GET /v1/catalog-import-jobs/{job_id}/item-failures` for the complete paginated failure list.

## Request, Response, And Error Model

### Resource model

- `catalog-import-job`: the single client-visible operation resource for one bulk import attempt.
- `upload session`: a one-time upload target attached to a job until the file is accepted for scanning.
- `item failure`: a row-level rejection record for jobs that finished processing all rows.

### `POST /v1/catalog-import-jobs`

Request:

```json
{
  "source_file": {
    "file_name": "catalog-2026-03-07.csv",
    "media_type": "text/csv",
    "size_bytes": 36800123,
    "checksum_sha256": "4f96b0a4f5602ef6c66b6bc4f8cf6efc9d1606d4d6e1ac9fc4d493cb7b6dfb0d"
  },
  "completion": {
    "mode": "webhook",
    "webhook_endpoint_id": "whe_01JV4RS90S8K6M4Q9FQGQ9JWKP"
  }
}
```

Semantics:

- Required header: `Idempotency-Key`.
- `source_file.media_type` must be `text/csv`.
- `source_file.size_bytes` must be `> 0` and `<= 52428800`.
- `source_file.checksum_sha256` is required and must be a lowercase hex SHA-256 digest.
- `completion.mode` enum: `poll`, `webhook`.
- `completion.webhook_endpoint_id` is required when `completion.mode=webhook`.
- A replay with the same idempotency key and the same payload returns the same `job_id` and current job state.
- A replay before upload completion may return a replacement active upload session if the previous unused upload session expired; the `job_id` must stay stable.
- A replay with the same idempotency key and a different payload returns `409 Conflict` with code `idempotency_conflict`.

Success response:

```json
{
  "id": "cij_01JV4S7N2Y9T6N7S43P3PZZN0E",
  "status": "pending",
  "phase": "awaiting_upload",
  "version": 1,
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-07T12:00:00Z",
  "source_file": {
    "file_name": "catalog-2026-03-07.csv",
    "media_type": "text/csv",
    "size_bytes": 36800123,
    "checksum_sha256": "4f96b0a4f5602ef6c66b6bc4f8cf6efc9d1606d4d6e1ac9fc4d493cb7b6dfb0d"
  },
  "completion": {
    "mode": "webhook",
    "webhook_endpoint_id": "whe_01JV4RS90S8K6M4Q9FQGQ9JWKP"
  },
  "upload": {
    "id": "ciu_01JV4S7NHE0CJVVKWB9G7T3M5G",
    "method": "PUT",
    "url": "https://uploads.example.com/u/ciu_01JV4S7NHE0CJVVKWB9G7T3M5G",
    "expires_at": "2026-03-07T12:30:00Z",
    "required_headers": {
      "Content-Type": "text/csv"
    },
    "max_size_bytes": 52428800
  },
  "summary": {
    "rows_total": null,
    "rows_processed": 0,
    "rows_succeeded": 0,
    "rows_failed": 0
  },
  "result": null,
  "failure": null,
  "consistency": {
    "model": "eventual",
    "as_of": "2026-03-07T12:00:00Z",
    "may_lag": true
  },
  "_links": {
    "self": "/v1/catalog-import-jobs/cij_01JV4S7N2Y9T6N7S43P3PZZN0E",
    "upload_completion": "/v1/catalog-import-jobs/cij_01JV4S7N2Y9T6N7S43P3PZZN0E/upload-completions"
  }
}
```

Response headers:

- `Location: /v1/catalog-import-jobs/{job_id}`
- `ETag: "1"`
- `X-Request-Id`
- `RateLimit-Limit`, `RateLimit-Remaining`, `RateLimit-Reset`

### Upload target contract

- The upload target is opaque and single-use.
- Request body is raw CSV bytes, not `multipart/form-data`.
- Required upload header: `Content-Type: text/csv`.
- Upload success is any `2xx` from the upload target.
- If the upload target rejects the request because it is expired or already consumed, the client must replay `POST /v1/catalog-import-jobs` with the same `Idempotency-Key` to recover the current job state or receive a replacement active upload session.
- Upload target failures are not `application/problem+json`; the authoritative control-plane recovery step is still the API job resource.

### `POST /v1/catalog-import-jobs/{job_id}/upload-completions`

Request:

```json
{
  "upload_id": "ciu_01JV4S7NHE0CJVVKWB9G7T3M5G"
}
```

Semantics:

- First successful call moves the job from `phase=awaiting_upload` to `phase=scanning` or `phase=queued`.
- Repeating the same call for the same `job_id` and `upload_id` is equivalent and returns the current job state.
- If the `upload_id` does not match the active upload session for the job, return `409 Conflict`.
- If the upload bytes are missing, unreadable, or the checksum does not match, the call may still return `202 Accepted`; the job then reaches terminal `status=failed` with a structured `failure`.

Success response:

- `202 Accepted` with `Location` to the job and `Retry-After: 5`.
- `200 OK` for an equivalent replay after upload completion was already accepted.

### `GET /v1/catalog-import-jobs/{job_id}`

Success response shape:

```json
{
  "id": "cij_01JV4S7N2Y9T6N7S43P3PZZN0E",
  "status": "succeeded",
  "phase": "terminal",
  "version": 7,
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-07T12:08:10Z",
  "started_at": "2026-03-07T12:02:00Z",
  "completed_at": "2026-03-07T12:08:10Z",
  "source_file": {
    "file_name": "catalog-2026-03-07.csv",
    "media_type": "text/csv",
    "size_bytes": 36800123,
    "checksum_sha256": "4f96b0a4f5602ef6c66b6bc4f8cf6efc9d1606d4d6e1ac9fc4d493cb7b6dfb0d"
  },
  "completion": {
    "mode": "webhook",
    "webhook_endpoint_id": "whe_01JV4RS90S8K6M4Q9FQGQ9JWKP"
  },
  "summary": {
    "rows_total": 10000,
    "rows_processed": 10000,
    "rows_succeeded": 9800,
    "rows_failed": 200
  },
  "result": {
    "outcome": "partial_success",
    "failures_available": true,
    "failures_url": "/v1/catalog-import-jobs/cij_01JV4S7N2Y9T6N7S43P3PZZN0E/item-failures"
  },
  "failure": null,
  "consistency": {
    "model": "eventual",
    "as_of": "2026-03-07T12:08:12Z",
    "may_lag": true
  }
}
```

Job semantics:

- `status` enum: `pending`, `running`, `succeeded`, `failed`, `canceled`.
- `phase` enum: `awaiting_upload`, `scanning`, `queued`, `importing`, `terminal`.
- `status=succeeded` means the system reached a row-level verdict for the whole uploaded file.
- `result.outcome` enum:
  - `complete_success`: every row was imported.
  - `partial_success`: at least one row imported and at least one row failed.
  - `no_rows_accepted`: processing completed, but zero rows were imported because every row failed business or validation checks.
- `status=failed` means the import process itself did not complete successfully. Examples: malware detected, checksum mismatch, unreadable CSV, infrastructure failure. In that case `failure` is required and `result` is `null`.
- `failure` object fields:
  - `code`: stable machine-readable code such as `malware_detected`, `checksum_mismatch`, `invalid_csv`, `processing_error`
  - `detail`: sanitized human-readable detail
  - `retryable`: boolean
  - `partial_application`: boolean indicating whether any row writes may already have been committed before failure
- `summary.rows_total` may stay `null` until parsing determines the number of rows.

Polling behavior:

- `GET` returns `ETag` based on `version`.
- `If-None-Match` returns `304 Not Modified`.
- `Cache-Control: no-store` is returned even though the server-side view may itself be cache-backed.
- `GET` does not guarantee read-after-write after upload completion or after webhook delivery.

### `GET /v1/catalog-import-jobs/{job_id}/item-failures`

Query parameters:

- `cursor`: opaque cursor from the previous page.
- `page_size`: optional, default `100`, max `500`.

Semantics:

- Stable ordering is ascending `row_number`.
- `row_number` is `1`-based and counts data rows, not the header row.
- Unknown query parameters fail with `400 Bad Request`.
- This endpoint is authoritative only when the parent job has `status=succeeded`.
- If the job is not yet in a terminal succeeded state, return `409 Conflict` with code `result_not_ready`.

Success response:

```json
{
  "data": [
    {
      "row_number": 17,
      "source_key": "SKU-123",
      "code": "invalid_price",
      "detail": "price must be greater than or equal to 0",
      "field": "price"
    },
    {
      "row_number": 18,
      "source_key": "SKU-124",
      "code": "unknown_category",
      "detail": "category code CAT-999 is not recognized",
      "field": "category_code"
    }
  ],
  "next_cursor": "g2wAAAABaANkAA1sYXN0X3Jvd19udW1iZXJLIhI=",
  "as_of": "2026-03-07T12:08:12Z"
}
```

### Webhook delivery contract

- Webhooks are sent only for jobs created with `completion.mode=webhook`.
- Delivery happens only when the job reaches a terminal state.
- Polling remains available even when webhook mode is selected.
- Delivery is at-least-once. Clients must tolerate duplicates.
- Any `2xx` response within `5s` acknowledges receipt.
- Non-`2xx` or timeout triggers retries on this schedule: immediate initial delivery, then `1m`, `5m`, `15m`, `1h`, `6h`, and `24h`.
- Stable deduplication key: webhook payload `id`.
- Signature headers:
  - `X-Webhook-Id`
  - `X-Webhook-Timestamp`
  - `X-Webhook-Signature`
- `X-Webhook-Signature` format: `v1=<hex-hmac-sha256(timestamp + "." + raw_body)>`.
- Clients must reject payloads whose `X-Webhook-Timestamp` is outside a `5 minute` replay window.

Webhook payload:

```json
{
  "id": "evt_01JV4SB3X8N8PC2XDC93YV9J6C",
  "type": "catalog-import.job.terminal",
  "occurred_at": "2026-03-07T12:08:10Z",
  "job_id": "cij_01JV4S7N2Y9T6N7S43P3PZZN0E",
  "job_version": 7,
  "status": "succeeded",
  "result": {
    "outcome": "partial_success",
    "rows_total": 10000,
    "rows_succeeded": 9800,
    "rows_failed": 200,
    "failures_url": "/v1/catalog-import-jobs/cij_01JV4S7N2Y9T6N7S43P3PZZN0E/item-failures"
  },
  "failure": null,
  "job_url": "/v1/catalog-import-jobs/cij_01JV4S7N2Y9T6N7S43P3PZZN0E"
}
```

Webhook ordering notes:

- A webhook may reflect `job_version=N` before `GET /v1/catalog-import-jobs/{job_id}` shows `version >= N`.
- If a partner receives a webhook and a later `GET` returns a smaller `version`, the partner is reading a stale projection and should retry later.

### Error model

All API errors use `Content-Type: application/problem+json`.

Problem fields:

- Required: `type`, `title`, `status`, `detail`
- Stable extensions: `code`, `request_id`, `errors`
- `errors` is used only for field-level request validation failures

Error mapping:

| Status | When used | Notes |
| --- | --- | --- |
| `400 Bad Request` | malformed JSON, trailing tokens, invalid cursor, unknown query parameter | strict decode failure |
| `401 Unauthorized` | missing or invalid auth | partner identity not established |
| `403 Forbidden` | authenticated partner lacks permission to import catalogs | capability failure |
| `404 Not Found` | unknown `job_id` or unknown / not-owned `webhook_endpoint_id` | avoids leaking ownership |
| `409 Conflict` | idempotency key reused with different payload, wrong `upload_id`, upload completion after terminal state, item failures requested before ready | state or idempotency conflict |
| `413 Payload Too Large` | upload target received more than `50 MiB` | upload-plane failure, not JSON API creation payload |
| `415 Unsupported Media Type` | `source_file.media_type` is not `text/csv` or upload uses the wrong content type | media validation |
| `422 Unprocessable Content` | semantic request error such as invalid checksum format or disabled webhook endpoint | syntactically valid, semantically invalid |
| `428 Precondition Required` | missing `Idempotency-Key` on job creation | explicit contract requirement |
| `429 Too Many Requests` | partner exceeded API rate limit | includes `Retry-After` |
| `503 Service Unavailable` | service cannot accept new jobs right now | retryable acceptance failure |

Example validation error:

```json
{
  "type": "https://api.example.com/problems/validation",
  "title": "Validation failed",
  "status": 422,
  "detail": "One or more request fields are invalid.",
  "code": "validation_error",
  "request_id": "req_01JV4S9ZQ1W1A7X3SY6EZTZ9S7",
  "errors": [
    {
      "field": "completion.webhook_endpoint_id",
      "code": "required_when_webhook",
      "detail": "webhook_endpoint_id is required when completion.mode is webhook"
    }
  ]
}
```

## Boundary And Cross-Cutting Policies

- Input media:
  - Control-plane endpoints accept `application/json`.
  - Source-file uploads accept only `text/csv`.
  - Compressed uploads such as `gzip` or `zip` are not part of `v1`.
- Size rules:
  - Maximum source file size is `50 MiB` (`52428800` bytes).
  - One file per job.
- Validation pipeline at the boundary:
  1. authenticate partner
  2. enforce route and rate limits
  3. strict JSON decode
  4. field validation and normalization
  5. accept job or upload completion
  6. async scan and processing
- Malware scanning:
  - Mandatory before any CSV parsing side effects or catalog writes.
  - A scan rejection produces terminal `status=failed`, `failure.code=malware_detected`, `failure.partial_application=false`.
- Idempotency:
  - Required only on `POST /v1/catalog-import-jobs`.
  - Deduplication TTL is `24h`.
  - Idempotency scope is partner + route + request payload.
- Concurrency / preconditions:
  - No client-visible mutable update endpoint is part of this contract, so `If-Match` is not used.
  - `GET` supports `If-None-Match` only.
- Correlation:
  - All API responses return `X-Request-Id`.
  - Stable async correlation fields are `job_id` and `version`.
  - Webhook payloads echo `job_id` and `job_version`.
- Rate limiting:
  - All JSON API endpoints return `RateLimit-Limit`, `RateLimit-Remaining`, and `RateLimit-Reset`.
  - `429` responses include `Retry-After`.
  - The upload target may enforce separate transport-level rate or size controls; clients recover through the job resource.

## Consistency And Async Notes

- Acceptance is strong only for `POST /v1/catalog-import-jobs` and `POST /v1/catalog-import-jobs/{job_id}/upload-completions`.
- `GET` endpoints are eventual because they read from a cache-backed view.
- The contract does not promise read-after-write after job creation, upload completion, or webhook delivery.
- `consistency.as_of` tells the caller how fresh the returned view is. `version` lets the caller detect stale reads after a newer webhook.
- `status=succeeded` is about process completion, not all-row success. Use `result.outcome` plus `summary` for row-level truth.
- `status=failed` is reserved for process-level failure. Clients must inspect `failure.partial_application` before assuming nothing was written.
- The item-failures collection is complete only when `status=succeeded`. A failed job may stop before producing a complete row-level verdict.

## Compatibility Notes

- Additive:
  - new optional response fields
  - new optional webhook payload fields
  - new optional problem-details extensions
- Behavior-changing:
  - changing the eventual-consistency disclosure or adding a strong-read guarantee
  - changing retry guidance, idempotency TTL, or webhook retry cadence
  - changing the meaning of `status=succeeded`, `result.outcome`, or `failure.partial_application`
- Breaking:
  - removing polling support
  - removing or renaming documented fields, headers, or enums without overlap
  - changing `job_id` stability on idempotent replay
  - changing the webhook signature algorithm or header names without a compatibility window
  - changing accepted source-file media type away from `text/csv`

- Client compatibility rule: request parsing is strict, but response parsing should ignore unknown fields.

## Open Questions And Risks

- Does a partner-managed webhook endpoint resource already exist, or does the same release need a public API to create and rotate webhook endpoints?
- Is the CSV schema fixed for all partners, or does the job request need a `schema_version` or `catalog_type` field?
- Should the API expose a refresh-upload-session endpoint explicitly, or is replaying the original idempotent create call enough for UX and supportability?
- How long are job resources and item-failure pages retained after completion?
- Is there a contractual freshness SLO for the cache-backed job view, or should `v1` expose only `version` and `consistency.as_of` without a max-lag promise?
- Do partners need a downloadable complete failure report file in addition to paginated `item-failures` responses?

# Partner Bulk Catalog Import API Contract

## Selected Decisions

| Topic | Viable options | Selected contract | Rejected option | Why | Compatibility note |
| --- | --- | --- | --- | --- | --- |
| Large-file intake | Direct multipart upload to API, or async job with upload slot | `POST /v1/partner-catalog-import-jobs` creates an async job and returns a single-use upload URL for the CSV | Direct `multipart/form-data` upload to the JSON API | 50 MB files plus mandatory malware scanning should not pin API handlers to large request bodies or pretend the import is synchronous | Baseline v1 behavior; switching to direct upload later would be a behavior change |
| Completion consumption | Polling only, webhook only, or polling plus optional webhook | Polling is always available; caller may additionally request a signed terminal-state webhook | Webhook-only completion | Partners need a recovery path when webhook delivery fails, and the cache-backed read model may lag briefly | Webhook support is additive if introduced later; removing polling would be breaking |
| Partial failures | One boolean, terminal job failure, or summary plus item-level failures | Job resource exposes summary counts; per-row failures are available via a paginated subresource | Single success flag with hidden row failures | Clients need to reconcile mixed outcomes without downloading the entire input again | Adding new failure codes is additive; changing failure item shape is behavior-sensitive |

## Resource And Endpoint Matrix

| Resource | Method and path | Semantics | Success | Retry / idempotency |
| --- | --- | --- | --- | --- |
| Import job collection | `POST /v1/partner-catalog-import-jobs` | Create an async import job and receive upload instructions for one CSV file | `202 Accepted` with `Location: /v1/partner-catalog-import-jobs/{job_id}` and initial job representation | `Idempotency-Key` required; same key + same normalized body returns the same `202` and job; same key + different body returns `409` |
| Import job | `GET /v1/partner-catalog-import-jobs/{job_id}` | Read current job state, summary, freshness metadata, and terminal result/failure details | `200 OK`; `304 Not Modified` when `If-None-Match` matches the current `ETag` | Safe and idempotent; callers should honor `Retry-After` while job is non-terminal |
| Import failure list | `GET /v1/partner-catalog-import-jobs/{job_id}/failures?cursor=...&page_size=...` | Read row-level failures for a completed job | `200 OK` | Safe and idempotent; cursor pagination; default `page_size=50`, max `200` |
| Partner webhook delivery | Outbound `POST` to the partner's configured endpoint | Notify the partner that the job reached a terminal state | Partner returns any `2xx` within `5s` to acknowledge receipt | At-least-once delivery; duplicates and out-of-order deliveries are possible |

## Upload / Initiation Flow

1. Client calls `POST /v1/partner-catalog-import-jobs` with file metadata and optional completion mode.
2. API creates the job in `awaiting_upload` state and returns a single-use upload URL plus required upload headers.
3. Client uploads the CSV bytes with `PUT` to the returned URL before `upload.expires_at`.
4. After the object is present, the service performs mandatory malware scanning.
5. Only after a successful scan may the job advance to `queued` and `running`.
6. Terminal state is discovered by polling `GET /v1/partner-catalog-import-jobs/{job_id}` or by the optional signed webhook.

Direct `multipart/form-data` upload to the JSON API is not part of v1.

## Request, Response, And Error Model

### `POST /v1/partner-catalog-import-jobs`

Required headers:

- `Authorization: Bearer <token>`
- `Idempotency-Key: <opaque-key>`
- `Content-Type: application/json`

Request body:

```json
{
  "file_name": "catalog-2026-03-07.csv",
  "declared_size_bytes": 18432123,
  "declared_media_type": "text/csv",
  "checksum_sha256": "f5fd3f7157c1f5d09f9d2f02fb1b20a5b0c4da4fd87d0f228ca5d10d9ed9f130",
  "partner_reference": "batch-2026-03-07-01",
  "completion": {
    "mode": "webhook",
    "webhook_endpoint_id": "whe_01JCV7J2K4M2Z0GQ2Y3V95M5SZ"
  }
}
```

Request rules:

- `declared_size_bytes` is required and must be `> 0` and `<= 52428800` (50 MiB).
- `declared_media_type` must be `text/csv`.
- `checksum_sha256` is required, lowercase hex, 64 chars.
- `completion.mode` is `poll` or `webhook`.
- When `completion.mode=webhook`, `webhook_endpoint_id` is required and must belong to the authenticated partner.
- Unknown JSON fields are rejected.

Success response:

```json
{
  "id": "imp_01JCV7P8W09W9A6Q8E2W1EJQW4",
  "partner_reference": "batch-2026-03-07-01",
  "status": "awaiting_upload",
  "version": 1,
  "created_at": "2026-03-07T10:00:00Z",
  "updated_at": "2026-03-07T10:00:00Z",
  "upload": {
    "method": "PUT",
    "url": "https://uploads.example.com/partner-import/imp_01JCV7P8W09W9A6Q8E2W1EJQW4",
    "required_headers": {
      "Content-Type": "text/csv",
      "X-Content-SHA256": "f5fd3f7157c1f5d09f9d2f02fb1b20a5b0c4da4fd87d0f228ca5d10d9ed9f130"
    },
    "expires_at": "2026-03-07T10:15:00Z",
    "max_size_bytes": 52428800,
    "accepted_media_types": [
      "text/csv"
    ]
  },
  "completion": {
    "mode": "webhook",
    "webhook_endpoint_id": "whe_01JCV7J2K4M2Z0GQ2Y3V95M5SZ"
  },
  "consistency": {
    "mode": "eventual",
    "as_of": "2026-03-07T10:00:00Z",
    "max_expected_lag_seconds": 10
  },
  "links": {
    "self": "/v1/partner-catalog-import-jobs/imp_01JCV7P8W09W9A6Q8E2W1EJQW4",
    "failures": "/v1/partner-catalog-import-jobs/imp_01JCV7P8W09W9A6Q8E2W1EJQW4/failures"
  }
}
```

Response headers:

- `Location: /v1/partner-catalog-import-jobs/{job_id}`
- `ETag: W/"1"`
- `Retry-After: 5`
- `X-Request-Id: <request-id>`

### Job Resource

`GET /v1/partner-catalog-import-jobs/{job_id}` returns:

- `id`: opaque import job identifier.
- `status`: one of `awaiting_upload`, `scanning`, `queued`, `running`, `succeeded`, `failed`, `expired`.
- `version`: monotonically increasing integer; increments on every visible state transition.
- `created_at`, `updated_at`: timestamps for job creation and latest included state transition.
- `completion`: requested completion mode.
- `consistency`: freshness disclosure for the cache-backed view.
- `result`: present only when `status=succeeded`; omitted otherwise.
- `failure`: present only when `status=failed` or `status=expired`.
- `links.failures`: stable link to the row-level failure list.

Successful terminal result:

```json
{
  "id": "imp_01JCV7P8W09W9A6Q8E2W1EJQW4",
  "status": "succeeded",
  "version": 6,
  "created_at": "2026-03-07T10:00:00Z",
  "updated_at": "2026-03-07T10:12:31Z",
  "completion": {
    "mode": "webhook",
    "webhook_endpoint_id": "whe_01JCV7J2K4M2Z0GQ2Y3V95M5SZ"
  },
  "consistency": {
    "mode": "eventual",
    "as_of": "2026-03-07T10:12:34Z",
    "max_expected_lag_seconds": 10
  },
  "result": {
    "outcome": "partial_failure",
    "summary": {
      "total_rows": 1000,
      "imported_count": 990,
      "failed_count": 10,
      "skipped_count": 0
    },
    "failures_truncated": false
  },
  "links": {
    "self": "/v1/partner-catalog-import-jobs/imp_01JCV7P8W09W9A6Q8E2W1EJQW4",
    "failures": "/v1/partner-catalog-import-jobs/imp_01JCV7P8W09W9A6Q8E2W1EJQW4/failures"
  }
}
```

Terminal operation-level failure:

```json
{
  "id": "imp_01JCV7P8W09W9A6Q8E2W1EJQW4",
  "status": "failed",
  "version": 4,
  "created_at": "2026-03-07T10:00:00Z",
  "updated_at": "2026-03-07T10:02:09Z",
  "consistency": {
    "mode": "eventual",
    "as_of": "2026-03-07T10:02:11Z",
    "max_expected_lag_seconds": 10
  },
  "failure": {
    "stage": "scan",
    "code": "malware_detected",
    "detail": "The uploaded file did not pass malware scanning.",
    "retryable": false
  },
  "links": {
    "self": "/v1/partner-catalog-import-jobs/imp_01JCV7P8W09W9A6Q8E2W1EJQW4"
  }
}
```

### Partial-Failure Representation

Mixed row outcomes do not force `status=failed`. The job may finish with:

- `status=succeeded` and `result.outcome=success` when every row was imported.
- `status=succeeded` and `result.outcome=partial_failure` when at least one row imported and at least one row failed.
- `status=succeeded` and `result.outcome=all_failed` when processing completed but zero rows imported because every row failed item-level validation.

`GET /v1/partner-catalog-import-jobs/{job_id}/failures` response:

```json
{
  "job_id": "imp_01JCV7P8W09W9A6Q8E2W1EJQW4",
  "result_outcome": "partial_failure",
  "page_size": 50,
  "next_cursor": "eyJvZmZzZXQiOjUwfQ",
  "items": [
    {
      "row_number": 24,
      "partner_item_key": "SKU-1234",
      "code": "invalid_price",
      "detail": "price must be greater than or equal to 0",
      "field": "price",
      "rejected_value": "-1"
    },
    {
      "row_number": 25,
      "partner_item_key": "SKU-1235",
      "code": "unknown_category",
      "detail": "category_id does not exist",
      "field": "category_id",
      "rejected_value": "cat-999"
    }
  ]
}
```

Item-level failure codes are an open set. Clients must not hard-fail on unknown `items[].code`.

### Problem Details Error Model

Errors use `application/problem+json`.

Example:

```json
{
  "type": "https://api.example.com/problems/unsupported-media-type",
  "title": "Unsupported media type",
  "status": 415,
  "detail": "declared_media_type must be text/csv",
  "code": "unsupported_media_type",
  "request_id": "req_01JCV7P2M7TANQ8KQ4V1N4B0D9",
  "errors": [
    {
      "field": "declared_media_type",
      "message": "must be text/csv"
    }
  ]
}
```

Error mapping:

| Status | When | Notes |
| --- | --- | --- |
| `400 Bad Request` | Malformed JSON, trailing tokens, or unknown fields | Strict decode failure |
| `401 Unauthorized` | Missing or invalid bearer token | No partner context from caller-supplied headers |
| `403 Forbidden` | Partner lacks import capability or references a webhook endpoint it does not own | Authorization failure |
| `404 Not Found` | Unknown `job_id` or unknown `webhook_endpoint_id` | Stable not-found semantics |
| `409 Conflict` | Same `Idempotency-Key` reused with a different request body | Distinct from validation failure |
| `413 Payload Too Large` | `declared_size_bytes` exceeds `52428800`, or uploaded object exceeds the signed upload limit | Upload URL may surface `413` directly |
| `415 Unsupported Media Type` | `declared_media_type` is not `text/csv`, or uploaded bytes are identified as a different media type | Zip archives and spreadsheets are rejected |
| `422 Unprocessable Content` | Semantically invalid request, such as malformed `checksum_sha256` or `completion.mode=webhook` without `webhook_endpoint_id` | Request shape is valid JSON but semantically invalid |
| `428 Precondition Required` | Missing `Idempotency-Key` on `POST /v1/partner-catalog-import-jobs` | Retry-unsafe create without dedup key is rejected |
| `429 Too Many Requests` | Partner exceeds create, poll, or failure-list quota | Includes `Retry-After`; may include `RateLimit-Limit`, `RateLimit-Remaining`, `RateLimit-Reset` |

Malware detection, CSV parse failure, and downstream import failure are not surfaced as synchronous `4xx` responses after job creation; they appear as terminal job state in `GET /v1/partner-catalog-import-jobs/{job_id}`.

## Boundary And Cross-Cutting Policies

- Auth context is derived from the validated bearer token. The API does not accept caller-chosen `partner_id` headers.
- `POST /v1/partner-catalog-import-jobs` is retry-safe by contract only when `Idempotency-Key` is supplied.
- Idempotency scope is partner + route + normalized request body. Deduplication TTL is `24h`.
- The upload URL is single-use and expires after `15m`.
- Accepted upload media type is `text/csv` only. Compressed archives, Excel workbooks, and multi-file bundles are rejected.
- The `50 MiB` limit applies to the uploaded object, not a compressed representation.
- Malware scanning is mandatory and blocks progression to `queued` or `running`.
- Polling clients should honor `Retry-After`; if absent, they should wait at least `5s` between `GET` requests for the same non-terminal job.
- `GET` supports `ETag` and `If-None-Match` for cache-friendly polling.
- `X-Request-Id` is returned on API responses. `job_id` and `version` are the stable correlation fields across polling and webhook delivery.

## Webhook Contract

Webhook delivery is optional and only used when `completion.mode=webhook`.

Outbound request:

- Method: `POST`
- `Content-Type: application/json`
- `X-Webhook-Id: evt_<opaque>`
- `X-Webhook-Timestamp: 2026-03-07T10:12:35Z`
- `X-Webhook-Signature: v1=<hex(hmac_sha256(secret, timestamp + "." + raw_body))>`
- `X-Webhook-Event: catalog.import.completed`

Payload:

```json
{
  "id": "evt_01JCV86Y4QJFXM8RG9GAGBW93Q",
  "type": "catalog.import.completed",
  "occurred_at": "2026-03-07T10:12:35Z",
  "job_id": "imp_01JCV7P8W09W9A6Q8E2W1EJQW4",
  "job_version": 6,
  "status": "succeeded",
  "result": {
    "outcome": "partial_failure",
    "summary": {
      "total_rows": 1000,
      "imported_count": 990,
      "failed_count": 10,
      "skipped_count": 0
    },
    "failures_url": "/v1/partner-catalog-import-jobs/imp_01JCV7P8W09W9A6Q8E2W1EJQW4/failures"
  },
  "links": {
    "job": "/v1/partner-catalog-import-jobs/imp_01JCV7P8W09W9A6Q8E2W1EJQW4"
  }
}
```

Webhook semantics:

- Sent only for terminal states: `succeeded`, `failed`, or `expired`.
- Delivery is at least once; duplicates and out-of-order delivery are possible.
- Deduplication key is `id` / `X-Webhook-Id`.
- Partner verifies the HMAC signature over `timestamp + "." + raw_body`.
- Replay window is `5m`; partners should reject timestamps older than `5m` or more than `5m` in the future.
- Sender timeout is `5s`.
- Any `2xx` response acknowledges the delivery.
- Retry schedule after non-`2xx` or timeout: `30s`, `2m`, `10m`, `1h`, `6h`, then stop.
- Webhook is a notification channel, not a separate source of truth; clients can always re-read the job resource.

## Consistency And Async Notes

State machine:

| Status | Meaning | Terminal |
| --- | --- | --- |
| `awaiting_upload` | Job exists; service is waiting for the partner to upload the CSV | No |
| `scanning` | File uploaded; malware scanning in progress | No |
| `queued` | Scan passed; job accepted for import processing | No |
| `running` | Import is actively parsing and applying rows | No |
| `succeeded` | Processing completed; inspect `result.outcome` for full vs partial success | Yes |
| `failed` | Operation-level failure prevented useful processing | Yes |
| `expired` | Partner did not upload the file before the upload slot expired | Yes |

Freshness disclosure:

- `GET /v1/partner-catalog-import-jobs/{job_id}` is an eventual-consistency read backed by a cache-backed projection.
- `consistency.as_of` states how fresh the visible projection is.
- `consistency.max_expected_lag_seconds` is the documented lag target for that view.
- The contract does not guarantee read-after-write after upload completion or webhook delivery.
- If a client receives a webhook with `job_version=N` and a subsequent `GET` returns `version < N`, the client is reading a stale projection and should retry.

## Compatibility Notes

- Additive changes in v1:
  - New optional top-level fields in job, result, failure, or webhook payloads.
  - New item-level failure codes.
  - Additional summary counters inside `result.summary`.
- Behavior-changing changes in v1:
  - Changing freshness guarantees or the meaning of `consistency.max_expected_lag_seconds`.
  - Changing retry cadence guidance or webhook retry schedule.
  - Changing the meaning of `result.outcome`.
- Breaking changes:
  - Lowering the file-size limit below `50 MiB`.
  - Accepting media types other than `text/csv` without versioning existing behavior.
  - Renaming webhook signature headers or changing the signing algorithm.
  - Removing polling support, the failure subresource, or required fields already defined in this contract.

## Open Questions And Risks

- Is `webhook_endpoint_id` backed by an existing partner-managed webhook-endpoint resource, or does this API also need a way to register/update endpoints?
- What retention window should apply to uploaded source files, job metadata, and `GET /failures` results?
- Does the CSV format need an explicit `schema_version` field so template evolution stays compatibility-safe?
- Is the freshness target of `10s` acceptable for the cache-backed job view, or does the UI need a tighter bound?
- Should v1 support client-initiated cancellation or replacement of an `awaiting_upload` / `queued` job, or is retry by creating a new job sufficient?

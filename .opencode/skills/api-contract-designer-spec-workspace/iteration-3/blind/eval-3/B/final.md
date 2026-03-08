# Bulk Catalog Import API Contract

## Contract Framing And Assumptions

- Audience: authenticated partner integrations and the partner-facing UI.
- Problem: accept one CSV upload up to `50,000,000` bytes, require malware scanning before any catalog effects, process asynchronously, expose a stable job resource, support signed completion webhooks for partners that opt in, represent partial row failures explicitly, and disclose that job reads come from a cache-backed view that can lag.
- Selected option: two-step upload session plus async job resource. Rejected direct `multipart/form-data POST /v1/catalog-import-jobs` because a `50 MB` file plus mandatory scanning makes synchronous-looking acceptance misleading and harder to retry safely.
- Selected option: polling via the job resource is always available; webhook delivery is an optional terminal notification mode on top of the same job resource. Rejected webhook-only because some partners and the UI still need read access and reconciliation.
- Selected option: webhook mode uses the partner's pre-registered default completion webhook target, not an arbitrary per-request callback URL. Rejected caller-supplied URLs in `v1` because partner-facing callbacks should prefer pre-registered or ownership-verified targets and the prompt does not require per-request URL flexibility.
- Selected option: partial row failures are represented as a terminal job status plus a paginated failures subresource. Rejected `207 Multi-Status` and an inline unbounded failure array because the import is asynchronous and the failure list can be large.
- Auth scope: partner identity and tenant binding come from authenticated credentials, not from caller-supplied partner IDs in the path or body.

## Resource And Endpoint Matrix

| Endpoint | Method | Success | Purpose | Consistency | Retry class |
| --- | --- | --- | --- | --- | --- |
| `/v1/catalog-import-upload-sessions` | `POST` | `201 Created` | Reserve one upload slot for a CSV object and return an opaque upload URL | strong | retry-safe by contract with `Idempotency-Key` |
| `/v1/catalog-import-upload-sessions/{session_id}` | `GET` | `200 OK` | Read upload-session state | strong | retry-safe by protocol |
| opaque `upload.url` returned by the session | `PUT` | `2xx` from upload target | Upload raw CSV bytes | strong | retry-safe until session expiry |
| `/v1/catalog-import-jobs` | `POST` | `202 Accepted` | Start malware scan and async import from an uploaded session | strong on acceptance | retry-safe by contract with `Idempotency-Key` |
| `/v1/catalog-import-jobs/{job_id}` | `GET` | `200 OK` | Read the import job and terminal summary from the cache-backed view | eventual | retry-safe by protocol |
| `/v1/catalog-import-jobs/{job_id}/failures` | `GET` | `200 OK` | Page through per-row failures for a terminal job | eventual | retry-safe by protocol |

### Upload / Initiate Flow

1. `POST /v1/catalog-import-upload-sessions`
2. `PUT` the CSV bytes to the returned opaque `upload.url`
3. `POST /v1/catalog-import-jobs`
4. `GET /v1/catalog-import-jobs/{job_id}` until terminal, or request webhook mode and reconcile against the same job resource if needed

### Upload Session States

| State | Meaning |
| --- | --- |
| `open` | Session exists and can accept exactly one upload. |
| `uploaded` | The object upload finished and can be consumed by one job. |
| `consumed` | A job has already been created from this session. |
| `expired` | The session can no longer accept or start work. |

### Job Statuses

| Status | Meaning |
| --- | --- |
| `queued` | Job accepted but not yet scanning. |
| `scanning` | Malware scanning and file-safety checks are running. No catalog effects have started. |
| `processing` | Scan passed and row evaluation/import is running. |
| `succeeded` | All rows were evaluated and applied. |
| `succeeded_with_errors` | All rows were evaluated; accepted rows were applied, rejected rows were not, and row failures are available via `/failures`. |
| `failed` | The job ended without applying catalog changes. Examples: malware detected, unreadable CSV, expired or invalid upload object, unrecoverable dependency or internal fault before commit. |

`failed` is intentionally distinct from `succeeded_with_errors`. If the service cannot guarantee that `failed` means no catalog mutations from that job were committed, this contract must be reopened before coding.

## Request, Response, And Error Model

### `POST /v1/catalog-import-upload-sessions`

Required headers:

- `Idempotency-Key`
- `Content-Type: application/json`

Request:

```json
{
  "file_name": "catalog-2026-03-07.csv",
  "media_type": "text/csv",
  "file_size_bytes": 49753211
}
```

Rules:

- `file_name`, `media_type`, and `file_size_bytes` are required.
- `media_type` must be exactly `text/csv`.
- `file_size_bytes` must be between `1` and `50,000,000`.
- Unknown JSON fields, malformed JSON, trailing tokens, and wrong field types fail with `400`.

Response `201 Created`:

- `Location: /v1/catalog-import-upload-sessions/cis_01JNV1T4K9V0Z0X9M5A3J4M1QY`

```json
{
  "id": "cis_01JNV1T4K9V0Z0X9M5A3J4M1QY",
  "status": "open",
  "file_name": "catalog-2026-03-07.csv",
  "media_type": "text/csv",
  "file_size_bytes": 49753211,
  "created_at": "2026-03-07T12:00:00Z",
  "expires_at": "2026-03-07T12:15:00Z",
  "upload": {
    "method": "PUT",
    "url": "https://uploads.example.invalid/cis_01JNV1T4K9V0Z0X9M5A3J4M1QY",
    "required_headers": {
      "Content-Type": "text/csv"
    }
  }
}
```

Contract notes:

- The opaque `upload.url` is outside the versioned REST surface. Clients may rely on its method, required headers, expiry, and status-code class only; they must not depend on a storage-specific error body schema.
- A successful upload means bytes were stored, not that the file passed validation or malware scanning.
- Actual uploaded size must not exceed the declared `file_size_bytes` or the hard `50,000,000` byte limit.
- A session may be consumed by exactly one job.

### `GET /v1/catalog-import-upload-sessions/{session_id}`

Response `200 OK` returns the same shape as create, plus:

- `uploaded_at` when `status=uploaded`
- `consumed_at` and `job_id` when `status=consumed`

### `POST /v1/catalog-import-jobs`

Required headers:

- `Idempotency-Key`
- `Content-Type: application/json`

Request:

```json
{
  "upload_session_id": "cis_01JNV1T4K9V0Z0X9M5A3J4M1QY",
  "completion_delivery": {
    "mode": "webhook"
  }
}
```

Rules:

- `upload_session_id` is required.
- `completion_delivery` is optional. If omitted, the default is `{"mode":"poll"}`.
- `completion_delivery.mode` is one of `poll` or `webhook`.
- `webhook` mode uses the partner's pre-registered default completion webhook target. If no verified target and active secret are configured for the partner account, the request fails with `422`.
- The referenced upload session must be in `uploaded` state. `open` or `consumed` sessions fail with `409`. `expired` sessions fail with `410`.

Response `202 Accepted`:

- `Location: /v1/catalog-import-jobs/job_01JNV1X5CBR4W1FVC0SP0V84ZQ`

```json
{
  "id": "job_01JNV1X5CBR4W1FVC0SP0V84ZQ",
  "status": "queued",
  "version": 1,
  "created_at": "2026-03-07T12:02:00Z",
  "updated_at": "2026-03-07T12:02:00Z",
  "upload_session_id": "cis_01JNV1T4K9V0Z0X9M5A3J4M1QY",
  "summary": {
    "total_items": null,
    "accepted_items": null,
    "rejected_items": null
  },
  "completion_delivery": {
    "mode": "webhook",
    "webhook": {
      "delivery_status": "pending",
      "attempt_count": 0,
      "last_attempt_at": null
    }
  }
}
```

### `GET /v1/catalog-import-jobs/{job_id}`

Response headers:

- `Content-Type: application/json`
- `Cache-Control: no-store`

Representative terminal response with partial failures:

```json
{
  "id": "job_01JNV1X5CBR4W1FVC0SP0V84ZQ",
  "status": "succeeded_with_errors",
  "version": 4,
  "created_at": "2026-03-07T12:02:00Z",
  "updated_at": "2026-03-07T12:08:31Z",
  "started_at": "2026-03-07T12:02:08Z",
  "completed_at": "2026-03-07T12:08:31Z",
  "upload_session_id": "cis_01JNV1T4K9V0Z0X9M5A3J4M1QY",
  "summary": {
    "total_items": 12500,
    "accepted_items": 12430,
    "rejected_items": 70
  },
  "failures": {
    "count": 70,
    "href": "/v1/catalog-import-jobs/job_01JNV1X5CBR4W1FVC0SP0V84ZQ/failures"
  },
  "completion_delivery": {
    "mode": "webhook",
    "webhook": {
      "delivery_status": "delivered",
      "attempt_count": 1,
      "last_attempt_at": "2026-03-07T12:08:33Z"
    }
  },
  "freshness": {
    "consistency": "eventual",
    "as_of": "2026-03-07T12:08:34Z"
  }
}
```

Job resource rules:

- `version` is a monotonically increasing integer for externally visible job-state transitions. Webhooks carry the same job `version`. If a client has observed a larger webhook `version` than the latest `GET` response, the cache-backed view has not caught up yet.
- `summary.*` counts remain `null` until the job reaches a terminal state.
- `failed` jobs expose a `failure` object with `code`, `message`, and `stage`; they do not expose committed-row counts.
- `succeeded_with_errors` means every row was evaluated. Accepted rows were committed. Rejected rows were not committed.
- `freshness.consistency` is always `eventual` for this endpoint.

### `GET /v1/catalog-import-jobs/{job_id}/failures`

Query parameters:

- `cursor`: opaque pagination cursor, optional
- `page_size`: optional, default `50`, maximum `200`

Rules:

- Unknown query parameters fail with `400`.
- Rows are returned in ascending `row_number`.
- Non-terminal jobs fail with `409`.
- `succeeded` jobs return `200` with `items: []`.

Response `200 OK`:

```json
{
  "items": [
    {
      "row_number": 18,
      "item_key": "SKU-12345",
      "code": "invalid_price",
      "message": "price must be greater than or equal to 0",
      "field": "price"
    },
    {
      "row_number": 41,
      "item_key": "SKU-98765",
      "code": "missing_required_field",
      "message": "name is required",
      "field": "name"
    }
  ],
  "next_cursor": "eyJyb3ciOjQxfQ",
  "page_size": 50,
  "freshness": {
    "consistency": "eventual",
    "as_of": "2026-03-07T12:08:35Z"
  }
}
```

Per-row failure rules:

- `code` is an open string enum. Clients must not assume the list is closed.
- `row_number` is the CSV row number after the header row.
- `item_key` is present only when the row contained a parseable business identifier.

### Error Model

All versioned REST endpoints under `/v1/...` use `application/problem+json`.

Problem Details shape:

```json
{
  "type": "https://api.example.invalid/problems/upload-session-expired",
  "title": "Upload session expired",
  "status": 410,
  "detail": "Upload session cis_01JNV1T4K9V0Z0X9M5A3J4M1QY expired before a job was created.",
  "code": "upload_session_expired",
  "request_id": "req_01JNV20DZZ8A6FJ2FQX3V9V1G2",
  "errors": []
}
```

Common mappings:

| Status | When | Example `code` |
| --- | --- | --- |
| `400` | malformed JSON, unknown fields, invalid query params | `invalid_request` |
| `401` | missing or invalid partner credentials | `unauthenticated` |
| `403` | partner authenticated but not allowed to import | `forbidden` |
| `404` | unknown job or upload session in caller scope | `not_found` |
| `409` | upload session in the wrong state, failure list requested before terminal, or idempotency-key payload mismatch | `upload_session_consumed` |
| `410` | upload session expired | `upload_session_expired` |
| `413` | declared or actual uploaded object exceeds `50,000,000` bytes | `payload_too_large` |
| `415` | declared or uploaded media type is not `text/csv` | `unsupported_media_type` |
| `422` | semantically invalid request, including webhook mode without a configured default target | `validation_error` |
| `429` | request-rate or active-job quota exceeded | `rate_limited` |
| `503` | temporary dependency outage during session creation or job acceptance | `service_unavailable` |

Negative-path rules:

- Malware detection is not surfaced as an HTTP error on `POST /v1/catalog-import-jobs`; it is surfaced as terminal `status=failed` with `failure.code=malware_detected`.
- CSV parse or schema problems discovered after acceptance are surfaced on the job resource, not by changing the original `202 Accepted`.
- `429` responses include `Retry-After`.
- Success responses never embed an error payload.

## Retry, Idempotency, And Concurrency Rules

- `POST /v1/catalog-import-upload-sessions` requires `Idempotency-Key`.
- `POST /v1/catalog-import-jobs` requires `Idempotency-Key`.
- Idempotency keys are scoped to partner plus route and retained for `24h`.
- Same key plus same request payload returns the same semantic outcome, including the same resource ID.
- Same key plus different payload fails with `409`.
- `GET` endpoints are retry-safe by protocol.
- Upload `PUT` calls are retryable until session expiry, but only one completed object is retained. If a client cannot tell whether an upload succeeded, it should `GET` the upload session before retrying.
- One upload session may back exactly one logical job. Replays that resolve to the original accepted job are allowed through idempotency; a new logical request against an already consumed session is rejected with `409`.
- Write-side optimistic concurrency via `ETag` is not part of `v1` because clients do not mutate session or job resources after creation.

## Async, Freshness, And Webhook Notes

- `202 Accepted` on job creation means only that the service accepted responsibility to scan and process the uploaded object. It does not mean the object passed malware scanning or that any catalog rows were imported.
- Malware scanning is mandatory and always happens before `processing`.
- `GET /v1/catalog-import-jobs/{job_id}` and `/failures` are explicit eventual-consistency reads served from a cache-backed view.
- `freshness.as_of` is the timestamp of the cached representation returned to the client.
- There is no read-after-write guarantee after `POST /v1/catalog-import-jobs`.

### Signed Completion Webhook Contract

- Webhook mode emits one logical terminal event type: `catalog-import.job.completed`.
- The event is sent for `succeeded`, `succeeded_with_errors`, and `failed`.
- Delivery is at least once. Duplicates are possible. Clients must deduplicate on `X-Webhook-Id`.
- Delivery order is not guaranteed relative to polling.
- The first delivery attempt is immediate after the job reaches a terminal state.
- Retry schedule after a non-`2xx` response or timeout: `1m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `8h`, then stop.
- Receiver timeout expectation: return any `2xx` within `5` seconds.
- Replay protection window: `5` minutes from `X-Webhook-Timestamp`.
- Signature algorithm: HMAC-SHA256 over `X-Webhook-Timestamp + "." + raw_body`.

Webhook headers:

- `X-Webhook-Id`
- `X-Webhook-Timestamp`
- `X-Webhook-Signature: v1=<hex-hmac-sha256>`
- `Content-Type: application/json`

Webhook body:

```json
{
  "id": "wh_01JNV24F2Q4G6YDS1X5QBGJY4Q",
  "type": "catalog-import.job.completed",
  "occurred_at": "2026-03-07T12:08:31Z",
  "job": {
    "id": "job_01JNV1X5CBR4W1FVC0SP0V84ZQ",
    "status": "succeeded_with_errors",
    "version": 4,
    "summary": {
      "total_items": 12500,
      "accepted_items": 12430,
      "rejected_items": 70
    },
    "failures": {
      "count": 70,
      "href": "/v1/catalog-import-jobs/job_01JNV1X5CBR4W1FVC0SP0V84ZQ/failures"
    },
    "href": "/v1/catalog-import-jobs/job_01JNV1X5CBR4W1FVC0SP0V84ZQ"
  }
}
```

Webhook payload rules:

- For `failed`, `job.summary` may be `null` and the payload instead includes `job.failure` with `code`, `message`, and `stage`.
- The webhook represents source-of-truth terminal state; the cache-backed `GET` view may still show an older `version`.
- Job-resource webhook delivery states are `pending`, `delivered`, and `exhausted`.

## Compatibility, Artifact Updates, And Handoffs

- Compatibility class: additive new `v1` surface.
- Safe additive changes inside `v1`: new optional response fields, new per-row or terminal failure `code` values, and new webhook signature versions added alongside `v1`.
- Behavior changes that require explicit review and likely a versioning or coexistence plan: changing accepted media types, changing the maximum upload size, weakening the `failed means no applied catalog changes` invariant, adding or renaming job statuses, changing rate-limit semantics, changing webhook retry semantics, or changing freshness guarantees.
- This document is the client-visible contract only. Malware engine choice, object-store wiring, worker topology, and cache implementation are intentionally out of scope unless they change these wire semantics.
- Adjacent follow-on work is still needed for webhook target registration and secret lifecycle, but that work should refine this contract rather than replace it.

## Open Questions, Risks, And Reopen Conditions

- Does `50 MB` mean decimal `50,000,000` bytes or binary `52,428,800` bytes? This contract currently assumes decimal.
- Is one pre-registered default completion webhook target per partner sufficient, or do partners need multiple named targets in `v1`?
- What freshness SLA, if any, should be documented for the cache-backed job view?
- How long should upload sessions, terminal jobs, failure pages, and webhook delivery metadata be retained?
- Is a single opaque `PUT` upload enough for poor network conditions, or is resumable upload a requirement for some partners?
- Reopen this contract before coding if the service cannot guarantee the `failed` invariant, cannot verify webhook target ownership out of band, or cannot expose a monotonic job `version` for webhook-versus-poll reconciliation.

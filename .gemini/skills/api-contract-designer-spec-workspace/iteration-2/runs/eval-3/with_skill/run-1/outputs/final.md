# Bulk Catalog Import API Contract

## Contract Framing And Assumptions

- Audience: authenticated partner systems and the partner-facing UI.
- Problem: accept one CSV up to `50,000,000` bytes, require malware scanning before any import effects, process asynchronously, expose a stable job resource, support optional signed completion webhooks, and disclose that `GET` state is served from a cache-backed view that may lag.
- Selected option: two-step upload session plus async job. Rejected direct `multipart/form-data POST /v1/catalog-import-jobs` because 50 MB uploads, mandatory malware scanning, and retry ambiguity make synchronous-looking acceptance misleading.
- Selected option: polling remains the baseline via `GET /v1/catalog-import-jobs/{job_id}`; an optional terminal webhook may be requested on job creation. Rejected webhook-only because not every partner can host a receiver. Rejected progress webhooks in `v1` because they add ordering noise without solving the completion use case.
- Selected option: per-item partial-success semantics. Rejected all-or-nothing because the prompt requires partial item failures. Rejected inline full failure arrays and `207 Multi-Status` because failure volume is unbounded and the contract is async.
- Auth scope: partner identity and tenant binding come from authenticated credentials, not caller-supplied partner IDs in the path or body.

## Resource And Endpoint Matrix

| Endpoint | Method | Success | Purpose | Consistency | Retry |
| --- | --- | --- | --- | --- | --- |
| `/v1/catalog-import-upload-sessions` | `POST` | `201 Created` | Reserve one upload slot for a CSV file and return an opaque upload URL | strong | retry-safe by contract with `Idempotency-Key` |
| `/v1/catalog-import-upload-sessions/{session_id}` | `GET` | `200 OK` | Inspect whether the session is still open, uploaded, consumed, or expired | strong | retry-safe by protocol |
| opaque `upload.url` from session | `PUT` | `2xx` from upload target | Upload raw CSV bytes | strong | retry-safe until session expiry |
| `/v1/catalog-import-jobs` | `POST` | `202 Accepted` | Start malware scan and async import from an uploaded session | strong on acceptance | retry-safe by contract with `Idempotency-Key` |
| `/v1/catalog-import-jobs/{job_id}` | `GET` | `200 OK` | Read job state and terminal summary | eventual | retry-safe by protocol |
| `/v1/catalog-import-jobs/{job_id}/failures` | `GET` | `200 OK` | Page through rejected rows for terminal jobs | eventual | retry-safe by protocol |

### Upload Flow

1. Client creates an upload session.
2. Client uploads the raw CSV bytes to the opaque `upload.url` returned by that session.
3. Client creates a job from the uploaded session.
4. Client polls the job resource or receives one signed terminal webhook if requested.

### Upload Session State Model

| State | Meaning |
| --- | --- |
| `open` | Session exists and can accept exactly one upload. |
| `uploaded` | Upload completed and may be consumed by one job. |
| `consumed` | A job has already been created from this session. |
| `expired` | The upload window elapsed before the session was consumed. |

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
- Unknown JSON fields, malformed JSON, and trailing tokens fail with `400`.

Response `201 Created`:

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

- The opaque `upload.url` is not part of the versioned REST surface. Only a `2xx` upload response means the file bytes were stored.
- Uploading the file does not mean the file is accepted for import. Acceptance happens only after job creation and subsequent malware scanning.
- A successful upload transitions the session to `uploaded`.
- A session may be consumed by exactly one job.

### `GET /v1/catalog-import-upload-sessions/{session_id}`

Response `200 OK` returns the latest upload-session resource with the same shape as create, plus:

- `uploaded_at` when `status=uploaded`
- `consumed_at` when `status=consumed`
- `job_id` when `status=consumed`

### `POST /v1/catalog-import-jobs`

Required headers:

- `Idempotency-Key`
- `Content-Type: application/json`

Request:

```json
{
  "upload_session_id": "cis_01JNV1T4K9V0Z0X9M5A3J4M1QY",
  "completion_webhook": {
    "url": "https://partner.example.com/hooks/catalog-imports"
  }
}
```

Rules:

- `upload_session_id` is required.
- `completion_webhook` is optional. If omitted, clients use polling only.
- `completion_webhook.url` must be an absolute `https` URL.
- The webhook is signed with the partner account's configured webhook secret. If the partner has no configured secret, `POST /v1/catalog-import-jobs` fails with `422`.
- The referenced upload session must be in `uploaded` state. `open` and `consumed` sessions fail with `409`. `expired` sessions fail with `410`.

Response `202 Accepted`:

- `Location: /v1/catalog-import-jobs/job_01JNV1X5CBR4W1FVC0SP0V84ZQ`
- `Content-Type: application/json`

```json
{
  "id": "job_01JNV1X5CBR4W1FVC0SP0V84ZQ",
  "status": "queued",
  "created_at": "2026-03-07T12:02:00Z",
  "updated_at": "2026-03-07T12:02:00Z",
  "upload_session_id": "cis_01JNV1T4K9V0Z0X9M5A3J4M1QY",
  "summary": {
    "total_items": null,
    "accepted_items": null,
    "rejected_items": null
  },
  "completion_webhook": {
    "url": "https://partner.example.com/hooks/catalog-imports",
    "delivery_status": "pending",
    "attempt_count": 0,
    "last_attempt_at": null
  }
}
```

### Job Status Model

| Status | Meaning |
| --- | --- |
| `queued` | Job accepted but not yet scanning. |
| `scanning` | Malware scanning and file safety checks are running. No catalog effects have started. |
| `processing` | Scan passed and row evaluation/import is running. |
| `succeeded` | All rows were evaluated and applied. |
| `completed_with_errors` | All rows were evaluated; accepted rows were applied and rejected rows are available via the failures subresource. |
| `failed` | The job ended without committing catalog changes. Causes include malware detection, unreadable file, expired upload, or unrecoverable dependency/internal failure. |

### `GET /v1/catalog-import-jobs/{job_id}`

Response headers:

- `Content-Type: application/json`
- `Cache-Control: no-store`

Representative terminal response with partial failures:

```json
{
  "id": "job_01JNV1X5CBR4W1FVC0SP0V84ZQ",
  "status": "completed_with_errors",
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
  "completion_webhook": {
    "url": "https://partner.example.com/hooks/catalog-imports",
    "delivery_status": "delivered",
    "attempt_count": 1,
    "last_attempt_at": "2026-03-07T12:08:33Z"
  },
  "freshness": {
    "consistency": "eventual",
    "view_as_of": "2026-03-07T12:08:34Z",
    "source_last_updated_at": "2026-03-07T12:08:31Z"
  }
}
```

Job resource rules:

- `summary.*` counts remain `null` until the job reaches a terminal state.
- `failed` jobs expose a `failure` object with `code`, `message`, and `stage`; they do not expose partial committed rows.
- `completed_with_errors` means every row was evaluated. Accepted rows were committed. Rejected rows were not committed.
- `freshness.consistency` is always `eventual` on `GET /v1/catalog-import-jobs/{job_id}` because the response is served from a cache-backed view.

### `GET /v1/catalog-import-jobs/{job_id}/failures`

Query parameters:

- `cursor`: opaque pagination cursor, optional.
- `page_size`: optional, default `50`, maximum `200`.

Ordering:

- Rows are returned in ascending `row_number`.
- Unknown query parameters fail with `400`.

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
    "view_as_of": "2026-03-07T12:08:35Z",
    "source_last_updated_at": "2026-03-07T12:08:31Z"
  }
}
```

Failures endpoint rules:

- Non-terminal jobs fail with `409`.
- `succeeded` jobs return `200` with `items: []`.
- Failure `code` is an open string enum. Clients must not assume the list is closed.

### Error Model

All REST endpoints under `/v1/...` use `application/problem+json`.

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

Common status mappings:

| Status | When | Example `code` |
| --- | --- | --- |
| `400` | malformed JSON, unknown fields, invalid query params | `invalid_request` |
| `401` | missing or invalid partner credentials | `unauthenticated` |
| `403` | partner authenticated but not allowed to import | `forbidden` |
| `404` | unknown job or upload session in caller scope | `not_found` |
| `409` | upload session not ready, already consumed, or idempotency payload mismatch | `upload_session_consumed` |
| `410` | upload session expired | `upload_session_expired` |
| `413` | actual uploaded object exceeds `50,000,000` bytes | `payload_too_large` |
| `415` | declared or uploaded media type is not `text/csv` | `unsupported_media_type` |
| `422` | semantically invalid request, including invalid webhook URL or missing configured webhook secret | `validation_error` |
| `429` | request-rate or active-job quota exceeded | `rate_limited` |
| `503` | temporary dependency outage during session or job creation | `service_unavailable` |

Negative-path rules:

- Malware detection is not surfaced as an HTTP error on `POST /v1/catalog-import-jobs`; it is surfaced as job `status=failed` with `failure.code=malware_detected`.
- Transient processing faults after acceptance are surfaced on the job resource, not by changing the original `202 Accepted`.
- Success responses never embed an error payload.

## Retry, Idempotency, And Concurrency Rules

- `POST /v1/catalog-import-upload-sessions` requires `Idempotency-Key`.
- `POST /v1/catalog-import-jobs` requires `Idempotency-Key`.
- Idempotency keys are scoped to partner plus route and are retained for `24h`.
- Same key plus same payload returns the same semantic outcome, including the same resource ID.
- Same key plus different payload fails with `409`.
- `GET` endpoints are retry-safe by protocol.
- Job resources are read-only after creation, so `If-Match` and write-side optimistic concurrency are not part of `v1`.
- A single upload session may back exactly one job. Replays that resolve to the same original job are allowed through idempotency. New logical job creation attempts against a consumed session are rejected.

## Async, Freshness, And Webhook Notes

- `202 Accepted` on job creation means only that the service accepted responsibility to scan and process the uploaded file. It does not mean the file passed malware scanning or that any rows were imported.
- Malware scanning is mandatory and always happens before row processing. A job never reaches `processing` before the scan passes.
- `GET /v1/catalog-import-jobs/{job_id}` and `GET /v1/catalog-import-jobs/{job_id}/failures` are explicitly eventual-consistency reads backed by a cache-backed view.
- `freshness.view_as_of` is the timestamp of the cached representation returned to the client.
- `freshness.source_last_updated_at` is the latest source-of-truth status transition known to the view. Absence of equality between these timestamps means the view may still be catching up.
- Clients must not assume read-after-write consistency after `POST /v1/catalog-import-jobs`.
- If a signed completion webhook is configured, the webhook reflects the source-of-truth terminal state and may arrive before the cached `GET` view converges.

### Signed Completion Webhook Contract

- Only one logical terminal event is emitted per job: `catalog-import.job.completed`.
- The event is sent for `succeeded`, `completed_with_errors`, and `failed` terminal states.
- Delivery is at least once. Duplicates are possible. Clients must deduplicate on webhook `id`.
- Delivery order is not guaranteed relative to polling responses.
- Receiver timeout expectation: respond with a `2xx` within `5` seconds.
- Retry schedule after non-`2xx` or timeout: `1m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `8h`, then stop.
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
    "status": "completed_with_errors",
    "summary": {
      "total_items": 12500,
      "accepted_items": 12430,
      "rejected_items": 70
    },
    "href": "/v1/catalog-import-jobs/job_01JNV1X5CBR4W1FVC0SP0V84ZQ"
  }
}
```

Webhook delivery state on the job resource:

- `pending`: webhook requested but not yet delivered successfully.
- `delivered`: last attempt succeeded with `2xx`.
- `exhausted`: retry schedule ended without a successful delivery.

## Compatibility, Artifact Updates, And Handoffs

- Compatibility class: new additive `v1` surface.
- Safe additive changes inside `v1`: new response object fields, new failure `code` values, and new signature versions added alongside `v1`.
- Changes that require explicit versioning or a reviewed behavior-change path: new job `status` values, different partial-failure semantics, different freshness semantics, different rate-limit semantics, or broader accepted media types.
- This document is the client-visible contract artifact. Secret provisioning, malware engine policy, storage retention, and worker internals stay outside the contract unless they change client-visible behavior.
- Security follow-on is still required for webhook secret lifecycle and callback allowlist policy, but that work should refine this contract rather than replace it.

## Open Questions, Risks, And Reopen Conditions

- Does product mean `50 MB` decimal (`50,000,000` bytes) or binary (`52,428,800` bytes)? This contract currently assumes decimal.
- Should `completion_webhook.url` be allowed per job, or should webhook destinations be pre-registered resources with IDs?
- What freshness SLA should be documented for the cache-backed job view, if any?
- How long should upload sessions, terminal jobs, failure pages, and webhook delivery metadata be retained?
- Is a single `PUT` upload sufficient for poor network conditions, or is a resumable upload protocol required later?
- If the service cannot uphold the `failed means no committed catalog changes` invariant, this contract must be reopened before coding.

# Bulk Catalog Import API Contract

## Contract Framing

- Audience: authenticated partner systems and the partner-facing UI.
- Problem: accept one CSV up to `50,000,000` bytes, require malware scanning before any catalog effects, process asynchronously, expose a stable job resource, support optional signed completion webhooks, and disclose that job reads come from a cache-backed view that may lag.
- Selected option: separate upload-session resource plus domain-specific import-job resource.
  Rejected option: direct `multipart/form-data POST /v1/catalog-import-jobs` because a `50 MB` upload plus mandatory malware scanning would blur transport success, file acceptance, and job acceptance into one misleading call.
- Selected option: domain-specific `/v1/catalog-import-jobs/{job_id}` instead of a generic `/v1/operations/{id}` resource.
  Rejected option: generic operations because clients need import-specific summary counts, row-failure paging, and webhook delivery state.
- Selected option: polling is always available, and callers may additionally request one signed terminal webhook per job.
  Rejected option: webhook-only completion because not every partner can host a receiver.
- Selected option: partial row failures are represented by terminal summary counts plus a paged failures subresource.
  Rejected option: inline full failure arrays and `207 Multi-Status` because failure volume is unbounded and the import is asynchronous.

## Resource And Endpoint Matrix

| Endpoint | Method | Success | Purpose | Consistency | Retry |
| --- | --- | --- | --- | --- | --- |
| `/v1/catalog-import-upload-sessions` | `POST` | `201 Created` | Reserve one upload slot and return an opaque upload target for the CSV bytes | strong | retry-safe by contract with `Idempotency-Key` |
| `/v1/catalog-import-upload-sessions/{session_id}` | `GET` | `200 OK` | Read upload-session state before or after upload | strong | retry-safe by protocol |
| `upload.url` from the session response | `PUT` | `2xx` from upload target | Upload raw CSV bytes to the opaque target | strong | retry-safe until the upload target expires |
| `/v1/catalog-import-jobs` | `POST` | `202 Accepted` | Start mandatory malware scanning and asynchronous import for an uploaded session | strong on acceptance | retry-safe by contract with `Idempotency-Key` |
| `/v1/catalog-import-jobs/{job_id}` | `GET` | `200 OK` | Read import job status, terminal summary, and webhook delivery state | eventual | retry-safe by protocol |
| `/v1/catalog-import-jobs/{job_id}/failures` | `GET` | `200 OK` | Page through rejected item rows for a terminal job | eventual | retry-safe by protocol |

`v1` does not expose job cancellation or webhook replay endpoints.

### End-To-End Flow

1. Client creates an upload session with file metadata.
2. Client uploads the raw CSV bytes to the opaque `upload.url`.
3. Client creates a catalog import job from the uploaded session and may request a completion webhook.
4. Client polls the job resource or waits for the signed terminal webhook.
5. If the job finishes with partial row failures, client reads `/failures` for row-level details.

### Upload Session State Model

| State | Meaning |
| --- | --- |
| `open` | Session exists and can accept exactly one file upload. |
| `uploaded` | File bytes were stored and are eligible for job creation. |
| `consumed` | A job already references this session. |
| `expired` | The session can no longer be used for upload or job creation. |

### Import Job State Model

| State | Meaning |
| --- | --- |
| `queued` | The job was accepted but scanning has not started. |
| `scanning` | Malware scanning and file-level safety checks are running. No catalog effects have started. |
| `processing` | The file passed scanning and row evaluation or import is running. |
| `succeeded` | All rows were accepted and applied. |
| `completed_with_errors` | All rows were evaluated. Accepted rows were applied. Rejected rows were not applied. |
| `failed` | The job ended without applying any catalog changes. Examples: malware detected, unreadable CSV, or unrecoverable service failure before commit. |

`succeeded`, `completed_with_errors`, and `failed` are terminal states.

## Request, Response, And Error Model

### `POST /v1/catalog-import-upload-sessions`

Required headers:

- `Idempotency-Key`
- `Content-Type: application/json`

Request body:

```json
{
  "file_name": "catalog-2026-03-07.csv",
  "media_type": "text/csv",
  "file_size_bytes": 49753211
}
```

Rules:

- `file_name`, `media_type`, and `file_size_bytes` are required.
- `media_type` must be `text/csv`.
- `file_size_bytes` must be at least `1` and at most `50,000,000`.
- Compressed uploads such as `application/zip` or `application/gzip` are not accepted in `v1`.
- Unknown fields, malformed JSON, and trailing tokens fail with `400`.

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
  "expires_at": "2026-03-07T13:00:00Z",
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

- `upload.url` is opaque and outside the versioned REST surface.
- A successful `PUT` to `upload.url` only means the bytes were stored. It does not mean the file passed malware scanning or that an import job exists.
- The upload target must reject uploads larger than `50,000,000` bytes with `413`.
- The upload target must reject media types other than `text/csv` with `415`.
- A session may back exactly one import job.

### `GET /v1/catalog-import-upload-sessions/{session_id}`

Response `200 OK` returns the upload-session resource, with these additional fields when available:

- `uploaded_at` when `status=uploaded`
- `consumed_at` when `status=consumed`
- `job_id` when `status=consumed`

### `POST /v1/catalog-import-jobs`

Required headers:

- `Idempotency-Key`
- `Content-Type: application/json`

Request body:

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
- The referenced upload session must belong to the authenticated partner and be in `uploaded` state.
- `open` or `consumed` upload sessions fail with `409`.
- `expired` upload sessions fail with `410`.
- If the caller requests a completion webhook and the partner account has no active webhook signing secret, the request fails with `422`.

Response `202 Accepted`:

- `Location: /v1/catalog-import-jobs/job_01JNV1X5CBR4W1FVC0SP0V84ZQ`
- `Retry-After: 5`

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

Acceptance rules:

- `202 Accepted` means the service accepted responsibility to scan and process the uploaded file.
- `202 Accepted` does not mean the file passed malware scanning.
- Malware detection and CSV content failures after job creation are surfaced on the job resource, not by rewriting the original `202` response.

### `GET /v1/catalog-import-jobs/{job_id}`

Response headers:

- `Content-Type: application/json`
- `Cache-Control: no-store`
- `Retry-After: 5` while the job is non-terminal

Representative partial-success response:

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
- `succeeded` returns `summary.rejected_items = 0` and no `failures.href`.
- `completed_with_errors` means every data row was evaluated. Accepted rows were committed. Rejected rows were skipped.
- `failed` returns a `failure` object with `code`, `message`, and `stage`.
- `failed` means no catalog changes from that job were committed.
- `GET /v1/catalog-import-jobs/{job_id}` never returns row-level failures inline.

Representative fatal failure object:

```json
{
  "failure": {
    "stage": "malware_scan",
    "code": "malware_detected",
    "message": "The uploaded file failed mandatory malware scanning."
  }
}
```

### `GET /v1/catalog-import-jobs/{job_id}/failures`

Query parameters:

- `cursor`: opaque pagination cursor, optional.
- `page_size`: optional, default `50`, maximum `200`.

Rules:

- Rows are returned in ascending `row_number`.
- `row_number` is a 1-based data-row index after the CSV header row.
- Unknown query parameters fail with `400`.
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
    "view_as_of": "2026-03-07T12:08:35Z",
    "source_last_updated_at": "2026-03-07T12:08:31Z"
  }
}
```

Failure row rules:

- `code` is an open string enum. Clients must not assume the set is closed.
- `item_key` is optional because some rows may fail before a stable business key can be parsed.
- Header-level CSV defects are surfaced as job-level `failed` errors, not as row-level failures.

### Error Model

All `/v1/...` REST endpoints use `application/problem+json`.

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
| `404` | unknown upload session or job in caller scope | `not_found` |
| `409` | upload session not ready, already consumed, or idempotency payload mismatch | `upload_session_consumed` |
| `410` | upload session expired | `upload_session_expired` |
| `413` | actual uploaded object exceeds `50,000,000` bytes | `payload_too_large` |
| `415` | declared or uploaded media type is not `text/csv` | `unsupported_media_type` |
| `422` | semantically invalid request, including invalid webhook URL or missing configured webhook secret | `validation_error` |
| `428` | missing required `Idempotency-Key` | `idempotency_key_required` |
| `429` | request-rate limit or active-import quota exceeded | `rate_limited` |
| `503` | transient dependency outage during session or job creation | `service_unavailable` |

Negative-path rules:

- Malware detection is never surfaced as an HTTP error on `POST /v1/catalog-import-jobs`; it is surfaced as `job.status=failed`.
- Transient processing faults after job acceptance are surfaced on the job resource, not by changing the original `202 Accepted`.
- Successful responses never embed an error payload.

## Boundary And Cross-Cutting Policies

- Authentication and tenant scoping come from validated partner credentials, not caller-supplied partner IDs in the path or body.
- `POST /v1/catalog-import-upload-sessions` and `POST /v1/catalog-import-jobs` require `Idempotency-Key`.
- Idempotency keys are scoped to partner plus route and retained for `24h`.
- Same idempotency key with the same request payload returns the same semantic outcome, including the same resource ID.
- Same idempotency key with a different payload fails with `409`.
- `GET` endpoints are retry-safe by protocol.
- Job resources are read-only after creation, so `If-Match` and write-side optimistic concurrency are not part of `v1`.
- `429` responses must include `Retry-After`.
- The upload target may also return transient `429` or `503`; clients may retry upload while the upload session is still `open`.
- Request validation pipeline is explicit: transport limit, strict decode, field validation, business-state validation, then async acceptance.
- CSV parsing, malware scanning, and per-row validation happen only after `POST /v1/catalog-import-jobs` accepts the job.

### Signed Completion Webhook Contract

- Only one logical terminal event type exists in `v1`: `catalog-import.job.completed`.
- The event is emitted for `succeeded`, `completed_with_errors`, and `failed`.
- Delivery is at least once. Duplicates are possible. Clients must deduplicate by webhook event `id`.
- Delivery order is not guaranteed relative to polling responses.
- Receiver timeout expectation: return a `2xx` within `5` seconds.
- Retry schedule after timeout or non-`2xx`: `1m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `8h`, then stop.
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

Webhook delivery states on the job resource:

- `pending`: webhook requested but no successful delivery yet.
- `delivered`: the last delivery attempt succeeded with `2xx`.
- `exhausted`: the retry schedule ended without a successful delivery.

Webhook delivery outcome does not change the import job outcome.

## Consistency And Async Notes

- `POST /v1/catalog-import-jobs` is strongly consistent only for job acceptance. It does not provide read-after-write visibility on the cache-backed `GET` view.
- Malware scanning is mandatory and always precedes row processing.
- `GET /v1/catalog-import-jobs/{job_id}` and `/failures` are explicitly eventual-consistency reads backed by a cache-backed view.
- `freshness.view_as_of` is the timestamp of the cached representation returned to the client.
- `freshness.source_last_updated_at` is the latest source-of-truth update reflected in that cached representation.
- Clients must not assume that polling immediately after `202 Accepted` returns the latest source-of-truth state.
- A signed terminal webhook may arrive before the cache-backed `GET` view shows the same terminal state.
- Clients that need the earliest terminal signal should prefer the signed webhook when they request one.

## Compatibility Notes

- Compatibility class: new additive `v1` surface.
- Safe additive changes inside `v1`: new response fields, new failure `code` values, new webhook signature versions added alongside `v1`, and new optional job metadata fields.
- Behavior changes that require explicit review before shipping in `v1`: new job status values, different `failed` versus `completed_with_errors` semantics, different freshness semantics, different retry or rate-limit semantics, or different webhook retry guarantees.
- Breaking changes that require a new major version: changing endpoint shapes incompatibly, removing fields, changing required headers, changing accepted media type rules incompatibly, or changing row-failure pagination semantics incompatibly.
- This contract intentionally keeps malware engine choice, storage retention internals, queue topology, and worker implementation details out of scope unless they change client-visible behavior.

## Open Questions And Risks

- Does product mean `50 MB` decimal (`50,000,000` bytes) or binary (`52,428,800` bytes)?
- Should completion webhooks use per-job URLs as specified here, or should partners register webhook endpoints once and refer to them by ID?
- What freshness SLA, if any, should the cache-backed job view promise to clients?
- How long should upload sessions, terminal jobs, failure pages, and exhausted webhook metadata be retained?
- Is one-shot `PUT` upload sufficient for expected partner network conditions, or will a resumable upload protocol be required later?
- Can the service guarantee the `failed means no catalog changes were committed` invariant for every fatal path? If not, this contract must be reopened before coding.

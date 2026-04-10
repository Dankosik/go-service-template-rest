# Async Operations And Webhooks

## When To Read This
Read this when work may outlive the request, completion time is variable, bulk processing is involved, clients need polling or callbacks, a webhook contract is requested, or `202 Accepted` is being considered. Keep this contract-first; hand off worker runtime, queue topology, saga orchestration, and webhook delivery implementation.

## Compact Principles
- Use async semantics when completion is often slow, variable, fan-out heavy, or uncertain at request time.
- `202 Accepted` means the service accepted responsibility for processing; it does not mean business success.
- Provide a durable client recovery path: operation resource, authoritative business resource with lifecycle state, webhook plus reconciliation read, or a documented combination.
- Prefer one control-plane resource per async lifecycle unless a second resource removes a concrete client ambiguity.
- Operation resources should expose stable fields such as `id`, `status`, `created_at`, `updated_at`, result reference, failure problem, and retention or expiry.
- Keep operation statuses reachable. Start with `pending`, `running`, `succeeded`, and `failed`; add `canceling`, `canceled`, or `expired` only if clients can observe those paths.
- Include `Location` for the operation or created resource and use `Retry-After` when polling cadence matters.
- Define what clients should do after a request timeout: replay with an idempotency key, poll by operation URI, or find by a client-supplied correlation field.
- For bulk work, define all-or-nothing vs per-item results and how large failure sets are paged or downloaded.
- Webhooks are push notifications, not the sole source of truth. Provide a reconciliation read or monotonic version so clients can recover missed, duplicate, or out-of-order events.
- Treat webhooks as at-least-once and possibly out of order. Define event IDs, dedup keys, timestamps, retry window, signature verification, and sender timeout.
- Prefer pre-registered or ownership-verified webhook destinations over arbitrary per-request callback URLs unless the product explicitly needs per-request callback URLs.
- In OpenAPI, use callbacks for request-tied outbound calls and the `webhooks` top-level object for provider-originated event shapes.

## Good Examples
```http
POST /v1/exports HTTP/1.1
Content-Type: application/json
Idempotency-Key: "exp_01J8V7Q9Y0T8F"

{
  "format": "csv",
  "filter": { "status": "paid" }
}

HTTP/1.1 202 Accepted
Location: /v1/operations/op_123
Retry-After: 30
Content-Type: application/json

{
  "id": "op_123",
  "status": "pending",
  "created_at": "2026-03-10T18:31:00Z"
}
```

```http
GET /v1/operations/op_123 HTTP/1.1

HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "op_123",
  "status": "succeeded",
  "result": {
    "type": "export",
    "href": "/v1/exports/file_456"
  },
  "updated_at": "2026-03-10T18:34:11Z",
  "expires_at": "2026-03-11T18:34:11Z"
}
```

```http
POST /client/webhooks/order-events HTTP/1.1
Content-Type: application/json
webhook-id: evt_789
webhook-timestamp: 1773177251
webhook-signature: v1,base64-signature

{
  "id": "evt_789",
  "type": "order.updated",
  "resource": "/v1/orders/ord_123",
  "resource_version": 17,
  "occurred_at": "2026-03-10T18:34:11Z"
}
```

Good contract note: the webhook event carries an event ID for deduplication and a resource version for reconciliation against reads.

## Bad Examples
```http
POST /v1/exports HTTP/1.1

HTTP/1.1 202 Accepted
```

Why it is bad: the client has no operation URI, no recovery rule, no polling cadence, and no way to learn final success or failure.

```json
{
  "status": "completed"
}
```

Why it can be bad: "completed" can mean succeeded, failed, canceled, or partially completed. Use terminal states that preserve outcome.

## Edge Cases That Often Fool Agents
- A queue enqueue attempt is not automatically durable acceptance. The contract should identify the point after which the service owns completion or failure reporting.
- Returning both an operation resource and a business resource can be useful, but define which one is authoritative for lifecycle and recovery.
- Cancellation does not imply rollback. Define whether cancellation is best effort, compensating, impossible after a point, or a terminal business state.
- Operation retention needs a fallback. If the operation expires, tell clients whether to read the business resource, use an export URL, or receive `410 Gone`.
- Webhook delivery must tolerate duplicate, late, and out-of-order events. A single "we send once" promise is brittle.
- Per-request callback URLs create trust-boundary and SSRF questions. If the API allows them, make ownership verification and target restrictions part of the public contract and hand off security details.
- OpenAPI can describe webhook shapes, but timing, retry schedule, and event ordering often need prose outside the schema.
- Do not expose internal queue names, worker IDs, or shard positions in operation IDs or event payloads.

## Source Links Gathered Through Exa
- RFC 9110, `202 Accepted`, `Location`, and `Retry-After` semantics: https://www.rfc-editor.org/rfc/rfc9110.html
- RFC 9421, HTTP Message Signatures: https://www.rfc-editor.org/rfc/rfc9421
- OpenAPI Specification v3.1.1: https://spec.openapis.org/oas/v3.1.1.html
- OpenAPI Learn, Providing Webhooks: https://learn.openapis.org/specification/webhooks.html
- OpenAPI Learn, Callback Example: https://learn.openapis.org/examples/v3.0/callback-example.html
- Microsoft Graph Long Running Operations pattern: https://github.com/microsoft/api-guidelines/blob/vNext/graph/patterns/long-running-operations.md
- Microsoft REST API Guidelines, long-running operations and webhooks: https://github.com/microsoft/api-guidelines/blob/master/Guidelines.md
- Standard Webhooks specification, community webhook signing and delivery conventions: https://github.com/standard-webhooks/standard-webhooks/blob/main/spec/standard-webhooks.md

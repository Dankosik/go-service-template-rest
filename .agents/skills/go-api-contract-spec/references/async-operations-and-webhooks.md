# Async Operations And Webhooks

## When To Load
Load this when work may outlive the request, completion time is variable, bulk processing is involved, clients need polling or callbacks, a webhook contract is requested, or `202 Accepted` is being considered.

## Decision Rubric
- Use async semantics when completion exceeds the caller's request-timeout or UX budget, is variable, is fan-out heavy, or is uncertain at request time. If no repo policy exists, treat `10s` as a rough trigger rather than a hard rule.
- RFC 9110 makes `202 Accepted` intentionally noncommittal about final outcome; this skill uses it only when the API also accepts durable recovery or reporting responsibility. It is not business success and not merely "enqueue attempted."
- Provide one durable client recovery path: operation resource, authoritative business resource with lifecycle state, webhook plus reconciliation read, or a documented combination.
- Prefer one control-plane resource per lifecycle unless a second resource removes a concrete client ambiguity.
- Operation resources should expose `id`, reachable `status`, `created_at`, `updated_at`, result reference, structured failure problem, and retention or expiry.
- Start statuses from what clients can observe: `pending`, `running`, `succeeded`, `failed`. Add `canceling`, `canceled`, or `expired` only if those paths are real.
- Include `Location` for the operation or created resource and `Retry-After` when polling cadence matters.
- Define timeout recovery: replay with idempotency key, poll by operation URI, read an authoritative resource, or find by a client-supplied correlation field.
- Bulk contracts must choose all-or-nothing or per-item results, including failure-set paging or download rules when large.
- Webhooks are notifications, not the source of truth. Include a reconciliation read or monotonic version for missed, duplicate, or out-of-order events.
- Prefer pre-registered or ownership-verified webhook targets over arbitrary per-request callback URLs unless the product requires callbacks.

## Imitate
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

{ "id": "op_123", "status": "pending", "created_at": "2026-03-10T18:31:00Z" }
```

Good: clients can recover after timeout, poll at a documented cadence, and distinguish accepted work from completed business success.

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

Good: event ID supports deduplication and `resource_version` supports reconciliation against reads.

## Reject
```http
POST /v1/exports HTTP/1.1

HTTP/1.1 202 Accepted
```

Bad: no operation URI, recovery rule, polling cadence, or final outcome path.

```json
{ "status": "completed" }
```

Bad: "completed" blurs succeeded, failed, canceled, and partially completed.

## Agent Traps
- A queue enqueue attempt is not automatically durable acceptance. Identify the boundary after which the service owns completion or failure reporting.
- Returning both an operation resource and a business resource can be useful; define which one is authoritative for lifecycle and recovery.
- Cancellation does not imply rollback. Say whether it is best effort, compensating, impossible after a point, or a terminal business state.
- Operation retention needs a fallback. After expiry, clients may need `410 Gone`, a business-resource read, or an export URL.
- Per-request callback URLs raise trust-boundary and SSRF questions. Keep that in the contract and hand off security details.
- In OpenAPI, use callbacks for request-tied outbound calls and top-level `webhooks` for provider-originated event shapes.

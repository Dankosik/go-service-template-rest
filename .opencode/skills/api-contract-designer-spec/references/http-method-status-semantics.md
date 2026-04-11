# HTTP Method And Status Semantics

## When To Load
Load this when choosing methods, mutation semantics, status codes, `201` vs `202` vs `204`, `Location`, `ETag`, `PUT` vs `PATCH`, content negotiation errors, or conditional request behavior.

## Decision Rubric
- Model the client-visible resource first; choose the method after the representation semantics are clear.
- `POST` creates a subordinate resource when the server assigns identity, or starts an operation resource when work outlives the request.
- `PUT` is full replacement of a known resource. Upsert is exception-only; if it exists, say so explicitly.
- `PATCH` is partial update and is incomplete until it defines patch media type, omitted fields, explicit `null`, empty objects, arrays, and immutable-field writes.
- Use `201 Created` plus `Location` when a new resource exists.
- Use `202 Accepted` only when the service accepted durable responsibility and the client has a completion or recovery path.
- Use `204 No Content` only when no body helps the client. A `204` response cannot carry a representation or problem body.
- Keep `409`, `412`, and `428` distinct. If the client supplied a stale `If-Match`, prefer `412`; if it omitted a required condition, prefer `428`.
- Use `415` for unsupported request content type and `406` when the server cannot produce an acceptable response representation.

## Imitate
```http
POST /v1/orders HTTP/1.1
Content-Type: application/json

HTTP/1.1 201 Created
Location: /v1/orders/ord_123
ETag: "v1"
Content-Type: application/json

{ "id": "ord_123", "status": "pending" }
```

Good: identity exists, `Location` points to the created resource, and the validator can feed later concurrency control.

```http
PATCH /v1/orders/ord_123 HTTP/1.1
Content-Type: application/merge-patch+json
If-Match: "v1"

{ "shipping_address": null }
```

Good only if `null` explicitly means "clear this field." Otherwise the contract should fail it or use a different patch format.

## Reject
```http
GET /v1/orders/ord_123/cancel HTTP/1.1

HTTP/1.1 200 OK
```

Bad: `GET` is being used for a requested mutation.

```http
PUT /v1/orders/ord_123 HTTP/1.1
Content-Type: application/json

{ "status": "paid" }
```

Bad unless documented as a full replacement shape or explicit upsert/partial policy. A partial `PUT` can silently clear fields.

## Agent Traps
- `202` is not a friendlier `200`; it needs an operation resource, authoritative business resource, webhook, or documented reconciliation read.
- A `Location` header on `201` usually points to the created resource; on `202`, it often points to an operation resource. Say which.
- `DELETE` idempotency does not require identical later responses. First delete can return `204`; later reads or deletes can be `404` or `410` if documented.
- Do not invent custom status codes. Use Problem Details extensions for API-specific detail.
- If multiple mutation surfaces coexist, define whether they share `ETag` space, stale-write behavior, and success statuses.

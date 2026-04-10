# HTTP Method And Status Semantics

## When To Read This
Read this when the task needs method choice, status-code selection, mutation semantics, `Location` or `ETag` behavior, `PUT` vs `PATCH`, `201` vs `202` vs `204`, content negotiation errors, or conditional request semantics. Stay contract-first; do not specify chi router topology or handler implementation.

## Compact Principles
- Model the client-visible resource first, then choose the method that matches the intended representation semantics.
- `GET`, `HEAD`, and `OPTIONS` are safe. Safe methods must not have requested side effects.
- `GET`, `HEAD`, `OPTIONS`, `PUT`, and `DELETE` are idempotent by HTTP method semantics. `POST` and `PATCH` are not idempotent unless the API adds a separate contract mechanism.
- Use `POST` to create subordinate resources when the server assigns identity, or to start an operation resource when work outlives the request.
- Use `PUT` only for full replacement of a known resource. If the API supports upsert, say so explicitly; otherwise a missing target should be an error.
- Use `PATCH` for partial updates and specify the patch media type. For JSON APIs, `application/merge-patch+json` is a common default; define omitted, `null`, empty object, and array behavior.
- Use `201 Created` when a new resource exists and send `Location` for the created resource.
- Use `202 Accepted` only when responsibility for later processing has been accepted but completion is not represented by the initial response.
- Use `204 No Content` when no response body helps the client. Do not send a problem or representation body with `204`.
- Use `303 See Other` when the result of a `POST` should be retrieved from another resource with `GET`.
- Use `409 Conflict` for state conflicts, `412 Precondition Failed` for supplied preconditions that evaluated false, and `428 Precondition Required` when the server requires a conditional request.
- Use `415 Unsupported Media Type` for unsupported request content type and `406 Not Acceptable` when response negotiation cannot produce an acceptable representation.
- If concurrency matters, define `ETag` emission on reads and successful writes, plus `If-Match` or `If-None-Match` rules on mutations.

## Good Examples
```http
POST /v1/orders HTTP/1.1
Content-Type: application/json

HTTP/1.1 201 Created
Location: /v1/orders/ord_123
ETag: "v1"
Content-Type: application/json

{
  "id": "ord_123",
  "status": "pending"
}
```

```http
PATCH /v1/orders/ord_123 HTTP/1.1
Content-Type: application/merge-patch+json
If-Match: "v1"

{ "shipping_address": null }

HTTP/1.1 200 OK
ETag: "v2"
Content-Type: application/json

{
  "id": "ord_123",
  "shipping_address": null,
  "status": "pending"
}
```

Contract note: the example is good only if `null` explicitly means "clear the field"; otherwise it must fail or use a different patch format.

## Bad Examples
```http
GET /v1/orders/ord_123/cancel HTTP/1.1

HTTP/1.1 200 OK
```

Why it is bad: `GET` is being used for a requested mutation.

```http
PUT /v1/orders/ord_123 HTTP/1.1
Content-Type: application/json

{ "status": "paid" }
```

Why it is bad unless explicitly documented: a partial body with `PUT` usually conflicts with full-replacement semantics and may accidentally clear fields.

## Edge Cases That Often Fool Agents
- `DELETE` being idempotent does not require identical response bodies on every call. The server may return `204` for the first delete and `404` or `410` later if the contract says so.
- `202` is not a nicer `200`; it means the client still needs a completion path, such as an operation resource, authoritative business resource, webhook, or documented reconciliation read.
- `204` cannot carry a response body. If the client needs the new representation, use `200` or `201`.
- `PATCH` without a patch format is incomplete. JSON Merge Patch treats `null` as removal unless the API documents a different semantic layer.
- `409` is not the right answer for every write failure. Use `412` when the client supplied a precondition that failed.
- `428` is useful when stale writes would be dangerous and the client omitted a required precondition.
- A `Location` header on `201` points at the created resource. A `Location` header on `202` often points at an operation resource; be explicit which it is.
- Do not use custom status codes. Select from registered status codes and use Problem Details extensions for API-specific detail.

## Source Links Gathered Through Exa
- RFC 9110, HTTP Semantics: https://www.rfc-editor.org/rfc/rfc9110.html
- RFC 6585, Additional HTTP Status Codes: https://www.rfc-editor.org/rfc/rfc6585.html
- OpenAPI Specification v3.1.1, status-code and HTTP API description rules: https://spec.openapis.org/oas/v3.1.1.html
- Microsoft REST API Guidelines, method and header conventions: https://github.com/microsoft/api-guidelines/blob/master/Guidelines.md

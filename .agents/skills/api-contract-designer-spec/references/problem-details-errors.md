# Problem Details And Error Contracts

## When To Read This
Read this when the API contract needs error payloads, validation failures, auth or concealment semantics, rate-limit or overload errors, field-level errors, or a consistent `application/problem+json` profile. Stay at the client-visible contract layer; do not turn this into handler, middleware, database, or logging design.

## Compact Principles
- Use `application/problem+json` as the default error representation when the API needs machine-readable error details beyond the status code.
- Treat the HTTP status code as authoritative. If a `status` member is present in the body, it must match the wire status sent by the origin server.
- Make `type` the stable machine identifier for the problem. Prefer an absolute URI under the API owner's control; avoid relative problem type URIs unless the base URI behavior is deliberate.
- Keep `title` stable for the problem type. Put occurrence-specific help in `detail`, and put machine-readable data in extensions.
- Do not ask clients to parse `detail`; define stable extension fields for data such as `code`, `request_id`, `errors`, `retry_after`, or conflict metadata.
- Use one consistent field-error shape. Include a pointer or field path, a stable code, and caller-facing detail; define ordering only if clients can rely on it.
- Choose the `400` vs `422` boundary once. A common contract split is malformed JSON, invalid media type, or unknown top-level structure as `400`; syntactically valid JSON with semantic field errors as `422`.
- Keep `409 Conflict`, `412 Precondition Failed`, and `428 Precondition Required` distinct when optimistic concurrency or state transitions are in play.
- Use `429 Too Many Requests` for client-visible rate limits and pair it with retry guidance when the client may retry later.
- Never return a success status with an embedded problem. A 2xx response means the request's HTTP-level outcome succeeded.
- Sanitize every error payload. No stack traces, SQL fragments, infrastructure names, secrets, or cross-tenant identifiers.

## Good Examples
```http
HTTP/1.1 422 Unprocessable Content
Content-Type: application/problem+json

{
  "type": "https://api.example.com/problems/validation",
  "title": "Request validation failed",
  "status": 422,
  "detail": "One or more fields need a different value.",
  "request_id": "req_01J8V7Q9Y0T8F",
  "errors": [
    {
      "code": "date_range.invalid",
      "pointer": "/data/attributes/end_date",
      "detail": "end_date must be on or after start_date."
    }
  ]
}
```

```http
HTTP/1.1 409 Conflict
Content-Type: application/problem+json

{
  "type": "https://api.example.com/problems/order-state-conflict",
  "title": "Order state does not allow this transition",
  "status": 409,
  "detail": "Only pending orders can be cancelled.",
  "code": "order.not_cancelable",
  "current_state": "shipped"
}
```

## Bad Examples
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "ok": false,
  "error": "cannot cancel"
}
```

Why it is bad: clients, proxies, SDKs, and retries will treat this as success unless every caller knows the private convention.

```http
HTTP/1.1 500 Internal Server Error
Content-Type: application/problem+json

{
  "type": "about:blank",
  "title": "Null pointer in order_repository.go line 231",
  "status": 500,
  "detail": "SQL timeout on host db-prod-7.internal"
}
```

Why it is bad: it leaks implementation details and gives no stable client action.

## Edge Cases That Often Fool Agents
- `about:blank` is a valid default problem type, but it adds no API-specific semantics. Do not use it for domain-specific validation or conflict cases that clients need to handle differently.
- Multiple field errors can fit one validation problem type, but unrelated problem types should not be bundled into a generic "many problems" payload without a clear client reason.
- Localization can change `title` or `detail`; stable machine decisions must come from `type`, `code`, or typed extensions.
- A cross-tenant missing-resource policy can intentionally return `404` for inaccessible resources, but the policy must be consistent and documented.
- `401` and `403` are not interchangeable. When the contract uses authentication challenges, include the appropriate auth header requirements.
- `415 Unsupported Media Type`, `406 Not Acceptable`, `405 Method Not Allowed`, and `413 Content Too Large` are client-visible HTTP contract decisions, not generic validation failures.
- Field-level pointers should name the request representation, not database columns or Go struct fields.
- If an intermediary might transform status codes, clients may keep the body's `status` for diagnostics, but the contract must still send the intended wire status.

## Source Links Gathered Through Exa
- RFC 9457, Problem Details for HTTP APIs: https://www.rfc-editor.org/rfc/rfc9457.html
- RFC 9110, HTTP Semantics: https://www.rfc-editor.org/rfc/rfc9110.html
- RFC 6585, Additional HTTP Status Codes: https://www.rfc-editor.org/rfc/rfc6585.html
- JSON Schema 2020-12 Core: https://json-schema.org/draft/2020-12/json-schema-core

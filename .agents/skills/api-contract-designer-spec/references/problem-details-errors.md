# Problem Details And Error Contracts

## When To Load
Load this when the API contract needs Problem Details, validation errors, auth or concealment status policy, field-level errors, rate-limit or overload errors, or sanitized negative paths.

## Decision Rubric
- Use `application/problem+json` when clients need a machine-readable error contract, not just a status code.
- If the API profile requires common members such as `type`, `title`, `status`, or `detail`, say so as a profile choice; RFC 9457 does not require every common member.
- Treat the wire status as authoritative; if the body has `status`, it must match.
- Make `type` and extension fields the stable machine contract. `title` and `detail` are for humans and diagnostics.
- Pick one `400` vs `422` split. A sharp default: malformed transport or JSON shape is `400`; syntactically valid JSON with semantic field errors is `422`.
- Keep `409`, `412`, and `428` separate: state conflict, failed supplied precondition, and missing required precondition.
- Choose one concealment policy for inaccessible resources. Cross-tenant lookups should not drift between `403` and `404` by endpoint.
- Never hide failure in a `2xx` body, and never leak stack traces, SQL, secrets, hostnames, shard IDs, or cross-tenant identifiers.

## Imitate
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

Good: clients can branch on `type` or `errors[].code`; the field pointer names the request representation, not a Go struct or DB column.

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

Good: this is a state conflict, not a validation typo or stale `If-Match`.

## Reject
```http
HTTP/1.1 200 OK
Content-Type: application/json

{ "ok": false, "error": "cannot cancel" }
```

Bad: clients, proxies, SDKs, and retries will treat the request as successful.

```json
{
  "type": "about:blank",
  "title": "Null pointer in order_repository.go line 231",
  "detail": "SQL timeout on host db-prod-7.internal"
}
```

Bad: it leaks implementation detail and gives clients no stable action.

## Agent Traps
- `about:blank` is valid but weak. Do not use it for domain problems clients must distinguish.
- Localization can change human text; machine decisions need stable `type`, `code`, or typed extensions.
- `401` and `403` are not interchangeable; authentication challenges can require headers.
- `405`, `406`, `413`, and `415` are API contract choices, not generic validation failures.
- Multiple field errors can share one validation problem; unrelated problem types need a client-facing reason to be bundled.

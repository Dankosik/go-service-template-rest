# Idempotency, Preconditions, And Retries

## When To Read This
Read this when a write can be retried, a client may see a timeout after a mutation, duplicate work is dangerous, optimistic concurrency matters, `ETag`/`If-Match` is in scope, or the contract needs retry classification. Keep the output at API semantics; do not design storage tables, locks, queues, or retry workers.

## Compact Principles
- Separate HTTP method idempotency from API-contract idempotency. `PUT` and `DELETE` are idempotent by method semantics; `POST` and `PATCH` need extra contract rules if clients may retry safely.
- For retryable non-idempotent writes, require an idempotency key or expose an operation resource with an unambiguous recovery read.
- Treat the IETF Idempotency-Key document as an active Internet-Draft until it is published as an RFC. Use it as strong design input, not as a final standard.
- Define key syntax, entropy expectations, tenant/account scope, operation scope, route or method scope, and retention/TTL.
- Default TTL in this skill remains `24h` unless the product contract needs a different duplicate-risk window.
- Compare retried payloads at the normalized contract level when insignificant JSON formatting or field ordering could differ.
- Same key plus same normalized payload should return an equivalent prior outcome once the first attempt reaches a durable outcome.
- Same key plus different normalized payload should fail with a stable conflict or validation problem.
- Same key while the first request is still in progress should have a documented response, often `409 Conflict` with retry or polling guidance.
- Define which failures reserve the key. Pre-decode errors usually should not reserve a key; accepted async work usually should.
- If lost updates matter, expose `ETag` on reads and require `If-Match` on high-contention updates.
- Use `412 Precondition Failed` when a supplied precondition is false. Use `428 Precondition Required` when the client omitted a required precondition.
- Define retry classification per endpoint: retry-safe by protocol, retry-safe by contract, retry only after read/poll, or retry-unsafe.

## Good Examples
```http
POST /v1/payments HTTP/1.1
Content-Type: application/json
Idempotency-Key: "pay_01J8V7Q9Y0T8F"

{
  "amount": "42.00",
  "currency": "USD",
  "order_id": "ord_123"
}

HTTP/1.1 201 Created
Location: /v1/payments/pay_456
```

Replay rule: if the same tenant sends the same normalized payload with the same key inside the TTL after completion, return an equivalent result and the same payment identity.

```http
PATCH /v1/orders/ord_123 HTTP/1.1
Content-Type: application/merge-patch+json
If-Match: "v4"

{ "shipping_address_id": "addr_789" }

HTTP/1.1 412 Precondition Failed
Content-Type: application/problem+json
```

Contract note: `412` means the supplied validator did not match the current representation; the client should read, reconcile, and retry with a fresh validator if appropriate.

## Bad Examples
```http
POST /v1/payments HTTP/1.1

HTTP/1.1 504 Gateway Timeout
```

Bad contract if no recovery rule exists: after a timeout, the client cannot know whether the payment exists, whether retry creates a duplicate, or what to read next.

```http
PATCH /v1/orders/ord_123 HTTP/1.1
If-Match: "stale"

HTTP/1.1 409 Conflict
```

Why it is usually bad: when an HTTP precondition is supplied and evaluates false, `412` is the more precise contract signal.

## Edge Cases That Often Fool Agents
- Idempotency keys must be scoped by tenant or account. Global key lookup can leak stored outcomes across callers.
- Low-entropy keys are attackable; the contract should require enough randomness and reject invalid formats.
- Post-TTL reuse is a real contract choice. Define whether the key is treated as new work after expiry and warn about duplicate risk when needed.
- If the first request is accepted for async processing, later async failure is still a stored outcome; replay should not create new work.
- Do not promise "exact same response" if timestamps or headers can change. "Equivalent outcome" plus stable resource identity is often more honest.
- Retry-After is advisory and only useful if clients know whether to retry the original write, poll an operation, or perform a read.
- `If-None-Match: *` can protect create-if-absent flows, while `If-Match` protects update-if-current flows.
- A request that fails strict JSON decoding should not usually burn an idempotency key, but a request accepted past the durable boundary usually should.

## Source Links Gathered Through Exa
- RFC 9110, HTTP Semantics, methods, validators, conditional requests, and Retry-After: https://www.rfc-editor.org/rfc/rfc9110.html
- RFC 6585, `428 Precondition Required` and `429 Too Many Requests`: https://www.rfc-editor.org/rfc/rfc6585.html
- IETF HTTPAPI Idempotency-Key Internet-Draft, work in progress: https://datatracker.ietf.org/doc/draft-ietf-httpapi-idempotency-key-header/
- RFC 9457, Problem Details error payloads: https://www.rfc-editor.org/rfc/rfc9457.html

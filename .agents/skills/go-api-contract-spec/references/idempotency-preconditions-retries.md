# Idempotency, Preconditions, And Retries

## When To Load
Load this when a write can be retried, a timeout may hide whether mutation happened, duplicate work is dangerous, optimistic concurrency matters, `ETag` or `If-Match` is in scope, or endpoints need retry classification.

## Decision Rubric
- Separate HTTP method idempotency from API-contract idempotency. `PUT` and `DELETE` have method idempotency; `POST` and `PATCH` need extra rules for safe retries.
- For retryable non-idempotent writes, require `Idempotency-Key` or expose a recovery read that cannot create duplicate work.
- Verify the current `Idempotency-Key` status before claiming RFC compliance; when it is still an Internet-Draft, use it as strong design input, not as a final RFC guarantee.
- Define key syntax, entropy expectations, tenant/account scope, operation scope, route or method scope, and TTL. This skill's `24h` TTL is a provider-inspired starting heuristic, not a standards default.
- For new `Idempotency-Key` contracts, prefer draft-compatible Structured Field string syntax, for example `Idempotency-Key: "8e03978e-40d5-43e8-bc93-6894a57f9324"`. Preserve an established unquoted or provider-specific convention only as an explicit compatibility choice.
- Compare retried payloads at the normalized contract level when irrelevant JSON formatting, object order, or defaults can differ.
- Same key plus same normalized payload should return an equivalent prior outcome after the durable boundary.
- Same key plus different normalized payload should fail as a stable caller-fixable validation problem, not an in-progress conflict. The current Idempotency-Key draft uses `422`; concurrent same-key attempts use `409`.
- Same key while the first attempt is still in progress needs its own response, often `409 Conflict` with polling or retry guidance.
- If an endpoint requires `Idempotency-Key` and the client omits it, treat that as a request-validation problem, typically `400` with a problem or documentation link. Do not reuse `428 Precondition Required`; HTTP preconditions are `If-Match`/`If-None-Match`-style validators, not idempotency keys.
- Define which failures reserve the key. Strict decode errors usually should not; accepted async work usually should.
- If lost updates matter, expose `ETag` on reads and successful writes, require `If-Match` on risky mutations, return `412` for false supplied preconditions, and return `428` when required preconditions are missing.

## Imitate
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

Good contract rule: within the TTL, the same tenant, key, and normalized payload returns an equivalent outcome and the same payment identity. A different payload returns a stable caller-fixable validation problem.

```http
PATCH /v1/orders/ord_123 HTTP/1.1
Content-Type: application/merge-patch+json
If-Match: "v4"

{ "shipping_address_id": "addr_789" }

HTTP/1.1 412 Precondition Failed
Content-Type: application/problem+json
```

Good: `412` tells the client to read, reconcile, and retry with a fresh validator if appropriate.

## Reject
```http
POST /v1/payments HTTP/1.1

HTTP/1.1 504 Gateway Timeout
```

Bad if no recovery rule exists: the client cannot know whether retrying creates a duplicate.

```http
PATCH /v1/orders/ord_123 HTTP/1.1
If-Match: "stale"

HTTP/1.1 409 Conflict
```

Usually bad: when a supplied HTTP precondition evaluates false, `412` is the sharper signal.

## Agent Traps
- Idempotency keys must be tenant-scoped. A global lookup can leak stored outcomes across callers.
- Low-entropy keys are abusable; the contract should reject invalid formats.
- Post-TTL reuse is a real product risk. Say whether the key starts new work and warn about duplicates when needed.
- Do not promise byte-identical replay responses if timestamps or headers can change. "Equivalent outcome" plus stable resource identity is often more honest.
- `Retry-After` is useless unless clients know whether to retry the write, poll an operation, or perform a read.
- `If-None-Match: *` protects create-if-absent flows; `If-Match` protects update-if-current flows.

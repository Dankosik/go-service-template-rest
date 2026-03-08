# Profile Editing API Contract

## Contract Framing And Assumptions

- Audience: first-party web and mobile clients editing the authenticated customer's own profile.
- Trust boundary: customer identity comes from validated auth context; the contract must not require clients to supply an arbitrary customer ID in the URI.
- Primary client-visible problem: replace `POST /v1/profile/update` with a resource-oriented contract that supports partial updates, deterministic validation failures, and safe concurrent edits.
- Consistency target: synchronous write acknowledgement; no async job resource is needed for normal profile edits.

### Nontrivial decision comparison

1. `PUT /v1/profile`
   - Rejected. Partial updates are a hard requirement, and `PUT` implies full replacement. Omitted fields would be ambiguous and create accidental-clearing risk.
2. `PATCH /v1/profile` with `application/merge-patch+json`
   - Selected. It matches partial-update semantics, keeps the resource model simple, and works cleanly with `ETag` + `If-Match`.
3. `PATCH /v1/customers/{customerId}/profile`
   - Rejected. For first-party self-service editing, the path parameter duplicates auth context, expands the authorization surface, and makes accidental cross-customer access easier to mis-specify.

## Resource And Endpoint Matrix

| Endpoint | Method | Purpose | Selected semantics |
| --- | --- | --- | --- |
| `/v1/profile` | `GET` | Read current authenticated customer's editable profile | Returns canonical profile representation and current strong `ETag` |
| `/v1/profile` | `PATCH` | Partially update current authenticated customer's profile | Requires `If-Match`; patch media type is `application/merge-patch+json` |
| `/v1/profile/update` | `POST` | Deprecated compatibility endpoint during rollout | Kept temporarily for existing clients; behavior aligned to the new contract where possible |

### Resource shape

- Treat the profile as a singleton resource owned by the authenticated customer.
- The representation contains customer-editable fields plus server-owned metadata such as `updated_at`.
- Mutable-field whitelist must be explicit in OpenAPI. Example editable fields: `first_name`, `last_name`, `display_name`, `phone`, `locale`, `marketing_preferences`.
- Server-owned or immutable fields such as `id`, `customer_id`, `email`, verification flags, and audit timestamps are read-only and must fail consistently if written.

### `PATCH /v1/profile` patch semantics

- Media type: `application/merge-patch+json`.
- Request body must be a JSON object. Arrays or scalar roots are rejected with `400`.
- Omitted fields mean "leave unchanged".
- Present non-null fields replace the existing value for that field.
- `null` clears a field only when that field is declared nullable in the contract.
- `null` for a non-nullable field returns `422`.
- Unknown fields return `400` with a stable machine-readable error code.

## Request, Response, And Error Model

### Successful responses

#### `GET /v1/profile`

- `200 OK`
- Headers:
  - `ETag: "<strong-profile-version>"`
- Body:

```json
{
  "id": "prof_123",
  "first_name": "Ana",
  "last_name": "Ng",
  "display_name": "Ana N.",
  "phone": "+14155550123",
  "locale": "en-US",
  "marketing_preferences": {
    "email_opt_in": true
  },
  "updated_at": "2026-03-07T12:34:56Z"
}
```

#### `PATCH /v1/profile`

- `200 OK`
- Headers:
  - `ETag: "<new-strong-profile-version>"`
- Body: full updated profile representation, not only changed fields.
- Rationale: clients need the committed server view and the new validator/concurrency token in one round trip.

### Error model

- Error media type: `application/problem+json`.
- Base required fields: `type`, `title`, `status`.
- Stable extensions for this surface:
  - `code`: endpoint-stable machine code
  - `request_id`: correlation identifier
  - `errors`: field-level validation entries for `422`

#### Field-level validation extension

```json
{
  "type": "https://api.example.internal/problems/profile-validation",
  "title": "Profile validation failed",
  "status": 422,
  "detail": "One or more profile fields are invalid.",
  "code": "profile_validation_failed",
  "request_id": "9ccecdfd92c94665a46406f8ba73cd77",
  "errors": [
    {
      "pointer": "/first_name",
      "code": "too_long",
      "message": "must be at most 100 characters"
    },
    {
      "pointer": "/phone",
      "code": "invalid_format",
      "message": "must be a valid E.164 phone number"
    }
  ]
}
```

#### Deterministic validation rules

- Use `400 Bad Request` for malformed JSON, unsupported top-level shape, unknown fields, duplicate mutually-exclusive inputs, and invalid `If-Match` syntax.
- Use `422 Unprocessable Content` for semantic field validation failures after successful decoding.
- For the same invalid payload, return the same `status`, top-level `code`, and field-level `errors` ordering.
- `errors` must be sorted deterministically by JSON Pointer, then by `code`.
- Immutable-field writes return `422` with `code: "immutable_field"`.

### Status code mapping

| Status | When |
| --- | --- |
| `200 OK` | Profile read or update succeeded |
| `400 Bad Request` | Malformed JSON, unknown field, invalid patch shape, invalid `If-Match` syntax |
| `401 Unauthorized` | Caller is unauthenticated |
| `404 Not Found` | Authenticated customer has no profile resource |
| `409 Conflict` | `Idempotency-Key` reused with different request material, or another non-precondition domain conflict |
| `412 Precondition Failed` | `If-Match` ETag does not match current profile version |
| `415 Unsupported Media Type` | `PATCH` is not sent as `application/merge-patch+json` |
| `422 Unprocessable Content` | Decoded payload fails profile validation rules |
| `428 Precondition Required` | `PATCH` omits required `If-Match` |
| `429 Too Many Requests` | Existing API rate limit policy applies |
| `500` / `503` | Existing platform problem-details mapping applies |

## Retry, Idempotency, And Concurrency Rules

- `GET /v1/profile` is retry-safe by protocol.
- `PATCH /v1/profile` is not protocol-idempotent.
- Optimistic concurrency is mandatory because concurrent edits are expected:
  - `GET /v1/profile` returns a strong `ETag`.
  - `PATCH /v1/profile` requires `If-Match` with exactly one current `ETag` value.
  - Missing `If-Match` returns `428 Precondition Required`.
  - Stale `If-Match` returns `412 Precondition Failed`.
  - Successful write returns the new `ETag`.
- `If-Match: *` is not supported on this endpoint because blind overwrite defeats the concurrency contract.
- `Idempotency-Key` is supported on `PATCH /v1/profile` and should be sent by clients that may retry after network ambiguity.
  - Scope: authenticated customer + method + route.
  - Same key + same request body + same `If-Match` returns the original success or error outcome.
  - Same key + different request body or different `If-Match` returns `409 Conflict`.
  - Default dedup retention target: `24h`.

## Async, Freshness, And Webhook Notes

- No async operation resource is needed. Profile edits are contractually synchronous.
- No webhook or callback behavior is part of this contract.
- Freshness model:
  - The response body of a successful `PATCH` is the committed server view for that write.
  - `GET /v1/profile` is intended to read the current canonical profile state.
  - Reopen this contract if the read path is cache-backed and may lag; in that case the API must disclose freshness explicitly with a field such as `as_of` or `last_updated_at`.

## Compatibility, Artifact Updates, And Handoffs

### Compatibility classification

- Add `GET /v1/profile` and `PATCH /v1/profile`: additive.
- Mark `POST /v1/profile/update` deprecated and introduce successor headers: additive.
- Future removal of `POST /v1/profile/update`: breaking and must wait for an explicit deprecation window or a major-version boundary.

### Legacy endpoint rollout contract

- Keep `POST /v1/profile/update` during rollout as a deprecated compatibility alias.
- Preserve the existing request shape for old clients during the coexistence window.
- Align its response and error model to the new contract:
  - return `application/problem+json` on errors,
  - emit deterministic validation codes,
  - return `ETag` on successful writes,
  - accept `If-Match` and `Idempotency-Key` when provided.
- Add deprecation signaling headers on every legacy response:
  - `Deprecation: true`
  - `Sunset: <rfc-8594-date>`
  - `Link: </v1/profile>; rel="successor-version"`
- During coexistence, if legacy callers omit `If-Match`, `POST /v1/profile/update` continues to process the write with legacy last-write-wins behavior, returns `200`, emits the new `ETag`, and remains explicitly non-canonical for concurrency-safe editing.

### Spec artifacts to update

- Primary API decision artifact: `specs/<feature-id>/30-api-contract.md` or the API section of `spec.md`.
- Wire contract: `api/openapi/service.yaml`
- Boundary-side effects only:
  - `specs/<feature-id>/50-security-observability-devops.md`
    - auth context source, request ID propagation, deprecation headers, rate-limit note
  - `specs/<feature-id>/55-reliability-and-resilience.md`
    - conditional-write retry policy, `Idempotency-Key` retention, stale-write failure semantics
  - `specs/<feature-id>/70-test-plan.md`
    - contract tests for merge-patch semantics, `412/428`, deterministic `422` ordering, legacy compatibility responses
  - `specs/<feature-id>/80-open-questions.md`
    - unresolved field mutability/nullability list, legacy sunset date, any cache-freshness uncertainty
  - `specs/<feature-id>/90-signoff.md`
    - accepted migration path, deprecation criteria, reopen conditions

### Adjacent handoffs

- No mandatory handoff right now.
- Reopen to `go-domain-invariant-spec` if profile-specific acceptance rules become materially more complex than field validation.
- Reopen to `go-security-spec` if customer-to-profile access rules are broader than self-service editing.

## Open Questions, Risks, And Reopen Conditions

### Open questions

- Which profile fields are nullable versus required-once-set?
- Is `email` editable on this surface or managed by a separate verified flow?
- What exact sunset date is acceptable for the legacy endpoint?

### Risks

- Allowing the legacy endpoint to coexist without hard `If-Match` enforcement keeps a temporary lost-update risk for old clients.
- If the service actually reads through a cache, the strong-read wording above would be inaccurate until freshness metadata is added.

### Reopen conditions

- Product requires bulk profile edits or async moderation/review before persistence.
- The profile resource is not guaranteed to exist for every authenticated customer.
- Mixed-client rollout shows old clients cannot adopt `ETag`/`If-Match` in the expected window.

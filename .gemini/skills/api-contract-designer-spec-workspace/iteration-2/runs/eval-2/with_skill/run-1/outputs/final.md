# Profile Editing API Contract

## Contract Framing And Assumptions

- Audience: first-party web and mobile clients editing the authenticated customer's own profile.
- Client-visible problem: replace `POST /v1/profile/update` with a resource-oriented contract that supports partial updates, deterministic validation failures, and safe concurrent edits.
- Resource ownership: the editable profile is a singleton resource bound to the authenticated subject, so the canonical URI should not require a caller-supplied customer ID.
- Consistency target: synchronous acceptance and synchronous committed response for normal profile edits. No async operation resource is needed for this contract.

### Nontrivial decision comparison

1. `PUT /v1/profile`
   - Rejected. The prompt requires partial updates, and `PUT` makes omitted fields ambiguous or accidentally destructive.
2. `PATCH /v1/profile`
   - Selected. It matches partial-update semantics, keeps the resource shape simple, and composes cleanly with `ETag` plus `If-Match`.
3. `PATCH /v1/customers/{customerId}/profile`
   - Rejected. For self-service profile editing, the path parameter duplicates auth context and widens the authorization surface without adding client value.
4. `PATCH /v1/profile` with `application/json-patch+json`
   - Rejected. Ordered patch operations and array-index manipulation are unnecessary here and make validation/error reporting harder to keep deterministic across clients.

## Resource And Endpoint Matrix

| Endpoint | Method | Purpose | Success statuses | Selected semantics |
| --- | --- | --- | --- | --- |
| `/v1/profile` | `GET` | Read the authenticated customer's editable profile | `200 OK`, `304 Not Modified` | Returns the canonical profile representation and a strong `ETag` |
| `/v1/profile` | `PATCH` | Partially update the authenticated customer's editable profile | `200 OK` | Canonical partial-update endpoint; requires `If-Match`; request media type is `application/merge-patch+json` |
| `/v1/profile/update` | `POST` | Deprecated compatibility endpoint during rollout | Existing legacy success status(s), typically `200 OK` | Compatibility-only bridge for old clients; not the canonical write surface |

### Resource shape

- Model the profile as a singleton resource at `/v1/profile`.
- The representation contains an explicit whitelist of customer-editable fields plus read-only server-managed metadata such as `updated_at`.
- Example editable fields: `first_name`, `last_name`, `display_name`, `phone`, `locale`, `marketing_preferences`.
- Read-only or separately-governed fields such as `id`, `customer_id`, `email`, verification flags, and audit timestamps are exposed as read-only in the representation and are not writable through this surface.

### `PATCH /v1/profile` patch semantics

- Media type: `application/merge-patch+json`.
- The request body must be a JSON object. Scalar or array roots return `400 Bad Request`.
- Omitted fields mean "leave unchanged".
- Present non-null fields replace the current value for that field.
- `null` clears a field only when that field is explicitly nullable in the contract.
- `null` for a non-nullable field returns `422 Unprocessable Content`.
- Unknown fields return `400 Bad Request`.
- Attempts to write read-only fields return `422 Unprocessable Content`.
- Arrays and nested objects are replaced at the field being patched; this contract does not support fine-grained array element operations.

## Request, Response, And Error Model

### Successful responses

#### `GET /v1/profile`

- `200 OK` with the full profile representation.
- `304 Not Modified` when `If-None-Match` matches the current entity tag.
- Headers:
  - `ETag: "<strong-profile-version>"`

#### `PATCH /v1/profile`

- Request headers:
  - `If-Match: "<strong-profile-version>"` is required.
  - `Idempotency-Key` is optional but recommended for retry after network ambiguity.
- `200 OK`
- Response body: the full committed profile representation, not just changed fields.
- Response headers:
  - `ETag: "<new-strong-profile-version>"`

### Error model

- Error media type: `application/problem+json`.
- Base shape extends the repo's existing `Problem` schema and keeps one stable profile for this surface.
- Required fields:
  - `type`
  - `title`
  - `status`
  - `detail`
- Stable extensions for this surface:
  - `request_id`
  - `code`
  - `errors` for field-level validation failures

### Field-level validation extension

```json
{
  "type": "https://api.example.internal/problems/profile-validation",
  "title": "Profile validation failed",
  "status": 422,
  "detail": "One or more profile fields are invalid.",
  "request_id": "9ccecdfd92c94665a46406f8ba73cd77",
  "code": "profile_validation_failed",
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

### Deterministic validation rules

- Use `400 Bad Request` for malformed JSON, invalid top-level patch shape, unknown fields, duplicate JSON keys if the decoder can detect them, and invalid `If-Match` syntax.
- Use `422 Unprocessable Content` for semantic validation failures after successful decoding, including immutable-field writes and domain field validation failures.
- For the same invalid request on the same API version, the server must return the same `status`, top-level `code`, and ordered `errors` entries.
- `errors` entries must be sorted deterministically by JSON Pointer, then by error `code`.
- `message` is human-readable; machine consumers should key on `code` and `pointer`.

### Status code mapping

| Status | When |
| --- | --- |
| `200 OK` | `GET` or `PATCH` succeeded |
| `304 Not Modified` | `GET` with matching `If-None-Match` |
| `400 Bad Request` | Malformed JSON, unknown field, invalid patch root, invalid precondition syntax |
| `401 Unauthorized` | Caller is unauthenticated |
| `404 Not Found` | The authenticated subject has no profile resource |
| `409 Conflict` | `Idempotency-Key` reused with different request material, or another non-precondition domain conflict |
| `412 Precondition Failed` | `If-Match` does not match the current profile `ETag` |
| `415 Unsupported Media Type` | `PATCH` is not sent as `application/merge-patch+json` |
| `422 Unprocessable Content` | The decoded payload fails profile validation rules |
| `428 Precondition Required` | `PATCH` omits required `If-Match` |
| `429 Too Many Requests` | Existing API rate-limit policy applies |
| `500` / `503` | Existing platform problem-details mapping applies |

## Retry, Idempotency, And Concurrency Rules

- `GET /v1/profile` is retry-safe by protocol.
- `PATCH /v1/profile` is retry-unsafe without an idempotency key.
- Canonical optimistic-concurrency contract:
  - `GET /v1/profile` returns a strong `ETag`.
  - `PATCH /v1/profile` requires exactly one `If-Match` value.
  - Missing `If-Match` returns `428 Precondition Required`.
  - Stale `If-Match` returns `412 Precondition Failed`.
  - `If-Match: *` is not supported because blind overwrite defeats the concurrency contract.
  - A successful write returns the new `ETag`.
- Optional `Idempotency-Key` support on `PATCH /v1/profile`:
  - Scope: authenticated customer + HTTP method + route.
  - Same key + same request body + same `If-Match` returns the original outcome.
  - Same key + different request body or different `If-Match` returns `409 Conflict`.
  - Retention target: `24h`.

## Async, Freshness, And Webhook Notes

- This contract is synchronous. `PATCH /v1/profile` should not return `202 Accepted`.
- The `PATCH` response body is the committed server view for that write.
- `GET /v1/profile` is intended to read the current canonical profile state. If the actual read path can lag because of cache or projection delay, this contract must be reopened to add explicit freshness disclosure such as `as_of`.
- No webhook or callback contract is part of this surface.

## Compatibility, Artifact Updates, And Handoffs

### Compatibility classification

- Add `GET /v1/profile`: additive.
- Add `PATCH /v1/profile`: additive.
- Mark `POST /v1/profile/update` deprecated: additive if its existing request and success response remain wire-compatible.
- Remove `POST /v1/profile/update`: breaking and requires a documented sunset or a major-version boundary.

### Legacy endpoint rollout contract

- Keep `POST /v1/profile/update` during the migration window as a deprecated compatibility endpoint.
- New web and mobile development should treat `/v1/profile` as the only canonical profile-editing surface.
- Preserve the legacy request shape and legacy success body during coexistence so existing clients do not break.
- Additive legacy-response headers are allowed:
  - `Deprecation: true`
  - `Sunset: <rfc-8594-date>`
  - `Link: </v1/profile>; rel="successor-version"`
  - `ETag: "<new-strong-profile-version>"`
- Legacy concurrency behavior:
  - If a legacy caller sends `If-Match`, honor it and return `412` on stale writes.
  - If a legacy caller omits `If-Match`, keep temporary last-write-wins behavior for backward compatibility.
  - Document explicitly that the legacy endpoint is not concurrency-safe unless `If-Match` is supplied.
- Legacy error behavior:
  - Do not silently replace the legacy wire error schema with the new `application/problem+json` model unless existing consumers are confirmed tolerant.
  - Deterministic problem-details validation is guaranteed on `PATCH /v1/profile`.
  - If the legacy endpoint can add stable machine codes without breaking consumers, treat that as additive; otherwise keep the legacy error contract until removal.
- Removal gate:
  - published deprecation window,
  - telemetry showing legacy traffic is at or near zero,
  - all supported web/mobile release trains migrated.

### Spec artifacts to update

- Canonical API decision artifact:
  - `specs/<feature-id>/30-api-contract.md` or the API section of `spec.md`
- Wire contract source of truth:
  - `api/openapi/service.yaml`
- Additional spec-package deltas only where this contract changes behavior:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
    - `If-Match` failure semantics, retry classification, `Idempotency-Key` retention
  - `specs/<feature-id>/70-test-plan.md`
    - merge-patch semantics, `304`, `412`, `428`, deterministic `422` ordering, legacy coexistence cases
  - `specs/<feature-id>/80-open-questions.md`
    - field mutability/nullability list, legacy sunset date, profile-existence assumption
  - `specs/<feature-id>/90-signoff.md`
    - deprecation acceptance criteria and removal gate

### Adjacent handoffs

- No mandatory adjacent handoff is required for this contract as written.
- Reopen to a domain-invariant pass if profile editing includes workflow rules beyond field validation.
- Reopen to a security-spec pass if the resource will be edited outside self-service authenticated-customer scope.

## Open Questions, Risks, And Reopen Conditions

### Open questions

- Which fields are editable, which are read-only, and which are nullable?
- Is `email` editable here, or is it governed by a separate verified change flow?
- Is the profile guaranteed to exist for every authenticated customer, or does `404` remain reachable?
- What sunset date is acceptable for `POST /v1/profile/update`?

### Risks

- Keeping the legacy endpoint writable without mandatory `If-Match` leaves a temporary lost-update risk for old clients.
- Mixed canonical and legacy error contracts during rollout increase client-support complexity, but they preserve backwards compatibility.
- If the service cannot expose a strong `ETag`, the proposed `If-Match` contract is not release-ready.

### Reopen conditions

- Profile writes may complete asynchronously or require moderation before commit.
- Reads are served from a lagging cache or projection and cannot provide the stated freshness behavior.
- The surface expands from self-service editing to admin or cross-customer editing.
- Product requires JSON Patch-style array operations or bulk profile edits.

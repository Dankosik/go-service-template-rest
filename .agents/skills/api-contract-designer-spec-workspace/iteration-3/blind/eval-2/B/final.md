# Customer Profile Editing API Contract

## Decision Summary

- Canonical resource: treat the authenticated customer's editable profile as a singleton resource at `GET /v1/profile` and `PATCH /v1/profile`.
- Selected write method: `PATCH`, not `PUT`, because clients need partial updates and omitted fields must not imply reset or deletion.
- Patch format: `application/merge-patch+json`.
- Concurrency: `GET /v1/profile` returns a strong opaque `ETag`; `PATCH /v1/profile` requires `If-Match`.
- Retry/idempotency: `PATCH /v1/profile` also requires `Idempotency-Key` so mobile/web retries after network ambiguity are safe by contract.
- Error model: one stable `application/problem+json` profile with deterministic machine-readable field errors.
- Rollout: add the new resource endpoints without removing `POST /v1/profile/update`; keep the old action endpoint as a deprecated compatibility bridge until migration is complete.

## Selected And Rejected Options

### Selected

1. `PATCH /v1/profile` with `application/merge-patch+json`
   - Why: matches partial-update semantics, keeps the contract resource-oriented, and avoids forcing full-document replacement.
2. `/v1/profile` as a singleton resource scoped by auth context
   - Why: web and mobile clients are editing their own profile; the server should derive ownership from identity, not from a caller-supplied customer ID.
3. Required conditional writes with `If-Match`
   - Why: concurrent edits already happen, so lost-update prevention must be explicit at the API boundary.

### Rejected

1. `PUT /v1/profile`
   - Rejected because full replacement conflicts with the partial-update requirement and makes omitted-field behavior dangerous under concurrency.
2. `PATCH /v1/profile` with `application/json-patch+json`
   - Rejected because operation-list patch documents add client complexity and make deterministic validation/errors harder than ordinary merge-patch.
3. `/v1/customers/{customer_id}/profile`
   - Rejected for the primary client contract because it duplicates auth context and widens the authorization surface without adding value for self-service clients.
4. Last-write-wins updates with no required precondition
   - Rejected because that preserves the old endpoint's main correctness flaw.

## Resource And Endpoint Matrix

| Endpoint | Purpose | Success statuses | Required request conditions | Retry / concurrency behavior | Compatibility class |
| --- | --- | --- | --- | --- | --- |
| `GET /v1/profile` | Read the authenticated customer's current editable profile | `200 OK`, `304 Not Modified` | Auth required | Safe and idempotent by protocol. Returns strong `ETag`. | Additive |
| `PATCH /v1/profile` | Partially update editable profile fields | `200 OK` | Auth required, `Content-Type: application/merge-patch+json`, `If-Match`, `Idempotency-Key` | Atomic per resource. Missing preconditions fail explicitly. Stale write fails distinctly. | Additive |
| `POST /v1/profile/update` | Deprecated compatibility endpoint for unmigrated clients | Preserve legacy success status/body during rollout | Existing auth behavior remains | Temporary bridge only. New concurrency guarantees apply only when callers opt into the new headers. | Additive now, eventual removal is breaking |

## Request, Response, And Error Model

### Canonical resource shape

- OpenAPI response schema: `CustomerProfile`.
- OpenAPI patch schema: `CustomerProfilePatch`.
- `CustomerProfilePatch` contains only editable fields and must reject undeclared properties.
- Server-managed or immutable fields remain outside `CustomerProfilePatch` and are `readOnly` in `CustomerProfile`.
- The contract does not invent the exact field list; the schema update must explicitly classify each field as editable, read-only, nullable, or non-nullable.

### `GET /v1/profile`

- Purpose: return the current profile for the authenticated customer.
- Request headers:
  - `If-None-Match` optional.
- Response headers:
  - `ETag: "<opaque-profile-version>"`
- Response behavior:
  - `200 OK` returns the full `CustomerProfile`.
  - `304 Not Modified` returns no body when `If-None-Match` matches the current `ETag`.

### `PATCH /v1/profile`

- Required headers:
  - `Content-Type: application/merge-patch+json`
  - `If-Match: "<etag-from-latest-read>"`
  - `Idempotency-Key: <opaque-client-generated-key>`
- Patch semantics:
  - omitted field: leave unchanged
  - present non-null field: replace that field value
  - present object field: merge by object member per JSON Merge Patch semantics
  - present array field: replace the full array
  - present `null`: clear only if that field is explicitly nullable and clearable in `CustomerProfilePatch`
  - unknown field: fail validation
  - immutable field: fail validation
  - patch application is atomic for the whole profile resource
- Success response:
  - `200 OK`
  - body: full updated `CustomerProfile`, not only changed fields
  - headers: current `ETag`
- No-op semantics:
  - a valid no-op patch still returns `200 OK` with the current representation and current `ETag`
  - no-op patches do not require a synthetic version bump

### Preconditions, concurrency, and idempotency

- `ETag` values are opaque to clients.
- `PATCH /v1/profile` requires exactly one `If-Match` value.
  - missing `If-Match` -> `428 Precondition Required`
  - stale or mismatched `If-Match` -> `412 Precondition Failed`
  - `If-Match: *` is not supported
- `PATCH /v1/profile` requires `Idempotency-Key`.
  - missing `Idempotency-Key` -> `428 Precondition Required`
  - same key + same authenticated subject + same route + same normalized patch body + same `If-Match` -> return an equivalent prior outcome
  - same key + different normalized patch body or different `If-Match` -> `409 Conflict`
- Idempotency retention target: `24h`.
- Successful writes always return the new current `ETag`.

### Stable error model

- Media type: `application/problem+json`.
- Base schema: extend the repo's existing `Problem` schema instead of introducing a second error envelope.
- Required top-level fields:
  - `type`
  - `title`
  - `status`
  - `detail`
- Stable extensions for this surface:
  - `request_id`
  - `code`
  - `errors`
- `errors` item shape:
  - `pointer`: JSON Pointer to the offending field
  - `code`: stable machine-readable validation code
  - `message`: human-readable explanation

### Deterministic validation behavior

- Use `400 Bad Request` for malformed JSON, trailing tokens, invalid header syntax, or other framing errors that prevent a valid patch document from being interpreted.
- Use `422 Unprocessable Content` for any syntactically valid payload that violates the contract, including unknown fields, immutable-field writes, invalid enum/format/length, or forbidden null clears.
- For the same invalid request on the same API version, the server must return the same HTTP status, the same top-level `code`, and the same ordered `errors` entries.
- `errors` must be sorted deterministically by `pointer`, then by `code`.
- Clients must key on `code` and `pointer`, not on `message`.

### Example validation failure

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

### Status code mapping

| Status | When it applies |
| --- | --- |
| `200 OK` | Successful `GET` or `PATCH` |
| `304 Not Modified` | `GET` with matching `If-None-Match` |
| `400 Bad Request` | Malformed JSON, trailing tokens, or invalid precondition/idempotency header syntax |
| `401 Unauthorized` | Missing or invalid authentication |
| `403 Forbidden` | Authenticated caller is not allowed to read or edit this profile |
| `404 Not Found` | No profile resource exists for the authenticated subject |
| `409 Conflict` | Idempotency key reuse with different request material, or another non-precondition business conflict |
| `412 Precondition Failed` | `If-Match` does not match the current profile `ETag` |
| `415 Unsupported Media Type` | Request is not `application/merge-patch+json` |
| `422 Unprocessable Content` | The decoded payload violates the profile contract |
| `428 Precondition Required` | Missing required `If-Match` or missing required `Idempotency-Key` |
| `429 Too Many Requests` | Existing rate-limit policy applies; `Retry-After` if enforced |
| `500` / `503` | Existing platform problem-details mapping applies |

## Boundary And Cross-Cutting Policies

- Auth context, not caller-supplied identity, determines which profile is read or updated.
- OpenAPI security for both new profile operations should use `bearerAuth`.
- Strict JSON defaults:
  - reject malformed payloads
  - reject trailing tokens
  - reject undeclared fields in `CustomerProfilePatch`
- Boundary order is explicit:
  1. auth and transport checks
  2. content-type validation
  3. strict JSON decode
  4. normalization
  5. semantic validation
  6. `If-Match` and `Idempotency-Key` checks
  7. business update
- `PATCH /v1/profile` is atomic per resource.
- Problem responses must not leak stack traces, SQL text, secrets, or infrastructure topology.
- If rate limiting exists on this surface, it must use `429` and should include `Retry-After`.

## Consistency And Async Notes

- This contract is synchronous; profile edits are not exposed as an operation resource.
- `PATCH /v1/profile` returns the committed server view for that write.
- `GET /v1/profile` is intended to be strong for the caller's own profile.
- After a successful `PATCH`, a follow-up `GET /v1/profile` by the same subject must reflect the new state and the new `ETag`.
- If the real implementation cannot provide that read-after-write behavior, the contract must be reopened to add freshness disclosure or an async redesign rather than silently weakening semantics.

## Compatibility Notes

### Canonical new surface

- All new web and mobile work should use:
  - `GET /v1/profile`
  - `PATCH /v1/profile`

### Legacy `POST /v1/profile/update` coexistence

- Keep `POST /v1/profile/update` during the migration window as a deprecated compatibility endpoint.
- Preserve its current request shape and current success body while coexistence lasts.
- Additive legacy response headers are allowed:
  - `Deprecation: true`
  - `Sunset: <rfc-8594-date>`
  - `Link: </v1/profile>; rel="successor-version"`
  - `ETag: "<current-profile-version>"`
- Legacy behavior during rollout:
  - if a caller sends `If-Match`, honor it
  - if a caller sends `Idempotency-Key`, honor it
  - if a caller omits those headers, preserve legacy behavior temporarily and document that the endpoint is not concurrency-safe in that mode
- Legacy error compatibility:
  - do not unilaterally replace the legacy wire error schema with the new `application/problem+json` model unless current consumers are confirmed tolerant
  - deterministic `application/problem+json` errors are guaranteed on `PATCH /v1/profile`
- Removal gate:
  - published deprecation window
  - telemetry showing legacy traffic is effectively gone
  - supported web/mobile release trains migrated

### Change classification

- Add `GET /v1/profile`: additive
- Add `PATCH /v1/profile`: additive
- Extend the shared `Problem` schema with deterministic validation fields: additive if existing consumers tolerate new properties
- Mark `POST /v1/profile/update` deprecated: additive if its request and success response remain compatible
- Require new precondition/idempotency headers on the old `POST`: breaking
- Remove `POST /v1/profile/update`: breaking

## Spec Artifacts To Update

1. `specs/<feature-id>/30-api-contract.md`
   - Canonical decision record for resource shape, `PATCH` selection, error model, preconditions, and rollout policy.
   - If this repo keeps a single-file feature spec instead of split artifacts, put the same content in the API contract / Decisions section of `specs/<feature-id>/spec.md`.
2. `api/openapi/service.yaml`
   - Add `GET /v1/profile` and `PATCH /v1/profile`.
   - Keep `POST /v1/profile/update` documented as deprecated during coexistence.
   - Add or extend components for:
     - `CustomerProfile`
     - `CustomerProfilePatch`
     - deterministic validation error items under the shared `Problem` schema
     - reusable headers for `ETag`, `If-Match`, `If-None-Match`, and `Idempotency-Key`
     - reusable responses for `412`, `415`, `422`, `428`, and `429`
   - Mark the new profile operations with `bearerAuth`.
   - Declare `application/merge-patch+json` on `PATCH`.
3. `specs/<feature-id>/80-open-questions.md` or the equivalent open-questions section in `spec.md`
   - Record unresolved field mutability/nullability, profile-existence behavior, and the approved sunset date for the legacy endpoint.

- Generated code such as `internal/api/openapi.gen.go` should be regenerated after the OpenAPI change is approved, but it is not a spec artifact and is intentionally not part of the contract record.

## Open Questions And Risks

### Open questions

- Which profile fields are editable, and which of them are nullable versus required-once-set?
- Is the profile guaranteed to exist for every authenticated customer, or should `404` remain part of the contract?
- Does the client-visible path prefix remain `/v1` as stated in the prompt, or must the service-wide `/api/v1` prefix be applied consistently to both new and legacy routes?
- What sunset date is acceptable for removing `POST /v1/profile/update`?

### Risks

- If the service cannot mint a strong `ETag`, the conditional-write contract is not ready.
- If legacy clients depend on the exact legacy error envelope, error-model convergence must wait until those clients migrate.
- If some fields need custom workflows such as verification or moderation, they may need separate sub-resources instead of generic merge-patch semantics.

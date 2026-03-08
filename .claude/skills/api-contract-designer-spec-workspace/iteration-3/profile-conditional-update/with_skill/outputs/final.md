# Customer Profile Editing API Contract

## Contract Framing And Assumptions

- Audience: first-party web and mobile clients editing the authenticated customer's own profile.
- Client-visible problem: replace `POST /v1/profile/update` with a resource-oriented contract that supports partial edits, deterministic validation failures, and safe concurrent updates.
- Canonical resource: the editable profile is a singleton owned by the authenticated subject, so the canonical URI does not take a caller-supplied customer ID.
- Consistency target: synchronous commit with strong read-after-write for the caller's own profile. No async operation resource is needed for this contract.

### Nontrivial choices

1. `PUT /v1/profile`
   - Rejected. Product needs partial updates, and `PUT` would make omitted-field behavior destructive or ambiguous.
2. `PATCH /v1/profile` with `application/merge-patch+json`
   - Selected. It matches partial-update semantics, keeps the surface resource-oriented, and works cleanly with `ETag` plus `If-Match`.
3. `PATCH /v1/customers/{customer_id}/profile`
   - Rejected. For self-service editing it duplicates auth context and unnecessarily widens the authorization surface.
4. `PATCH /v1/profile` returning `204 No Content`
   - Rejected. Clients need the normalized committed representation and the new `ETag` after each successful edit.

## Resource And Endpoint Matrix

| Endpoint | Method | Purpose | Success statuses | Required request conditions | Notes |
| --- | --- | --- | --- | --- | --- |
| `/v1/profile` | `GET` | Read the authenticated customer's editable profile | `200 OK`, `304 Not Modified` | Auth required | Returns canonical representation and strong `ETag` |
| `/v1/profile` | `PATCH` | Partially update editable profile fields | `200 OK` | Auth required, `Content-Type: application/merge-patch+json`, `If-Match`, `Idempotency-Key` | Canonical write surface; atomic per resource |
| `/v1/profile/update` | `POST` | Deprecated compatibility endpoint during rollout | Preserve existing legacy success status/body | Existing auth rules stay intact | Compatibility bridge only; not the canonical write contract |

### Resource shape

- OpenAPI response schema: `CustomerProfile`.
- OpenAPI request schema: `CustomerProfilePatch`.
- `CustomerProfilePatch` contains only editable fields and rejects undeclared properties.
- Read-only or separately-governed fields such as `id`, `customer_id`, verified contact attributes, and audit metadata remain `readOnly` in `CustomerProfile` and are not writable through this surface.

### `PATCH /v1/profile` patch semantics

- Media type: `application/merge-patch+json`.
- The request body must be a JSON object. Scalar or array roots return `400 Bad Request`.
- Omitted field: leave unchanged.
- Present non-null field: replace that field value.
- Present object field: merge by object member according to JSON Merge Patch.
- Present array field: replace the whole array.
- Present `null`: clear the field only when that field is explicitly nullable and clearable in `CustomerProfilePatch`; otherwise return `422 Unprocessable Content`.
- Unknown field: return `400 Bad Request`.
- Attempt to write a read-only field: return `422 Unprocessable Content`.
- Patch application is atomic for the whole profile resource.

## Request, Response, And Error Model

### `GET /v1/profile`

- `200 OK` returns the full `CustomerProfile`.
- `304 Not Modified` returns no body when `If-None-Match` matches the current entity tag.
- Response headers:
  - `ETag: "<opaque-profile-version>"`

### `PATCH /v1/profile`

- Required headers:
  - `Content-Type: application/merge-patch+json`
  - `If-Match: "<etag-from-latest-read>"`
  - `Idempotency-Key: <opaque-client-generated-key>`
- Success response:
  - `200 OK`
  - body: full committed `CustomerProfile`, not just changed fields
  - headers: current `ETag`
- No-op semantics:
  - if the patch is valid but makes no effective change after normalization, return `200 OK` with the current representation and current `ETag`
  - no-op patches do not require a synthetic version bump

### Deterministic validation and evaluation order

- Request processing order for `PATCH /v1/profile` is:
  1. auth and transport checks
  2. content-type check
  3. strict JSON decode
  4. normalization
  5. semantic validation
  6. `If-Match` evaluation
  7. mutation
- Result: malformed or semantically invalid requests return `400` or `422` consistently even if the supplied `ETag` is stale.
- For the same invalid request on the same API version, the server must return the same HTTP status, top-level problem `code`, and ordered field-level `errors`.

### Stable error model

- All new-contract failures use `application/problem+json`.
- Base schema extends the existing repository `Problem` schema; do not introduce a competing error envelope.
- Required fields:
  - `type`
  - `title`
  - `status`
  - `detail`
- Stable extensions:
  - `code`
  - `request_id`
  - `errors`
- `errors` item shape:
  - `pointer`: JSON Pointer to the offending field
  - `code`: stable machine-readable validation code
  - `message`: human-readable explanation
- `errors` entries must be sorted deterministically by `pointer`, then by `code`.

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
      "pointer": "/display_name",
      "code": "too_long",
      "message": "must be at most 80 characters"
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

| Status | When |
| --- | --- |
| `200 OK` | Successful `GET` or `PATCH` |
| `304 Not Modified` | `GET` with matching `If-None-Match` |
| `400 Bad Request` | Malformed JSON, trailing tokens, invalid patch root, unknown field, or invalid precondition header syntax |
| `401 Unauthorized` | Caller is unauthenticated |
| `403 Forbidden` | Caller is authenticated but not permitted to edit this profile |
| `404 Not Found` | No profile exists for the authenticated subject |
| `409 Conflict` | `Idempotency-Key` reuse with different request material, or another non-precondition domain conflict |
| `412 Precondition Failed` | `If-Match` does not match the current profile `ETag` |
| `415 Unsupported Media Type` | Request is not `application/merge-patch+json` |
| `422 Unprocessable Content` | Well-formed payload fails semantic validation |
| `428 Precondition Required` | Missing required `If-Match` or `Idempotency-Key` |
| `429 Too Many Requests` | Existing API throttling policy applies |
| `500` / `503` | Existing platform problem-details mapping applies |

## Retry, Idempotency, And Concurrency Rules

- `GET /v1/profile` is retry-safe by protocol semantics.
- `PATCH /v1/profile` is retry-safe by contract only when the caller supplies `Idempotency-Key`; otherwise automatic retry is not supported.
- Concurrency contract:
  - `GET /v1/profile` returns a strong opaque `ETag`.
  - `PATCH /v1/profile` requires exactly one `If-Match` value.
  - Missing `If-Match` returns `428 Precondition Required`.
  - Stale or mismatched `If-Match` returns `412 Precondition Failed`.
  - `If-Match: *` is not supported because blind overwrite defeats the lost-update guarantee.
  - Successful writes return the new `ETag`.
- Idempotency contract for `PATCH /v1/profile`:
  - key scope: authenticated subject + HTTP method + route
  - retention target: `24h`
  - same key + same normalized payload + same `If-Match` => return an equivalent outcome
  - same key + different normalized payload or different `If-Match` => `409 Conflict`

## Async, Freshness, And Webhook Notes

- This contract is synchronous. `PATCH /v1/profile` must not return `202 Accepted`.
- After a successful `PATCH`, a subsequent `GET /v1/profile` by the same authenticated subject must return the committed state and current `ETag`.
- No webhook or callback contract is part of this surface.
- If the implementation cannot provide strong read-after-write and instead serves a lagging projection or cache, this contract must be reopened and augmented with explicit freshness disclosure such as `as_of`.

## Compatibility, Artifact Updates, And Handoffs

### Compatibility strategy for `POST /v1/profile/update`

- Keep `POST /v1/profile/update` during the migration window as a deprecated compatibility endpoint.
- Preserve its current request shape and success body/status during coexistence so existing clients stay wire-compatible.
- Additive response headers are allowed on the legacy endpoint:
  - `Deprecation: true`
  - `Sunset: <rfc-8594-date>` once approved
  - `Link: </v1/profile>; rel="successor-version"`
  - `ETag: "<current-profile-version>"`
- Legacy concurrency behavior:
  - if a caller supplies `If-Match`, honor it and return `412` on stale writes
  - if a caller omits `If-Match`, keep temporary last-write-wins behavior for backward compatibility
  - document explicitly that the legacy endpoint is not concurrency-safe unless `If-Match` is supplied
- Legacy retry behavior:
  - if a caller supplies `Idempotency-Key`, honor it using the same dedup rules as `PATCH`
  - do not make `Idempotency-Key` mandatory on the old `POST` during coexistence; that would be breaking for unmigrated clients
- Legacy error behavior:
  - deterministic `application/problem+json` is guaranteed for `GET /v1/profile` and `PATCH /v1/profile`
  - do not silently replace the legacy error envelope unless existing consumers are verified tolerant of the change
- Removal gate:
  - published deprecation window
  - telemetry shows legacy traffic is at or near zero
  - supported web and mobile clients are migrated

### Compatibility classification

- Add `GET /v1/profile`: additive.
- Add `PATCH /v1/profile`: additive.
- Mark `POST /v1/profile/update` deprecated while preserving its wire contract: additive.
- Make `If-Match` or `Idempotency-Key` mandatory on the old `POST`: breaking.
- Remove `POST /v1/profile/update`: breaking.

### Spec artifacts to update

- Canonical API decision artifact:
  - `specs/<feature-id>/30-api-contract.md`
  - or, if the feature uses the repository's single-file spec format, the API section of `specs/<feature-id>/spec.md`
- Wire-contract source of truth:
  - `api/openapi/service.yaml`
  - add `GET /v1/profile` and `PATCH /v1/profile`
  - keep `POST /v1/profile/update` documented as `deprecated: true` during coexistence
  - add or extend `CustomerProfile`, `CustomerProfilePatch`, `Problem`, shared headers (`ETag`, `If-Match`, `If-None-Match`, `Idempotency-Key`), and shared responses for `304`, `412`, `422`, `428`, and `429`
  - declare `bearerAuth` on the new operations
  - declare `application/merge-patch+json` on `PATCH`
- If split spec artifacts are used, sync only the affected deltas:
  - `55-reliability-and-resilience.md`: retry class, idempotency retention, `412`/`428` behavior
  - `70-test-plan.md`: merge-patch semantics, deterministic error ordering, legacy coexistence, `304`/`412`/`422`/`428`
  - `80-open-questions.md`: editable/nullability list, legacy sunset date, profile-existence assumption
  - `90-signoff.md`: deprecation exit criteria

### Adjacent handoffs

- No mandatory adjacent handoff is required for this contract as written.
- Reopen to `go-domain-invariant-spec` if profile editing includes workflow rules beyond field validation, such as verified-contact state transitions.
- Reopen to `go-security-spec` if the surface expands beyond self-service editing by the authenticated subject.

## Open Questions, Risks, And Reopen Conditions

### Open questions

- Which fields are editable, which are read-only, and which are nullable/clearable?
- Is contact-email editing in scope here, or handled by a separate verified change flow?
- Is a profile resource guaranteed to exist for every authenticated subject, or is `404` a real outcome?
- What sunset date is acceptable for `POST /v1/profile/update`?

### Risks

- Keeping the legacy endpoint writable without mandatory `If-Match` leaves a temporary lost-update risk for unmigrated clients.
- Mixed canonical and legacy error contracts during rollout increase support complexity, but they preserve backward compatibility.
- If the service cannot emit a strong opaque `ETag`, the conditional-write contract is not release-ready.

### Reopen conditions

- Profile writes become asynchronous or require moderation before commit.
- Reads are served from a lagging projection or cache and cannot provide strong read-after-write semantics.
- The surface expands from self-service editing to admin or cross-customer editing.
- Product requires fine-grained array operations or bulk profile updates.

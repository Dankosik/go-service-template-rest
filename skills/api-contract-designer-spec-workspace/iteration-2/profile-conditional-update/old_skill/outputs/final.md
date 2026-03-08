# Customer Profile Edit API Contract

## Decision Summary

- Canonical resource: model the editable customer profile as the authenticated caller's singleton resource at `GET /v1/profile` and `PATCH /v1/profile`.
- Write method: use `PATCH`, not `PUT`, because product requires partial updates and omitted fields must not imply deletion or reset.
- Patch media type: `application/merge-patch+json`.
- Concurrency: `GET /v1/profile` returns an opaque `ETag`; `PATCH /v1/profile` requires `If-Match`.
- Retry behavior: `PATCH /v1/profile` is retry-safe by contract only when the caller also supplies `Idempotency-Key`.
- Error model: use a stable `application/problem+json` profile with deterministic field-level validation errors.
- Rollout: add the new resource endpoints without removing `POST /v1/profile/update`; keep the old action endpoint as a deprecated compatibility bridge until the deprecation window closes.

## Selected And Rejected Options

### Selected

- `PATCH /v1/profile` with `application/merge-patch+json`
  - Why: it matches the partial-update requirement, keeps the contract resource-oriented, and avoids forcing mobile or web clients to send the full profile document on every edit.
- `/v1/profile` as a singleton resource scoped by authenticated identity
  - Why: first-party clients are editing their own profile; the server should derive ownership from auth context instead of trusting a caller-supplied profile or customer ID.
- Required conditional write with `If-Match`
  - Why: concurrent edits already happen, so lost-update prevention must be part of the contract instead of a best-effort runtime detail.

### Rejected

- `PUT /v1/profile`
  - Rejected because full replacement conflicts with the stated need for partial updates and makes omitted-field semantics dangerous during concurrent edits.
- `PATCH /v1/profile` with `application/json-patch+json`
  - Rejected because operation-list patch documents add unnecessary client complexity for routine profile editing and couple clients too tightly to document structure.
- `/v1/me/profile`
  - Rejected because it is only a stylistic alias over authenticated identity and does not improve resource clarity enough to justify another path convention.
- `/v1/customers/{customer_id}/profile`
  - Rejected for the primary web/mobile contract because it exposes an identity parameter that first-party self-service clients should not have to send.
- Last-write-wins `PATCH` with no required precondition
  - Rejected because it does not protect concurrent edits and would quietly preserve the main flaw of the old action endpoint.

## Resource And Endpoint Matrix

| Endpoint | Purpose | Success | Required request conditions | Retry / concurrency | Compatibility class |
| --- | --- | --- | --- | --- | --- |
| `GET /v1/profile` | Read the authenticated customer's current profile | `200 OK`, `304 Not Modified` when `If-None-Match` matches | Auth required | Safe and idempotent by protocol. Returns `ETag`. | Additive |
| `PATCH /v1/profile` | Partially update editable profile fields | `200 OK` with the full updated profile and current `ETag` | Auth required, `Content-Type: application/merge-patch+json`, `If-Match`, `Idempotency-Key` | Atomic per resource. Missing precondition is explicit. Stale precondition fails distinctly. | Additive |
| `POST /v1/profile/update` | Deprecated compatibility endpoint for unmigrated clients | Preserve current success status and body during rollout | Existing auth rules stay intact | Temporary bridge only. No new canonical guarantees unless caller opts into new headers. | Additive now, eventual removal is breaking |

## Request, Response, And Error Model

### Canonical Resource Representation

- Response schema name in OpenAPI: `CustomerProfile`.
- Request patch schema name in OpenAPI: `CustomerProfilePatch`.
- `CustomerProfilePatch` must contain only editable fields and must reject undeclared properties.
- Mutability should be explicit in schema terms:
  - editable fields are present in `CustomerProfilePatch`
  - immutable or server-managed fields stay out of `CustomerProfilePatch` and remain `readOnly` in `CustomerProfile`

### `GET /v1/profile`

- Purpose: fetch the current profile for the authenticated subject.
- Request headers:
  - `If-None-Match` optional.
- Response headers:
  - `ETag: "<opaque-version>"`
  - `Cache-Control` policy may be set by the service, but must not weaken the read-after-write guarantee for the authenticated caller.
- Response behavior:
  - `200 OK` returns the full `CustomerProfile` representation.
  - `304 Not Modified` returns no body when `If-None-Match` matches the current entity tag.
- Consistency contract:
  - this endpoint is strong-consistency for the caller's own profile
  - after a successful `PATCH`, a subsequent `GET /v1/profile` by the same subject must reflect the new state and the new `ETag`

### `PATCH /v1/profile`

- Required headers:
  - `Content-Type: application/merge-patch+json`
  - `If-Match: "<etag-from-latest-read>"`
  - `Idempotency-Key: <opaque-client-generated-key>`
- Patch semantics:
  - omitted field: no change
  - present non-null field: replace that field value
  - present object field: merge by object member, per JSON Merge Patch semantics
  - present array field: replace the whole array
  - present `null`: clear the field only if that field is explicitly nullable and clearable in `CustomerProfilePatch`; otherwise fail validation
  - unknown field: fail validation
  - immutable field: fail validation
  - patch application is atomic for the full profile resource
- Success response:
  - `200 OK`
  - body: full updated `CustomerProfile` representation
  - headers: current `ETag`
- No-op semantics:
  - if the patch is valid but makes no effective change after normalization, return `200 OK` with the current representation and current `ETag`
  - no-op patches do not require a synthetic version bump

### Preconditions, Concurrency, And Retry Semantics

- `ETag` must be opaque to clients.
- `If-Match` is mandatory on `PATCH`.
  - missing `If-Match` -> `428 Precondition Required`
  - stale or mismatched `If-Match` -> `412 Precondition Failed`
- `Idempotency-Key` is mandatory on `PATCH`.
  - missing `Idempotency-Key` -> `428 Precondition Required`
  - same key plus same authenticated subject, route, and normalized payload -> return an equivalent outcome
  - same key plus different payload -> `409 Conflict`
- Idempotency retention target: `24h`.
- Successful writes always return the current `ETag`.

### Status Code Mapping

| Status | When it applies |
| --- | --- |
| `200 OK` | Successful `GET` or `PATCH` |
| `304 Not Modified` | `GET` with matching `If-None-Match` |
| `400 Bad Request` | Malformed JSON, invalid syntax, trailing tokens, or otherwise unreadable request framing |
| `401 Unauthorized` | Missing or invalid authentication |
| `403 Forbidden` | Authenticated caller is not allowed to edit this profile |
| `404 Not Found` | No profile resource exists for the authenticated subject |
| `409 Conflict` | Idempotency key reuse with different payload, or another business-state conflict unrelated to a stale `ETag` |
| `412 Precondition Failed` | `If-Match` does not match the current profile `ETag` |
| `415 Unsupported Media Type` | Request is not `application/merge-patch+json` |
| `422 Unprocessable Content` | Deterministic semantic validation failure |
| `428 Precondition Required` | Missing required `If-Match` or missing required `Idempotency-Key` |
| `429 Too Many Requests` | Rate limit exceeded; include `Retry-After` when enforced |

### Stable Error Model

- New-contract failures use `Content-Type: application/problem+json`.
- Base schema: extend the repository's existing `Problem` schema rather than introducing a competing envelope.
- Required top-level fields:
  - `type`
  - `title`
  - `status`
  - `detail`
- Stable extensions:
  - `code`
  - `request_id`
  - `errors`
- `errors` item shape:
  - `path`: JSON Pointer to the offending field
  - `code`: stable machine-readable validation code
  - `message`: human-readable explanation
- Deterministic validation rules:
  - the same invalid payload must always map to the same HTTP status and the same top-level `code`
  - return all detected validation failures, not only the first one
  - sort `errors` deterministically by `path`, then by `code`
  - do not silently ignore unknown or immutable fields
  - do not leak stack traces, SQL text, internal topology, or secrets

## Boundary And Cross-Cutting Policies

- Auth context, not caller-supplied identity, determines which profile is being read or changed.
- Operation security in OpenAPI should require `bearerAuth` for both `GET /v1/profile` and `PATCH /v1/profile`.
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
  6. conditional-write check
  7. business update
- Rate limiting, if enforced, must use `429` and should provide `Retry-After`.
- Request correlation should remain visible through `request_id` in problem responses.

## Consistency And Async Notes

- This contract is synchronous and strong-consistency by design.
- No operation resource is needed.
- If the underlying implementation cannot provide read-after-write consistency for `GET /v1/profile` after `PATCH /v1/profile`, that is a contract mismatch and should be resolved before release rather than hidden as a runtime detail.
- If future profile fields need long-running processing or non-merge semantics, model them as separate sub-resources or explicit async operations instead of weakening this endpoint's semantics.

## Compatibility Notes

### Canonical New Surface

- All new web and mobile integrations should use:
  - `GET /v1/profile`
  - `PATCH /v1/profile`

### Legacy `POST /v1/profile/update` Coexistence

- Keep `POST /v1/profile/update` during the migration window as a deprecated compatibility endpoint.
- Preserve its current request and success response shape during that window.
- Add migration-friendly response headers on the legacy endpoint where safe:
  - current `ETag`
  - successor `Link` header pointing to `/v1/profile`
  - `Sunset` once a retirement date is approved
- Legacy endpoint behavior during rollout:
  - route overlapping business logic through the same validation rules as the new endpoint where possible
  - if a legacy caller supplies `If-Match`, honor it
  - if a legacy caller supplies `Idempotency-Key`, honor it
  - if a legacy caller omits those headers, keep legacy behavior only as a temporary compatibility exception and document that it does not provide the new lost-update guarantee
- Do not make `If-Match` or `Idempotency-Key` mandatory on `POST /v1/profile/update` in the same rollout; that would be a breaking change for unmigrated clients.

### Change Classification

- Add `GET /v1/profile`: additive
- Add `PATCH /v1/profile`: additive
- Extend `Problem` with stable field-error details: additive if existing consumers tolerate new properties
- Mark `POST /v1/profile/update` deprecated: additive if request and success response compatibility are preserved
- Require `If-Match` or `Idempotency-Key` on the old `POST`: breaking
- Remove `POST /v1/profile/update`: breaking

## Spec Artifact Updates

### Required Contract Artifacts

- `api/openapi/service.yaml`
  - Add `GET /v1/profile` and `PATCH /v1/profile`.
  - Keep `POST /v1/profile/update` documented as deprecated during coexistence.
  - Add or extend components for:
    - `CustomerProfile`
    - `CustomerProfilePatch`
    - `Problem.errors`
    - shared headers for `ETag`, `If-Match`, `If-None-Match`, `Idempotency-Key`
    - shared responses for `412`, `422`, `428`, and `429`
  - Mark new profile operations with `bearerAuth`.
  - Declare `application/merge-patch+json` on `PATCH`.

### Generated / Runtime Contract Artifacts

- `internal/api/openapi.gen.go`
  - Regenerate after the OpenAPI contract changes.
- `internal/infra/http/openapi_contract_test.go`
  - Extend runtime contract coverage for:
    - `GET /v1/profile`
    - `PATCH /v1/profile`
    - problem content type on profile failures
    - precondition and validation status mappings

## Open Questions And Risks

- This contract assumes one editable profile per authenticated subject. If a subject can own multiple editable profiles, the URI must introduce an explicit resource identifier instead of using `/v1/profile`.
- This contract assumes profile edits can complete synchronously with strong read-after-write semantics. If writes fan out to eventually consistent systems, the API contract must disclose that explicitly before release.
- JSON Merge Patch replaces arrays as whole values. If product needs element-level append/remove semantics for complex collections, those collections should become separate sub-resources rather than overloading generic profile patching.
- The exact legacy error envelope for `POST /v1/profile/update` is not specified here. If existing clients depend on it, keep that envelope during rollout even while aligning validation rules underneath.

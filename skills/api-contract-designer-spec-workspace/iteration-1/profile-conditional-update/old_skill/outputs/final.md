# Customer Profile Edit API Contract

## Decision Summary

- Resource shape: model the editable customer profile as a singleton resource owned by the authenticated subject at `/v1/profile`.
- Write method: use `PATCH /v1/profile`, not `PUT`, because clients need partial updates and omitted fields must not imply deletion or reset.
- Patch format: use `application/merge-patch+json`.
- Concurrency: `GET /v1/profile` returns an opaque `ETag`; `PATCH /v1/profile` requires `If-Match`.
- Retry behavior: `PATCH /v1/profile` is retry-safe by contract only when the caller also supplies `Idempotency-Key`.
- Error model: use one stable `application/problem+json` profile with deterministic field-level validation errors.
- Compatibility: introduce the new resource endpoints additively and keep `POST /v1/profile/update` as a deprecated compatibility endpoint during rollout. Removing the old endpoint later is a breaking change.

## Selected And Rejected Options

### Selected

- `PATCH /v1/profile` with JSON Merge Patch.
  - Why: partial edits are a core requirement, JSON Merge Patch is the default partial-update format, and it keeps clients focused on resource fields instead of operation lists.

### Rejected

- `PUT /v1/profile`
  - Rejected because full replacement would force web/mobile clients to send the entire profile representation and would make omitted-field semantics dangerous during concurrent edits.
- `PATCH /v1/profile` with `application/json-patch+json`
  - Rejected because operation-style patch documents add unnecessary client complexity for ordinary field editing and couple clients to document structure too tightly.
- `/v1/customers/{customer_id}/profile`
  - Rejected for the primary client contract because profile ownership is derived from authenticated identity; first-party web/mobile clients should not have to send their own customer identifier.

## Resource And Endpoint Matrix

| Endpoint | Purpose | Success | Retry / concurrency | Compatibility class |
| --- | --- | --- | --- | --- |
| `GET /v1/profile` | Read the authenticated customer profile | `200 OK`, `304 Not Modified` when `If-None-Match` matches | Safe and idempotent by protocol. Returns `ETag`. | Additive |
| `PATCH /v1/profile` | Partially update editable profile fields | `200 OK` with the full updated profile representation and new `ETag` | Requires `If-Match` and `Idempotency-Key`. Atomic per resource. | Additive |
| `POST /v1/profile/update` | Deprecated compatibility surface for old clients only | Keep existing success schema during rollout | Legacy bridge; not the long-term contract surface | Additive now, eventual removal is breaking |

## Request, Response, And Error Model

### `GET /v1/profile`

- Response headers:
  - `ETag: "<opaque-version>"`
- Conditional read:
  - `If-None-Match` supported.
  - Matching validator returns `304 Not Modified`.
- Response body:
  - The canonical current profile representation for the authenticated customer.
  - The exact field schema belongs in OpenAPI; this contract decision does not invent new profile fields.

### `PATCH /v1/profile`

- Required headers:
  - `Content-Type: application/merge-patch+json`
  - `If-Match: "<etag-from-latest-read>"`
  - `Idempotency-Key: <opaque-client-generated-key>`
- Patch semantics:
  - Omitted field: no change.
  - Present non-null field: replace that field value.
  - Present field with `null`: clear the field only if that field is explicitly nullable / clearable in the schema; otherwise fail validation.
  - Arrays are replaced as whole-array values when present.
  - Unknown fields fail validation.
  - Immutable fields fail validation.
  - Patch application is atomic for the whole profile resource.
- Success response:
  - `200 OK`
  - Body: full updated profile representation.
  - Headers: new `ETag` for the updated version.

### Status Code Mapping

| Status | When it applies |
| --- | --- |
| `200 OK` | Successful `GET` or `PATCH` |
| `304 Not Modified` | `GET` with matching `If-None-Match` |
| `400 Bad Request` | Malformed JSON, invalid JSON syntax, trailing tokens, or structurally unreadable payload |
| `401 Unauthorized` | Missing or invalid authentication |
| `403 Forbidden` | Authenticated caller is not allowed to edit this profile |
| `404 Not Found` | No profile resource exists for the authenticated subject |
| `409 Conflict` | Business-state conflict unrelated to `ETag` mismatch |
| `412 Precondition Failed` | `If-Match` does not match the current profile `ETag` |
| `415 Unsupported Media Type` | Request is not `application/merge-patch+json` |
| `422 Unprocessable Content` | Deterministic validation failure: unknown field, immutable field, type/format/range violation, or illegal `null` clear |
| `428 Precondition Required` | Missing required `If-Match` or missing required `Idempotency-Key` |
| `429 Too Many Requests` | Rate limit hit; include `Retry-After` when enforced |

### Stable Error Model

- All new-contract errors use `Content-Type: application/problem+json`.
- Required top-level fields:
  - `type`
  - `title`
  - `status`
  - `detail`
- Stable extensions:
  - `code`
  - `request_id`
  - `errors` for field-level validation failures
- `errors` item shape:
  - `path`: JSON Pointer to the offending field
  - `code`: stable machine-readable validation code
  - `message`: human-readable explanation
- Determinism rules for validation failures:
  - return all detected validation errors, not just the first one
  - order `errors` deterministically by `path`, then by `code`
  - keep `code` values stable across web and mobile clients
  - do not include stack traces, SQL text, internal topology, or secrets

## Boundary And Cross-Cutting Policies

- Resource ownership comes from authenticated identity, not from a caller-supplied customer ID in the URI.
- Strict JSON behavior:
  - reject malformed payloads
  - reject trailing tokens
  - reject unknown fields
- Validation pipeline at the contract boundary:
  - content-type and basic transport checks
  - strict JSON decode
  - normalization
  - semantic validation
  - business update
- `PATCH` requests without both required write preconditions are invalid:
  - `If-Match` prevents lost updates
  - `Idempotency-Key` makes mobile/web retries safe after ambiguous network failures
- Idempotency behavior:
  - dedup scope must include authenticated subject, method/route, and request payload
  - same key plus same payload returns an equivalent outcome
  - same key plus different payload returns `409 Conflict`
  - default dedup retention target: `24h`

## Consistency And Async Notes

- This contract should be exposed as a strong-consistency resource boundary:
  - after a successful `PATCH`, a subsequent `GET /v1/profile` for the same subject must reflect the new state and new `ETag`
- No async operation resource is needed for this edit flow.
  - If implementation cannot reliably complete within normal synchronous request latency, the contract must be redesigned explicitly instead of hiding async work behind `200`.

## Compatibility Notes

### New Stable Contract

- First-party web/mobile clients should move to:
  - `GET /v1/profile`
  - `PATCH /v1/profile`

### Legacy Compatibility Endpoint

- Keep `POST /v1/profile/update` during the migration window as a deprecated compatibility endpoint.
- Mark it deprecated in OpenAPI and expose deprecation metadata:
  - `Deprecation: true`
  - `Sunset: <date>`
  - `Link: </v1/profile>; rel="successor-version"`
- Compatibility rules:
  - preserve the existing success payload shape for old clients during the grace period
  - route legacy writes through the same validation and domain-update behavior as the new endpoint
  - honor `If-Match` if a migrated caller supplies it
  - if old clients cannot yet supply `If-Match`, keep current legacy write behavior only as a temporary exception and document that it does not provide the new lost-update guarantee

### Change Classification

- Add `GET /v1/profile`: additive
- Add `PATCH /v1/profile`: additive
- Mark `POST /v1/profile/update` deprecated: additive if response compatibility is preserved
- Tighten old `POST` semantics to require `If-Match`: breaking for unmigrated callers
- Remove old `POST /v1/profile/update`: breaking

## Spec Artifact Updates

### Required

- `specs/<feature-id>/30-api-contract.md`
  - Canonical decision record for resource shape, `PATCH` choice, preconditions, error model, and compatibility classification.
- `api/openapi/service.yaml`
  - Add `GET /v1/profile` and `PATCH /v1/profile`.
  - Model required headers: `If-Match`, `If-None-Match`, `Idempotency-Key`.
  - Add the shared `application/problem+json` schema and field-error extension.
  - Mark `POST /v1/profile/update` as deprecated and document successor link / sunset metadata.

### Update If Present In The Feature Spec Set

- `70-test-plan.md`
  - Add contract-level cases for stale `ETag`, missing preconditions, idempotency replay, deterministic validation ordering, and legacy/new endpoint parity during rollout.
- rollout or migration section (`90-rollout-and-migration.md` or equivalent)
  - Record the coexistence window, deprecation signal, and explicit rule for when the old endpoint can be removed.

## Open Questions And Risks

- Confirm whether the authenticated subject always has exactly one profile. If not, the URI may need an explicit resource identifier.
- Confirm whether current legacy clients can send `If-Match` and `Idempotency-Key`; if they cannot, the temporary legacy exception must be time-boxed and treated as a known concurrency risk.
- Confirm whether any profile fields need custom patch semantics beyond ordinary merge-patch replacement; if yes, those fields should be modeled as separate sub-resources instead of overloading generic patch behavior.

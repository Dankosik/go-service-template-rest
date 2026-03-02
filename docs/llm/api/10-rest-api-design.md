# REST API design instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Designing or changing REST/JSON API contracts
  - Adding or reviewing HTTP endpoints, URI structures, query parameters, and status codes
  - Defining pagination, filtering, partial/full updates, bulk operations, or long-running operations
  - Defining idempotency, retries, optimistic concurrency, ETags, and preconditions
  - Standardizing API error model and consistency semantics for clients
- Do not load when: The task is purely internal implementation detail with no API contract change

## Purpose
- This document defines repository defaults for production REST APIs in Go services.
- The goal is predictable API behavior, safe retries, and reviewable contract evolution.
- Defaults in this document are mandatory unless an ADR explicitly approves an exception.

## Baseline assumptions
- External API style: REST over HTTP with JSON payloads.
- Contract source of truth: OpenAPI.
- API versioning default: major version in URI prefix (`/v1`, `/v2`).
- Error media type: `application/problem+json` (RFC 9457 style).
- Response media type default: `application/json; charset=utf-8`.

## Required inputs before generating endpoints
Resolve these first. If missing, apply defaults and state assumptions.

- API audience: external/public or internal-only HTTP clients
- Consistency model per endpoint: strong read-after-write or eventual
- Retry expectations: can clients auto-retry this operation or not
- Concurrency conflict policy: last-write-wins or precondition required
- Pagination requirement: cursor pagination only or explicit ADR for offset
- Bulk semantics: synchronous bounded batch or async operation resource

## Resource modeling and URI conventions

### Resource modeling defaults
- Model business resources, not RPC actions.
- Use collection and item resources as primary shape:
  - collection: `/v1/orders`
  - item: `/v1/orders/{order_id}`
  - sub-collection only when ownership is real: `/v1/customers/{customer_id}/orders`
- Prefer flat resource space plus filters over deeply nested paths.
- Keep URI depth small (default: max 2 resource levels after version prefix).

### URI conventions
- URI prefix MUST start with API major version (`/v1`).
- Path segments MUST be lowercase and kebab-case.
- Collection names MUST be plural nouns.
- Path parameters MUST be opaque identifiers (`{order_id}`, not `{db_pk}`).
- Do not include verbs in URI (`/createOrder`, `/orders/delete` are invalid).
- Do not leak DB or infrastructure details in URI (`/tables/orders`, `/shards/7/orders` are invalid).
- Do not encode transport quirks into domain URIs (`/orders?rpc=1` is invalid).

### Decision rules: resource vs action
Use this order.

1. If operation is CRUD on a domain entity, model as collection/item resource.
2. If operation triggers long or bulk processing, model a new operation/job resource.
3. Use action-like endpoint only if resource modeling is impossible, and only with ADR plus explicit idempotency/retry contract.

## HTTP method semantics and status codes

### Method defaults
- `GET`: read item/collection, safe and idempotent.
- `POST`: create resource in collection or create operation/job resource.
- `PUT`: full replacement of target item, idempotent by contract.
- `PATCH`: partial update of target item.
- `DELETE`: delete target item, idempotent by contract.

### Status code defaults
- `200 OK`: successful read/update with response body.
- `201 Created`: successful create with `Location` header.
- `202 Accepted`: accepted async/long-running request, not completed yet.
- `204 No Content`: successful operation without response body (often `DELETE`, sometimes `PUT`/`PATCH`).
- `304 Not Modified`: conditional `GET` with matching `If-None-Match`.
- `400 Bad Request`: malformed request or invalid query syntax.
- `401 Unauthorized`: missing/invalid authentication.
- `403 Forbidden`: authenticated but not allowed.
- `404 Not Found`: resource does not exist.
- `409 Conflict`: business/domain conflict or idempotency payload mismatch.
- `412 Precondition Failed`: `If-Match` / precondition does not hold.
- `415 Unsupported Media Type`: unsupported `Content-Type`.
- `422 Unprocessable Content`: syntactically valid, semantically invalid payload.
- `428 Precondition Required`: precondition is mandatory but missing.
- `429 Too Many Requests`: rate limit exceeded.
- `500 Internal Server Error`: unexpected server failure.
- `503 Service Unavailable`: temporary unavailability.
- `504 Gateway Timeout`: upstream timeout in dependency chain.

## PUT and PATCH semantics

### PUT defaults
- `PUT` means full replacement of resource representation.
- Missing mutable fields in payload are treated as explicit replacement behavior, not implicit "keep old value".
- `PUT` MUST be idempotent: same request repeated produces equivalent final state.
- Default for non-existing target on `PUT`: `404 Not Found`.
- `PUT` upsert is allowed only if explicitly documented in contract (never implicit).

### PATCH defaults
- Default patch media type: `application/merge-patch+json` (JSON Merge Patch).
- Accept `application/json-patch+json` only if operation-level patches are required and documented.
- `PATCH` MUST specify only mutable fields; unknown/immutable fields return `400` or `422`.
- `PATCH` atomicity is per resource: either full patch applied or none.
- If patch semantics are non-idempotent, mark endpoint retry-unsafe and require idempotency key for retried clients.

## Pagination, filtering, sorting, and field selection

### Pagination defaults
- Default pagination style: cursor/token pagination.
- Standard request fields: `page_size`, `page_token`.
- Standard response field: `next_page_token`.
- Default `page_size`: 50.
- Max `page_size`: 200.
- Sorting MUST be deterministic; add stable tiebreaker (`id`) when needed.
- Offset pagination (`limit` + `offset`) is exception-only and requires ADR (typically admin/reporting endpoints with bounded depth).

### Filtering defaults
- Filters use explicit query params on whitelist fields.
- Unknown filter params MUST fail with `400`.
- Type/format validation is mandatory for every filter.
- Default operator set should be explicit in docs (`eq`, `in`, range boundaries).
- Filtering semantics MUST be stable across releases inside same major version.

### Sorting defaults
- Use `sort` query parameter.
- Prefix `-` means descending order (for example `sort=-created_at`).
- Unsupported sort fields MUST fail with `400`.

### Field selection defaults
- Optional sparse response can use `fields` query parameter.
- `fields` MUST be whitelist-validated.
- Sensitive/internal fields must never be selectable.

## Bulk operations

### Default strategy
- For large or expensive bulk writes, create an async bulk job resource.
- Recommended pattern:
  - `POST /v1/order-import-jobs`
  - `202 Accepted`
  - `Location: /v1/order-import-jobs/{job_id}`

### Synchronous bulk (exception path)
- Allow only for bounded payload size (default max 100 items).
- Contract MUST define one mode explicitly:
  - all-or-nothing
  - per-item partial success with per-item status payload
- Do not hide partial failures behind a single success flag.
- Bulk write endpoints MUST document retry and idempotency semantics.

## Idempotency, retries, and concurrency control

### Retry classification (mandatory per endpoint)
- Retry-safe by HTTP semantics: typically `GET`, `HEAD`.
- Retry-safe by contract: `PUT`, `DELETE`, or explicitly designed `POST`/`PATCH`.
- Retry-unsafe: operations with non-idempotent side effects unless protected by idempotency key.

### Idempotency key defaults
- Retry-unsafe endpoints that clients may retry MUST support `Idempotency-Key`.
- Default deduplication TTL: 24h.
- Key scope MUST include tenant/account + route + operation.
- Same key + same payload => return equivalent result.
- Same key + different payload => `409 Conflict`.
- Idempotency decision and retention policy MUST be documented in OpenAPI.

### ETag and optimistic concurrency defaults
- Mutable resources SHOULD return `ETag` on `GET`.
- Conditional `GET` with `If-None-Match` SHOULD support `304 Not Modified`.
- Concurrent updates SHOULD use `If-Match` with current ETag.
- If precondition is required but missing, return `428 Precondition Required`.
- If `If-Match` fails, return `412 Precondition Failed`.
- Successful write SHOULD return updated `ETag`.

## Long-running operations and async acknowledgement

### When to switch to async
Use async acknowledgement when any condition is true:
- Expected processing time is usually above 2 seconds
- Execution time is highly variable
- Operation fans out to multiple downstream systems
- Caller does not require immediate finalized state

### Async acknowledgement pattern
- Initial response: `202 Accepted`.
- Include `Location` header with operation status resource URI.
- Optionally include `Retry-After` for polling guidance.
- Operation resource SHOULD expose:
  - `id`
  - `status` (`pending`, `running`, `succeeded`, `failed`, `canceled`)
  - `created_at`, `updated_at`
  - `result` reference on success
  - structured `error` on failure
- On completion, either:
  - return final state from operation resource, or
  - return/indicate canonical result URI and allow `303 See Other`.

## Error model consistency

### Default error format
- Use one HTTP error format only: `application/problem+json`.
- Required fields:
  - `type`
  - `title`
  - `status`
  - `detail`
  - `instance` (when available)
- Optional extensions (stable across API):
  - `code` (domain error code)
  - `request_id`
  - `errors` (field-level validation details)

### Domain-to-HTTP mapping defaults
- Validation failure -> `400` or `422` (choose one per API and keep consistent)
- Auth missing/invalid -> `401`
- Permission denied -> `403`
- Not found -> `404`
- Precondition/version conflict -> `412` or `409` by semantics
- Rate limit -> `429`
- Upstream timeout -> `504`
- Upstream unavailable -> `503`

### Error model rules
- Never return `200` with embedded error payload.
- Never return stack traces, SQL text, or internal topology to clients.
- Error response shape must stay stable across endpoints.

## Eventual consistency disclosure

### Contract disclosure rules
- Every endpoint MUST declare consistency model: `strong` or `eventual`.
- For eventual endpoints, docs MUST include staleness expectation (for example, target propagation window at p95).
- Write endpoints that materialize asynchronously MUST use async acknowledgement pattern, not fake immediate success.
- Read endpoints over eventually consistent projections SHOULD provide freshness metadata (`as_of`, `last_updated_at`) where feasible.

### Behavioral rules
- Do not claim read-after-write guarantee if not provided.
- Do not silently change consistency behavior in same major version.
- If consistency model changes, treat as behavioral contract change and require explicit review decision.

## REST anti-patterns (review blockers)
Treat each item as a blocker unless an ADR explicitly accepts the risk.

- Action-heavy endpoints:
  - bad: `/v1/orders/create`, `/v1/payments/execute`
  - required fix: model resource or operation resource
- Ambiguous semantics:
  - bad: `PUT` used for partial update, `POST /resources/{id}` for create without contract
  - required fix: strict method semantics and explicit status mapping
- Unsafe retries:
  - bad: client retries `POST` blindly, server creates duplicates
  - required fix: retry classification + idempotency key contract
- Leaky transport details:
  - bad: URI/fields expose table names, shard IDs, internal queue names
  - required fix: domain-oriented contract, hide infrastructure internals
- Non-deterministic list behavior:
  - bad: pagination without stable ordering
  - required fix: deterministic sort and stable cursor contract
- Error-shape drift:
  - bad: each endpoint returns custom error JSON
  - required fix: single problem details profile

## MUST / SHOULD / NEVER

### MUST
- MUST use major-version URI prefix (`/v1`).
- MUST model resources with noun-based URIs and consistent HTTP semantics.
- MUST define status codes, retries, idempotency, and consistency per endpoint.
- MUST enforce cursor pagination defaults for new list/search endpoints.
- MUST use one error model (`application/problem+json`) across entire API surface.
- MUST document async behavior via `202` + operation/status resource.

### SHOULD
- SHOULD return ETags for mutable resources and support conditional requests.
- SHOULD require `If-Match` on high-contention updates.
- SHOULD expose bounded filtering/sorting with strict validation.
- SHOULD use async bulk job resources instead of large synchronous bulk writes.
- SHOULD include freshness metadata for eventually consistent read models.

### NEVER
- NEVER encode verbs/actions as default endpoint style.
- NEVER change semantics of methods/status codes per endpoint "by convenience".
- NEVER allow retry-unsafe endpoints without explicit idempotency policy.
- NEVER leak internal implementation details through public contract.
- NEVER return success status for failed operation.

## Review checklist
Before approving REST contract or endpoint changes, verify:

- Resource model is domain-oriented and URI naming follows conventions
- Version prefix and major-version policy are respected
- Method semantics (`GET/POST/PUT/PATCH/DELETE`) are unambiguous
- Status code usage is correct and complete for success/error paths
- `PUT`/`PATCH` semantics are explicit and tested for edge cases
- Pagination/filter/sort rules are deterministic and bounded
- Bulk operation semantics (atomicity, limits, partial failures) are explicit
- Retry classification exists for every endpoint
- Idempotency key behavior (scope, TTL, conflict handling) is documented
- ETag/precondition behavior is implemented where concurrency matters
- Long-running flows use `202` + operation resource pattern
- Error format is consistent with `application/problem+json`
- Eventual consistency model and staleness disclosure are present in docs
- No anti-patterns from blocker list are introduced

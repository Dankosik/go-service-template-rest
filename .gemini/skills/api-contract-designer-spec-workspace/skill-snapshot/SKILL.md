---
name: api-contract-designer-spec
description: "Design clear REST API contracts for Go services: resource modeling, HTTP semantics, errors, idempotency, concurrency, async behavior, consistency disclosure, and compatibility-safe evolution."
---

# API Contract Designer Spec

## Purpose
Define or review API behavior so clients and implementers have one clear contract for resources, methods, statuses, errors, retries, concurrency, and consistency.

## Scope
- design resource model, URI shape, and versioned endpoint structure
- define HTTP method semantics and status-code mapping
- define request, response, and error contracts
- define pagination, filtering, sorting, and sparse-field rules
- define idempotency, retry classification, and conflict behavior
- define optimistic concurrency and precondition behavior
- define async and long-running operation semantics
- define consistency and freshness disclosure at the API boundary
- define cross-cutting boundary behavior such as validation, limits, auth context, correlation, and rate limiting
- classify contract changes as additive, behavior-changing, or breaking

## Boundaries
Do not:
- redesign service decomposition, storage topology, or distributed orchestration as the primary output
- prescribe low-level handler, middleware, or client implementation details as the main result
- treat observability, security, or reliability operations as the primary domain unless they are exposed at the API boundary
- push contract-important behavior into “runtime details” that clients are still expected to depend on

## Core Defaults
- Default external style is REST over HTTP with JSON payloads and OpenAPI as the contract source of truth.
- API major version belongs in the URI prefix.
- Error format default is `application/problem+json`.
- Cursor pagination is the default; offset pagination is exception-only.
- Long-running or variable-latency work should use explicit async acknowledgment with an operation resource.
- Cross-cutting behavior should be explicit at the contract boundary, not inferred from implementation.

## Expertise

### Resource Modeling And URI Semantics
- Model business resources, not RPC actions.
- Use collection and item shape by default.
- Use sub-collections only when ownership is real.
- Keep URI depth small and stable.
- Use lowercase kebab-case path segments and plural collection nouns.
- Use opaque identifiers; do not leak DB or topology details into URIs.
- Avoid embedding implementation detail such as table names, shard IDs, or queue names.

### HTTP Method And Status Semantics
- `GET` is safe and idempotent.
- `POST` creates a collection resource or an operation resource.
- `PUT` is full replacement and idempotent by contract.
- `PATCH` is partial update and must define patch semantics.
- `DELETE` is idempotent by contract.
- Make success and failure mappings explicit across the full surface.
- Never return a success status with an embedded error payload.

### Update Semantics
- `PUT` means full replacement; omitted mutable fields are not implicitly preserved.
- Default `PUT` on a missing target is `404`; upsert is exception-only and must be explicit.
- Default patch media type is `application/merge-patch+json`.
- `application/json-patch+json` is justified only when operation-style patch semantics are really required.
- Unknown or immutable fields in patch input should fail consistently.
- Patch application should be atomic per resource.

### Query Semantics: Pagination, Filtering, Sorting, Sparse Fields
- Sorting should be deterministic and include a stable tie-breaker when needed.
- Default `page_size` is `50`; default max is `200` unless a different contract is justified.
- Filters are whitelist-based; unknown filters should fail.
- Filter and sort field types must be validated at the contract level.
- Sorting syntax uses `sort`, with descending order via `-field`.
- Sparse field selection is whitelist-only; sensitive or internal fields are never selectable.

### Error Model And Disclosure Discipline
- Use one stable Problem Details profile across the API surface.
- Required fields: `type`, `title`, `status`, `detail`; `instance` when available.
- Optional stable extensions may include `code`, `request_id`, and field-level `errors`.
- Keep the choice between `400` and `422` consistent and documented once.
- Error payloads must be sanitized: no stack traces, SQL text, secrets, or infrastructure topology.

### Retry Classification And Idempotency
- Classify every endpoint as retry-safe by protocol, retry-safe by contract, or retry-unsafe.
- Retry-unsafe operations that may be retried by clients should require `Idempotency-Key`.
- Default idempotency dedup TTL: `24h`.
- Key scope should include tenant or account, operation, and route or method.
- Same key with same payload returns equivalent outcome.
- Same key with different payload returns conflict.
- Missing required idempotency key should map to an explicit precondition-style failure.

### Concurrency And Preconditions
- Mutable resources should expose `ETag` on reads when concurrent updates matter.
- Conditional reads with `If-None-Match` should support `304`.
- High-contention writes should require `If-Match`.
- Missing required precondition should fail explicitly.
- Failed precondition should fail distinctly from general conflict.
- Successful writes should return the updated `ETag` when concurrency control is used.

### Async And Long-Running Operations
- Async contract is mandatory when duration is often greater than a couple of seconds, fan-out exists, or completion time is highly variable.
- Start endpoint should return `202 Accepted` plus an operation-status location.
- Operation resources should define:
  - `id`
  - `status`
  - `created_at`
  - `updated_at`
  - success result reference
  - structured failure details
- Default status enum: `pending`, `running`, `succeeded`, `failed`, `canceled`.
- Poll guidance may include `Retry-After`.
- Never hide async side effects behind fake synchronous success.

### Consistency And Freshness Disclosure
- Each endpoint should declare whether behavior is `strong` or `eventual`.
- Eventual endpoints should disclose expected propagation or freshness behavior.
- Read models that converge asynchronously should expose freshness fields such as `as_of` or `last_updated_at`.
- Do not claim read-after-write guarantees unless they are explicitly provided.
- A consistency-model change inside the same major version is a behavior change and requires explicit treatment.

### Cross-Cutting Boundary Contracts
- Keep validation, normalization, limits, auth context, idempotency, correlation, rate-limit behavior, async behavior, and webhook behavior explicit at the boundary.
- Make boundary pipeline order explicit:
  - transport limits
  - strict decode
  - normalization
  - semantic validation
  - business logic
- Strict JSON defaults:
  - reject unknown fields
  - reject trailing tokens
  - fail malformed payloads consistently
- Default input limits should be explicit and reviewable.
- Tenant context should come from validated identity, not arbitrary caller headers.
- Correlation should support trace context and a stable request ID.
- Rate-limit behavior should define `429` semantics and `Retry-After` guidance.

### File Uploads And Webhooks
- Upload contracts should define media type, size/type limits, sync vs async processing, and publish-after-scan behavior when scanning is required.
- Large upload default should be upload-session or presigned flow rather than huge direct multipart.
- Webhooks and callbacks should assume at-least-once delivery and tolerate duplicates or out-of-order delivery.
- Webhook contracts should define signature verification, replay window, retry schedule, dedup key, and sender timeout expectations.

### Distributed, Data, And Cache Implications At The API Boundary
- Do not encode hidden global ACID assumptions in API semantics.
- Expose long cross-service processes as process or operation state, not fake immediate completion.
- Classify behavior as immediate local invariant vs convergent cross-service process.
- Keep contract changes rollout-safe for mixed-version deployments.
- For cache-accelerated reads, disclose staleness and fallback behavior instead of treating cache expiry as a correctness mechanism.

### Observability Contract
- Make request and operation correlation observable through the contract where appropriate.
- Use low-cardinality route templates for logs, metrics, and spans.
- Async surfaces should preserve a stable correlation identity through retries and DLQ transitions.
- Telemetry-related contract fields must stay bounded and reviewable.

### Compatibility And Evolution
- Preserve compatibility by default.
- Classify each change as:
  - `additive`
  - `behavior-change`
  - `breaking`
- Give stronger scrutiny to semantics changes than to payload growth.
- Do not quietly change error mapping, consistency behavior, or retry semantics inside a supposedly stable contract.

## Decision Quality Bar
Major API recommendations should make the following explicit:
- the client-facing problem
- at least two viable options when the decision is nontrivial
- selected and rejected options
- compatibility class
- method, status, and error semantics
- retry, idempotency, and concurrency behavior
- consistency and freshness behavior
- fail-path semantics and open risks

## Deliverable Shape
Return API work in a compact, reviewable form:
- `Resource And Endpoint Matrix`
- `Request, Response, And Error Model`
- `Boundary And Cross-Cutting Policies`
- `Consistency And Async Notes`
- `Compatibility Notes`
- `Open Questions And Risks`

## Escalate When
Escalate if:
- resource ownership or client audience is still ambiguous
- mutating behavior lacks retry, idempotency, or concurrency rules
- async or eventual-consistency behavior is hidden behind synchronous-looking APIs
- boundary behavior such as validation, auth context, limits, or rate limiting is unspecified
- a breaking change cannot be versioned, communicated, or justified cleanly

---
name: api-contract-designer-spec
description: "Design API-contract-first specifications for Go services. Use when planning or revising client-visible REST API behavior before coding and you need explicit resource modeling, HTTP method/status semantics, request/response/error contracts, pagination/filter semantics, idempotency/retry/concurrency rules, async behavior, consistency disclosure, and compatibility-safe evolution. Skip when the task is local code implementation, service decomposition, chi router topology, SQL schema/migration design, or low-level observability/security operations tuning."
---

# API Contract Designer Spec

## Purpose
Turn product and behavior changes into one client-visible API contract that is explicit enough for OpenAPI, implementation, tests, and rollout to converge without semantic drift.

## Scope
Use this skill to define or review REST API behavior before coding:
- resource and URI model
- HTTP methods, statuses, and mutation semantics
- request, response, and error contracts
- pagination, filtering, sorting, sparse-field, and bulk-result semantics
- retry classification, idempotency, and optimistic-concurrency rules
- async or long-running operation behavior
- consistency and freshness disclosure at the API boundary
- boundary-visible validation, limits, auth-context, correlation, and rate-limit semantics
- compatibility-safe evolution and deprecation strategy

## Boundaries
Do not:
- redesign service decomposition, storage topology, or distributed orchestration as the primary output
- take ownership of chi router topology, SQL schema and migration design, or worker runtime wiring as the main result
- prescribe low-level handler, middleware, repository, or client implementation as the deliverable
- push client-visible behavior into “implementation details later”

## Escalate When
Escalate if resource ownership, client audience, consistency model, retry expectations, or rollout compatibility cannot be made explicit, or if API-visible behavior depends on unresolved routing, security, distributed, or data/cache decisions.

## Core Defaults
- REST over HTTP with JSON payloads; OpenAPI is the wire-contract source of truth.
- Keep API major version in the URI prefix.
- Use `application/problem+json` as the default HTTP error model.
- Prefer resource or operation resources over action-RPC endpoints.
- Cursor pagination is the default; offset pagination is exception-only.
- Prefer honest async acknowledgement over fake synchronous success.
- Treat the prompt's stated client problem as the contract budget. Do not widen media types, enum values, flows, or control surfaces unless they remove a concrete ambiguity.
- Missing contract facts become explicit assumptions or blockers, not implementation guesses.
- Keep final API decisions in `30-api-contract.md` or the API section of `spec.md`.

## Expertise

### Contract Framing And Consumer Model
- Resolve affected clients, trust boundary, consistency expectations, retry behavior, and ownership assumptions first.
- Separate immediate acceptance from eventual completion. If write acknowledgement and final business state differ, model them separately.
- Distinguish actor identity, tenant scope, and business resource references. Auth-derived tenancy does not by itself remove legitimate business identifiers from the contract.
- Choose between CRUD resource, sub-resource, and operation/job resource based on client semantics, not internal handler shape.
- Do not introduce extra accepted media types, terminal statuses, or companion endpoints just for “completeness”; every addition needs a prompt-backed or repo-default-backed reason.
- For nontrivial contract questions, compare at least two viable options before selecting one.

### Resource Modeling And URI Semantics
- Model business resources, not RPC verbs.
- Use collection and item shape by default.
- Use sub-collections only when ownership is real.
- Keep URI depth small and stable.
- Use lowercase kebab-case path segments and plural collection nouns.
- Use opaque identifiers; do not leak DB, topology, or queue details into URIs.
- When bulk or long-running behavior exists, prefer a job or operation resource instead of overloading an existing item endpoint.

### HTTP Method, Status, And Mutation Semantics
- `GET` is safe and idempotent.
- `POST` creates a collection resource or starts an operation resource.
- `PUT` is full replacement and idempotent by contract.
- `PATCH` is partial update and must define patch semantics explicitly.
- `DELETE` is idempotent by contract.
- Use `201 Created` with `Location` for successful creates.
- Use `202 Accepted` when the work is accepted but not complete.
- Use `204 No Content` only when no response body is useful.
- Keep `409 Conflict`, `412 Precondition Failed`, and `428 Precondition Required` distinct.
- `PUT` on a missing target defaults to `404`; upsert is exception-only and must be explicit.
- Default patch media type is `application/merge-patch+json`.
- Unknown or immutable mutable-field writes should fail consistently.
- Endpoint matrices, examples, and detailed rules must agree. If idempotent replay, conditional success, or legacy coexistence changes the returned success status, surface it where clients scan first.
- Do not hide partial failure behind a generic success flag.

### Query, Pagination, And Representation Semantics
- Sorting must be deterministic and include a stable tie-breaker when needed.
- Default `page_size` is `50`; default max is `200` unless a different contract is justified.
- Filters are whitelist-based; unknown filters should fail.
- Filter and sort field types must be validated at the contract level.
- Sorting syntax uses `sort`, with descending order via `-field`.
- Sparse field selection is whitelist-only; sensitive or internal fields are never selectable.
- Multi-item outcomes must define whether they are all-or-nothing or per-item result shapes.

### Error Model And Negative-Path Semantics
- Use one stable Problem Details profile across the API surface.
- Required fields are `type`, `title`, `status`, and `detail`; include `instance` when available.
- Stable extensions may include `code`, `request_id`, and field-level `errors`.
- Choose the `400` vs `422` boundary once and keep it consistent.
- Make auth, media-type, payload-size, precondition, rate-limit, and dependency-failure mappings explicit.
- Error payloads must be sanitized: no stack traces, SQL text, secrets, or infrastructure topology.
- Never return a success status with an embedded error payload.

### Retry Classification, Idempotency, And Preconditions
- Classify every endpoint as retry-safe by protocol, retry-safe by contract, or retry-unsafe.
- Retry-unsafe operations that may be retried by clients should require `Idempotency-Key`.
- Default idempotency dedup TTL is `24h`.
- Key scope should include tenant or account, operation, and route or method.
- Same key with same payload returns equivalent outcome.
- Same key with different payload returns conflict.
- Mutable resources should expose `ETag` on reads when lost updates matter.
- Conditional reads with `If-None-Match` should support `304`.
- High-contention writes should require `If-Match`.
- Missing required preconditions should fail explicitly rather than collapsing into a generic conflict.
- Successful writes should return the updated `ETag` when concurrency control is used.

### Async, Bulk, Upload, And Webhook Contracts
- Async contract is mandatory when duration is often greater than a couple of seconds, fan-out exists, or completion time is highly variable.
- Start endpoints should return `202 Accepted` plus an operation-status location and may include `Retry-After`.
- Once the business request is accepted, keep one clear control-plane resource for the async lifecycle unless a second resource removes a concrete client ambiguity.
- Operation resources should define `id`, `status`, `created_at`, `updated_at`, success result reference, and structured failure details.
- Use only operation or job statuses the contract can actually reach. `pending`, `running`, `succeeded`, and `failed` are a common baseline; add `canceled` only when cancellation is a real client-visible path.
- For large uploads, prefer an upload-session or presigned flow rather than huge direct multipart requests.
- Do not broaden accepted upload media types beyond the prompt or established API policy just to be generous.
- Upload contracts should define media type, size limits, file-type rules, and publish-after-scan behavior when scanning is required.
- Bulk write contracts must choose all-or-nothing or per-item partial-success semantics explicitly.
- Webhooks and callbacks should assume at-least-once delivery, duplicates, and possible reordering.
- For partner-facing or cross-boundary callbacks, prefer pre-registered or ownership-verified callback targets over arbitrary per-request URLs unless the prompt explicitly makes caller-supplied URLs part of the contract.
- When webhooks and eventual read models coexist, include a monotonic version, freshness token, or equivalent reconciliation aid so clients can compare push delivery with lagging reads.
- Webhook contracts should define signature verification, replay window, retry schedule, dedup key, and sender timeout expectations.

### Consistency And Freshness Disclosure
- Each endpoint should declare whether behavior is `strong` or `eventual`.
- Eventual endpoints should disclose expected propagation or freshness behavior.
- Read models that converge asynchronously should expose freshness fields such as `as_of` or `last_updated_at` when feasible.
- Cache-backed reads are contract-visible when freshness can lag or fail over.
- Do not claim read-after-write guarantees unless they are explicitly provided.
- A consistency-model change inside the same major version is a behavior change and requires explicit treatment.

### API Boundary Contracts
- Keep validation, normalization, limits, auth context, tenant binding, idempotency, correlation, rate-limit behavior, and overload semantics explicit at the boundary.
- Make boundary pipeline order explicit: transport limits, strict decode, normalization, semantic validation, then business logic.
- Strict JSON defaults are reject unknown fields, reject trailing tokens, and fail malformed payloads consistently.
- Tenant context should come from validated identity, not arbitrary caller headers.
- Correlation should support trace context plus a stable request ID.
- Rate-limit behavior should define `429` semantics and `Retry-After` guidance when throttling is temporary.

### Artifact Alignment And Adjacent Handoffs
- Keep final API decisions in `30-api-contract.md` or the API section of `spec.md`; sync only affected deltas into `50`, `55`, `70`, `80`, and `90`.
- Keep artifact updates and adjacent handoffs concise and secondary unless the prompt explicitly asks for full spec-package propagation. The client-visible contract remains the main deliverable.
- If route topology, middleware order, `404/405/OPTIONS`, or CORS behavior becomes primary, hand off to `go-chi-spec`.
- If business acceptance rules or state-transition semantics are still unclear, hand off to `go-domain-invariant-spec`.
- If authn, authz, tenant isolation, or threat-class controls exceed API-surface scope, hand off to `go-security-spec`.
- If storage ownership, migrations, or cache correctness become primary, hand off to `go-data-architect-spec` or `go-db-cache-spec`.
- If completion semantics depend on cross-service orchestration or reconciliation, hand off to `go-distributed-architect-spec`.
- Adjacent skills inform the contract; they do not replace ownership of client-visible API semantics.

### Compatibility And Evolution
- Preserve compatibility by default.
- Classify each change as `additive`, `behavior-change`, or `breaking`.
- Give stronger scrutiny to status, error, retry, idempotency, precondition, async, and consistency changes than to payload growth alone.
- Keep contract changes rollout-safe for mixed-version deployments.
- Action-endpoint cleanup needs an explicit deprecation or coexistence strategy if clients already depend on it.
- Do not quietly change error mapping, consistency behavior, or retry semantics inside a supposedly stable contract.

## Decision Quality Bar
For every major API recommendation, include:
- the client-facing problem and affected audience
- at least two viable options when the decision is nontrivial
- the selected option and at least one explicit rejection reason
- endpoint, method, status, and error semantics
- retry, idempotency, and precondition behavior
- async, consistency, and freshness behavior
- any invented status, media type, or companion surface only when it has an explicit client-facing reason
- artifact updates and adjacent-skill handoffs
- compatibility class, assumptions, risks, and reopen conditions

## Deliverable Shape
Return API work in a compact, reviewable form:
- `Contract Framing And Assumptions`
- `Resource And Endpoint Matrix`
- `Request, Response, And Error Model`
- `Retry, Idempotency, And Concurrency Rules`
- `Async, Freshness, And Webhook Notes`
- `Compatibility, Artifact Updates, And Handoffs`
- `Open Questions, Risks, And Reopen Conditions`

## Escalate Or Reject
- action-like endpoints retained without a clear justification
- mutating surfaces that still lack retry, idempotency, or precondition semantics
- async or eventual-consistency behavior hidden behind synchronous-looking success
- boundary behavior such as validation, auth context, limits, or rate limiting left implicit
- compatibility impact or deprecation path omitted for an API-visible change
- critical contract ambiguity deferred to coding

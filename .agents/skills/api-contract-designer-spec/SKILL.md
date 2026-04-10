---
name: api-contract-designer-spec
description: "Design API-contract-first specifications for Go services. Use when planning or revising client-visible REST API behavior before coding and you need explicit resource modeling, HTTP method/status semantics, request/response/error contracts, pagination/filter semantics, idempotency/retry/concurrency rules, async behavior, consistency disclosure, and compatibility-safe evolution. Skip when the task is local code implementation, service decomposition, chi router topology, SQL schema/migration design, or low-level observability/security operations tuning."
---

# API Contract Designer Spec

## Purpose
Turn product and behavior changes into one client-visible API contract that is explicit enough for OpenAPI, implementation, tests, and rollout to converge without semantic drift.

## Specialist Stance
- Treat the API as a client-visible compatibility contract, not a handler sketch.
- Trace every nontrivial recommendation through request parsing, response shape, error semantics, retry behavior, and client migration impact.
- Prefer the smallest contract surface that solves the caller problem and keeps future compatibility honest.
- Hand off routing, storage, security, distributed completion, and worker-runtime decisions when they become the primary owner of the hard question.

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
- REST over HTTP with JSON payloads; OpenAPI must mirror the approved wire contract rather than outrank it.
- Keep API major version in the URI prefix.
- Use `application/problem+json` as the default HTTP error model.
- Prefer resource or operation resources over action-RPC endpoints.
- Cursor pagination is the default; offset pagination is exception-only.
- Prefer honest async acknowledgement over fake synchronous success.
- Treat the prompt's stated client problem as the contract budget. Do not widen media types, enum values, flows, or control surfaces unless they remove a concrete ambiguity.
- Missing contract facts become explicit assumptions or blockers, not implementation guesses.

## Reference Files
Use reference files lazily. Read only the file that matches the contract symptom in front of you, then synthesize the guidance into the task's `spec.md` or API decision output. Do not paste reference examples wholesale into deliverables.

| Task symptom | Read |
| --- | --- |
| Error payloads, Problem Details, validation errors, auth/concealment status policy, field-level errors, sanitized negative paths | `references/problem-details-errors.md` |
| HTTP method selection, `PUT` vs `PATCH`, create/update/delete status codes, `201`/`202`/`204`, `Location`, `ETag`, content negotiation status codes | `references/http-method-status-semantics.md` |
| List endpoints, cursor or offset pagination, filters, sort syntax, sparse fields, `total_count`, collection links, multi-item result semantics | `references/pagination-filtering-sorting.md` |
| Write retries, `Idempotency-Key`, timeout recovery, idempotency TTL/scope, `If-Match`, `If-None-Match`, `409` vs `412` vs `428` | `references/idempotency-preconditions-retries.md` |
| Long-running work, `202 Accepted`, operation resources, polling, retention, bulk operations, callbacks, webhooks, event deduplication | `references/async-operations-and-webhooks.md` |
| Compatibility class, status/error behavior changes, pagination changes, enum/nullability/default changes, URI versioning, Deprecation/Sunset, coexistence | `references/compatibility-and-versioning.md` |

If several symptoms apply, read the smallest set of references that covers the contract risk. Keep the final recommendation contract-first and hand off chi routing, SQL schema, worker runtime, distributed orchestration, and implementation details to their owners.

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

### Representation And Lifecycle Semantics
- Define the canonical read representation, write-only inputs, read-only outputs, server-assigned fields, and whether omitted versus explicit `null` have different meaning.
- Distinguish accepted, persisted, externally observable, and terminal business state. If those moments differ, expose them separately instead of collapsing them into one ambiguous status.
- Define lifecycle states and legal transitions at the API boundary; if a state cannot be reached or observed by clients, do not put it in the public contract.
- Use stable encodings for identifiers, timestamps, money, percentages, and high-precision numbers. When rounding matters, do not default to floating-point JSON numbers unless the prompt or established API policy explicitly requires them.
- Default timestamps to RFC 3339 UTC and state timestamp precision when concurrency, ordering, or webhook reconciliation depends on it.
- Treat enum shape as contract surface. If future values are plausible, either state that clients must tolerate unknown values or avoid pretending the enum is closed.

### HTTP Method, Status, And Mutation Semantics
- `GET` is safe and idempotent.
- `POST` creates a collection resource or starts an operation resource.
- `PUT` is full replacement and idempotent by contract.
- `PATCH` is partial update and must define omitted versus `null` versus empty semantics, array replacement behavior, and whether writes to read-only fields fail or are ignored.
- `DELETE` is idempotent by contract.
- Use `201 Created` with `Location` for successful creates.
- Use `202 Accepted` when the work is accepted but not complete.
- Use `204 No Content` only when no response body is useful.
- Keep `409 Conflict`, `412 Precondition Failed`, and `428 Precondition Required` distinct.
- `PUT` on a missing target defaults to `404`; upsert is exception-only and must be explicit.
- Default patch media type is `application/merge-patch+json`.
- Unknown or immutable mutable-field writes should fail consistently.
- When multiple mutation surfaces can change the same resource during migration, define whether they share the same version source, `ETag` space, and stale-write behavior or are intentionally inconsistent during coexistence.
- Endpoint matrices, examples, and detailed rules must agree. If idempotent replay, conditional success, or legacy coexistence changes the returned success status, surface it where clients scan first.
- Do not hide partial failure behind a generic success flag.

### Query, Pagination, And Collection Semantics
- Sorting must be deterministic and include a stable tie-breaker when needed.
- Default `page_size` is `50`; default max is `200` unless a different contract is justified.
- Filters are whitelist-based; unknown filters should fail.
- Filter and sort field types must be validated at the contract level.
- Sorting syntax uses `sort`, with descending order via `-field`.
- Cursor contracts should state snapshot versus live-pagination behavior, duplicate or skip risk under concurrent writes, cursor expiry behavior, and the client recovery rule when a cursor is rejected.
- Sparse field selection is whitelist-only; sensitive or internal fields are never selectable.
- `total_count`, if exposed, must be identified as exact, approximate, delayed, or omitted by design rather than implied.
- Multi-item outcomes must define whether they are all-or-nothing or per-item result shapes.

### Error Model And Negative-Path Semantics
- Use one stable Problem Details profile across the API surface.
- Required fields are `type`, `title`, `status`, and `detail`; include `instance` when available.
- Stable extensions may include `code`, `request_id`, and field-level `errors`.
- Choose the `400` vs `422` boundary once and keep it consistent.
- Validation behavior should say whether unknown fields are rejected, whether one or all caller-fixable field errors are returned, and whether field-error ordering is deterministic.
- Make auth, media-type, payload-size, precondition, rate-limit, and dependency-failure mappings explicit.
- Surface `405`, `406`, `410`, and `415` explicitly when they are client-visible instead of collapsing them into generic `400` or `404` behavior.
- Differentiate caller-fixable rejection, state conflict, missing precondition, overload or transient dependency failure, and accepted-but-later-failed work.
- Choose a concealment policy for inaccessible resources once and keep it consistent. Cross-tenant or unauthorized lookups should not randomly alternate between `403` and `404`.
- Error payloads must be sanitized: no stack traces, SQL text, secrets, or infrastructure topology.
- Never return a success status with an embedded error payload.

### Retry Classification, Idempotency, And Preconditions
- Classify every endpoint as retry-safe by protocol, retry-safe by contract, or retry-unsafe.
- For every non-idempotent write, define the durable acceptance boundary: which outcomes mean no durable work exists, and which outcomes mean the client must poll, read, or replay a stored result.
- Retry-unsafe operations that may be retried by clients should require `Idempotency-Key`.
- Default idempotency dedup TTL is `24h`.
- Key scope should include tenant or account, operation, and route or method.
- Define payload comparison at the normalized contract level, not only at raw-byte level, when retries may differ in insignificant formatting.
- Same key with same payload returns equivalent outcome.
- Same key with different payload returns conflict.
- When same-key replay hits a stored terminal result, define whether the contract returns `200 OK`, `201 Created`, or `202 Accepted`; do not leave terminal replay semantics implicit.
- When replay returns a stored outcome, say whether `Location`, `ETag`, operation IDs, and advisory headers such as `Retry-After` are identical or merely equivalent.
- Distinguish failures that do not reserve the idempotency key from accepted attempts that do reserve it and later fail during async processing.
- Define post-TTL reuse behavior for expired idempotency keys when that choice affects duplicate-work risk.
- Mutable resources should expose `ETag` on reads when lost updates matter.
- Conditional reads with `If-None-Match` should support `304`.
- High-contention writes should require `If-Match`.
- Missing required preconditions should fail explicitly rather than collapsing into a generic conflict.
- Successful writes should return the updated `ETag` when concurrency control is used.

### Async, Bulk, Upload, And Webhook Contracts
- Async contract is mandatory when duration is often greater than a couple of seconds, fan-out exists, or completion time is highly variable.
- `202 Accepted` should mean the service accepted responsibility for durable processing, not merely that it attempted to enqueue work.
- Start endpoints should return `202 Accepted` plus an operation-status location and may include `Retry-After`.
- Once the business request is accepted, keep one clear control-plane resource for the async lifecycle unless a second resource removes a concrete client ambiguity.
- If returning both the operation resource and the authoritative business-resource reference reduces retry ambiguity, do that explicitly.
- Operation resources should define `id`, `status`, `created_at`, `updated_at`, success result reference, and structured failure details.
- Operation resources should define expiry or retention behavior and the fallback discovery path once the operation record ages out.
- Use only operation or job statuses the contract can actually reach. `pending`, `running`, `succeeded`, and `failed` are a common baseline; add `canceled` only when cancellation is a real client-visible path.
- For large uploads, prefer an upload-session or presigned flow rather than huge direct multipart requests.
- Do not broaden accepted upload media types beyond the prompt or established API policy just to be generous.
- Upload contracts should define media type, size limits, file-type rules, and publish-after-scan behavior when scanning is required.
- Bulk write contracts must choose all-or-nothing or per-item partial-success semantics explicitly.
- For partial-success bulk contracts, define request-level terminal status, accepted and rejected counters, per-item correlation keys, and how large failure sets are paged or downloaded.
- Webhooks and callbacks should assume at-least-once delivery, duplicates, and possible reordering.
- For partner-facing or cross-boundary callbacks, prefer pre-registered or ownership-verified callback targets over arbitrary per-request URLs unless the prompt explicitly makes caller-supplied URLs part of the contract.
- When webhooks and eventual read models coexist, include a monotonic version, freshness token, or equivalent reconciliation aid so clients can compare push delivery with lagging reads.
- Webhook contracts should define signature verification, replay window, retry schedule, dedup key, and sender timeout expectations.

### Consistency And Freshness Disclosure
- Each endpoint should declare whether behavior is `strong` or `eventual`.
- Eventual endpoints should disclose expected propagation or freshness behavior.
- Read models that converge asynchronously should expose freshness fields such as `as_of` or `last_updated_at` when feasible.
- Cache-backed reads are contract-visible when freshness can lag or fail over.
- If one read path is authoritative and another is a lagging projection, say which one clients should use for timeout recovery, reconciliation, and read-after-write expectations.
- Do not claim read-after-write guarantees unless they are explicitly provided.
- A consistency-model change inside the same major version is a behavior change and requires explicit treatment.

### API Boundary Contracts
- Keep validation, normalization, limits, auth context, tenant binding, idempotency, correlation, rate-limit behavior, and overload semantics explicit at the boundary.
- Make boundary pipeline order explicit: transport limits, strict decode, normalization, semantic validation, then business logic.
- Strict JSON defaults are reject unknown fields, reject trailing tokens, and fail malformed payloads consistently.
- Tenant context should come from validated identity, not arbitrary caller headers.
- Correlation should support trace context plus a stable request ID.
- Rate-limit behavior should define `429` semantics and `Retry-After` guidance when throttling is temporary.
- Admin, debug, or override controls should not piggyback on general client endpoints unless they are explicitly part of the public contract.

### Boundaries And Handoffs
- Own client-visible API semantics; do not turn this into chi routing, SQL schema, worker runtime, or service-decomposition design.
- When used inside a repository workflow, hand final API decisions back to the orchestrator's chosen decision artifact; this skill does not own artifact propagation.
- Recommend OpenAPI or generated-surface updates only when they follow from the contract decision, and do not claim they were updated without tool evidence.
- Hand off when routing, domain invariants, security, data/cache ownership, or distributed completion semantics become primary. Adjacent skills inform the contract; they do not replace ownership of client-visible API semantics.

### Compatibility And Evolution
- Preserve compatibility by default.
- Classify each change as `additive`, `behavior-change`, or `breaking`.
- Give stronger scrutiny to status, error, retry, idempotency, precondition, async, and consistency changes than to payload growth alone.
- Treat enum-value expansion, default-value changes, nullability tightening, cursor or sort-contract changes, and weaker freshness guarantees as compatibility decisions, not harmless cleanup.
- “Additive” is not automatically safe if common clients use closed-world validation, exhaustive enum switches, or strict response decoders.
- Keep contract changes rollout-safe for mixed-version deployments.
- Action-endpoint cleanup needs an explicit deprecation or coexistence strategy if clients already depend on it.
- When legacy and replacement endpoints coexist, define whether they share idempotency space, ETag or precondition behavior, and status-code semantics.
- When coexistence is temporary, define which surface is authoritative for validation rules, concurrency semantics, and deprecation milestones instead of letting old and new routes silently diverge.
- Use explicit deprecation signaling when relevant, such as documented timelines and standard headers like `Deprecation` or `Sunset`, rather than burying migration in prose only.
- Do not quietly change error mapping, consistency behavior, or retry semantics inside a supposedly stable contract.

## Decision Quality Bar
For every major API recommendation, include:
- the client-facing problem and affected audience
- at least two viable options when the decision is nontrivial
- the selected option and at least one explicit rejection reason
- endpoint, method, status, and error semantics
- representation shape, lifecycle states, and any authoritative-versus-projection read split
- retry, idempotency, and precondition behavior
- durable acceptance boundary and timeout-recovery semantics for non-idempotent writes
- async, consistency, and freshness behavior
- deterministic validation semantics and any legacy-surface coexistence rules when they affect callers
- if old and new surfaces coexist, a short comparison of which semantics are shared versus temporarily divergent
- any invented status, media type, or companion surface only when it has an explicit client-facing reason
- adjacent-skill handoffs when the contract depends on another seam
- compatibility class, assumptions, risks, and reopen conditions

## Deliverable Shape
Return API work in a compact, reviewable form:
- `Contract Framing And Assumptions`
- `Resource And Endpoint Matrix`
- `Request, Response, And Error Model`
- `Retry, Idempotency, And Concurrency Rules`
- `Async, Freshness, And Webhook Notes`
- `Compatibility, Legacy Coexistence, Artifact Updates, And Handoffs`
- `Open Questions, Risks, And Reopen Conditions`

## Escalate Or Reject
- action-like endpoints retained without a clear justification
- mutating surfaces that still lack retry, idempotency, or precondition semantics
- async or eventual-consistency behavior hidden behind synchronous-looking success
- boundary behavior such as validation, auth context, limits, or rate limiting left implicit
- compatibility impact or deprecation path omitted for an API-visible change
- critical contract ambiguity deferred to coding

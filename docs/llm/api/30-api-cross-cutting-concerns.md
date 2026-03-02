# API cross-cutting concerns instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Designing or changing request validation, normalization, and payload limits
  - Designing auth context propagation (identity, tenant, scopes), idempotency, retry semantics, and correlation IDs
  - Defining rate limiting, request size enforcement, and overload behavior
  - Designing file upload endpoints, webhooks, callbacks, and long-running operation semantics
  - Defining what must be implemented in middleware/interceptors vs contract (OpenAPI/proto)
- Do not load when: The task is a local implementation detail with no API boundary, no transport behavior, and no cross-cutting contract impact

## Purpose
- This document defines repository defaults for API cross-cutting behavior.
- Goal: consistent validation, security context, retries, limits, async behavior, and reviewability across all endpoints.
- Defaults in this document are mandatory unless an ADR explicitly approves an exception.

## Baseline assumptions
- External/public APIs: HTTP + JSON with OpenAPI as source of truth.
- Internal synchronous APIs: gRPC/Connect allowed; canonical gRPC status model applies.
- Trust boundary starts at edge entrypoint; internal traffic is not trusted by default.
- Cross-cutting behavior must be implemented in both places:
  - contract level (`OpenAPI`/`proto` + docs)
  - runtime level (HTTP middleware / gRPC interceptors)

## Required inputs before changing endpoint behavior
Resolve these first. If missing, apply defaults and state assumptions.

- Audience and trust boundary (external, partner, internal service)
- Identity model (`end-user`, `service`, or both)
- Tenant isolation requirement and tenant key source
- Retry expectations (client retries? proxy retries? both?)
- Payload profile (JSON vs multipart, expected sizes, upload volumes)
- Completion semantics (sync <= 2s or async/LRO)
- Webhook/callback requirement (sender, receiver, security mode)

## Concern-to-layer mapping (mandatory)
Every concern must be visible in both contract and runtime.

| Concern | Contract MUST define | Runtime MUST enforce |
|---|---|---|
| Validation | schema constraints, required fields, allowed enums/formats | strict decoding + schema/semantic validation before business logic |
| Normalization | which fields are normalized and how | deterministic normalization before validation/storage |
| Input limits | max sizes, media types, error statuses | header/body/query limits, streaming limits, bounded parsing |
| Auth context | security schemes, required claims/metadata | token/cert validation, principal extraction, tenant binding |
| Authorization boundary | auth requirements per operation | object-level checks at use-case boundary |
| Idempotency/retry | endpoint retry class, required key headers/fields, conflict semantics | dedup store, payload fingerprint, conflict handling, TTL |
| Correlation/tracing | request/trace headers and response echo policy | request ID generation, trace context propagation, log binding |
| Rate limits | 429/`RESOURCE_EXHAUSTED` semantics, retry headers | token-bucket/quota checks on tenant/principal route keys |
| Async semantics | `202` pattern, operation resource schema, polling/callback rules | durable enqueue/orchestration, status transitions, cancellation |
| Webhooks/callbacks | delivery model, signature scheme, retry schedule, dedup key | signature verification, replay window, dedup, retry worker |

## Request validation, normalization, and input size enforcement

### Enforcement pipeline (order is mandatory)
1. Enforce transport limits (headers, URI/query length, body bytes, media type).
2. Parse/decode with strict mode.
3. Normalize only fields explicitly declared as normalizable.
4. Validate schema constraints and semantic rules.
5. Call business logic only after steps 1-4 succeed.

### Validation defaults
- Validation MUST happen at API boundary before domain/use-case execution.
- HTTP JSON decoding MUST:
  - reject unknown fields by default,
  - reject trailing JSON tokens,
  - return `400` for malformed JSON.
- HTTP semantic validation failures MUST return:
  - `422` by default for semantically invalid but well-formed payload,
  - `400` for malformed syntax or unsupported query/filter shape.
- gRPC validation MUST use schema-level validators (for example, protovalidate) and map failures to `INVALID_ARGUMENT`.
- Request validation errors MUST use stable machine-readable shape:
  - HTTP: `application/problem+json` with `errors` field for per-field details,
  - gRPC: canonical status + structured details when needed.

### Normalization defaults
- Normalization MUST be deterministic and documented per field.
- Default normalization rules:
  - trim leading/trailing whitespace for human-entered text fields where whitespace is not meaningful,
  - apply Unicode NFC normalization for user text that participates in equality/search,
  - lowercase only fields explicitly defined as case-insensitive (for example, email if contract says case-insensitive).
- NEVER normalize security-sensitive or opaque values:
  - tokens, signatures, passwords, opaque IDs, idempotency keys.
- Normalization rules MUST be applied before validation that depends on canonical form.

### Input size enforcement defaults
- HTTP server MUST enforce explicit limits:
  - `MaxHeaderBytes`: `16 KiB` default,
  - max request URI (path + query): `4 KiB` default at edge/gateway,
  - JSON request body max: `1 MiB` default,
  - multipart direct upload max: `10 MiB` default.
- On over-limit:
  - body too large: `413 Content Too Large`,
  - headers too large: `431 Request Header Fields Too Large`,
  - URI too long: `414 URI Too Long`.
- Body limits MUST be enforced before decode (`http.MaxBytesReader` in HTTP handlers).
- For HTTP/1.1 edge-facing endpoints, requests with both `Transfer-Encoding` and `Content-Length` MUST be rejected (`400`) and connection closed.

### Decision rules
Use this order.

1. If payload is JSON and <= `1 MiB`, use strict JSON decode + schema validation.
2. If binary upload is needed and <= `10 MiB`, use `multipart/form-data`.
3. If upload likely exceeds `10 MiB` or is bursty, switch to upload-session/presigned URL flow.
4. If parser/memory pressure is possible, prefer streaming parser and avoid buffering full payload in memory.

## Identity/tenant context, idempotency keys, correlation IDs, and retry semantics

### Auth context defaults
- Every authenticated request MUST build one principal context object with at least:
  - `subject_id`
  - `subject_type` (`end_user` or `service`)
  - `tenant_id` (when multi-tenant)
  - `roles` and/or `scopes`
  - `auth_method` (`bearer`, `mtls`, etc.)
  - `request_id` and trace identifiers
- Authorization MUST be fail-closed (`deny by default`).
- Object-level authorization is mandatory for resource-by-ID access (ownership/tenant boundary checks).

### Identity source and trust rules
- HTTP identity default: `Authorization: Bearer <token>`.
- gRPC identity/correlation default: metadata fields (auth + tracing metadata).
- Service identity default for internal calls: mTLS workload identity where available.
- NEVER trust client-provided `X-User-Id` / `X-Tenant-Id` / role headers as source of truth unless issued by a trusted gateway and cryptographically protected.

### Tenant context propagation
- Tenant context MUST come from validated identity claims or signed internal credential, not arbitrary headers.
- If tenant is required and missing, reject request with:
  - HTTP: `401` or `403` by auth semantics,
  - gRPC: `UNAUTHENTICATED` or `PERMISSION_DENIED`.
- Cross-tenant access MUST be explicitly forbidden unless endpoint contract says otherwise.

### Correlation and tracing defaults
- Accept W3C trace context (`traceparent`) when present; generate new trace when absent.
- Generate/propagate request correlation ID (`X-Request-ID`) for every request.
- Return request ID in response headers and include it in error payload extensions (`request_id`) and logs.
- Correlation IDs are observability metadata only; NEVER use them as auth/authz input.

### Retry classification and idempotency defaults
- Every endpoint MUST be classified as:
  - retry-safe by protocol semantics,
  - retry-safe by contract,
  - retry-unsafe.
- Retry-unsafe operations that clients may retry MUST require idempotency key:
  - HTTP: `Idempotency-Key` header,
  - gRPC/Connect: `request_id` or `idempotency_key` field.
- Idempotency defaults:
  - dedup TTL: `24h`,
  - scope: `tenant_id + operation + route/method`,
  - same key + same payload => return equivalent result,
  - same key + different payload => conflict (`409` / `ABORTED`).
- Missing required idempotency key:
  - HTTP default: `428 Precondition Required`,
  - gRPC default: `FAILED_PRECONDITION`.
- Retry policy defaults for clients/inter-service calls:
  - no retry by default,
  - allow at most `1` retry for transient failures only (`408`, `429`, `502`, `503`, `504`, `UNAVAILABLE`, `RESOURCE_EXHAUSTED`, `DEADLINE_EXCEEDED`),
  - exponential backoff with jitter (`50ms` base, `250ms` max),
  - never retry validation/auth/authz/not-found/conflict failures.

## Rate limiting and overload semantics

### Defaults
- Exceeded rate/quota MUST return:
  - HTTP: `429 Too Many Requests`,
  - gRPC: `RESOURCE_EXHAUSTED`.
- `Retry-After` SHOULD be returned for temporary throttling windows.
- Optional rate headers may be exposed (`RateLimit-Limit`, `RateLimit-Remaining`, `RateLimit-Reset`) if documented.
- Limit key default is composite:
  - `tenant_id + principal_id(or service_id) + route`.
- Separate policies SHOULD exist for:
  - burst protection (short window),
  - sustained quota (long window),
  - expensive endpoints (custom weights).

### Decision rules
1. If overload is policy/quota-based, use `429`.
2. If dependency outage/unavailability causes rejection, use `503`.
3. If payload too large, use `413` (not `429`/`503`).

## File upload rules

### Contract requirements
- Contract MUST declare:
  - accepted media type (`multipart/form-data` or upload-session flow),
  - max file size,
  - allowed file types,
  - processing semantics (sync vs async scan/publish).
- Large-file path default: upload-session/presigned URL + finalize endpoint.

### Runtime requirements
- Always enforce HTTP body limit before multipart parsing.
- Stream file content; do not read entire file into memory.
- Generate storage filename on server side; do not trust client filename for storage path.
- Store outside webroot/public static path by default.
- Validate type using allowlist + content sniffing; do not trust only client `Content-Type`.
- If malware/content scan is required, publish file only after scan result (`pending -> approved/rejected`).

### Response semantics
- Immediate accepted processing can return `202` + operation resource.
- Validation/type/size errors return `400`/`413`/`415`/`422` by exact failure type.

## Webhooks and callback patterns

### Delivery semantics defaults
- Webhooks are `at-least-once` by default:
  - duplicates are possible,
  - out-of-order delivery is possible,
  - delays are possible.
- Event envelope SHOULD be standardized (CloudEvents recommended).
- Every event MUST have stable event ID for consumer deduplication.

### Security defaults
- Webhook payloads MUST be signed (for example HMAC over timestamp + raw body).
- Receiver MUST verify signature and enforce replay window (default `5 minutes`).
- Delivery MUST use HTTPS only.
- Secrets/tokens MUST NOT be included in payload body.

### Sender/receiver behavior
- Receiver should return `2xx` only after durable persistence/enqueue.
- Receiver `5xx` means retryable failure; `4xx` means non-retryable contract failure.
- Sender retries MUST use bounded exponential backoff + jitter and max delivery window (default `24h`).
- Delivery attempt metadata (`event_id`, `attempt`, `timestamp`) MUST be logged with correlation IDs.

### Callback-specific rules
- Callback URL registration MUST include ownership verification handshake.
- Callback auth MUST be explicit (signed webhook or mTLS or OAuth token).
- Callback endpoint SLA/timeout MUST be documented; default sender timeout `5s`.

## Long-running operations and async semantics

### When async is mandatory
Use async/LRO when any condition is true:
- expected processing time often exceeds `2s`,
- operation fans out to multiple dependencies,
- completion latency is highly variable,
- side effects are processed by background workers.

### Contract pattern defaults
- Start request returns `202 Accepted`.
- `Location` header points to operation status resource.
- Operation resource MUST include:
  - `id`
  - `status` (`pending`, `running`, `succeeded`, `failed`, `canceled`)
  - `created_at`, `updated_at`
  - `result` (or result URI) on success
  - structured `error` on failure
- Polling guidance SHOULD include `Retry-After`.
- Operation start endpoints MUST follow idempotency rules.

### Completion and consistency rules
- Do not return fake immediate success for async side effects.
- If resulting data is eventually consistent, contract MUST disclose expected staleness window.
- On completion, service may return final representation directly or `303` to canonical resource URI.

## Middleware/interceptors blueprint (mandatory baseline)

### HTTP middleware order (default)
1. Request ID + trace context extraction/generation
2. Transport guards (header/URI/body size limits, content-type checks, TE+CL rejection)
3. Authentication
4. Principal/tenant context injection
5. Rate limit/quota check
6. Idempotency pre-check (when required)
7. Handler-level decode/normalize/validate
8. Authorization (object-level in handler/use-case boundary)
9. Error mapping to Problem Details
10. Access logging/metrics/tracing finalization

### gRPC interceptor order (default)
1. Trace/correlation interceptor
2. AuthN interceptor
3. Principal/tenant context interceptor
4. Rate limit interceptor
5. Validation interceptor
6. Idempotency interceptor (for mutating methods)
7. AuthZ checks at service/use-case boundary
8. Error/status mapping interceptor
9. Metrics/logging interceptor

## Cross-cutting anti-patterns (review blockers)
Treat each item as a blocker unless an ADR explicitly accepts the risk.

- Validation inside business logic only, with no boundary validation
- Accepting unknown JSON fields silently for security-sensitive writes
- Missing body/header limits on public endpoints
- Trusting caller-supplied identity/tenant headers without cryptographic trust
- No object-level authorization for resource-by-ID endpoints
- Retry-unsafe `POST` without idempotency key policy
- Unbounded retries or retrying non-transient failures
- Rate limit implemented but no `429` semantics in contract
- File upload endpoint that trusts original filename/path or reads full file into memory
- Webhooks treated as exactly-once or ordered by default
- Async operations returning `200` as if finalized when work is still pending
- Missing correlation ID propagation in logs/errors/traces

## MUST / SHOULD / NEVER

### MUST
- MUST validate, normalize, and enforce input limits at API boundary before use-case execution.
- MUST define and enforce principal + tenant context for every authenticated request.
- MUST require and enforce idempotency keys for retry-unsafe retried operations.
- MUST classify retry semantics per endpoint and document them in contract.
- MUST return `429`/`RESOURCE_EXHAUSTED` for quota throttling and document retry guidance.
- MUST define explicit upload limits and secure file-handling pipeline.
- MUST define webhook delivery as at-least-once and require dedup by event ID.
- MUST use `202` + operation resource for long-running work.
- MUST implement cross-cutting behavior in both middleware/interceptors and contract artifacts.

### SHOULD
- SHOULD expose request ID in every error response and major log event.
- SHOULD standardize webhook/event payload envelope (CloudEvents recommended).
- SHOULD use separate limits for read/write and expensive routes.
- SHOULD expose staleness/freshness metadata for eventually consistent reads.
- SHOULD keep default limits conservative and override per endpoint only with explicit rationale.

### NEVER
- NEVER treat correlation/request IDs as identity or authorization input.
- NEVER trust raw user input for SQL fragments, file paths, callback URLs, or policy fields.
- NEVER accept infinite-timeout outbound calls in production request paths.
- NEVER return success status for operations that are only queued but not completed unless using explicit async contract semantics.
- NEVER leak tokens, secrets, or internal infrastructure details in client-facing errors.

## Review checklist
Before approving API changes with cross-cutting impact, verify:

- Validation exists at boundary and strict decoding behavior is explicit
- Normalization rules are documented and deterministic
- Header/URI/body/file limits are explicit and enforced
- Auth context includes subject + tenant and is derived from trusted credentials
- Object-level authorization exists for ID-based access paths
- Retry class exists for each endpoint and matches actual side effects
- Idempotency key requirements, scope, TTL, and conflict behavior are implemented and documented
- Correlation/trace propagation is implemented and visible in logs/errors
- Rate limit semantics (`429`/`RESOURCE_EXHAUSTED`, `Retry-After`) are documented and tested
- File upload contract includes media type, limits, storage safety, and scan semantics
- Webhook/callback contract includes signature, replay window, dedup key, retries, and timeout
- Long-running operations use `202` + operation resource with clear status transitions
- Middleware/interceptor chain includes all mandatory cross-cutting controls
- No blocker anti-patterns are introduced

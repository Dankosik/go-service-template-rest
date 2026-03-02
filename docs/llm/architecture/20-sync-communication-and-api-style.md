# Sync communication and API style instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Designing or changing synchronous communication between services
  - Choosing between REST/JSON, gRPC, and Connect
  - Defining timeout, deadline, retry, idempotency, error, and pagination behavior
  - Designing internal vs external API boundaries, gateway/BFF shape, and client ownership
- Do not load when: The task is a local implementation detail without network/API contract impact

## Purpose
- This document defines repository defaults for synchronous request-reply communication.
- The goal is predictable, reviewable behavior for reliability and API evolution.
- The defaults below are mandatory unless an ADR explicitly approves an exception.

## Transport and sync-hop decision rules
Use this order. Do not choose transport first.

### 1) Decide if a synchronous hop is needed
Do not add a synchronous hop by default. Use async patterns (event, queue, outbox, LRO) when any point is true:
- The operation is not in the user-facing critical path
- Expected duration is usually more than 2 seconds or highly variable
- The use case requires fan-out to multiple downstream services in one request
- The workflow can tolerate eventual consistency
- Failure of the downstream dependency should not block the caller immediately

If sync is still required, continue to transport selection.

### 2) Default transport selection
- External/public APIs: REST/JSON with OpenAPI contract
- Internal service-to-service APIs: gRPC with Protobuf contract
- Connect: choose when you need one Protobuf contract plus mixed client/protocol support (gRPC, gRPC-Web, Connect) or HTTP/1.1-friendly operation

### 3) Hard boundaries
- Do not expose internal service APIs directly to external clients.
- Publish external APIs through gateway and/or BFF.
- Do not mix REST and gRPC/Connect for the same use case without explicit ownership and contract boundaries.

## Synchronous call defaults

### Deadline and timeout policy
- Every outbound sync call MUST have an explicit deadline.
- Never use infinite timeouts (`0` timeout or no deadline).
- Propagate inbound context deadlines downstream.
- Default end-to-end budget for interactive APIs: 2500ms.
- Default per-hop deadline:
  - Read/query call: 300ms
  - Write/command call: 1000ms
  - Absolute per-hop cap: 2000ms
- Budget rule for downstream call:
  - `outbound_deadline = min(per-hop default, remaining_inbound_budget - 100ms)`
  - If remaining budget is less than 150ms, fail fast instead of calling downstream.
- Connect calls MUST set `connect-timeout-ms`.

### Retry policy
- Default is no retry.
- Retries are allowed only for retry-safe operations and transient failures.
- Transient failure examples:
  - HTTP: `408`, `429`, `502`, `503`, `504`
  - gRPC/Connect: `UNAVAILABLE`, `RESOURCE_EXHAUSTED`, `DEADLINE_EXCEEDED`
- Max retry attempts by default: 1 retry (2 total attempts).
- Backoff policy default: exponential backoff with jitter, 50ms base, 250ms max.
- Never retry on:
  - Validation or contract errors
  - Authentication or authorization failures
  - Not-found and business conflicts
  - Caller cancellation
- Retry behavior MUST be documented in the API contract and client docs.

### Idempotency policy
- Every sync endpoint/method MUST be classified as:
  - Retry-safe by protocol semantics
  - Retry-safe by contract
  - Retry-unsafe
- Retry-unsafe operations MUST implement idempotency keys:
  - HTTP: `Idempotency-Key` header
  - gRPC/Connect: `request_id` or `idempotency_key` field in request
- Default deduplication retention (TTL): 24h.
- Key scope MUST include tenant (or account), operation, and route/method.
- Same key + same payload returns equivalent result.
- Same key + different payload returns conflict:
  - HTTP: `409`
  - gRPC/Connect: `ABORTED`

### Error model defaults
- External HTTP errors: RFC 9457 `application/problem+json`.
- Internal gRPC/Connect errors: canonical gRPC status codes.
- Use one deterministic domain error mapping. Minimum required mapping:
  - Invalid input: `400` / `INVALID_ARGUMENT`
  - Unauthenticated: `401` / `UNAUTHENTICATED`
  - Forbidden: `403` / `PERMISSION_DENIED`
  - Not found: `404` / `NOT_FOUND`
  - Conflict/precondition: `409` / `FAILED_PRECONDITION` or `ABORTED`
  - Rate limit: `429` / `RESOURCE_EXHAUSTED`
  - Upstream timeout: `504` / `DEADLINE_EXCEEDED`
  - Upstream unavailable: `503` / `UNAVAILABLE`
- Never return `200 OK` or gRPC success with error payload semantics.
- Never leak stack traces, SQL errors, or internal topology in client-facing messages.

### Pagination defaults for sync list endpoints
- New list/search endpoints MUST use token (cursor) pagination.
- Standard fields:
  - Request: `page_size`, `page_token`
  - Response: `next_page_token`
- Default `page_size`: 50.
- Max `page_size`: 200.
- Sorting MUST be deterministic and documented.
- Offset pagination is not a default. Allow only by explicit ADR (for admin/reporting use cases).

## Internal vs external contracts, gateway, BFF, and ownership

### Contract boundaries
- Internal contracts:
  - Source of truth: Protobuf (or explicitly approved internal OpenAPI)
  - Versioned and reviewed with compatibility checks
- External contracts:
  - Source of truth: OpenAPI
  - Stable deprecation and versioning policy
- Do not reuse internal contracts as public contracts without explicit adaptation layer.

### Gateway defaults
- Gateway owns edge concerns:
  - Authentication entry checks
  - Rate limiting and request size controls
  - TLS termination and edge security policies
  - Request ID propagation
- Gateway does not own domain business rules.

### BFF defaults
- Use BFF when client-specific aggregation/shape is required (web/mobile/partner differences).
- BFF is owned by the corresponding client platform team.
- BFF must not become a second domain service:
  - No core domain invariants in BFF
  - No direct access to other service databases
  - No hidden write orchestration without explicit contract

### Client ownership
- Every externally consumed API MUST declare:
  - Owner team
  - Intended client classes
  - Backward-compatibility window and deprecation policy
- Every internal API MUST declare:
  - Producer team
  - Consumer teams/services
  - SLO and timeout contract
- No unowned shared API surfaces.

## Anti-patterns
Treat each item as a review blocker unless an ADR explicitly accepts the risk.

- Chatty services: one user request causes many fine-grained sync calls (N+1 over network)
- Unknown retry semantics: no explicit retry-safe/unsafe classification
- No deadlines: outbound calls without explicit timeout/deadline
- Cascading synchronous chains: long dependency chains where one failure propagates widely
- Direct external access to internal services without gateway/BFF control
- Multiple incompatible error formats in one API surface
- Unbounded list endpoints without enforced pagination limits
- Sync calls for naturally asynchronous workflows (batch processing, long-running jobs)

## MUST / SHOULD / NEVER

### MUST
- MUST decide first if sync is needed before selecting transport.
- MUST use explicit deadlines for every outbound sync call.
- MUST document retry and idempotency semantics per endpoint/method.
- MUST use one error model per surface (HTTP Problem Details, gRPC canonical codes).
- MUST enforce token-based pagination defaults for list endpoints.
- MUST define owner teams for each API contract and client group.
- MUST route external traffic through gateway/BFF, not directly to domain services.

### SHOULD
- SHOULD keep synchronous chains short (target: max 2 downstream hops in critical path).
- SHOULD fail fast when remaining deadline budget is too small.
- SHOULD use async request-reply/LRO for long operations.
- SHOULD keep gateway and BFF free of domain business invariants.
- SHOULD enforce contract checks (lint/breaking) in CI for internal and external API specs.

### NEVER
- NEVER add retries to retry-unsafe operations without idempotency keys.
- NEVER use infinite timeout defaults.
- NEVER return success responses with embedded error payloads.
- NEVER expose internal transport contracts as public APIs without an explicit boundary layer.
- NEVER accept ambiguous ownership of API clients and contracts.

## Review checklist
Before approving API or sync communication changes, verify:
- Sync vs async decision is explicit and justified
- Transport choice (REST vs gRPC/Connect) follows defaults or has ADR
- End-to-end deadline budget and per-hop deadlines are documented
- Retry policy is explicit, bounded, and aligned with idempotency class
- Idempotency behavior includes key scope, TTL, and conflict handling
- Error mapping is deterministic and consistent across endpoints
- Pagination follows token defaults with documented limits
- External exposure path uses gateway/BFF and has clear ownership
- No chatty call graphs or cascading sync chains are introduced
- No blocked anti-patterns are present without approved exceptions

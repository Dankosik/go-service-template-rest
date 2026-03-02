---
name: api-contract-designer-spec
description: "Design API-contract-first specifications for Go services in a spec-first workflow. Use when planning or revising REST API behavior before coding and you need clear resource modeling, HTTP semantics, error model, idempotency/retry/concurrency rules, and cross-cutting API contract requirements. Skip when the task is local code implementation, pure architecture decomposition, SQL schema/migration design, CI/container setup, or detailed observability/security operations tuning."
---

# API Contract Designer Spec

## Purpose
Create a clear, reviewable API specification package before implementation. Success means API behavior is explicit, consistent, and directly translatable into implementation and tests without contract ambiguities.

## API Contract Hard Skills

### Mission
- Define API behavior so implementation and tests have no contract-level ambiguity.
- Preserve client compatibility by default and classify each change as additive, behavior-changing, or breaking.
- Express reliability, consistency, and security semantics at the API boundary, not as hidden runtime assumptions.

### Default posture
- External API default: REST over HTTP with JSON payloads and OpenAPI as the contract source of truth.
- API major version MUST be in URI prefix (`/v1`, `/v2`).
- Error format default is `application/problem+json`; response format default is `application/json; charset=utf-8`.
- Cursor pagination is default (`page_size`, `page_token`, `next_page_token`); offset pagination is exception-only.
- Async acknowledgement (`202` + operation resource) is default for long-running or variable-latency operations.
- Cross-cutting behavior MUST be defined in contract artifacts and expected in runtime enforcement.

### Resource modeling and URI semantics
- Model business resources, not RPC actions.
- Use collection/item shape by default (`/v1/orders`, `/v1/orders/{order_id}`).
- Use sub-collections only when ownership is real (`/v1/customers/{customer_id}/orders`).
- Keep URI depth small (default max two resource levels after version prefix).
- Use lowercase kebab-case path segments and plural collection nouns.
- Use opaque path identifiers (`{order_id}`), never DB/internal topology tokens.
- Never leak implementation details in contract URIs (table names, shard IDs, queue names).

### HTTP method and status semantics
- `GET` is safe and idempotent for reads.
- `POST` creates a collection resource or operation/job resource.
- `PUT` is full replacement and idempotent by contract.
- `PATCH` is partial update and MUST define patch document semantics.
- `DELETE` is idempotent by contract.
- Status mapping MUST be explicit for all success and fail paths (`200`, `201`, `202`, `204`, `304`, `400`, `401`, `403`, `404`, `409`, `412`, `415`, `422`, `428`, `429`, `500`, `503`, `504`).
- Never return success status with embedded error payload.

### Update semantics (`PUT` and `PATCH`)
- `PUT` means full replacement; omitted mutable fields are not implicitly preserved.
- Default `PUT` on missing target is `404`; upsert is allowed only if explicitly documented.
- `PATCH` default media type is `application/merge-patch+json`.
- `application/json-patch+json` is allowed only for explicitly justified operation-level patch semantics.
- Unknown/immutable fields in patch input MUST fail (`400` or `422`, consistently).
- Patch application MUST be atomic per resource.

### Query semantics: pagination, filtering, sorting, sparse fields
- Sorting MUST be deterministic; add stable tiebreaker when needed.
- Default `page_size` is `50`; max `page_size` is `200` unless explicitly overridden.
- Filters are whitelist-based; unknown filters MUST fail with `400`.
- Filter and sort field types/formats MUST be validated at contract level.
- Sorting syntax uses `sort`, descending uses `-field`.
- Sparse field selection (`fields`) is whitelist-only; sensitive/internal fields are never selectable.

### Error model and disclosure discipline
- Use one stable Problem Details profile across the entire API surface.
- Required fields: `type`, `title`, `status`, `detail`; `instance` when available.
- Optional stable extensions: `code`, `request_id`, `errors` (field-level details).
- Validation status choice (`400` vs `422`) MUST be consistent across API surface and documented once.
- Error payloads MUST be sanitized: no stack traces, SQL text, secrets, or infrastructure topology.

### Retry classification and idempotency policy
- Every endpoint MUST be classified as retry-safe by protocol, retry-safe by contract, or retry-unsafe.
- Retry-unsafe operations that may be retried by clients MUST require `Idempotency-Key`.
- Idempotency default dedup TTL is `24h`.
- Idempotency key scope MUST include tenant/account + operation + route/method.
- Same key with same payload returns equivalent result.
- Same key with different payload returns conflict (`409`).
- Missing required idempotency key SHOULD map to `428 Precondition Required`.

### Concurrency and preconditions
- Mutable resources SHOULD provide `ETag` on reads.
- Conditional reads with `If-None-Match` SHOULD support `304`.
- High-contention writes SHOULD require `If-Match`.
- Missing required precondition MUST return `428`.
- Failed precondition MUST return `412`.
- Successful writes SHOULD return updated `ETag`.

### Async/LRO contract semantics
- Async contract is mandatory when expected duration is often >2s, fan-out exists, or completion is highly variable.
- Start endpoint returns `202 Accepted` with `Location` header for operation status resource.
- Operation resource MUST define `id`, `status`, `created_at`, `updated_at`, success result reference, and structured failure error.
- Status enum defaults: `pending`, `running`, `succeeded`, `failed`, `canceled`.
- Poll guidance MAY include `Retry-After`.
- Completion MAY return final payload or canonical resource reference (`303` allowed).
- Never hide async side effects behind fake synchronous success.

### Consistency and freshness disclosure
- Each endpoint MUST declare consistency mode (`strong` or `eventual`).
- Eventual endpoints MUST declare staleness expectation (target propagation window).
- Eventual read models SHOULD expose freshness fields (`as_of`, `last_updated_at`).
- Do not claim read-after-write guarantees unless explicitly provided.
- Consistency-model changes inside same major API version are behavior changes and require explicit sign-off.

### API cross-cutting boundary contracts
- Validation/normalization/limits/auth/idempotency/correlation/rate-limit/async/webhook concerns MUST be split into:
  - contract obligations (`30-api-contract.md` and impacted artifacts)
  - runtime enforcement expectations (middleware/interceptor boundary).
- Boundary pipeline order MUST be explicit: transport limits -> strict decode -> normalization -> semantic validation -> business logic.
- Strict JSON behavior defaults: reject unknown fields, reject trailing tokens, `400` for malformed payloads.
- Default input limit contracts:
  - `MaxHeaderBytes` `16 KiB`
  - URI length `4 KiB`
  - JSON body `1 MiB`
  - multipart upload `10 MiB`
- Over-limit status defaults MUST be explicit (`413`, `414`, `431`).
- Auth context contract MUST include subject, subject type, tenant, scopes/roles, auth method, and correlation fields.
- Tenant context MUST come from validated identity, not arbitrary caller headers.
- Correlation contract MUST include `traceparent` support and response `X-Request-ID`.
- Rate-limit contract MUST define `429` semantics and `Retry-After` guidance.

### File upload and webhook contract skills
- Upload contracts MUST define media type, size/type limits, processing mode (sync or async), and publish-after-scan semantics when scanning is required.
- Large upload default contract is upload-session/presigned flow, not direct huge multipart payloads.
- Webhook/callback contracts default to at-least-once delivery with duplicate/out-of-order tolerance.
- Webhook contracts MUST define signature verification, replay window (default 5m), retry schedule, dedup key, and sender timeout expectations.

### Distributed consistency and data/cache implications at API boundary
- Do not encode implicit global ACID assumptions in API semantics.
- Expose long cross-service workflows as operation/process state, not fake immediate completion.
- Explicitly classify invariant behavior as immediate (local) vs convergent (cross-service process).
- Contract changes must remain rollout-safe for mixed-version deployments (expand/contract compatibility window).
- For cache-accelerated reads, declare staleness/fallback semantics and avoid using cache expiry timing as correctness mechanism.

### Observability contract obligations
- Request/operation correlation MUST be observable from contract semantics (`X-Request-ID`, trace context expectations, error `request_id`).
- API-facing logs/metrics/traces MUST use low-cardinality route templates; raw IDs/paths are not valid metric dimensions.
- Async API surfaces SHOULD carry stable `correlation_id` and attempt semantics through retries/DLQ transitions.
- Telemetry-related contract fields must be bounded and reviewable (no unbounded identifiers in metric labels).

### Decision quality bar (evidence threshold)
- For each nontrivial decision, provide at least two options and one rejected option with reason.
- Decision record MUST include: compatibility class, status mapping, retry/idempotency, concurrency/preconditions, consistency semantics, and fail-path behavior.
- Every major decision MUST map impacts across `30/40/50/55/70/80/90`.
- If evidence is missing, mark `[assumption]` and create explicit owner-bound unblock item in `80-open-questions.md`.

## Scope And Boundaries
In scope:
- design REST resource model, URI shape, and versioned endpoint structure
- define HTTP method semantics and status-code mapping
- define request/response/error contracts (`application/problem+json`)
- define pagination/filter/sort/field-selection rules
- define idempotency/retry classification and conflict semantics
- define optimistic concurrency semantics (ETag, `If-Match`, `If-None-Match`, preconditions)
- define async/LRO contract (`202`, operation resource, polling/callback semantics)
- define API-level consistency disclosure (`strong` vs `eventual`, staleness notes)
- define API boundary cross-cutting requirements (validation/normalization/limits, auth+tenant context contract, correlation IDs, rate-limit semantics, file upload/webhook contracts)
- classify API changes as additive, behavior-changing, or breaking

Out of scope:
- service/module decomposition and ownership boundaries
- SQL physical schema, DDL/migration scripts, and storage internals
- distributed orchestration internals (saga choreography/orchestration implementation)
- runtime implementation details in middleware/interceptors/handlers
- CI/CD pipeline design and container runtime hardening
- detailed SLI/SLO targets, alert tuning, and on-call policy
- benchmark/profiling plans and low-level performance tuning

## Working Rules
1. Determine the current phase and target gate from `docs/spec-first-workflow.md`, then align contract depth and outputs to that phase.
2. Load minimal context and stop when six API axes are source-backed: endpoint semantics, error model, retry/idempotency/concurrency, cross-cutting boundary behavior, async/consistency semantics, compatibility evolution.
3. Normalize task scope into a contract surface matrix: resource, operation, method, URI, audience, compatibility impact.
4. Use hard-skill defaults from this file as baseline; deviations require explicit rationale and decision record.
5. Build endpoint-level contract matrix first, then derive narrative sections; never skip matrix-level completeness.
6. For each nontrivial decision, compare at least two options and record one rejected option with reason.
7. Record major decisions with IDs (`API-###`), owner, compatibility class, and cross-domain impact.
8. Mark missing critical facts as `[assumption]`; validate within the same pass or move to `80-open-questions.md` with owner and unblock condition.
9. Synchronize API decisions with impacted artifacts (`50/55/70/80/90`) and include explicit `updated` or `no changes required` status for each impacted artifact.
10. Run final consistency pass: no hidden "decide in implementation" items for API boundary behavior.

## API Decision Protocol
For every major API decision, document:
1. decision ID (`API-###`) and phase context
2. owner role
3. client-facing problem statement
4. options (minimum two)
5. selected option and rationale
6. at least one rejected option and rejection reason
7. compatibility class (`additive`, `behavior-change`, `breaking`)
8. status/error mapping and fail-path semantics
9. retry/idempotency/concurrency semantics
10. consistency/freshness semantics (`strong` vs `eventual`, staleness disclosure)
11. cross-domain impacts (`30/40/50/55/70/80/90`)
12. residual risks and reopen conditions

## Output Expectations
- Primary artifact:
  - `30-api-contract.md` with mandatory sections:
    - `Contract Scope And Compatibility`
    - `Resource And Endpoint Matrix`
    - `Method And Status Semantics`
    - `Validation, Normalization, And Input Limits Contract`
    - `Auth, Tenant, Correlation, And Rate-Limit Contract`
    - `Request/Response And Error Model`
    - `Idempotency And Retry Policy`
    - `Concurrency And Preconditions`
    - `Async/LRO Semantics`
    - `Consistency And Freshness Disclosure`
    - `Change Classification And Compatibility Notes`
- Conditional alignment artifacts (update when impacted by API decisions):
  - `50-security-observability-devops.md`
  - `55-reliability-and-resilience.md`
  - `70-test-plan.md`
  - `80-open-questions.md`
  - `90-signoff.md`
- Conditional artifact status format for `50/55/70`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification linked to `API-###`
  - for `updated`, list changed sections with linked `API-###`
- Decision format for major API decisions:
  - decision ID (`API-###`)
  - owner role
  - context/problem
  - options (minimum two)
  - selected option with rationale
  - at least one rejected option with rejection reason
  - compatibility class (additive/behavior-change/breaking)
  - retry/idempotency/concurrency semantics
  - consistency/freshness semantics
  - error/fail-path semantics
  - impacts on architecture/data/security/operability
  - risks and reopen conditions
- Language: match user language when possible.
- Detail level: concrete and reviewable with client-visible behavior explicitly stated.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when six API axes are covered with source-backed inputs: endpoint semantics, error model, retry/idempotency/concurrency, cross-cutting boundary behavior, async/consistency semantics, compatibility evolution.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase section, and target gate first
  - read additional sections only when needed for unresolved contract decisions
- `docs/llm/api/10-rest-api-design.md`
- `docs/llm/api/30-api-cross-cutting-concerns.md`

Load by trigger:
- AuthN/AuthZ, tenant, service identity:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Sync/async interaction and distributed consistency implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Data/cache implications of API contracts:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Observability implications on API boundary:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prioritize trigger-loaded docs over always-loaded docs for that decision.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and track reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` in the same pass or promote it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- `30-api-contract.md` contains all mandatory sections from this skill.
- Endpoint matrix is complete for all affected operations: method, URI, statuses, error profile, retry class, idempotency requirement, preconditions, async mode, consistency mode.
- Cross-cutting API requirements are defined and internally consistent.
- Breaking/behavior changes are clearly classified and justified.
- All major API decisions include `API-###`, owner, selected option, and at least one rejected option with reason.
- API uncertainties are resolved or tracked in `80-open-questions.md` with owner and unblock condition.
- Impacted `50/55/70` artifacts have explicit status and decision links.
- `30` is synchronized with impacted `50/55/70/80/90` artifacts with no contradictions.
- No hidden API decisions are deferred to coding.

## Anti-Patterns
Use these preferred patterns to prevent anti-pattern drift:
- keep resource-oriented API modeling and justify every action-style exception explicitly
- keep one stable `application/problem+json` profile across similar failure classes
- classify retry/idempotency/concurrency semantics for every mutating endpoint
- expose async side effects through explicit `202` + operation-resource semantics
- declare consistency mode and staleness behavior for every read/write surface
- keep cross-cutting boundary controls explicit in contract artifacts, not implicit runtime behavior
- close contract uncertainties in spec artifacts or track them as owner-bound open questions

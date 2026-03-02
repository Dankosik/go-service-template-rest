---
name: api-contract-designer-spec
description: "Design API-contract-first specifications for Go services in a spec-first workflow. Use when planning or revising REST API behavior before coding and you need clear resource modeling, HTTP semantics, error model, idempotency/retry/concurrency rules, and cross-cutting API contract requirements. Skip when the task is local code implementation, pure architecture decomposition, SQL schema/migration design, CI/container setup, or detailed observability/security operations tuning."
---

# API Contract Designer Spec

## Purpose
Create a clear, reviewable API specification package before implementation. Success means API behavior is explicit, consistent, and directly translatable into implementation and tests without contract ambiguities.

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
1. Determine the current phase and target gate from `docs/spec-first-workflow.md`, then align all contract decisions to that phase.
2. Load only the minimal required context and stop when four API axes are source-backed: endpoint semantics, error model, retry/idempotency/concurrency, and cross-cutting API behavior.
3. Normalize the task into explicit contract scope: affected resources/endpoints, client-facing behavior changes, and compatibility expectations.
4. Use `30-api-contract.md` as the primary artifact and keep it as the source of truth for API behavior.
5. For each nontrivial API decision, compare at least two options and select one explicitly.
6. Record major API decisions with IDs (`API-###`) and include rationale, trade-offs, compatibility class, and cross-domain impact.
7. Mark missing critical facts as `[assumption]`; validate them in the same pass or move them to `80-open-questions.md` with owner and unblock condition.
8. Synchronize API decisions with impacted artifacts (`50/55/70/80/90`) and keep implementation-facing behavior fully specified.

## Output Expectations
- Primary artifact:
  - `30-api-contract.md` with mandatory sections:
    - `Contract Scope And Compatibility`
    - `Resource And Endpoint Matrix`
    - `Method And Status Semantics`
    - `Request/Response And Error Model`
    - `Idempotency And Retry Policy`
    - `Concurrency And Preconditions`
    - `Async/LRO Semantics`
    - `Consistency And Freshness Disclosure`
- Conditional alignment artifacts (update when impacted by API decisions):
  - `50-security-observability-devops.md`
  - `55-reliability-and-resilience.md`
  - `70-test-plan.md`
  - `80-open-questions.md`
  - `90-signoff.md`
- Decision format for major API decisions:
  - decision ID (`API-###`)
  - context/problem
  - options (minimum two)
  - selected option with rationale
  - compatibility class (additive/behavior-change/breaking)
  - retry/idempotency/concurrency semantics
  - error/fail-path semantics
  - impacts on architecture/data/security/operability
  - risks and reopen conditions
- Language: match user language when possible.
- Detail level: concrete and reviewable with client-visible behavior explicitly stated.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when all four API axes are covered with source-backed inputs.

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

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.

## Definition Of Done
- `30-api-contract.md` contains all mandatory sections from this skill.
- API behavior is explicit for all affected endpoints: method, URI, status codes, error shape, retry class, and consistency behavior.
- Cross-cutting API requirements are defined and internally consistent.
- Breaking/behavior changes are clearly classified and justified.
- API uncertainties are resolved or tracked in `80-open-questions.md` with owner and unblock condition.
- `30` is synchronized with impacted `50/55/70/80/90` artifacts with no contradictions.
- No hidden API decisions are deferred to coding; all critical unknowns are either resolved or tracked as `[assumption]`/open question.

## Anti-Patterns
- designing API around internal tables or infrastructure topology
- mixing resource-oriented API with implicit RPC semantics without rationale
- omitting retry/idempotency/conflict semantics
- inconsistent error schema across similar failure types
- returning fake synchronous success for async side effects
- shifting contract uncertainty into implementation without open-question tracking

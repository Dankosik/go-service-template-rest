# Service boundaries and decomposition instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Designing or changing service boundaries
  - Splitting a module into multiple services
  - Merging services back into a modular monolith
  - Reviewing architecture for coupling, ownership, and transactional correctness
- Do not load when: The task is a small local code change inside an already stable service boundary

## Purpose
- This document defines how to decompose a system into services, bounded contexts, and ownership boundaries.
- The goal is independent evolution without creating a distributed monolith.

## Boundary selection model
Use the four-axis model for every proposed boundary. Do not decide from one axis alone.

### 1) Domain boundary
- Start with business capability and bounded context, not tables or CRUD entities.
- A service should represent one coherent business capability.
- Bounded context is a domain modeling unit; it is not automatically a separate deployable service.
- Use domain analysis/context mapping before proposing service extraction.

### 2) Data ownership boundary
- Every critical entity must have one explicit owner service (single source of truth).
- Other services access that data via API/events, not direct database access.
- New service boundaries are valid only when data ownership can be isolated.

### 3) Team ownership boundary
- One service should have one clear owning team responsible end-to-end: code, data, operations, SLA, and API contracts.
- If ownership is split across teams, the boundary is likely wrong or premature.
- Prefer service boundaries that align with team communication boundaries (service-per-team / Conway alignment).

### 4) Transaction boundary
- If a use case requires strong ACID invariants across candidates on most requests, keep them in one service/module.
- Cross-service workflows should use local transactions plus explicit consistency patterns (for example saga/outbox), not implicit distributed ACID assumptions.
- Do not assume 2PC/distributed transactions as a default microservice mechanism.
- Avoid dual writes (DB write + direct publish/call in the same logical step without atomic linkage).

## Decision rule: new service vs module
Create a new service only when all points are true:
- Domain boundary is stable and meaningful
- Data ownership can be isolated
- A single team can own the service end-to-end
- Independent deploy cadence and scaling are real requirements
- Cross-boundary workflows can tolerate eventual consistency

Otherwise keep the functionality in a modular monolith or as a module inside an existing service.

## Signals of distributed monolith and bad decomposition
Treat these as red flags:
- Frequent coordinated multi-service releases for one feature
- Regular cascade failures (downstream failure breaks multiple upstream services)
- Chatty synchronous call chains in the core request path
- Hidden coupling via shared schema/table reads
- "Service per entity/table" decomposition
- No clear source of truth for key entities
- Most business flows requiring cross-service ACID expectations
- Shared domain logic package forcing lock-step upgrades across services
- Frequent coordinated schema migrations across services

## When not to extract a microservice
Prefer modular monolith/module-in-service when:
- Domain boundaries are still changing
- Team size or operational maturity is low
- ACID consistency across the proposed split is mandatory
- Most requests would require synchronous multi-service orchestration
- Independent scaling/release cadence is not required yet
- One team can effectively own the module inside the existing service

## Shared database and shared business logic policy
Default policy is prohibition.

### Shared database
- Do not share schema/tables across services.
- Do not read another service database directly, including read-only access.
- Sharing physical DB infrastructure is allowed only if logical ownership and schema isolation remain strict.

### Shared business logic
- Do not place domain models and business rules in shared libraries used by multiple services.
- Shared libraries are acceptable only for cross-cutting concerns (for example logging, metrics, HTTP client utilities, tracing helpers).

### Exception process
Allow an exception only with explicit written justification:
- ADR with problem statement, alternatives, trade-offs, owner, and review date
- Clear sunset/migration plan to remove coupling
- Explicit risk acceptance by owning teams
- No silent or "temporary" exceptions without a deadline

## MUST / SHOULD / NEVER

### MUST
- MUST describe boundaries in terms of business capability and bounded context.
- MUST define data owner and source of truth for every key entity.
- MUST evaluate boundary proposals on domain, data ownership, team ownership, and transaction boundaries.
- MUST document consistency model for cross-service workflows.
- MUST prove independent deployability before introducing a new service.

### SHOULD
- SHOULD default to modular monolith early in greenfield until boundaries stabilize.
- SHOULD prefer coarse-grained APIs/events over chatty synchronous interactions.
- SHOULD revisit boundaries when release coupling appears.
- SHOULD keep internal module boundaries explicit inside a service (for example by package/module structure).
- SHOULD treat distributed systems overhead (operations, observability, contract coordination) as a first-class cost in decomposition decisions.

### NEVER
- NEVER split services by table or CRUD shape alone.
- NEVER share database schema/tables between services without an approved ADR exception.
- NEVER access another service data store directly.
- NEVER introduce shared domain business logic libraries across services without approved ADR exception.
- NEVER rely on 2PC/distributed ACID as a default cross-service strategy.
- NEVER implement cross-service side effects via dual write when consistency matters.
- NEVER hide coordinated deployment requirements behind "independent services" language.

## Review checklist
Before approving boundary changes, verify:
- Boundary is defined by business capability and bounded context
- Data ownership and source of truth are explicit
- Team ownership and operational responsibility are explicit
- Transaction model is explicit (local ACID vs saga/eventual consistency)
- Independent deployability is real, not assumed
- No shared DB/schema access is introduced
- No shared domain business logic coupling is introduced
- No default 2PC/distributed ACID assumption is introduced
- No dual write path is introduced for cross-service side effects
- Distributed monolith risk was assessed and documented

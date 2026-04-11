# Architecture Anti Patterns

## When To Load
Load this when a proposed architecture smells like premature microservices, distributed monolith, shared database, service-per-table, direct cross-service DB reads, dual writes, retry storms, fragile fallback, permanent compatibility shims, or technology-led topology.

Use it as a challenge checklist. Reject or reopen the decision when the smell affects ownership, invariant preservation, failure containment, or rollout safety. Keep the response architecture-first and route lower-level remediation to the right specialist.

## Decision Examples

### Example 1: "It is only reads" shared database coupling
Context: A team proposes a read service that directly joins multiple owners' operational databases.

Selected option: Treat this as a shared database/distributed monolith smell. Use owner APIs, API composition, event-fed projections, or a derived read model with a support owner and staleness contract.

Rejected options:
- Direct reads of private service tables as steady state.
- Shared credentials that let multiple services access each other's tables.
- Calling the read service independent while it still couples every owner schema.

Evidence that would change the decision:
- The system is explicitly still a modular monolith and logical table ownership is enforced under one deployment and one team.
- The read store is a derived copy owned by the consuming surface, not the owners' operational database.
- A temporary migration bridge has a removal date, access controls, and reconciliation plan.

Failure modes and rollback implications:
- Schema changes become coordinated releases across services.
- Long read queries can create runtime coupling through locks, pool exhaustion, or noisy neighbors.
- Rollback requires removing database access, replacing consumers with contracts, and proving no hidden queries remain.

### Example 2: Indefinite dual writes and compatibility shims
Context: During extraction, both old and new paths write the same business state. A "temporary" compatibility topic is proposed with no removal condition.

Selected option: Declare one authoritative writer per phase. If dual-read, shadow write, or bridge events are used, bound them with reconciliation ownership, drift metrics, and contraction criteria.

Rejected options:
- Two active write owners for one invariant-bearing entity.
- Permanent compatibility topics, adapters, or data copy jobs.
- Treating event publication as proof of decoupling while consumers still require coordinated deploys.

Evidence that would change the decision:
- One write path is explicitly non-authoritative and used only for comparison.
- The bridge is a finite migration artifact with owner, exit metric, and deletion task.
- Business accepts a write freeze or one-time cutover instead of coexistence.

Failure modes and rollback implications:
- Divergent writes make neither side trustworthy; rollback requires reconciliation and possibly customer repair.
- Permanent shims become hidden source-of-truth routes.
- Once consumers depend on a compatibility topic, removing it can become harder than the original extraction.

### Example 3: Untested fallback under dependency failure
Context: A service uses a cache or projection for normal reads and falls back to a primary database or external provider when the fast path is unavailable.

Selected option: Prefer improving the primary path, pushing data proactively, failing fast with a controlled degraded response, or turning fallback into a regularly exercised failover mode. If fallback remains, prove capacity and test it under realistic failure.

Rejected options:
- Fallback that triggers only during rare incidents and sends all traffic to a dependency that cannot handle it.
- Fallback that changes semantics silently.
- Fallback that bypasses authorization, tenant isolation, or freshness rules.

Evidence that would change the decision:
- The fallback path is exercised continuously in production and sized as a valid mode, not a panic path.
- The dependency can handle full fallback load and failure injection proves it.
- The degraded response is safer than retrying or fallback and is accepted by the business.

Failure modes and rollback implications:
- Fallback can turn a partial outage into a full-site outage by amplifying load.
- Rare code paths collect latent bugs; test and observe them before relying on them.
- Rollback may be a breaker/flag that disables fallback and preserves the primary system for recovery.

### Example 4: Retry storm and fragile sync chain
Context: A proposed flow adds several synchronous service calls with retries at each hop and no deadline budget.

Selected option: Bound the critical path with end-to-end deadline, per-hop timeout, retry budget, idempotency classification, and circuit or bulkhead policy. Move non-final work to async execution when immediate finality is not required.

Rejected options:
- Retrying every error at every layer.
- Missing or very long timeouts on remote calls.
- Treating a circuit breaker as a replacement for business exception handling.
- Queueing work without a DLQ and owner.

Evidence that would change the decision:
- Calls are local/in-process and do not consume remote failure budget.
- The operation is idempotent and retried at one controlled layer with backoff and jitter.
- A synchronous dependency is required for a hard invariant and fits the deadline budget.

Failure modes and rollback implications:
- Retrying at multiple layers multiplies traffic during dependency distress.
- Slow dependencies can consume threads/connections and cause unrelated failures.
- Rollback should remove the new sync hop or disable retry amplification, not just lower traffic after overload begins.

## Source Links Gathered Through Exa
- Microservices.io, "Shared database": https://microservices.io/patterns/data/shared-database.html
- Microservices.io, "Database per service": https://microservices.io/patterns/data/database-per-service.html
- Microservices.io, "Transactional outbox": https://microservices.io/patterns/data/transactional-outbox.html
- AWS Builders' Library, "Avoiding fallback in distributed systems": https://aws.amazon.com/builders-library/avoiding-fallback-in-distributed-systems/
- AWS Builders' Library, "Timeouts, retries, and backoff with jitter": https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter
- Azure Architecture Center, "Retry Storm antipattern": https://learn.microsoft.com/en-us/azure/architecture/antipatterns/retry-storm/
- Azure Architecture Center, "Circuit Breaker pattern": https://learn.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker
- Azure Architecture Center, "Bulkhead pattern": https://learn.microsoft.com/en-us/azure/architecture/patterns/bulkhead
- Martin Fowler, "Circuit Breaker": https://martinfowler.com/bliki/CircuitBreaker.html


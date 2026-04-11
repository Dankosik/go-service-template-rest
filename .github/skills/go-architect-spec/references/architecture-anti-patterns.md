# Architecture Anti Patterns

## Behavior Change Thesis
When loaded for an architecture smell, this file makes the model turn the smell into a specific blocker, accepted risk, or reopen condition instead of giving a generic "be careful" warning.

## When To Load
Load when a proposed architecture smells like premature microservices, distributed monolith, shared database, service-per-table, direct cross-service DB reads, dual writes, retry storms, fragile fallback, permanent compatibility shims, or technology-led topology.

## Decision Rubric
- Block the smell when it changes ownership, invariant preservation, failure containment, release autonomy, or rollback truth.
- Convert "maybe okay" smells into explicit acceptance criteria: owner, duration, signal, exit condition, and proof command or operational drill.
- Prefer smaller architecture moves when the smell solves the wrong pressure: module seam for ownership sprawl, projection for reads, worker runtime for batch work, bounded retry for dependency distress.
- Route low-level remediation to specialist skills; keep this file at architecture risk and decision level.

## Imitate

### "It Is Only Reads" Shared Database Coupling
Context: a read service directly joins multiple owners' operational databases.

Choose: treat it as shared database/distributed-monolith risk. Use owner APIs, API composition, event-fed projections, or a derived read model with support owner and staleness contract.

Copy: this names why read-only still couples schemas, releases, connection pools, locks, and noisy-neighbor behavior.

### Indefinite Dual Writes
Context: during extraction, both old and new paths write the same business state. A temporary compatibility topic has no removal condition.

Choose: declare one authoritative writer per phase. Bound comparison writes or bridge events with owner, drift metric, reconciliation rule, and contraction task.

Copy: this rejects "temporary" as a substitute for ownership.

### Untested Fallback
Context: a service normally uses cache/projection but falls back to primary DB or provider when the fast path fails.

Choose: prefer safer degradation, proactive data push, primary-path hardening, or a continuously exercised failover mode. If fallback remains, prove capacity and semantics under realistic failure.

Copy: this avoids turning partial outage into full outage by concentrating load on a dependency.

### Retry Storm
Context: a flow adds several synchronous calls with retries at each hop and no deadline budget.

Choose: bound the critical path with end-to-end deadline, per-hop timeout, one controlled retry layer, idempotency classification, and bulkhead/circuit behavior. Move non-final work async when finality is not required.

Copy: this turns reliability smell into architecture constraints instead of sprinkling retries.

## Reject
- "The shared DB is fine because consumers only read." Bad because independent deployability still breaks through private schema coupling.
- "We will remove the shim later" with no owner or exit metric. Bad because later is not an architecture state.
- "Fallback is safer than failing." Bad when fallback semantics, authorization, freshness, and capacity are not proved.
- "Circuit breaker fixes retry storms." Bad because a breaker is not a deadline, idempotency, or business exception policy.
- "Use microservices because the repo is large." Bad because codebase size alone does not prove domain, data, or runtime independence.

## Agent Traps
- Do not over-correct every smell into a hard rejection. Some are valid as bounded migration mechanisms.
- Do not call a shim temporary unless the plan includes deletion criteria and consumer inventory.
- Do not accept cross-service table reads as steady state just because the query is convenient.
- Do not recommend fallback without checking whether it bypasses auth, tenant isolation, freshness, or dependency capacity.
- Do not let "we already have Kafka/gRPC/Temporal" become the architecture reason.

# Universal Disciplines

Curated non-triggerable depth references shared across services. A discoverable
`.agents/skills` method owns invocation; the active phase owns authority,
artifact, review, proof, output, and completion. Each entry below only selects
one decision-changing branch and never reselects those owners.

Raw upstream material is not part of the active instruction graph. Re-vendor it
outside this directory for comparison, then curate only repository-portable
policy here.

| Discipline | Decision pressure | Reached from |
| --- | --- | --- |
| [auth-access-control](auth-access-control/SKILL.md) | Principal, credential, permission, tenant, revocation | `go-security` |
| [cache-engineering](cache-engineering/SKILL.md) | Cache value, key, freshness, fill, invalidation, degradation | `go-db-cache` |
| [concurrency-control](concurrency-control/SKILL.md) | Durable-state interleaving, arbitration, fencing | `go-concurrency` |
| [distributed-system-design](distributed-system-design/SKILL.md) | Cross-component topology, state, failure, evolution | `go-system-architecture`, `go-distributed` |
| [durable-background-jobs](durable-background-jobs/SKILL.md) | Durable acceptance, lease, effect, recovery | `go-reliability` |
| [external-api-integration](external-api-integration/SKILL.md) | Provider request identity, ambiguity, convergence | `go-reliability`, `go-api-contract` |
| [postgres-performance](postgres-performance/SKILL.md) | PostgreSQL bottleneck, cause, intervention | `go-db-cache`, `go-performance` |
| [postgres-schema-design](postgres-schema-design/SKILL.md) | Relational identity, invariant, migration | `go-data-architecture` |
| [production-diagnosis](production-diagnosis/SKILL.md) | Incident localization, causal falsification, recovery | `go-systematic-debugging` |
| [reliable-messaging](reliable-messaging/SKILL.md) | Broker acceptance, effect, acknowledgement, recovery | `go-distributed` |

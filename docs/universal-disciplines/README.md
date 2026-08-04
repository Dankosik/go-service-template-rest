# Universal Disciplines

Deep, technology-anchored engineering disciplines shared across services. They are **non-triggerable depth references**, not catalog skills: the `.agents/skills` specialist lenses own invocation and route here when a request needs the full discipline, so the skill catalog stays within its discovery budget. Each directory keeps the upstream `SKILL.md` process — leading lens, stage chain, completion criteria, executable falsifiers — plus its own progressive-disclosure references.

Source of truth: [Dankosik/agent-skills](https://github.com/Dankosik/agent-skills). Update by re-vendoring, not by local edits, so the copies stay mergeable.

| Discipline | Depth it owns | Reached from |
| --- | --- | --- |
| [auth-access-control](auth-access-control/SKILL.md) | Credentials, session/token lifecycle, permission models, tenant isolation, revocation, deny-path proof | `go-security` |
| [cache-engineering](cache-engineering/SKILL.md) | Freshness contracts, invalidation, stampedes, hot keys, degraded modes, measured cache value | `go-db-cache` |
| [concurrency-control](concurrency-control/SKILL.md) | Lost updates, check-then-act races, locking choice, fencing, leader/singleton overlap on durable state | `go-concurrency` |
| [distributed-system-design](distributed-system-design/SKILL.md) | Forces-driven composition: boundaries, consistency, capacity, failure models, migration | `go-system-architecture`, `go-distributed` |
| [durable-background-jobs](durable-background-jobs/SKILL.md) | Leases, visibility timeouts, poison jobs, schedules, backfills, crash recovery | `go-reliability` |
| [external-api-integration](external-api-integration/SKILL.md) | Provider boundaries: request identity, ambiguous outcomes, webhooks, reconciliation | `go-reliability`, `go-api-contract` |
| [postgres-performance](postgres-performance/SKILL.md) | Baseline-to-delta evidence loop for PostgreSQL latency, locks, and capacity | `go-db-cache`, `go-performance` |
| [postgres-schema-design](postgres-schema-design/SKILL.md) | Invariant-first relational modeling, constraints, safe migrations | `go-data-architecture` |
| [production-diagnosis](production-diagnosis/SKILL.md) | Cross-service incident loop: symptom contract, localization, falsified cause, verified recovery | `go-systematic-debugging` |
| [reliable-messaging](reliable-messaging/SKILL.md) | Broker delivery: publish/consume guarantees, ordering, redrive, replay, effect proof | `go-distributed` |

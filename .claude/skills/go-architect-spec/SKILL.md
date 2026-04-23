---
name: go-architect-spec
description: "Design architecture-first specifications for Go services. Use when planning new features, refactors, service extractions, or behavior changes before coding and you need explicit boundary ownership, workload-driven sync/async design, invariant and consistency rules, failure/degradation model, and rollout-safe implementation sequencing. Reach for this whenever the hard part is deciding module vs service boundaries, long-running workflow ownership, read/write topology, or modular-monolith vs microservice trade-offs. Skip when the task is a local code fix, low-level API/DB/security implementation, test-case authoring, or CI/container configuration."
---

# Go Architect Spec

## Purpose
Turn ambiguous service changes into explicit architecture decisions that remain correct under growth, failure, backlog pressure, and mixed-version rollout, and express them as a compact architecture section rather than drifting into API, schema, or tool detail.

## Specialist Stance
- Treat architecture as ownership, invariants, workload shape, and failure behavior before topology.
- Prefer the smallest boundary change that preserves source-of-truth clarity and operability.
- Challenge service extraction, broker adoption, and workflow-engine choices unless workload and ownership evidence justify them.
- Hand off API, data, security, reliability, and delivery details when they become primary rather than architectural consequences.

## Scope
Use this skill to define or review service-level architecture decisions:
- boundaries and ownership
- decomposition into modules, runtimes, and services
- sync vs async interaction style
- write authority, read topology, and consistency model
- resilience, degradation, and rollout shape

## Boundaries
Do not:
- reduce the task to local code changes or low-level implementation detail
- redesign endpoint payload minutiae, physical schema details, cache tuning, or CI/container setup as the primary output
- approve a new service boundary without ownership, transaction-boundary, runtime-isolation, and operational-readiness proof
- invent stricter SLOs, freshness budgets, or operational thresholds than the prompt supplies unless they are clearly marked as assumptions
- leave critical architecture choices implicit for implementation to discover later

## Escalate When
Escalate if the recommendation depends on unresolved ownership, missing invariant or write-authority assumptions, undefined failure behavior, unclear rollout compatibility, or cross-domain trade-offs that materially affect API, data, security, or operability.

## Reference Loading
Load at most one reference by default, and only when its behavior-change thesis matches the task. A reference should make you choose a more discriminating architecture answer than the `SKILL.md` body alone; if the prompt only needs the broad rule, stay in this file.

Keep the main output architecture-first. Route API payloads, SQL migration mechanics, low-level resilience tuning, and CI/container details to specialist skills instead of expanding architecture references into documentation dumps.

| Symptom in the prompt | Load | Behavior change |
| --- | --- | --- |
| Boundary placement, write ownership, shared-data pressure, or Go package layout being treated as architecture | [boundary-decomposition-examples.md](references/boundary-decomposition-examples.md) | Choose invariant and ownership boundaries instead of service-per-entity, direct DB reads, or generic packages. |
| Debate between modular monolith, internal module, separate worker/runtime, or true service extraction | [modular-monolith-vs-service-extraction.md](references/modular-monolith-vs-service-extraction.md) | Apply the all-conditions extraction test instead of treating traffic or team preference as sufficient. |
| Request-path vs queue, saga, process manager, choreography, orchestration, or workflow engine | [sync-async-workflow-ownership.md](references/sync-async-workflow-ownership.md) | Name the process owner, pivot, and client-visible completion model instead of choosing a broker/tool first. |
| CQRS, read services, projections, materialized views, search indexes, exports, dashboards, aggregators, or stale reads | [read-write-topology-and-projections.md](references/read-write-topology-and-projections.md) | Keep projections derived-only with freshness and bypass rules instead of letting query convenience become write truth. |
| External provider, partner lifecycle, webhook state, vendor status vocabulary, or ambiguous third-party result semantics | [external-provider-anti-corruption.md](references/external-provider-anti-corruption.md) | Normalize provider evidence behind a local lifecycle owner instead of importing vendor states as internal truth. |
| Ownership move, service extraction rollout, source-of-truth change, mixed-version window, canary, shadow read, or rollback boundary | [rollout-and-migration-patterns.md](references/rollout-and-migration-patterns.md) | Select phased compatibility with one authoritative writer per phase instead of big-bang cutover or indefinite bridges. |
| Premature microservices, distributed monolith, shared database, service-per-table, direct cross-service DB reads, dual writes, retry storm, fragile fallback, or permanent shim smell | [architecture-anti-patterns.md](references/architecture-anti-patterns.md) | Convert the smell into a blocker, accepted risk, or reopen condition with concrete failure consequences. |

## Core Defaults
- Prefer modular monolith boundaries until service extraction is justified by domain capability, data ownership, team ownership, transaction boundaries, and runtime isolation needs.
- Prefer one explicit source of truth per invariant-bearing entity or process.
- Prefer runtime splits, bounded worker pools, queues, projections, or read replicas before service splits when the problem is mainly batch work, fan-out, or read scale.
- Prefer local ACID inside one service-owned datastore; use explicit eventual-consistency patterns across services.
- Prefer additive, compatibility-first evolution (`expand -> migrate/backfill -> contract`) over big-bang replacement.
- Treat operational overhead, observability cost, and release coordination as first-class costs in every decomposition decision.

## Architecture Facts To Lock First
Before recommending topology, make these facts explicit:
- which invariants are truly hard and who owns them
- which step is the irreversible or non-compensable pivot
- who owns write truth, who owns read projections, and which views are derived only
- what work belongs on the request path and what should move to background execution
- what actually dominates scale: contention, read fan-out, payload size, hot keys or hot tenants, queue depth, external latency, or team isolation
- what evidence exists for the choice: latency budget, QPS and burstiness, read/write ratio, freshness SLA, data growth, and RPO/RTO expectations
- which degradation modes are acceptable and which must fail closed
- which mixed-version, migration, or rollout windows already constrain the design

## Expertise

### Workload Shape And Topology
- Classify the dominant workload before choosing architecture: request/response, long-running job, bursty fan-out, stream processing, reconciliation, or operator-driven workflow.
- Do not mistake high read volume, heavy CPU, or large batch/export jobs for proof that a new service boundary is needed.
- Separate read-scaling problems from write-ownership problems. Read replicas, caches, search indexes, materialized projections, and worker runtimes can change topology without creating a new domain owner.
- Model hot paths, hot keys, hot tenants, backlog growth, and payload-size pressure explicitly. A service split that leaves the real bottleneck untouched is not an architecture improvement.
- Use the supplied freshness and latency constraints as the decision boundary. If a tighter number is useful, mark it as an assumption instead of presenting it as an established fact.
- Reject technology-led decisions such as “use Kafka because we have Kafka” or “split a service because traffic is rising” unless the workload and ownership model actually require them.

### Boundaries And Decomposition
- Use a four-axis boundary prompt for every boundary decision: domain capability, data ownership, team ownership, and transaction boundary.
- Require explicit source-of-truth ownership for each critical entity and process.
- Internal module seams may follow invariant ownership, change cadence, or failure isolation, not only entity names.
- When modular-monolith seams are the hard part, express each module in terms of `owns truth`, `must not own`, `sync seam`, `async seam`, and `extraction posture` if that removes ambiguity.
- For modular-monolith work, make the orchestration or application layer explicit when it coordinates multiple modules, and keep subdomain truth inside the owning modules.
- If one module owns process truth, do not automatically collapse that into the wiring layer. Describe the application/orchestration layer separately when that distinction prevents peer-module coupling.
- If dependency direction matters to keep seams real, state it directly rather than leaving it implied.
- Use anti-corruption adapters when an external or legacy model would otherwise leak across a boundary and distort local domain rules.
- Reject service-per-table, service-per-CRUD, shared-schema decomposition, and cross-service direct DB access by default.
- Approve service extraction only when independent deployability, ownership, scaling, runtime isolation, operational readiness, and accepted consistency trade-offs are all explicitly justified.
- Detect distributed-monolith signals early: coordinated releases, chatty call chains, shared schema coupling, cross-service table reads, or hidden shared business logic.
- Distinguish service boundaries from runtime boundaries. Separate processes or binaries are sometimes enough; a new service should not be the default answer to every isolation problem.

### External Boundaries And Anti-Corruption
- Treat external providers as semi-trusted evidence sources, not as authorities for internal lifecycle truth.
- Normalize provider results inside the owning module or service. Do not let partner statuses, payload fields, or failure vocabulary become the internal or public source-of-truth model by accident.
- Name the anti-corruption boundary explicitly when provider instability, mixed-version partner behavior, or domain-language mismatch could leak into the core workflow.
- Keep retry, reconciliation, timeout, and operator-repair ownership with the local boundary, not with the provider's semantics.

### State, Invariants, And Pivots
- Build an explicit invariant register before selecting sync/async boundaries or service topology.
- Separate `local_hard_invariant` from `cross_process_invariant` and keep hard invariants inside one local transaction boundary whenever possible.
- Identify the irreversible or non-compensable pivot in every multi-step flow and design recovery around it.
- Model long-running or failure-prone flows with explicit durable execution state: monotonic transitions, durable identity, timers, and reconciliation ownership.
- Derived projections, caches, search indexes, and exports may accelerate reads, but they must not quietly become write authorities.
- Require one active owner for retries, stuck detection, manual repair, and convergence in any multi-step workflow.

### Sync Communication And Critical Path
- Prove that synchronous hops are required before selecting transport or adding a service-to-service dependency.
- Define the critical path, end-to-end deadline budget, and per-hop budget before approving a sync call chain.
- Keep request paths short and non-chatty. If a design needs multiple remote calls in sequence, justify why the path still meets latency and failure goals.
- If the outcome does not need immediate finality, prefer a job, long-running operation, or resource-status pattern over stretching the request path.
- Make retry policy explicit per operation; bound retries and classify non-retry cases.
- Require idempotency design for retry-unsafe operations.
- Choose API transport from consumer reach, contract tooling, streaming, latency, and operability. REST/OpenAPI often fits public or browser-facing surfaces; gRPC/Protobuf often fits controlled internal RPC when strong schemas, streaming, or low-latency transport matter.
- Do not place remote calls after a non-compensable pivot unless the recovery and reconciliation model is explicit.

### Async, Queueing, And Workflow Engines
- Choose async because of latency variability, fan-out, buffering, backpressure, human-in-the-loop work, cancellation, or retry isolation, not because a broker exists.
- Classify each async handoff explicitly as an event or a command and align ownership with that intent.
- Choose orchestration when one owner must track timers, retries, cancellation, reconciliation, or operator actions. Choose choreography only when independent reactions do not need one authoritative process state.
- Use pub/sub for independent domain reactions and queues for owned work distribution.
- Require transactional outbox or an equivalent atomic linkage when a DB state change must emit a message.
- When duplicate processing can change business state or trigger irreversible side effects, require consumer idempotency, durable dedup or inbox handling, bounded retries, poison-message handling, and clear DLQ ownership.
- Prefer an internal durable state machine or process manager only when the workflow is locally owned and simple enough that state, timers, retries, and repair paths stay reviewable.
- Consider a workflow engine or durable-execution platform when long timers, human tasks, cross-owner orchestration, replay/debug needs, fleet-wide operations, or hand-rolled retry/state persistence would become the architecture.

### Data And Read Topology
- Separate command authority, query projections, and analytical or export views. Only one surface should own correctness for writes.
- Avoid cross-service joins in write paths. For hot read paths, use explicit aggregators, BFFs, or service-owned projections instead of ad hoc cross-service querying.
- When approving a query runtime or read service, state the rule in one line: who owns write truth, what is derived-only, and which correctness-critical paths must bypass the projection.
- If one datastore remains shared, enforce strict logical ownership by module or service and forbid shared writes as steady state.
- For long-running exports, reporting, or scans, define a stable read fence or documented consistency boundary instead of making vague exact-snapshot claims.
- Treat caches and search indexes as performance tools with a staleness contract, not as hidden sources of truth.

### Distributed Consistency And Evolution
- Keep hard invariants inside one local transaction boundary whenever possible.
- For multi-step or cross-service flows, define each step contract: trigger, local transaction scope, idempotency key, timeout, retry class, and compensation or forward-recovery rule.
- Identify the pivot transaction and enforce compensable-before / retryable-after rules.
- Collapse a flow back into one ownership boundary if the distributed design adds coordination cost without independent ownership, scaling, or deployability benefits.
- Require reconciliation ownership, cadence, and repair path for critical eventual-consistency flows.
- Reject dual writes, hidden invariant ownership, and distributed locks as the primary correctness mechanism.
- Never allow indefinite dual writes or permanent compatibility shims. If a bridge is temporarily required for migration, bound its authority, reconciliation, and removal criteria up front.
- Make schema, state, and event evolution additive-first; require tolerant readers, mixed-version compatibility, and explicit contraction criteria.
- Prefer shadow, dark-read, dual-read, canary, or strangler cutover phases over one-shot boundary moves when data or ownership is shifting; define exit criteria and irreversible checkpoints explicitly.

### Resilience, Degradation, And Release Safety
- Classify dependencies by criticality before selecting fallback behavior.
- Define per-dependency timeout, retry budget, bulkhead, fallback mode, and observability signals.
- Propagate deadlines explicitly and fail fast when remaining budget is insufficient.
- Bound queues, leases, and concurrency. Make overload shedding, noisy-neighbor protection, and blast-radius isolation explicit.
- Define degradation modes, activation conditions, and deactivation criteria.
- Require graceful startup and shutdown semantics for stateful workers, consumers, and long-running jobs.
- Make rollback authority and rollback limits explicit whenever a change is not trivially reversible.

### Cross-Domain Consequences
- Record downstream effects only when they force a new decision, handoff, or proof obligation before the current architecture recommendation is usable.
- API: make consistency disclosure, idempotency, long-running-operation behavior, and compatibility windows explicit when architecture changes external behavior; otherwise record `no new decision required in API`.
- Data: keep data ownership boundaries clear, justify datastore choices by access patterns, and frame cache or projection use by correctness and staleness contract only when those choices are architecture-critical now.
- Security: define trust boundaries, identity propagation model, tenant isolation, and fail-closed authorization expectations when architecture changes who enforces them.
- Operability: require the minimum logging, metrics, traces, and debuggability needed to operate the design safely.
- Delivery: ensure the architecture can actually be enforced by CI, release gates, migration controls, and runtime assumptions.

## Decision Quality Bar
For every major architecture recommendation, include:
- the problem and constraints
- the dominant workload and invariant drivers
- whether a real `live fork` exists
- when a `live fork` exists, the viable options, the selected option, and at least one explicit rejection reason
- when no `live fork` exists, the chosen repo-consistent approach and why it is the only decision that needs current architecture treatment
- who owns write truth and which views are derived only
- when external providers matter, how their semantics are normalized and prevented from becoming lifecycle truth
- trade-offs, risks, and control mechanisms
- measurable acceptance boundaries
- rollout strategy and rollback limits
- explicit reopen or extraction criteria when proposing a read runtime, separate runtime, or future service split
- for runtime split vs real service extraction, use an all-conditions test when precision matters and state what rollback does and does not revert
- any invented numeric target marked as an assumption rather than a silent fact
- assumptions, blockers, and reopen conditions

## Deliverable Shape
When writing the architecture spec or review, cover:
- context, scope, and non-goals
- boundary and ownership model
- an ownership matrix for internal modules when modular-monolith seams are central to the task
- dependency direction when internal seams or orchestration placement matters
- workload shape, critical path, and runtime topology
- sync or async interaction style
- command/query authority split when read projections are involved
- consistency model, invariants, and state-machine expectations
- anti-corruption or provider-boundary rules when external systems affect domain behavior
- failure, degradation, and rollout strategy
- only the downstream API, data, security, operability, or delivery effects that force a new decision, handoff, or proof obligation
- explicit `no new decision required in <domain>` notes when an adjacent domain is affected but not decision-critical now

## Escalate Or Reject
- a new service boundary without ownership, transaction-boundary, and runtime-isolation proof
- a read model, cache, or search index quietly becoming write authority
- a sync call chain without critical-path budgets, retry semantics, and idempotency classification
- a correctness-bearing async design without atomic message linkage, idempotent consumption or dedup, bounded retries, or DLQ/replay ownership
- a distributed flow without invariant ownership, pivot definition, and explicit state model
- a migration that relies on indefinite dual writes, permanent compatibility shims, or manual heroics
- a workflow-engine or broker recommendation based on tool familiarity instead of workload evidence
- any architecture decision left for coding to discover later

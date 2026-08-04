---
name: distributed-system-design
description: "Forces-driven design and review of cross-component distributed systems. Use for service boundaries, interactions, data ownership, consistency, replication, partitioning, scalability, availability, multi-region behavior, failure isolation, capacity, migration, or architectural tradeoffs. Route a single messaging, external-API, background-job, cache, or PostgreSQL mechanism to its dedicated skill; use this skill to compose multiple mechanisms into one system contract."
---

# Forces-Driven Distributed System Design

Design from **forces**, not from a catalog:

`requirements -> forces -> estimates -> simplest topology -> contracts -> failure model -> evolution -> proof`

Every component, edge, and pattern is **earned** by a named requirement or constraint. The strongest design is the simplest topology whose contracts survive the stated load, failures, and evolution.

## Choose the depth

Begin at the requested decision and follow only branches that can change its verdict or make the result unsafe. Reuse supplied evidence and settled decisions.

- **End-to-end design, readiness, migration, or material guarantee change:** trace the full in-scope chain from forces through proof.
- **Narrow review, diagnosis, or correction:** stay on the affected journey and its decision-changing dependencies; leave unrelated capacity, topology, migration, and failure work out of scope.
- **Unmet need for distribution:** keep the current deployable, store, or synchronous path and name the measurable condition that would justify the next boundary.

Preserve unresolved inputs as labelled assumptions, variables, or gaps. The sections below are a causal map, not mandatory report headings.

## Authority and evidence

For review, diagnosis, design, or planning, inspect available requirements, code, schemas, diagrams, telemetry, incidents, and deployment artifacts while preserving state. For build or change requests, make in-scope local changes and run non-destructive validation.

Provisioning, deployment, traffic or failover changes, data movement, production load or fault tests, credential changes, and destructive actions require explicit authorization for the exact target and bounds. Verify an authorized production action with fresh readback.

Check material provider-, product-, version-, and pattern-specific claims against current primary documentation. Separate facts from inference, and keep proposed, validated locally, deployed, and verified live states distinct.

Stop when the requested artifact meets its applicable criteria or a concrete blocker or authorization boundary is reached. Name the exact missing evidence, decision, or next action.

This skill owns system composition and cross-component contracts. Hand detailed mechanism work to the matching skill while retaining the end-to-end requirement. When that deep dive is a real next step, record `<skill> -> unresolved question or evidence` in the output; omit ceremonial handoffs that change no decision.

| Concern | Handoff |
| --- | --- |
| Broker delivery, ordering, redrive, replay | `reliable-messaging` |
| Third-party API or webhook boundary | `external-api-integration` |
| Durable worker, lease, schedule, backfill | `durable-background-jobs` |
| Cache key, freshness, invalidation, degradation | `cache-engineering` |
| PostgreSQL relations and constraints | `postgres-schema-design` |
| PostgreSQL query or capacity bottleneck | `postgres-performance` |

## 1. Frame the forces

Turn the request, supplied evidence, and settled decisions into:

- the functional boundary, actors, critical operations, and business invariants;
- latency, availability, durability, freshness, throughput, RPO, and RTO targets per critical journey;
- data classification, tenant and authorization boundaries, residency, retention, and audit requirements;
- current topology, migration constraints, team ownership, delivery horizon, budget, and operated technologies;
- the requested decision or artifact and the evidence needed to call it ready;
- facts, inferences, assumptions, and unresolved decisions.

Use supplied targets exactly. Preserve a missing target as a variable or bounded assumption.

Surface conflicting requirements. Keep the stricter safety contract as the conservative assumption, or present the alternatives and the decision needed to choose.

## 2. Quantify the load envelope

For each design-driving path, estimate only dimensions that can change the topology:

- average, peak, and burst request or event rate;
- service time and concurrent in-flight work;
- payload, fan-out, network throughput, and cross-region traffic;
- write/read ratio, storage growth, retention, replication, and index or derived-data amplification;
- tenant or key skew, hot partitions, batch size, and retry or replay amplification;
- degraded and recovery load after a dependency, zone, or region returns.

Show formulas, units, ranges, and sensitivity to the largest assumptions. Read [references/capacity-and-topology.md](references/capacity-and-topology.md) when capacity, decomposition, sharding, cells, or independent scaling may affect the design.

Keep rate, payload, fan-out, retention, retry amplification, and sustainable service rate independent until evidence relates them. Do not silently collapse several intents, deliveries, or effects into one source operation.

## 3. Draw the simplest viable topology

Start from the current system, or from one deployable and one authoritative store for a greenfield design. Earn each added network boundary, copy, queue, cache, partition, service, region, or control plane through independent scaling, failure containment, data or trust ownership, deployment lifecycle, geographic locality, or a different consistency or latency contract.

For every component, record:

- one responsibility and the force that requires it;
- owned state and source-of-truth status;
- synchronous and asynchronous interfaces;
- scaling unit, failure domain, trust boundary, and operator;
- dependencies needed to serve or recover.

## 4. Define state and interaction contracts

For each edge, specify protocol and direction, operation or message identity, schema and compatibility, deadline, retry ownership, ordering, concurrency or backpressure, idempotency, acknowledgement or response semantics, authentication and authorization, and observability.

For each critical operation, trace the read and write path and name:

- the authoritative state and transaction boundary;
- the consistency and freshness required by that operation;
- replicas, partitions, derived copies, and their lag or conflict behavior;
- the response exposed during success, uncertainty, degradation, and recovery;
- reconciliation for any state that can diverge.

Read [references/data-and-coordination.md](references/data-and-coordination.md) when state crosses a process, partition, replica, datastore, or region, or when ordering, consensus, distributed transactions, sagas, CQRS, or materialized views are considered.

## 5. Break the design

Read [references/resilience-and-load.md](references/resilience-and-load.md) whenever the design has a remote dependency, retry, queue, failover, autoscaling, or availability target.

Build a failure matrix for critical components and edges:

| Failure or overload | Detection | Containment/degraded behavior | Recovery/reconciliation | User-visible result | Signal and test |
| --- | --- | --- | --- | --- | --- |

Include slow and partial failures, exhausted resources, retry amplification, backlog growth, stale or duplicate data, process and host loss, dependency and control-plane loss, zone or region loss where applicable, and security or tenant-isolation failures. Recalculate remaining capacity under the failure scenario; redundancy that cannot carry failover load does not meet an availability target.

## 6. Compare alternatives and earn patterns

When the choice is material, compare the simplest viable design with its strongest practical alternative; for a review, compare the current design with the smallest viable correction. Evaluate both against requirements, capacity, consistency, failure containment, operational burden, security, cost, delivery time, and future change. End the comparison when no practical alternative could change the decision.

Record each material pattern decision as:

`forces -> chosen pattern -> guarantee -> cost/new failure mode -> rejected alternative -> falsifier`

Prefer a managed or already-operated primitive when it provides the required contract. A familiar pattern that changes no decision is reference, not architecture.

## 7. Design evolution and proof

Read [references/evolution-and-multi-region.md](references/evolution-and-multi-region.md) when the design changes an existing system, spans regions, promises disaster recovery, or requires mixed-version operation.

Define:

- version compatibility and coexistence of old and new components;
- data migration, backfill, validation, cutover, and cleanup, with an owner and removal condition for every temporary writer, adapter, flag, shadow copy, job, and compatibility path;
- the last rollback-safe state, the first new-only or irreversible write, who may cross that boundary, and the roll-forward or repair path after it;
- deployment order, blast-radius controls, and abort signals;
- load, consistency, fault, recovery, security, and restore tests, including a lost response after each authoritative commit and retry with the same logical identity;
- production signals that prove SLOs and reveal saturation or divergence.

Map each high-risk assumption to current evidence or to a named validation with environment, workload, expected observation, stopping rule, and owner.

## 8. Report the design

Lead with the verdict. Preserve the requirements, decisions, evidence, material caveats, and next action; trim introductions, repetition, generic pattern explanations, and optional background first. Keep facts separate from inference.

For a narrow task, use the compact interface: `verdict -> affected contract -> cause or decision -> smallest correction -> proof/gap`.

For any full-chain task—end-to-end design, readiness, migration, or material guarantee change—read [references/reporting.md](references/reporting.md) for the full report interface and completion criterion. Include a Mermaid diagram when three or more components or failure domains interact.

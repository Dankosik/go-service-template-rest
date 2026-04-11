# Modular Monolith Vs Service Extraction

## Behavior Change Thesis
When loaded for modular-monolith vs service-extraction pressure, this file makes the model run an all-conditions extraction test and often choose module or runtime isolation, instead of treating rising traffic, team preference, or package sprawl as enough to create a service.

## When To Load
Load when a prompt asks whether to keep a Go service as a modular monolith, create internal packages/modules, split a separate runtime/worker, or extract a true service.

## Decision Rubric
- Default to modular monolith when ownership, datastore, transaction boundary, and release cadence are still shared.
- Choose a separate worker/runtime when workload isolation, scheduling, CPU, queue depth, or request-path protection is the main problem and write truth stays put.
- Choose a true service only when domain capability, exclusive data authority, team/support ownership, independent deployability, runtime isolation, and accepted consistency trade-offs all hold.
- State rollback truthfully: module refactors usually roll back by code path; service extraction may leave routing, data authority, in-flight work, and compatibility windows behind.
- Name extraction posture and reopen criteria, so "not now" is not mistaken for "never".

## Imitate

### Onboarding Owned By One Team
Context: applicant profile, document verification, sanctions screening, and reviewer decisions live in one codebase and one Postgres database. One team owns the whole flow. A proposal wants four microservices.

Choose: keep a modular monolith. Define modules around invariant ownership and workflow roles, with an application/orchestration layer if it coordinates the process. Use logical data ownership rules and explicit internal contracts.

Copy: this gives the implementation a real seam without adding network, data migration, and release coordination costs before they buy anything.

### Batch Export Pressure
Context: long-running exports scan large ranges and threaten request-path latency.

Choose: separate worker runtime, bounded queue, read replica, or stable read fence first. The worker owns execution and backpressure; the core service still owns write truth.

Copy: this distinguishes workload isolation from domain ownership.

### Read-Only Storefront Rendering
Context: high-throughput read-only rendering has narrow data access and stricter runtime constraints than merchant administration.

Choose: a separate runtime or service can be justified if inputs are stable, the capability is derived-only, and independent scaling or performance controls materially reduce risk.

Copy: this permits runtime separation without dragging mutable core workflow state across the boundary.

## Reject
- "Traffic is high, so make a microservice." Bad because the bottleneck might be read fan-out, CPU, cache misses, or batch pressure rather than ownership.
- "Use technical layers as modules: handlers, services, repos." Bad because business changes still cut across every layer and no owner owns truth.
- "Split now, keep both services using the old tables." Bad because the service boundary is fake until data authority and schema contracts are exclusive or explicitly transitional.
- "A separate worker is basically a service extraction." Bad because a worker can be a runtime boundary under the same owner and datastore.

## Agent Traps
- Do not conflate "module", "binary", "runtime", and "service"; choose the smallest boundary that solves the actual pressure.
- Do not require every module to be extraction-ready on day one. Some modules exist to restore local ownership clarity.
- Do not ignore operational cost. A service needs on-call, observability, deploy, migration, and compatibility ownership.
- Do not leave "future extraction" vague; name the evidence that would reopen it.

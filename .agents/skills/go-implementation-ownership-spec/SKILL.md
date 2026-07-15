---
name: go-implementation-ownership-spec
description: "Use when accepted domain, API, data, architecture, and proof decisions need a coherent Go package, file, dependency, sequence, cleanup, and proof design; Own implementation mechanism and placement; Skip when system topology or domain policy is unresolved, or root technical-design orchestration is requested."
---

# Go Implementation Ownership Spec

Load the [shared specialist contract](../specialist-contract.md), then apply this Go-placement boundary. Use the accepted [system/integration design](../../../docs/spec-first-workflow/phases/system-integration-design.md) and [Go ownership design](../../../docs/spec-first-workflow/phases/go-code-ownership-design.md) as inputs, not policy to rediscover.

## Outcome And Boundary

Integrate accepted domain, API, data, architecture, security, reliability, rollout, and proof decisions into one implementation design. Own Go package/file responsibility, dependency direction, placement of access to accepted sources of truth and other repository-level mechanisms, generated/manual source authority, change sequence, cleanup, test owner, and executable proof.

Do not choose or revise service/component topology, public API behavior, business invariants, physical data policy, trust rules, lifecycle/retry budgets, distributed semantics, delivery policy, or other unresolved behavior. Stop on those gaps before package placement; a package diagram cannot settle an upstream policy fork.

## Ownership Core

1. Start from observable behavior, accepted topology, sources of truth, contracts, sequence, failures, consistency, and operational boundaries; map each responsibility to one narrow owner.
2. Choose the smallest repository mechanism that realizes accepted decisions before naming packages. Preserve established seams and current Go/stdlib patterns; apply the canonical [research method](../../../docs/spec-first-workflow/phases/research.md#method) only for a real unresolved mechanism choice.
3. Keep domain truth independent of transport, persistence, and generated adapters. Point dependencies inward or toward explicit consumer-owned seams; reject cycles, generic shared packages, speculative interfaces, and layer-per-type scaffolding.
4. Name canonical authored sources and generated outputs. Change generators/config/specs before derived files, and never split policy between generated and manual owners.
5. Order implementation by dependency and safe transition: source-of-truth changes, adapters/composition, compatibility or migration steps, tests, rollout-sensitive checks, then removal of replaced code, stale tests/config, and temporary bridges.
6. Assign proof to the narrow owner: observable behavior and failure paths, package/dependency conformance, generated drift, migration/compatibility, and cleanup. Do not substitute a package compile for contract or system proof.

Prefer one `design/overview.md`; split a focused artifact only when it creates a real owner or review boundary.

## Return And Stop

Return accepted inputs and constraints; accepted sources of truth and their access owners; mechanism mapping; package/file owners and dependency direction; authored/generated authority; call and failure sequence; implementation order; cleanup; tests and commands; assumptions; and reopen conditions. The design is ready when planning can name files, owners, dependencies, proof, and deletions without making a design decision.

Return no new decision when accepted design already fixes those details. Stop and name the required API, system architecture, domain, data, security, reliability, distributed, delivery, or proof owner whenever implementation would otherwise invent policy. Reject speculative abstractions and any placement proposal that silently changes topology or wire behavior.

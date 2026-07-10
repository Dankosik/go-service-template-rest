---
name: go-design-spec
description: "Assemble and reconcile integrated technical-design context for Go services when separate design depth is triggered. Use when `spec.md` is specification-review-approved but the work still needs coherent system/integration design, Go code ownership design, task-local `design/` artifacts, cross-domain reconciliation before the mandatory technical design review, or read-only technical-design-review analysis before `planning-and-task-breakdown`. Skip when the task is a local code fix, pure spec authoring, direct-path work, lean-local work with sufficient compact design in reviewed `spec.md`, implementation coding, post-code review execution, or CI/container setup."
---

# Go Design Spec

## Trigger And Scope

Use this skill to integrate task-local technical design after specification review when separate design depth is triggered, or as the design-integrity lens in a read-only technical-design review. Reconcile system/integration behavior, Go code ownership, architecture, contracts, data, security, reliability, observability, delivery, cleanup, and test ownership without reopening the approved problem frame.

Do not author `spec.md`, implementation tasks, phase cards, code, post-code review, or tooling/process detail. Keep lean-local design in reviewed `spec.md` when its `Compact Design` is sufficient; do not create a design bundle for appearance.

## Approved Input And Design Boundary

Require specification-review-approved `spec.md`, the current shape/trigger record, `docs/repo-architecture.md` when repository ownership matters, current task-local design artifacts, and relevant specialist decisions/evidence. Missing or contradictory approved behavior reopens specification rather than becoming a design assumption.

Separate design depth is ordered: system/integration design decides observable service behavior, contracts, external systems, data/source-of-truth, sequence, failures, validation, and rollout; Go code ownership design then assigns package/file responsibility, dependency direction, local abstractions, cleanup, and test ownership without changing that behavior. Each triggered checkpoint records its own `Design fan-out` result before review-ready handoff.

## Integrated Design Invariants

1. **Artifact ownership stays stable.** `spec.md` owns approved decisions; compact or split `design/` owns implementation-critical technical context; `tasks.md` later consumes both. Do not duplicate authority across them.
2. **Observable behavior precedes code placement.** Complete system/integration choices before Go ownership choices. Planning must not invent architecture, contract, sequence, failure, rollout, package/file, cleanup, or test ownership.
3. **Every responsibility has one source and owner.** Name runtime/generated authorities, allowed dependency direction, adapter boundaries, explicit non-owners, replacement cleanup, and retained legacy owner/reason/proof/exit condition.
4. **Complexity must remove present risk.** Prefer explicit Go-native boundaries, direct control flow, narrow consumer-owned interfaces, and focused same-package seams. Reject speculative layers, generic shared packages, interface-per-struct, manager/factory chains, and simplification that weakens a contract.
5. **Contract changes close explicitly.** REST/OpenAPI, event payload, caller-visible status/error/idempotency/retry/async/freshness/compatibility, generated contract, or material internal-interface changes resolve to `design/contracts/`, `compact_sufficient`, `not_expected`, or `blocked` with source and proof carrier.
6. **Failure and side-effect order are first-class.** Runtime sequence names sync/async boundaries, deadlines, retries, atomic linkage, idempotency, partial failure, recovery, shutdown, degraded behavior, and validation at the scenarios where they matter.
7. **Cross-domain contradictions are resolved, not blended.** Preserve specialist ownership, classify downstream seams as decision, constraint, proof, follow-up, or no new decision, and reopen the actual owner when a planning-critical fork remains.
8. **Review readiness is falsifiable.** The smallest coherent bundle must let technical design review and then planning proceed without rediscovery; missing source/owner/sequence/proof or unresolved fan-out blocks readiness.

Before selecting an architecture, workflow, integration, resilience, consistency, data-flow, or abstraction shape, perform Pattern Fit Diligence with concrete pattern descriptions and real-use examples, task forces, repository/operational/proof fit, and idiomatic Go fit. Before selecting custom infrastructure, a runtime dependency, or material helper/abstraction, compare current stdlib, established repository patterns, mature OSS, and custom code with current maintenance, adoption, license, security, transitive-cost, API-stability, and boundary-fit evidence.

## Symptom-Driven Reference Selector

Load repository/task artifacts first. Then load at most one reference by default; use more only for independent pressures. State which design choice the reference is expected to change.

| Symptom or decision pressure | Load | Behavior change |
| --- | --- | --- |
| Bundle shape, conditional artifacts, status/indexing, or spec/design/task ownership is unclear. | [design-bundle-assembly.md](references/design-bundle-assembly.md) | Produce the minimum triggered, indexed bundle instead of filler artifacts or disguised spec/planning prose. |
| Package/file responsibility, source responsibility, generated authority, dependency direction, cleanup, or test owner is unclear. | [component-and-ownership-maps.md](references/component-and-ownership-maps.md) | Name concrete owners and non-owners instead of shared helpers, generated authority drift, or vague placement. |
| Runtime order, side effects, sync/async boundary, retries, failure, recovery, or partial completion is unclear. | [runtime-sequence-and-failure-points.md](references/runtime-sequence-and-failure-points.md) | Write scenario-level flow and failure ownership instead of a happy-path arrow chain. |
| Specialist outputs disagree across architecture, API, data, security, reliability, observability, delivery, or QA. | [cross-domain-reconciliation.md](references/cross-domain-reconciliation.md) | Resolve contradictions through selected/rejected options, owner, and proof obligations instead of vague compromise. |
| Review-ready/planning-ready status, design fan-out, review verdict, proof carryover, or reopen conditions are unclear. | [design-readiness-and-planning-handoff.md](references/design-readiness-and-planning-handoff.md) | Block or qualify readiness explicitly instead of making planning rediscover design gaps. |
| A layer, interface, helper, adapter, shared package, dependency, or claimed simplification changes ownership or impact radius. | [complexity-and-abstraction-tradeoffs.md](references/complexity-and-abstraction-tradeoffs.md) | Require present-day complexity reduction and preserved guarantees instead of future-proof indirection or false simplification. |

## Required Evidence And Deliverable

For each material design decision, record the approved source, current repository evidence, live fork or settled default, selected/rejected options and patterns, runtime/generated source of truth, owner/non-owner, dependency direction, sequence and failure consequence, cleanup/test owner, proof carrier, assumptions, and reopen trigger.

Produce only triggered artifacts:

- lean `Compact Design` or `design/overview.md` when sufficient;
- `design/system-integration.md` and `design/go-code-ownership.md` when separate checkpoints are triggered;
- focused component, sequence, ownership, data-model, dependency-graph, contract, test-plan, or rollout artifacts only when their owner trigger requires them.

In technical-design-review mode, use the [shared review finding envelope](../../../docs/subagent-contract.md#shared-review-finding-envelope) and classify findings as `blocks_planning`, `reopens_design`, `reopens_spec`, `accepted_risk_candidate`, `proof_obligation`, or `record_only`; recommend `PASS`, `CONCERNS`, or `FAIL`. Name the exact planning decision made unsafe, the strongest counterargument or simpler alternative, and why the verdict is not stronger or weaker. Review does not rewrite design.

## Success, Escalation, And Stop Conditions

Success means the design bundle is internally consistent, minimal, owner-specific, testable, operable, cleanup-complete, and ready for the distinct technical-design review; review success means planning can create implementation-ready tasks without inventing decisions.

Stop or escalate when the spec is unstable; repository baseline is missing; design checkpoints disagree; required fan-out is blocked/ineligible; a contract/data/security/reliability/delivery/test owner still has a live planning-critical fork; or proof and rollout cannot be made honest. Reject filler artifacts, hidden generated authority, cross-service dual writes/ACID, vague package placement, unexplained legacy surfaces, unapproved dependencies/patterns, and `PASS` when planning must still design.

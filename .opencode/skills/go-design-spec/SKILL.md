---
name: go-design-spec
description: "Assemble and reconcile the integrated technical-design bundle for Go services. Use when `spec.md` is approved but non-trivial work still needs coherent task-local `design/` artifacts and cross-domain reconciliation before `planning-and-task-breakdown`. Skip when the task is a local code fix, pure spec authoring, direct-path work with an explicit design-skip rationale, implementation coding, review execution, or CI/container setup."
---

# Go Design Spec

## Purpose
Act as the integrator for task-local technical design: reconcile architecture, API, data, reliability, security, observability, and testing implications; reduce accidental complexity; and leave `design/` stable enough for planning without reopening the approved problem frame.

## Scope
Use this skill to run an integrated technical-design pass: reduce accidental complexity, remove contradictions, preserve maintainability, keep architecture, API, data, reliability, security, observability, and testing implications coherent, and leave the task-local design stable enough for task breakdown.

## Boundaries
Do not:
- replace domain-specific expert decisions with generic style advice
- treat this skill as final `spec.md` assembly; `spec-document-designer` owns `spec.md`
- make new problem-framing decisions that belong back in `spec.md` or the orchestrator
- produce task breakdown, phase cards, or coder execution sequencing; that belongs to `planning-and-task-breakdown`
- introduce new complexity without proving what risk or ambiguity it removes
- drift into implementation coding, review execution, or tooling/process detail as the main output
- leave cross-domain contradictions unresolved inside the design bundle

## Escalate When
Escalate if:
- `spec.md` is missing, unstable, or still contradicts itself in planning-critical ways
- the design is internally inconsistent or key assumptions differ across domains
- a required design artifact cannot be completed honestly without reopening `spec.md`
- critical behavior is not testable, operable, or rollout-safe
- repository baseline context from `docs/repo-architecture.md` materially matters and has not been loaded yet

## Reference Files
Use references lazily. Load repo-native task artifacts and repository docs first, then open at most one reference by default: the one that matches the active design pressure. Load multiple references only when the task clearly spans independent pressures, such as both runtime failure sequencing and a new abstraction boundary.

References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Each file exists to change a likely design choice. If a reference exposes a missing final decision, escalate to the orchestrator or the appropriate specialist instead of deciding inside this integrator skill. If a reference exposes missing execution sequencing, hand off to `planning-and-task-breakdown` instead of writing `tasks.md`.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| The design bundle shape is unclear, conditional artifacts are being created "for completeness", or `spec.md`, `design/`, and `tasks.md` are starting to absorb each other's jobs. | [design-bundle-assembly.md](references/design-bundle-assembly.md) | Makes the model produce a minimal, indexed design bundle with real artifact triggers instead of filler artifacts or disguised spec/planning content. |
| `design/component-map.md` or `design/ownership-map.md` needs package responsibility, source-of-truth, generated-code, or dependency-direction decisions. | [component-and-ownership-maps.md](references/component-and-ownership-maps.md) | Makes the model name concrete owners and stable boundaries instead of inventing shared helpers, treating generated files as authorities, or hiding ownership in "common" packages. |
| `design/sequence.md` needs runtime order, failure points, side effects, retries, sync/async boundaries, or partial-failure policy. | [runtime-sequence-and-failure-points.md](references/runtime-sequence-and-failure-points.md) | Makes the model write scenario-level runtime flow with failure ownership instead of a happy-path arrow chain or ambiguous sync/async finality. |
| Specialist outputs or design artifacts disagree across architecture, API, data, security, reliability, observability, delivery, or QA. | [cross-domain-reconciliation.md](references/cross-domain-reconciliation.md) | Makes the model reconcile by selected option, rejected options, and proof obligations instead of smoothing contradictions into a vague compromise. |
| The bundle is about to be marked planning-ready or handed to `planning-and-task-breakdown`. | [design-readiness-and-planning-handoff.md](references/design-readiness-and-planning-handoff.md) | Makes the model block or qualify readiness with artifact status, risks, and reopen conditions instead of saying "done enough" while planning must rediscover design. |
| A proposed layer, interface, helper, adapter, shared package, or simplification may change ownership or widen impact radius. | [complexity-and-abstraction-tradeoffs.md](references/complexity-and-abstraction-tradeoffs.md) | Makes the model require present-day complexity reduction and contract preservation instead of future-proof indirection or simplification that weakens guarantees. |

## Specialist Stance
- `spec.md` owns decisions, `design/` owns task-local technical context, and `tasks.md` consumes approved `spec.md + design/` for the implementation handoff.
- Prefer the simplest explicit design that satisfies current requirements and preserves change locality.
- Treat accidental complexity as a blocker when it increases integration risk or widens impact radius without clear benefit.
- Prefer additive, compatibility-first evolution over big-bang replacement.
- Preserve specialist ownership: integrate and challenge domain decisions, but do not replace architecture, data, security, observability, or QA expertise.
- Prefer one coherent design-bundle handoff over scattered partial notes that still force planning to rediscover technical context.
- Keep `design/overview.md` as the bundle entrypoint instead of repeating the same story in every artifact; when the bundle is planning-bound, its artifact index should include status and trigger rationale for required and plausible conditional artifacts.

## Boundaries And Handoffs
This is a technical-design integrator, not a workflow owner:
- use repository artifacts when they are present, but do not redefine when phases start or stop
- if `spec.md` is missing or unstable, hand back to specification instead of inventing decisions inside design
- if planning or implementation details appear, keep only the design constraints that planning must consume and hand execution sequencing to `planning-and-task-breakdown`
- if one domain seam becomes the real hard problem, hand off to that specialist instead of flattening it into a generic integrated design note

## Expertise

### Design Bundle Assembly
- Produce or tighten the required core artifacts for non-trivial work:
  - `design/overview.md` for chosen approach, artifact index with planning-bound artifact status and conditional trigger rationale, unresolved seams, and readiness summary
  - `design/component-map.md` for affected packages, modules, or components; responsibilities; and what changes versus what stays stable
  - `design/sequence.md` for call order, sync or async boundaries, failure points, side effects, and parallel versus sequential behavior
  - `design/ownership-map.md` for source-of-truth ownership, allowed dependency direction, and responsibility boundaries
- Add conditional artifacts only when their trigger is real:
  - `design/data-model.md` when persisted state, schema, cache contract, projections, replay behavior, or migration shape changes
  - `design/dependency-graph.md` when dependency shape or generated-code flow changes or a coupling risk must be made explicit
  - `design/contracts/` when API, event, generated, or material internal interface contracts change
- Call out when `test-plan.md` or `rollout.md` must exist before planning can start, but do not turn this skill into execution planning.

### Complexity And Maintainability
- Avoid speculative abstractions, indirection layers, interface-per-struct patterns, and service-manager-factory chains that do not remove concrete present-day complexity.
- Require every abstraction to justify:
  - what problem it removes now
  - why a simpler alternative was rejected
  - what maintenance and change-radius cost it introduces
- Prefer explicit boundaries, explicit control flow, and predictable dependency direction over hidden magic.
- Optimize for local change paths and bounded impact radius.

### Boundary And Ownership Consistency
- When boundaries are touched, check them against domain capability, data ownership, team ownership, and transaction boundary.
- Require explicit source-of-truth ownership for critical entities and cross-service flows.
- Reject design narratives that quietly rely on shared-schema coupling, cross-service direct DB access, or cross-service ACID.
- Surface distributed-monolith signals early: coordinated releases, chatty dependency graphs, hidden shared logic, or cross-service flow ownership ambiguity.

### Sync And API Seams
- Verify sync vs async choice before discussing transports or endpoints.
- For sync seams, require explicit deadline budgets, retry or no-retry classes, side-effect idempotency policy, and error model; add pagination behavior only for collection or list semantics.
- Guard against action-RPC drift hiding inside nominally resource-oriented APIs.
- Make eventual-consistency disclosure explicit when sync read behavior depends on async convergence.

### Async And Distributed Seams
- Require explicit event vs command intent and a justified choice of pub/sub vs queue.
- For side-effecting async flows, require a named atomicity and idempotency or dedup model, such as transactional outbox plus idempotent consumer or an equivalent guarantee.
- When cross-service invariants exist, require an explicit process or saga state model.
- Make compensation or forward-recovery semantics explicit for each critical distributed step.
- Reject dual writes and unscoped exactly-once assumptions. If a platform claims exactly-once behavior, state the guarantee boundary and require idempotent handling for external side effects.

### Data, Cache, And Evolution Integrity
- Keep local transaction boundaries explicit and aligned with ownership boundaries.
- Require rollout-sensitive persisted-state evolution to use `expand -> backfill/verify -> contract` with a mixed-version compatibility window.
- Require cache decisions to preserve correctness: clear staleness contract, tenant-safe keying, invalidation/fallback behavior, and no hidden dependency on exact TTL timing.
- Do not allow data/cache assumptions to silently break domain behavior during rollout.

### Security, Observability, Delivery, And Reliability Seams
- Require trust boundaries, validation expectations, and fail-closed authorization assumptions to be explicit where they affect behavior.
- Require observability to remain actionable: trace/log/metric correlation must survive changed critical paths, and metric cardinality must stay bounded.
- Ensure proposed design remains enforceable by CI, migration validation, contract checks, and release controls.
- Require per-dependency timeout, retry, fallback, overload, and rollback assumptions for critical paths.
- Reject designs that depend on heroic manual operations or undocumented release choreography.

## Design Readiness Bar
For each planning-critical design recommendation that chooses between real trade-offs, make clear:
- the complexity symptom or integration risk
- at least two viable options
- the selected option and at least one explicit rejection reason
- trade-offs across simplicity, flexibility, cost, risk, and change impact
- cross-domain impact on architecture, API, data, security, observability, reliability, and testing
- assumptions, blockers, and reopen conditions

## Deliverable Shape
When writing or reviewing the integrated technical-design bundle, cover:
- the required core `design/` artifacts and any triggered conditional artifacts
- contradictions across domains
- simplification opportunities
- abstractions or layers that should be removed, merged, or made explicit
- what changes versus what remains stable
- runtime sequence, ownership boundaries, and any data, contract, or dependency edges that planning must respect
- downstream consequences for API, data, reliability, security, observability, and testing
- what must loop back into `spec.md` before planning can safely begin
- whether `design/` is stable enough for `planning-and-task-breakdown`
- the planning handoff boundary and any reason the next session must reopen `spec.md` instead of moving forward
- unresolved design risks that should block implementation

## Escalate Or Reject
- missing or unstable `spec.md`
- any hidden “decide later in coding” system-level gap
- contradictory assumptions left unresolved across domain specs
- a new abstraction or layer with no measurable simplification outcome
- simplification that weakens API, data, reliability, or security contracts
- migration, cache, retry, or degradation assumptions that are not rollout-safe
- design rationale based on taste instead of workload, constraints, and operating cost

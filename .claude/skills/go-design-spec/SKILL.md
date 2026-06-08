---
name: go-design-spec
description: "Assemble and reconcile integrated technical-design context for Go services when separate design depth is triggered. Use when `spec.md` is approved but the work still needs coherent task-local `design/` artifacts, one `design/overview.md`, cross-domain reconciliation before the mandatory technical design review, or read-only technical-design-review analysis before `planning-and-task-breakdown`. Skip when the task is a local code fix, pure spec authoring, direct-path work, lean-local work with sufficient compact design in `spec.md`, implementation coding, post-code review execution, or CI/container setup."
---

# Go Design Spec

## Purpose
Act as the integrator for task-local technical design: reconcile architecture, API, data, reliability, security, observability, and testing implications; reduce accidental complexity; and leave compact or split design context stable enough for mandatory technical design review by closing the current decision frontier without reopening the approved problem frame.

## Outcome-First Operating Rules
- Start by naming the skill-specific outcome, success criteria, constraints, available evidence, and stop rule.
- Treat workflow steps as decision rules, not a ritual checklist. Follow exact order only when this skill or the repository contract makes the sequence an invariant.
- Use the minimum context, references, tools, and validation loops that can change the deliverable; stop expanding when the quality bar is met.
- Before acting, resolve prerequisite discovery, lookup, or artifact reads that the outcome depends on; parallelize only independent evidence gathering and synthesize before the next decision.
- Prefer bounded assumptions and local evidence over broad questioning; ask only when a missing fact would change correctness, ownership, safety, or scope.
- When evidence is missing or conflicting, retry once with a targeted strategy or label the assumption, blocker, or reopen target instead of treating absence as proof.
- Finish only when the requested deliverable is complete in the required shape and verification or a clearly named blocker/residual risk is recorded.

## Scope
Use this skill to run an integrated technical-design pass or technical-design-review analysis: reduce accidental complexity, remove contradictions, preserve maintainability, keep architecture, API, data, reliability, security, observability, and testing implications coherent, and leave the task-local design stable enough for review and later task breakdown without expanding every visible downstream effect into new design work.

## Boundaries
Do not:
- replace domain-specific expert decisions with generic style advice
- treat this skill as final `spec.md` assembly; `spec-document-designer` owns `spec.md`
- make new problem-framing decisions that belong back in `spec.md` or the orchestrator
- produce task breakdown, phase cards, or coder execution sequencing; that belongs to `planning-and-task-breakdown`
- introduce new complexity without proving what risk or ambiguity it removes
- drift into implementation coding, post-code review execution, or tooling/process detail as the main output
- leave cross-domain contradictions unresolved inside the design bundle

## Escalate When
Escalate if:
- `spec.md` is missing, unstable, or still contradicts itself in planning-critical ways
- the design is internally inconsistent or key assumptions differ across domains
- required compact or split design context cannot be completed honestly without reopening `spec.md`
- critical behavior is not testable, operable, or rollout-safe
- repository baseline context from `docs/repo-architecture.md` materially matters and has not been loaded yet

## Reference Files
Use references lazily. Load repo-native task artifacts and repository docs first, then open at most one reference by default: the one that matches the active design pressure. Load multiple references only when the task clearly spans independent pressures, such as both runtime failure sequencing and a new abstraction boundary. This limit applies to this skill's reference files, not to consuming multiple specialist lane outputs produced by the workflow.

References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Each file exists to change a likely design choice. If a reference exposes a missing final decision, escalate to the orchestrator or the appropriate specialist instead of deciding inside this integrator skill. If a reference exposes missing execution sequencing, hand off to `planning-and-task-breakdown` instead of writing `tasks.md`.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| The design bundle shape is unclear, conditional artifacts are being created "for completeness", or `spec.md`, `design/`, and `tasks.md` are starting to absorb each other's jobs. | [design-bundle-assembly.md](references/design-bundle-assembly.md) | Makes the model produce a minimal, indexed design bundle with real artifact triggers instead of filler artifacts or disguised spec/planning content. |
| `design/component-map.md` or `design/ownership-map.md` needs package responsibility, source-of-truth, generated-code, or dependency-direction decisions. | [component-and-ownership-maps.md](references/component-and-ownership-maps.md) | Makes the model name concrete owners and stable boundaries instead of inventing shared helpers, treating generated files as authorities, or hiding ownership in "common" packages. |
| `design/sequence.md` needs runtime order, failure points, side effects, retries, sync/async boundaries, or partial-failure policy. | [runtime-sequence-and-failure-points.md](references/runtime-sequence-and-failure-points.md) | Makes the model write scenario-level runtime flow with failure ownership instead of a happy-path arrow chain or ambiguous sync/async finality. |
| Specialist outputs or design artifacts disagree across architecture, API, data, security, reliability, observability, delivery, or QA. | [cross-domain-reconciliation.md](references/cross-domain-reconciliation.md) | Makes the model reconcile by selected option, rejected options, and proof obligations instead of smoothing contradictions into a vague compromise. |
| The bundle is about to be marked review-ready, planning-ready, or handed to `planning-and-task-breakdown`. | [design-readiness-and-planning-handoff.md](references/design-readiness-and-planning-handoff.md) | Makes the model block or qualify readiness with artifact status, risks, review-gate needs, and reopen conditions instead of saying "done enough" while planning must rediscover design. |
| A proposed layer, interface, helper, adapter, shared package, or simplification may change ownership or widen impact radius. | [complexity-and-abstraction-tradeoffs.md](references/complexity-and-abstraction-tradeoffs.md) | Makes the model require present-day complexity reduction and contract preservation instead of future-proof indirection or simplification that weakens guarantees. |

## Specialist Stance
- `spec.md` owns decisions, lean `Compact Design` or triggered `design/` owns task-local technical context, and `tasks.md` consumes approved decisions plus required design context for the implementation handoff.
- Separate design depth must pass technical design review before planning; this skill may prepare the design for that gate or support a read-only review lane, but it does not make final approval decisions.
- When supporting a technical-design-review lane, classify each finding as `blocks_planning`, `reopens_design`, `reopens_spec`, `accepted_risk_candidate`, `proof_obligation`, or `record_only`, then recommend `PASS`, `CONCERNS`, or `FAIL` with status rationale.
- In technical-design-review mode, judge whether planning can produce implementation-ready tasks without inventing architecture, ownership, contract, sequencing, rollout, or validation policy. If planning would need to make a design choice, the review should reopen design or specification instead of passing with a proof-only concern.
- Before selecting custom infrastructure, a new runtime dependency, or a meaningful helper/abstraction, compare the current Go standard library, established repository patterns, and mature open-source options. Design may proceed only when the selected option and rejected alternatives have current evidence for maintenance, adoption, license, security, transitive dependency cost, API stability, and repository-boundary fit.
- Before selecting an architecture, workflow, integration, resilience, consistency, data-flow, or abstraction shape, perform Pattern Fit Diligence. Search for known applicable design or system-design patterns, read concrete descriptions and real-use examples, compare candidates against the task's forces, repository boundaries, operational proof path, and idiomatic Go fit, and record selected and rejected patterns. Design may proceed with a custom shape only when known patterns fail a concrete requirement.
- Prefer the simplest explicit design that satisfies current requirements and preserves change locality.
- Treat accidental complexity as a blocker when it increases integration risk or widens impact radius without clear benefit.
- Prefer additive, compatibility-first evolution over big-bang replacement.
- Preserve specialist ownership: integrate and challenge domain decisions, but do not replace architecture, data, security, observability, or QA expertise.
- For non-trivial split or review-bound design, do not flatten independently planning-critical domain seams into one integrator pass. Request or consume multiple narrow read-only specialist lane outputs by default when more than one domain can change ownership, contracts, persistence, failure behavior, validation, or rollout.
- Domain impact alone is not a fan-out trigger. If the approved artifacts and repository evidence show the domain has no new decision to make, keep it inside the integrated pass and classify it as `constraint_only`, `proof_only`, `follow_up_only`, or `no new decision required`.
- If the integrator proceeds locally without specialist fan-out, record `Design fan-out rationale:` with the candidate seams considered, collapsed seam classifications, and escalation seams.
- Prefer one coherent design handoff over scattered partial notes that still force planning to rediscover technical context.
- Keep lean-local design merged in `spec.md` when it is sufficient; otherwise keep `design/overview.md` as the entrypoint instead of repeating the same story in every artifact. When the bundle is review-bound, its artifact index should include status and trigger rationale for required and plausible conditional artifacts.
- Keep deep design and corner-case coverage, but distinguish `must decide now` from `dependent consequence only`.
- Open another domain or artifact only when the unresolved point materially changes ownership, correctness model, public contract, safe sequencing, or rollout shape for the current bundle.
- When another domain is affected but unchanged, record `constraint_only`, `proof_only`, `follow_up_only`, or explicit `no new decision required in <domain>` rather than growing the bundle into a parallel specialist package.
- Treat split design artifacts as required questions only when split design is triggered, not equal-length documents. Uneven depth is normal: the hard issue may live in one artifact while another stays short because the boundary is stable.
- Prefer the smallest coherent bundle that lets technical design review and then planning proceed honestly. Do not inflate unaffected surfaces just to make the bundle look balanced.

## Boundaries And Handoffs
This is a technical-design integrator, not a workflow owner:
- use repository artifacts when they are present, but do not redefine when phases start or stop
- if `spec.md` is missing or unstable, hand back to specification instead of inventing decisions inside design
- if planning or implementation details appear, keep only the design constraints that technical design review and planning must consume and hand execution sequencing to `planning-and-task-breakdown`
- if one or more domain seams become the real hard problem, hand off to the relevant specialist lane set instead of flattening them into a generic integrated design note

## Expertise

### Design Bundle Assembly
- Produce or tighten split core artifacts when their trigger is real:
  - `design/overview.md` for chosen approach, artifact index with review-bound artifact status and conditional trigger rationale, unresolved seams, and readiness summary
  - `design/component-map.md` for affected packages, modules, generated surfaces, adapters, and components; responsibilities; what changes versus what stays stable; and which plausible surfaces are intentionally not touched
  - `design/sequence.md` for call order, sync or async boundaries, failure points, side effects, recovery or retry boundaries when relevant, and parallel versus sequential behavior
  - `design/ownership-map.md` for source-of-truth ownership, allowed dependency direction, generated-code authority, adapter responsibility, and explicit non-owners for critical behavior
- Keep each triggered artifact only as detailed as its current issue demands. A narrow change can justify very short component, sequence, or ownership notes as long as they explicitly preserve unchanged boundaries and planning does not need to infer what stayed stable.
- For replacement designs, make source-of-truth and generated-artifact consequences explicit: which old code, tests, fixtures, configs, docs, generated outputs, skills, agents, or mirrors are removed, refactored into the active path, retained with owner/reason/proof/exit condition, or routed back to specification because removal changes an approved boundary.
- Add conditional artifacts only when their trigger is real:
  - `design/data-model.md` when persisted state, schema, cache contract, projections, replay behavior, or migration shape changes
  - `design/dependency-graph.md` when dependency shape or generated-code flow changes or a coupling risk must be made explicit
  - `design/contracts/` when API, event, generated, or material internal interface contracts change
- Call out when technical design review, `test-plan.md`, or `rollout.md` must exist before planning can start, but do not turn this skill into execution planning.
- For lean-local work, prefer `spec.md` `Compact Design` or one `design/overview.md` when that honestly answers affected surfaces, ownership/source-of-truth, and sequence/failure behavior.

### Complexity And Maintainability
- Avoid speculative abstractions, indirection layers, interface-per-struct patterns, and service-manager-factory chains that do not remove concrete present-day complexity.
- Prefer maintained OSS over custom infrastructure when it satisfies the approved contract with lower ownership cost and acceptable license/security/adoption signals. Prefer custom code only when stdlib, established repo patterns, and mature OSS candidates fail a concrete contract, ownership, operational, or integration requirement.
- Require every abstraction to justify:
  - what problem it removes now
  - why a simpler alternative was rejected
  - what maintenance and change-radius cost it introduces
- Require every selected design or system-design pattern to justify:
  - what task force or failure mode it addresses now
  - which known alternatives were rejected and why
  - how the pattern stays idiomatic in Go: explicit control flow, small interfaces, context-aware I/O, package ownership, and simple composition
  - what proof will show the implementation preserved the pattern's guarantee rather than only its vocabulary
- Prefer explicit boundaries, explicit control flow, and predictable dependency direction over hidden magic.
- Optimize for local change paths and bounded impact radius.

### Boundary And Ownership Consistency
- When boundaries are touched, check them against domain capability, data ownership, team ownership, and transaction boundary.
- Require explicit source-of-truth ownership for critical entities and cross-service flows.
- Require retained legacy surfaces to have a current owner, reason, proof of continued need, and exit condition; otherwise design must choose removal/refactor or reopen the owning phase.
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
For each planning-critical design recommendation, make clear:
- the complexity symptom or integration risk
- whether a real `live fork` exists, meaning two plausible approaches would materially change ownership, interfaces, persistence shape, async model, operability, or rollout
- when a `live fork` exists, the viable options, the selected option, and at least one explicit rejection reason
- when no `live fork` exists, the chosen repo-consistent approach and why no competing option needs current design treatment
- for dependency-sensitive choices, the selected stdlib/repo-pattern/OSS/custom option, rejected options, current evidence signals, and why planning should not reopen library selection
- for pattern-sensitive choices, the selected design/system pattern or straightforward repo-native design, rejected pattern alternatives, source descriptions or examples, applicability, Go-fit, and why planning should not reopen pattern selection
- trade-offs across simplicity, flexibility, cost, risk, and change impact
- only the downstream effects that force a new decision, handoff, or proof obligation in architecture, API, data, security, observability, reliability, or testing
- assumptions, blockers, and reopen conditions only when they affect the current bundle or the first safe implementation slice
- avoid widening the bundle just to produce symmetrical coverage across artifacts or domains

## Technical Design Review Decision Quality
When this skill is used for read-only technical design review, do not only list defects. Make the gate decision auditable:

- Name the planning decision at risk for each material finding: ownership, interface, persistence, runtime sequence, failure/recovery, validation, rollout, dependency direction, or implementation proof.
- Identify any real `live fork`, meaning two plausible options would materially change the task ledger. If a live fork exists and the design has not selected one, recommend `FAIL` and reopen `technical design` or `specification`.
- For `CONCERNS`, prove the issue is a bounded risk that implementation can validate without choosing a missing design. Name the exact proof obligation and where planning should carry it.
- For `FAIL`, name the smallest reopen target, the decision or artifact that must change, and the condition a follow-up review must verify.
- For `PASS`, name the falsification checks performed, such as source-of-truth ownership, dependency direction, sequence/failure behavior, validation path, rollout assumptions, and conditional artifact triggers.
- For dependency-sensitive design, include dependency/OSS due-diligence in the falsification checks; missing current evidence for a new dependency or custom infrastructure is a planning blocker, not a coding preference.
- For pattern-sensitive design, include Pattern Fit Diligence in the falsification checks; missing known-pattern comparison for an invented design shape is a planning blocker, and cargo-cult pattern naming without task-specific Go/repository fit should reopen design.
- Include one strongest counterargument or simpler alternative for each material recommendation and explain why it does not change the result.
- Downgrade taste, local style, or optional cleanup to `record_only` unless it creates concrete planning or production-readiness risk.

## Deliverable Shape
When writing or reviewing the integrated technical-design bundle, cover:
- the required compact or split `design/` artifacts and any triggered conditional artifacts
- contradictions across domains
- simplification opportunities
- abstractions or layers that should be removed, merged, or made explicit
- for technical-design-review mode, the reviewed packet, classified findings, required reopen targets or planning proof obligations, and recommended gate result
- what changes versus what remains stable
- runtime sequence, ownership boundaries, and any data, contract, or dependency edges that planning must respect
- dependency/OSS due-diligence outcome when relevant: selected and rejected stdlib, repository-pattern, OSS, and custom options with current evidence and planning consequences
- Pattern Fit Diligence outcome when relevant: selected and rejected design/system patterns, source descriptions or examples, applicability, Go/repository fit, and planning consequences
- legacy-surface treatment when replacement work is in scope: remove/refactor/retain decisions, generated or mirror source-of-truth order, and retained-surface exit conditions
- downstream consequences as:
  - `forces new decision`
  - `forces handoff`
  - `forces proof obligation`
  - `no new decision required`
- concise stable or unchanged notes where a required artifact has little to say beyond preserved boundaries
- what must loop back into `spec.md` before planning can safely begin
- whether compact or split design context is stable enough for technical design review and later `planning-and-task-breakdown`
- the technical-design-review handoff boundary and any reason the next session must reopen `spec.md` instead of moving forward
- unresolved design risks that should block implementation

## Escalate Or Reject
- missing or unstable `spec.md`
- any hidden “decide later in coding” gap that would change ownership, correctness, contract, sequencing, or rollout
- contradictory assumptions left unresolved across domain specs
- a new abstraction or layer with no measurable simplification outcome
- a new dependency, custom infrastructure decision, or material helper/abstraction without recorded stdlib, repository-pattern, and mature-OSS due diligence
- an invented architecture, workflow, integration, resilience, consistency, data-flow, or abstraction shape without recorded Pattern Fit Diligence
- a named design/system pattern applied without evidence that its forces match this task and its implementation can stay idiomatic in Go
- simplification that weakens API, data, reliability, or security contracts
- migration, cache, retry, or degradation assumptions that are not rollout-safe
- design rationale based on taste instead of workload, constraints, and operating cost

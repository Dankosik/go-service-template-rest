# Spec-First Workflow (Orchestrator/Subagent-First)

## 1. Purpose

This document is the detailed runtime companion to [AGENTS.md](../AGENTS.md) for tasks that need more than a quick local fix:

- non-trivial or agent-backed work,
- preserved research,
- multi-session resume,
- `technical design` and later.

It explains artifact placement, timing, relationships, and resume order.

`AGENTS.md` stays compact; this document carries the detailed mechanics.

### Authority

`AGENTS.md` stays authoritative for role ownership, invariants, subagent protocol, and stage gates. If the two documents diverge, follow `AGENTS.md` and then repair this file.

### Repository Baseline

When the task depends on stable repository boundaries or runtime flows, load [docs/repo-architecture.md](./repo-architecture.md) before writing task-local design.

## 2. What This Workflow Does Not Change

More artifacts do not change ownership:

- **Orchestrator:** owns framing, routing, final decisions, implementation, validation, and artifact authority.
- **Subagents:** stay read-only and advisory.
- **Skills:** remain optional tools. For subagent-internal research or review skills, the orchestrator routes by skill name and the subagent loads the skill body inside its own pass.
- **Agent instructions vs skills:** agent instructions own scope, mode routing, and handoff; when a chosen skill defines a procedure or output shape, the skill owns that procedure or output shape.
- **`spec.md`:** remains the canonical decisions artifact.

The added artifacts provide control and technical context around `spec.md`, not another authority chain.

The `workflow-status` skill is a read-only status and next-action helper over those artifacts. It can summarize current phase, blockers, allowed writes, stop rule, implementation-readiness status, and implementation-start status, but it is not a session phase, gate, approval record, or replacement for `workflow-plan.md`, `workflow-plans/<phase>.md`, `spec.md`, `design/`, or `tasks.md`.

The `workflow-plan-adequacy-challenge` skill is a read-only challenger for generated workflow-control artifacts. It helps the orchestrator decide whether `workflow-plan.md` and the active `workflow-plans/<phase>.md` are sufficient for the actual task before handoff; it does not approve the handoff or edit files.

## 3. Artifact Model

### Layout

Smallest valid task-local layout: `specs/<feature-id>/spec.md`

Repository-wide stable architecture baseline: `docs/repo-architecture.md`

Non-trivial task-local bundle:

```text
specs/<feature-id>/
  workflow-plan.md
  workflow-plans/
    workflow-planning.md   # conditional; dedicated pre-research routing
    research.md            # conditional; dedicated research routing
    specification.md
    technical-design.md
    planning.md
    review-phase-1.md           # conditional
    validation-phase-1.md       # conditional
  spec.md
  design/
    overview.md
    component-map.md
    sequence.md
    ownership-map.md
    data-model.md          # conditional
    dependency-graph.md    # conditional
    contracts/             # conditional
  research/
    <topic>.md
  tasks.md
  test-plan.md             # conditional
  rollout.md               # conditional
```

### Artifact Purposes

- **`workflow-plan.md`:** Master routing and control artifact. It tracks cross-phase status: execution shape, current phase, artifact status, blockers, next session, next-session context bundle, links to phase workflow plans, and resume order. Required for non-trivial or agent-backed work. The next-session context bundle is an always-present field for non-trivial workflow plans: it either says the default resume order is sufficient or names the exact task-specific files the next session should read and why each one matters. It links to artifacts instead of copying their content.
- **`workflow-plans/<phase>.md`:** Phase-local workflow plan for one named phase only. It tracks that phase's local orchestration, order/parallelism, fan-in/challenge path when relevant, completion marker, stop rule, next action, and local blockers. Required for each named non-trivial phase that the task actually uses, including `workflow-planning` or `research` when those are dedicated sessions.
- **`spec.md`:** Canonical decision record: approved framing, scope, constraints, decisions, and accepted open questions. Always.
- **`design/`:** Task-local technical design bundle between `spec.md` and task breakdown. It explains how the approved change fits the repository architecture and what implementation must preserve. Required for non-trivial work unless explicitly skipped with rationale.
- **`tasks.md`:** Executable task ledger and final implementation handoff derived from approved `spec.md + design/`. It owns an optional compact implementation handoff header plus markdown checkboxes with stable task IDs, checkpoint labels when useful, optional `[P]` markers only for safe parallel work, concrete action and file/package surfaces, dependency markers when nontrivial, and proof expectations. Task items may use short continuation lines for dependency, proof, accepted concern, or reopen detail when one-line bullets would become dense, but they must remain executable ledger items rather than design notes. Expected by default for non-trivial implementation work; direct-path or tiny work may skip it only with an explicit waiver.
- **Implementation readiness:** Planning-phase exit gate recorded as `PASS`, `CONCERNS`, `FAIL`, or `WAIVED`. The status belongs in `workflow-plan.md`, the result and stop or handoff rule belong in `workflow-plans/planning.md`, and `tasks.md` may carry only a short reference when useful.
- **`research/*.md`:** Preserved evidence, comparisons, and validated research context. These files support decisions but do not own them. Create only when the task is long, ambiguous, or benefits from reusable research memory.
- **Artifact status vocabulary:** Use `approved`, `draft`, `missing`, `blocked`, `waived`, `not expected`, or `conditional` as the task needs. `waived` requires an eligible direct/local rationale and scope; `not expected` requires a short trigger-based reason when the artifact would otherwise be plausible; `conditional` means a later phase must decide the trigger instead of creating the artifact early.
- **`test-plan.md`:** Expanded validation strategy when test obligations are too large or too layered to fit cleanly inside `tasks.md`. Conditional.
- **`rollout.md`:** Operational rollout and migration notes when delivery order, compatibility, or recovery choreography matters. Conditional.

### Artifact Rules

- **Repository baseline:** [docs/repo-architecture.md](./repo-architecture.md) is the stable repository baseline, not a task-local design file.
- **Workflow control:** `workflow-plan.md` stays live through research, synthesis, `technical design`, planning, coding/execution, and validation; `workflow-plans/<phase>.md` stays phase-local and must not turn into a second `design/` bundle or `tasks.md`.
- **Pre-code phase plans:** Dedicated workflow-planning and research sessions use `workflow-plans/workflow-planning.md` and `workflow-plans/research.md`. Later pre-code phases normally get `workflow-plans/specification.md`, `workflow-plans/technical-design.md`, and `workflow-plans/planning.md`.
- **Post-code phase-control files:** Coding/execution does not get a phase-control file. Create `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` only when a named multi-session review or validation checkpoint truly needs phase-local routing beyond `workflow-plan.md` and `tasks.md`.
- **Artifact ownership:** `spec.md` stores decisions, `design/` stores task-specific technical context, and `tasks.md` stores executable task state and the implementation handoff. Do not make any of them absorb another artifact's job.
- **Research shape:** `research/*.md` should be flexible and evidence-oriented; there is no mandatory universal template. When preserved, a research note should make the question or scope, findings with evidence and limits, conflicts or open points, source notes, and handoff implication visible enough that later synthesis does not need chat memory. Copy the evidence discipline, not the headings.
- **Research fan-in home:** Store the durable fan-in summary in one owning place, normally `workflow-plans/research.md` for routing plus selected `research/*.md` for reusable evidence. Other files should link or summarize status only; do not copy the full fan-in narrative into both master workflow control and research notes.
- **No duplicate authority:** Do not duplicate the same authority across artifacts. Link instead.
- **Task-local examples:** Completed bundles under `specs/` can be useful examples of how a task used the workflow, but they are historical task-local records, not universal templates or alternate authority. Copy only the pattern that fits the current task and keep trigger decisions local. If an older bundle mentions a task-specific supplemental note, do not treat that note as a new workflow artifact type unless this document or `AGENTS.md` names it.

### Artifact Shape Matrix

Use this matrix as the context-first minimum. It is not a rigid template; omit empty sections and keep task-specific depth proportional to risk.

| Artifact | Minimum context to preserve | Must not own | Resume role |
| --- | --- | --- | --- |
| `workflow-plan.md` | Current phase, phase status, session boundary, next session, always-present next-session context bundle, artifact status with trigger rationale, blockers, accepted risks, reopen targets, and active gate status. | Final decisions, technical design, executable tasks, raw research or review transcripts. | First file to read; tells the next session where to resume and which artifacts matter. |
| `workflow-plans/<phase>.md` | Phase-local status, order or lanes, completion marker, stop rule, next action, local blockers, parallelizable work, and phase-specific gate or handoff state. | Cross-phase authority, `spec.md` decisions, design bundle content, task ledger entries, implementation logs. | Second file to read; tells the next session how the active phase is locally routed. |
| `spec.md` | Context, scope and non-goals, constraints, decisions, labeled assumptions or open questions, handoff link, validation obligations, and evidence-backed outcome only after proof. | Raw research dumps, component maps, runtime sequence design, task lists, validation success claims before proof. | Canonical decision record for all later design, planning, implementation, review, and validation work. |
| `design/overview.md` | Chosen approach, artifact index with required artifact status, conditional trigger rationale when planning-bound, unresolved seams, and readiness summary. | Final behavior decisions, task IDs, implementation sequence, raw research. | Design entrypoint for planning; links to the rest of the bundle without copying it. |
| `design/component-map.md` | Affected packages, modules, generated surfaces, adapters, stable surfaces, and responsibility changes. | T001-style execution order or speculative package rewrites. | Shows planning which surfaces change and which must remain stable. |
| `design/sequence.md` | Runtime order, sync or async boundaries, side effects, failure points, recovery or retry boundaries when relevant, and parallel versus sequential behavior. | Happy-path-only prose or implementation task sequencing. | Preserves flow context so planning does not rediscover ordering and failure semantics. |
| `design/ownership-map.md` | Source-of-truth ownership, dependency direction, generated-code authority, adapter responsibility, and explicit non-owners for critical behavior. | "Decide during implementation" ownership gaps or new product decisions. | Preserves boundary context for planning and review. |
| Conditional `design/*`, `test-plan.md`, `rollout.md` | Trigger and scope, artifact-specific obligations, exit criteria, and links back to the owner artifact or task IDs. | Just-in-case filler or duplicate runtime contract authority. | Loaded only when triggered or named in the next-session context bundle. |
| `tasks.md` | Optional compact implementation handoff, markdown checkboxes, stable task IDs, checkpoint labels, safe `[P]` markers, concrete action and surface, dependencies when nontrivial, proof expectations, and reopen notes when needed. | New spec or design decisions, broad strategy memos, raw validation transcripts. | Final executable handoff before coding. |
| `research/*.md` | Question or scope, findings with evidence and limits, conflicts or weak evidence, source notes, and handoff implication. | Final decisions, design sequences, task lists, raw command dumps, link dumps. | Optional evidence memory for synthesis and later audit; supports but does not approve decisions. |

### Context-First Quality Bar

Every phase artifact should preserve enough durable context for the next LLM session to continue from files, without chat memory or rediscovery:

- name the current phase, owning artifact, next action, and stop rule;
- distinguish facts, decisions, assumptions, blockers, evidence, accepted risks, and reopen targets;
- link to the owning artifact instead of copying another artifact's content;
- name the next-session context bundle even when it only says the default resume order is enough;
- record why plausible optional artifacts are `not expected`, `conditional`, or `waived`;
- keep proof obligations proportional to the claim and changed surface;
- make the next-session route explicit enough that implementation, review, or validation does not need to reconstruct workflow intent from prose;
- avoid placeholders, raw transcripts, generic "run tests" language, stale status, and completion claims without fresh evidence.

### Artifact-Producing Vs Artifact-Consuming Phases

#### Pre-code phases

Pre-code phases (`workflow planning`, `research`, `specification`, `technical design`, and `planning`) may create only the approved workflow/process artifact set:

- `workflow-plan.md`
- `workflow-plans/<phase>.md`
- `research/*.md`
- `spec.md`
- `design/`
- `tasks.md`
- optional `test-plan.md`
- optional `rollout.md`

If later review or validation phase-control files are genuinely needed, planning records that need and creates only the named optional routing files before coding starts. Planning must not create coding phase-control files.

#### Post-code phases

Post-code work (`coding/execution`, `review`, `reconciliation`, and `validation`) consumes that bundle.

Implementation and reconciliation may create approved codebase files such as code, tests, migrations, configs, generation inputs, and generated output when the task ledger requires them. Review remains read-only and advisory. Validation gathers proof rather than fixes. Post-code work may update only existing control or closeout surfaces such as `workflow-plan.md`, an active review or validation phase file when one exists, checkpoint state in existing `tasks.md`, and `spec.md` `Validation` or `Outcome`.

If required context is missing, stop, record the reopen in existing control artifacts, and reopen the appropriate earlier phase in a new session instead of inventing new artifacts here.

#### Direct-path exceptions

Direct-path exceptions may skip parts of the pre-code artifact bundle with explicit rationale, but they do not authorize creating new workflow/process artifacts mid-implementation or mid-validation.

## 4. Default `spec.md` Shape

The default sections are:

1. `Context`
2. `Scope / Non-goals`
3. `Constraints`
4. `Decisions`
5. `Open Questions / Assumptions`
6. `Task Breakdown / Handoff Link`
7. `Validation`
8. `Outcome`

Rules:

- Merge sections when that makes the file clearer.
- Do not create empty headings.
- Keep final decisions in `Decisions`.
- Keep research evidence in `research/*.md` when it is worth preserving.
- Keep only a short task-breakdown or handoff link in `spec.md`.
- In `Open Questions / Assumptions`, label uncertainty by unblock path when it affects future sessions:
  - `[assumption]` for a bounded assumption the workflow can proceed with and revisit if false;
  - `[accepted_risk]` for a known risk deliberately carried forward with proof obligations or limits;
  - `[requires_user_decision]` for product, business, or policy choices repository evidence cannot decide;
  - `[targeted_research]` for missing evidence that research can answer;
  - `[defer_to_design]` only when behavior is decided but component, sequence, or ownership detail belongs in `design/`;
  - `[reopen_spec_if_false]` for downstream discoveries that would invalidate a spec-level assumption.
- Before implementation, `Validation` records forward-looking proof obligations. `Outcome` is omitted, clearly pending, or evidence-backed only after fresh validation; do not write success language before proof exists.

## 5. The Design Bundle Between `spec.md` And Task Breakdown

For non-trivial work, `technical design` is an explicit stage between `specification` and `planning`.

The design bundle carries task-specific technical context that `spec.md` should not absorb:

- repository fit,
- participating components,
- runtime sequence,
- ownership and source-of-truth boundaries,
- stable areas,
- correctness or rollout risks.

Load order for design work:

1. Read [docs/repo-architecture.md](./repo-architecture.md) when stable repository boundaries or flows matter.
2. Read the approved `spec.md`.
3. Produce the task-local `design/` bundle.
4. Break implementation work down from approved `spec.md + design/`.

Required core design artifacts for non-trivial work:

- `design/overview.md` — design entrypoint, chosen approach, artifact index, unresolved seams, and readiness summary. When the bundle is planning-bound, its artifact index should show required artifact status plus trigger rationale for conditional artifacts, including why plausible optional artifacts are `not expected`, `conditional`, or `waived`.
- `design/component-map.md` — affected packages, modules, generated surfaces, adapters, and components; what changes; what remains stable; which surfaces are intentionally not touched.
- `design/sequence.md` — call order, sync or async boundaries, failure points, side effects, recovery or retry boundaries when relevant, and parallel versus sequential behavior.
- `design/ownership-map.md` — source-of-truth ownership, allowed dependency direction, generated-code authority, adapter responsibility, and explicit non-owners for critical behavior.

Conditional artifacts and trigger rules:

- `design/data-model.md` — create when the task changes persisted state, schema, cache contract, projections, replay behavior, or migration shape.
- `design/dependency-graph.md` — create when the task changes module or package dependency shape, generated-code dependency flow, or introduces a coupling risk that must be made explicit.
- `design/contracts/` — create when the task changes API contracts, event contracts, generated contracts, or material internal interfaces between subsystems. This folder is design-only context for the task, not an authoritative runtime contract source; canonical sources like `api/openapi/service.yaml`, generation inputs, and other repository-owned contract artifacts still win.
- `test-plan.md` — create when validation obligations are too large or multi-layered to fit cleanly inside `tasks.md`.
- `rollout.md` — create when the task needs migration sequencing, backfill and verify choreography, mixed-version compatibility, or explicit deploy and failback notes.

Minimum `test-plan.md` content when triggered:

- trigger and scope: why `tasks.md` cannot hold the validation obligations cleanly;
- proof obligations grouped by changed surface, failure path, or phase;
- planned commands or manual proof shape when known, with any environment limits;
- exit criteria and reopen target when proof is missing, failing, or narrower than the claim.

Minimum `rollout.md` content when triggered:

- trigger and scope: migration, mixed-version, compatibility, delivery order, or recovery reason;
- rollout sequence: expand, backfill, verify, switch, cleanup, or deploy/failback steps as applicable;
- safety checks, operator-visible states, and rollback or forward-recovery conditions;
- links back to `spec.md`, `design/`, and task IDs that own execution detail.

Design-bundle rules:

- `design/overview.md` is the entrypoint and link surface for the bundle; it indexes the design artifacts without duplicating their contents, and it keeps conditional-artifact status and trigger rationale visible for planning.
- Create conditional artifacts only when their trigger is real.
- Keep technical design in `design/`; do not push it back into `spec.md`.
- Record design artifact status in `workflow-plans/technical-design.md` and master `workflow-plan.md`.
- Tiny or `direct path` work may skip the design bundle only with an explicit design-skip rationale.

Planning-bound `design/overview.md` artifact indexes should be scannable without opening every design file just to rediscover artifact status. A compact shape is enough:

```markdown
## Artifact Index

- `design/component-map.md`: approved; package surfaces and stable areas are mapped.
- `design/sequence.md`: approved; request flow, failure points, side effects, and retry boundaries are covered.
- `design/ownership-map.md`: approved; source-of-truth, dependency direction, and generated-code authority are named.
- `design/data-model.md`: not expected; no persisted state, cache contract, projection, replay, or migration shape changes.
- `design/contracts/`: not expected; public and internal contract authorities are unchanged.
- `test-plan.md`: not expected; proof obligations fit in `tasks.md`.
- `rollout.md`: not expected; no migration, mixed-version window, compatibility choreography, or failback note is in scope.

## Readiness Summary

Planning may start from the approved `spec.md` plus the approved required design artifacts. Execution order and task IDs still belong in `tasks.md`.
```

## 6. Execution Loop

Typical path:

- `intake -> workflow planning -> research -> synthesis -> specification(clarification when non-trivial) -> technical design -> planning -> coding/execution from tasks.md -> validation -> done`

Path variations:

- **Idea-shaped work:** Adds `idea refinement` after `intake`.
- **Full orchestrated work:** Adds `synthesis(candidate -> challenge -> final)` and `specification(candidate -> clarification -> approved)`, and may add `review -> reconciliation` before validation.

Very small work may collapse several stages into one local pass, but stage names still drive artifact expectations and resume logic.

### Session-Bounded Phases

For non-trivial work, sessions are phase-scoped by default. Named workflow-control phases map to `workflow-plans/workflow-planning.md`, `workflow-plans/research.md`, `workflow-plans/specification.md`, `workflow-plans/technical-design.md`, `workflow-plans/planning.md`, and, when used, `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md`. Coding/execution is not modeled as its own workflow-control phase and does not use a dedicated phase file.

**Rule:** One session = one phase for non-trivial work unless an upfront `direct path` or `lightweight local` waiver was recorded before the boundary is crossed.

When the phase reaches its completion marker, update the owning artifact, current `workflow-plans/<phase>.md`, and master `workflow-plan.md`, then stop; begin the next phase in a new session.

This is a control rule, not a second state machine: the workflow states stay the same, and the session-bounded phase tells later sessions where to resume and where not to drift.

### 6.1 Idea Refinement and Framing

If the request is idea-shaped, refine it before treating it as implementation-ready.

If it is already concrete, frame:

- what must change,
- scope and non-goals,
- constraints or risk hotspots,
- success checks,
- missing facts.

### 6.2 Workflow Planning

Before any subagent call on non-trivial or agent-backed work, the orchestrator writes master `workflow-plan.md` plus the current phase workflow plan at `workflow-plans/<phase>.md`. A dedicated workflow-planning session uses `workflow-plans/workflow-planning.md`.

The master file owns cross-phase control:

- execution shape,
- current phase,
- artifact status,
- blockers,
- next-session routing,
- phase-plan links/status,
- whether later `design/`, `tasks.md`, `test-plan.md`, or `rollout.md` artifacts are expected.

The phase file owns only local orchestration:

- research mode,
- lanes,
- order or parallelism,
- fan-in or challenge path,
- status,
- completion marker,
- stop rule,
- next action,
- blockers,
- parallelizable work.

A lane's selected skill is a skill name or `no-skill` routing decision. It is not a requirement for the orchestrator to load the full `SKILL.md`; pass the selected name to the subagent and let that lane load the skill if it uses one.

If the selected skill defines an exact deliverable shape, the subagent follows that shape. Otherwise it uses the agent file's fallback return block. Recommended handoffs should classify the next action as one of `spawn_agent`, `reopen_phase`, `needs_user_decision`, `accept_risk`, `record_only`, or `no_action`, then name the target owner or artifact.

Section 7 defines the minimum shape in more detail.

For non-trivial or agent-backed work, the orchestrator runs a workflow plan adequacy challenge after generating or substantially repairing the master and active phase workflow plans, before treating the phase plan as sufficient for handoff. Tiny/direct-path work may skip this with an explicit rationale.

Adequacy gate mechanics:

- The gate reviews `workflow-plan.md`, the active `workflow-plans/<phase>.md`, and any optional review or validation phase-control files whose sufficiency affects handoff.
- The orchestrator invokes one read-only challenger lane with exactly one skill: `workflow-plan-adequacy-challenge`.
- The challenger checks task-specific sufficiency: routing, research mode, lane ownership, artifact expectations, blockers, stop rules, completion marker, next action, next-session handoff, and master/phase consistency.
- Findings must say what is insufficient, why it matters, what could fail, whether it blocks phase handoff or is recordable, and exactly what the orchestrator should add or clarify.
- The challenger must not edit artifacts, approve readiness, create a second `spec.md`, `design/`, or `tasks.md`, or turn the pass into generic checklist coverage.
- The orchestrator reconciles findings by repairing workflow-control artifacts, recording accepted risk or an eligible waiver, or reopening the appropriate earlier phase. Blocking findings prevent phase-complete handoff until reconciled.

For tiny local work, a brief explicit skip rationale in the main flow is enough instead of a full master `workflow-plan.md` plus `workflow-plans/`.

### 6.3 Research, Synthesis, and Specification

Research may stay local or fan out to read-only subagents, depending on the workflow plan. A dedicated research session uses `workflow-plans/research.md` for lane routing, fan-in state, blockers, and the stop rule; reusable evidence lives in `research/*.md` only when it helps later synthesis or resume.

If a subagent result is required for synthesis, review fan-in, or a user-requested agent-backed answer, treat short timeouts as “still running” and keep polling up to 20 minutes per cycle unless the lane is clearly hung, superseded, or canceled.

After research:

- synthesize comparable claims,
- resolve or track key assumptions,
- run pre-spec challenge when risk or ambiguity justifies it,
- run the autonomous `spec-clarification-challenge` gate before non-trivial `spec.md` approval,
- stabilize final decisions in `spec.md` only after planning-critical clarification items are reconciled,
- preserve reusable evidence in `research/*.md` when worth keeping.

`spec.md` should be stable enough that `technical design` can derive task-local context without reopening core problem framing by default.

Clarification gate mechanics:

- The gate is inside `specification`, not a new workflow phase.
- The orchestrator prepares a compact bundle with problem frame, scope and non-goals, candidate decisions, constraints, validation expectations, assumptions or open questions, and relevant research links.
- The orchestrator invokes one read-only subagent lane, preferably `challenger-agent`, using exactly one skill: `spec-clarification-challenge`.
- The subagent returns approval-focused questions classified as `blocks_spec_approval`, `blocks_specific_domain`, or `non_blocking_but_record`, with next actions such as `answer_from_existing_evidence`, `targeted_research`, `expert_subagent`, `accept_risk`, `defer_to_design`, or `requires_user_decision`.
- The orchestrator answers from existing evidence where possible, or reopens one read-only targeted research or expert lane per question when evidence is missing.
- Do not ask the human during the normal loop. If the point is truly external product or business policy, record `requires_user_decision` and leave `spec.md` blocked or partially draft.
- If material decisions changed or a major seam reopened and was resolved, rerun the clarification challenge once.
- Store final answers in existing `spec.md` sections only: stable outcomes in `Decisions`, assumptions in `Open Questions / Assumptions`, and proof consequences in `Validation`. Do not add raw subagent transcripts to `spec.md`.
- `workflow-plans/specification.md` records clarification status, lane used, targeted research status, resolution status, and approval or block rationale. `workflow-plan.md` records `spec.md` status and clarification gate status.

### 6.4 Technical Design

`technical design` begins after `spec.md` is stable enough to support design work and any required clarification gate is resolved or explicitly waived by an eligible direct/local exception. If the clarification gate is blocked, route to the recorded upstream reopen target instead of starting design.

It:

- uses [docs/repo-architecture.md](./repo-architecture.md) when repository baseline context matters,
- produces the required core `design/` artifacts,
- adds conditional design artifacts only when triggered,
- records approval state in `workflow-plans/technical-design.md` and master `workflow-plan.md`.

If design work exposes a missing decision or unstable problem boundary, loop back to `spec.md` instead of silently letting design redefine the task.

### 6.5 Planning

Planning is separate from workflow planning: workflow planning decides orchestration and artifact expectations; task breakdown decides executable implementation order after decisions and design are stable.

Enter planning only when:

- minimum viable framing is explicit,
- workflow planning chose `local` or `fan-out`,
- `spec.md` is stable,
- non-trivial specification clarification is reconciled or explicitly waived by an eligible direct/local exception,
- required `design/` artifacts are approved or an explicit design-skip rationale exists,
- higher-risk pre-spec challenge is reconciled or explicitly waived.

For non-trivial work, `planning-and-task-breakdown` should consume approved `spec.md + design/` and produce `tasks.md` as the executable task ledger and final implementation handoff. For `direct path` work, the explicit plan may stay as 1-3 concise lines in the main flow.

Planning is the last artifact-producing phase before code:

- the workflow/design/planning bundle must exist or be explicitly waived before the first implementation session starts,
- `tasks.md` is created or repaired only here when expected; post-code work may update existing checkbox/progress state but must reopen planning instead of inventing a missing required task ledger,
- dedicated coding phase files are not created or expected,
- any optional `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` must be explicitly justified by named multi-session routing before coding starts,
- `tasks.md` must slice execution into small reviewable, verification-bound increments; single-pass broad implementation needs explicit rationale.

Minimum `tasks.md` content:

- optional compact handoff header when it helps the next implementation session: consumed artifacts, readiness status, first task or checkpoint, named `CONCERNS` proof obligations, and reopen target;
- markdown checkboxes,
- stable task IDs such as `T001`,
- phase or checkpoint label,
- optional `[P]` only when safe to parallelize,
- short action,
- exact file path when known or a narrow package/artifact surface when exact file choice is genuinely design-time unknown,
- dependency marker when nontrivial,
- proof or verification expectation.

Prefer vertical, reviewable slices and avoid generic tasks such as "implement feature." A task item may use concise continuation lines when dependency, proof, accepted concern, or reopen detail would otherwise make a single-line checkbox hard to scan; it must still remain one executable ledger item. If exact tasking requires a missing design decision, reopen `technical design` instead of inventing the task.

Implementation readiness is the planning exit check, not a separate workflow phase. It runs after expected `tasks.md` is ready and before handoff to implementation.

Status values:

- `PASS`: implementation may start.
- `CONCERNS`: implementation may start only with named accepted risks and proof obligations.
- `FAIL`: implementation must not start; route to the named earlier phase.
- `WAIVED`: allowed only for tiny, direct-path, or prototype work with explicit rationale and scope.

Gate checks:

- `spec.md` is approved, or explicitly waived for eligible tiny/direct-path work.
- Required `design/` artifacts are approved, or a design-skip rationale is recorded.
- `tasks.md` is approved when non-trivial, unless an explicit eligible waiver exists.
- Triggered `test-plan.md` and `rollout.md` exist, or are explicitly not expected.
- Optional review or validation phase workflow files were created during planning only when named multi-session routing requires them.
- Material blockers are resolved, or explicitly accepted under `CONCERNS`.
- The validation and proof path is explicit.
- No unresolved high-impact open question remains that could change correctness, ownership, rollout, or validation.

Artifact placement:

- `workflow-plan.md` records the readiness status.
- `workflow-plans/planning.md` records the gate result and stop or handoff rule.
- `tasks.md` may carry a short readiness reference when useful.

### Session-Boundary Gate

For non-trivial work, a session may advance only the `Current phase` recorded in master `workflow-plan.md` and the matching `workflow-plans/<phase>.md`.

At completion:

- reconcile blocking workflow plan adequacy challenge findings, or record the eligible direct/local skip rationale,
- update the owning artifact, current phase workflow plan, and master `workflow-plan.md`,
- mark `Session boundary reached: yes`,
- set `Ready for next session` appropriately,
- record `Next session starts with`,
- stop.

If unfinished, leave the phase `in_progress` or `blocked`. `Direct path` work and any upfront `lightweight local` waiver may collapse boundaries only when recorded before the boundary is crossed.

### 6.6 Coding/Execution, Review, and Validation

Coding/execution happens in the main flow under orchestrator control and consumes approved `spec.md`, `design/`, existing `tasks.md` when expected, optional `test-plan.md`, optional `rollout.md`, and any pre-created review or validation phase workflow files. It is not modeled as a named workflow-control phase and does not use a separate phase plan.

Implementation rules:

- Create code, tests, migrations, configs, generation inputs, and generated artifacts only when the approved task ledger requires them.
- Update only existing control and checkpoint artifacts, including checkbox/progress state in existing `tasks.md` when the current task slice uses it.
- Do not create new workflow/process/planning/design/temp artifacts or ad hoc progress markdown.
- If coding exposes a real task-breakdown or design gap, or required `tasks.md` is missing, stop, record the reopen in existing control artifacts, and reopen the relevant earlier phase in a new session instead of silently drifting.

Review stays read-only and risk-driven. When planning created `workflow-plans/review-phase-N.md`, that file stays routing-focused: it records review scope, read-only lanes, finding status, orchestrator reconciliation status, accepted risks, blockers or reopen targets, validation implications, completion marker, and stop rule. Finding disposition should be compact and orchestrator-owned: finding ID or short label, source lane, disposition (`accepted`, `fixed in reconciliation`, `accepted risk`, `reopen`, or `no_action`), target artifact or task when applicable, and validation implication. It must not become a second review transcript, task ledger, design bundle, or decision record.

Reconciliation applies orchestrator-approved fixes or accepted-risk records from review inside the existing task ledger and control surfaces. If review exposes a missing spec, design, planning, or task-ledger decision, record the reopen target instead of letting review output become the new authority.

Validation is also artifact-consuming:

- use fresh evidence against the approved artifact bundle,
- update only existing closeout surfaces such as `workflow-plan.md`, the active `workflow-plans/validation-phase-N.md` when one was created before coding, progress state in existing `tasks.md` when needed, and `spec.md` `Validation` or `Outcome`,
- do not create new workflow/process/planning/design/temp artifacts,
- if proof exposes an upstream gap or expected control artifact is missing, reopen the right earlier phase instead of inventing a closeout artifact.

When a validation phase-control file exists, keep it closeout-focused: closeout claim, proof inputs, command or manual proof scope, allowed future writes, phase status, blockers or reopen target, completion marker, next action, and stop rule. The authoritative proof record still belongs in `spec.md` `Validation` and `Outcome`; the phase file only routes the validation checkpoint.

Closeout is not complete until the artifacts reflect reality:

- `workflow-plan.md` shows the current phase or completion state,
- the current `workflow-plans/<phase>.md` shows the local handoff state,
- `workflow-plan.md` reflects what phase completed or remains,
- `tasks.md`, when used, reflects only real checkbox/progress state and not new task invention,
- `spec.md` records actual outcome and remaining open questions,
- `Validation`/`Outcome` reflect what was actually proved.

## 7. Master `workflow-plan.md` And Phase Workflow Plans

The repository uses two workflow-control layers for non-trivial work:

- master `workflow-plan.md` for cross-phase control
- `workflow-plans/<phase>.md` for one phase only

Read the master file first, then the current phase workflow plan.

### 7.1 Master `workflow-plan.md`

`workflow-plan.md` owns runtime control across phases.

Minimum context-first control record:

- `Current phase`;
- `Phase status`;
- `Session boundary reached`;
- `Ready for next session`;
- `Next session starts with`;
- `Next session context bundle` as an always-present field: either `Default resume order is sufficient` with any narrow additions, or exact artifact paths and one-line reasons for task-specific resume context;
- artifact status table with trigger rationale for plausible optional artifacts;
- blockers, accepted assumptions, accepted risks, and reopen targets that still affect routing;
- adequacy or clarification gate status when the current handoff depends on it.

At minimum, it answers:

- current phase,
- phase status (`pending`, `in_progress`, `blocked`, or `complete` as the phase lifecycle requires),
- session-boundary state,
- next-session readiness and starting phase,
- next-session context bundle, including the explicit default-resume case when no task-specific bundle is needed,
- artifact status (`approved`, `draft`, `missing`, `blocked`, `waived`, `not expected`, or `conditional` as applicable),
- blockers,
- phase workflow plan status,
- implementation-readiness status,
- workflow plan adequacy challenge status when required,
- default resume order.

Use `Phase status` for lifecycle state only, such as `pending`, `in_progress`, `blocked`, or `complete`. Use a separate `Task state` or `Routing state` line for outcomes such as `done`, `reopened`, or `reopened-to-specification`, so later sessions do not confuse a backward route with the phase's local completion state.

### 7.2 `workflow-plans/<phase>.md`

`workflow-plans/<phase>.md` owns one phase's local orchestration:

- lanes or tracks,
- order or parallelism,
- completion marker,
- explicit out-of-scope work,
- stop rule,
- next action,
- parallelizable work,
- local blockers,
- implementation-readiness gate result and stop or handoff rule when the phase is `planning`,
- review scope, finding status, reconciliation status, accepted risks, or reopen target when the phase is `review`,
- workflow plan adequacy challenge status and resolution when required.

It is phase-local routing, not a replacement for `spec.md`, `design/`, or `tasks.md`.

Recommended update cadence:

- After framing or workflow planning: update the master file with execution shape, current phase, blockers, next-session routing, phase-plan links/status, and artifact expectations; update the current phase file with local orchestration, lanes, completion marker, and stop rule; run or explicitly waive the workflow plan adequacy challenge before handoff.
- After synthesis/specification: update `spec.md` status in the master file, record clarification gate status, and record any blocker that prevents leaving `workflow-plans/specification.md`.
- After `technical design` or planning: record approved design artifacts, expected `tasks.md` status, and implementation-readiness status in the master file and current phase file; during planning, also create or repair `tasks.md` and create only those review or validation phase workflow files whose named multi-session routing is genuinely needed.
- After each coding/execution checkpoint: update the master file and existing `tasks.md` checkbox/progress state when the task ledger is in use. If a needed workflow/process artifact is missing, reopen the relevant earlier phase instead of creating it mid-implementation.
- After review or reconciliation: update only existing control/checkpoint artifacts with review scope, findings status, orchestrator reconciliation, accepted risks, blockers, reopen targets, and validation implications. Do not paste raw review transcripts or invent new tasks in review control files.
- After any phase-complete handoff: reconcile blocking workflow plan adequacy challenge findings, mark `Session boundary reached`, `Ready for next session`, `Next session starts with`, and the always-present next-session context bundle in the master file, and close the current phase workflow plan.
- After validation: record completion or remaining blockers in the master file and the active validation phase workflow plan when one already exists; update `spec.md` `Validation` and `Outcome` to match the actual proof; update existing `tasks.md` checkbox/progress state only when already in use. If a required `tasks.md` is missing, reopen the relevant earlier phase instead of creating it during closeout.

### 7.3 Phase Workflow Plan Minimums

These are minimum routing shapes for `workflow-plans/<phase>.md`. They are examples, not templates to paste wholesale. Preserve the fields and ownership meaning, but adjust headings, grouping, and depth to the task; do not add empty lines for fields that do not carry routing value.

`workflow-planning`:

```text
Phase: workflow-planning
Phase status: complete | blocked
Execution shape: direct path | lightweight local | full orchestrated
Research mode: local | fan-out | not expected, with rationale
Lane plan: one owned question, role, evidence target, and skill or no-skill per lane when fan-out is expected
Completion marker: workflow routing and artifact expectations are explicit
Stop rule: do not start research or later phases in this session unless an upfront waiver exists
Next action: research, specification, or recorded direct-path execution
```

`research`:

```text
Phase: research
Phase status: complete | blocked
Research mode: local | fan-out, with why it fits
Lanes or local tracks: owned question, source surface, status, and single skill or no-skill
Fan-in: where the durable summary lives and which `research/*.md` notes were preserved, if any
Completion marker: must-answer-now questions handled or explicitly deferred
Stop rule: do not write `spec.md`, `design/`, or `tasks.md`
Next action: specification, challenge, targeted re-research, or blocked reopen
```

When research preserves reusable evidence, split routing from evidence memory instead of writing a research-shaped second spec:

```markdown
workflow-plans/research.md:
Fan-in: RQ1 and RQ2 handled from route files, generated handler tests, repository methods, and migration history; `research/state-ownership.md` preserved because later specification needs the source trail and limits.
Next action: specification with pre-spec challenge expected.
Stop rule: do not write `spec.md`, `design/`, or `tasks.md` in this research session.

research/state-ownership.md:
Question: Which repository surfaces own persisted job state and terminal transitions?
Findings: repository methods include tenant-scoped writes; no existing cancelled terminal state was found.
Evidence limits: comparable code exists for one feature only, so product intent remains uncertain.
Handoff implication: specification must decide whether cancellation is in scope; this note does not decide it.
```

`specification`:

```text
Phase: specification
Phase status: complete | blocked
Input sources: framing, research notes, direct evidence, and prior workflow artifacts used
Spec readiness: why `spec.md` is approved, draft, or blocked
Clarification gate: lane used, status, resolution, rerun status when needed, or eligible waiver
Completion marker: `spec.md` is approved or the unblock path is explicit
Stop rule: do not start `technical design` in this session unless an upfront waiver exists
Next action: technical-design, targeted research, requires_user_decision, or blocked reopen
```

`technical-design`:

```text
Phase: technical-design
Phase status: complete | blocked
Design pass type: fresh, resumed, or repair
Design artifacts: required and triggered conditional artifacts with status and trigger rationale
Planning readiness: why planning can start or which spec/design blocker remains
Completion marker: required design artifacts are approved or blocked with reopen target
Stop rule: do not write `tasks.md` or begin implementation
Next action: planning or named reopen target
```

`planning`:

```text
Phase: planning
Phase status: complete | blocked
Artifact outputs: `tasks.md`, triggered `test-plan.md` or `rollout.md`, and named review or validation phase files
Implementation readiness: PASS | CONCERNS | FAIL | WAIVED with required rationale
Adequacy challenge: status and resolution, or eligible direct/local skip rationale
Completion marker: approved task ledger and readiness gate, or explicit reopen target
Stop rule: do not begin implementation in this session unless an upfront waiver exists
Next action: first task ID, implementation checkpoint, or named reopen target
```

`review-phase-N` when planning created it:

```text
Phase: review-phase-N
Phase status: pending | in_progress | complete | blocked
Review scope: approved checkpoint, diff, and artifact bundle being reviewed
Read-only lanes: one question and one skill or no-skill per lane
Finding status: compact orchestrator-owned disposition, not raw transcripts
Reconciliation state: accepted, fixed in reconciliation, accepted risk, reopen, or no_action
Validation implications: proof additions or limits created by reconciled findings
Stop rule: do not edit files or create tasks from review output
Next action: reconciliation, validation, or named reopen target
```

`validation-phase-N` when planning created it:

```text
Phase: validation-phase-N
Phase status: pending | in_progress | complete | blocked
Closeout claim: exact phase or task claim under proof
Proof inputs: approved artifacts, task IDs, review notes, and optional `test-plan.md` or `rollout.md`
Commands or manual proof scope: fresh evidence required for the claim
Allowed future writes: existing closeout surfaces only
Blockers or reopen target: failed, missing, stale, or too-narrow proof
Stop rule: do not implement fixes or create missing process artifacts
Next action: close task, narrow claim, or reopen the right earlier phase
```

Minimal split example:

```text
workflow-plan.md:
Current phase: technical-design; Session boundary reached: no; Ready for next session: no
Next session starts with: planning; Phase workflow plans: specification complete; technical-design active; planning pending
Next session context bundle: spec.md for approved decisions; design/overview.md and required design maps for planning constraints
Artifacts: spec.md approved; design/ draft; tasks.md missing

workflow-plans/technical-design.md:
Phase status: in_progress; Completion marker: required design artifacts approved
Next action: finish sequence and ownership mapping; Stop rule: do not begin planning in this session
```

## 8. Resume Order And Stage Inference

### Resume order

In a later session, read artifacts in this order:

1. `workflow-plan.md`
2. current `workflow-plans/<phase>.md`
3. the `Next session context bundle` from `workflow-plan.md`; if it says the default resume order is sufficient, continue with the default phase artifact order below
4. phase artifacts in the order the current phase needs them:
   - `spec.md`
   - [docs/repo-architecture.md](./repo-architecture.md) when the task depends on stable repository architecture context
   - `design/overview.md`
   - remaining required design artifacts plus any triggered conditional design files
   - `tasks.md` when present or expected
   - optional `test-plan.md`, `rollout.md`, and selected `research/*.md`

If the task was intentionally small enough to skip some artifacts, read the recorded skip rationale before assuming the artifact is merely missing.

### How to infer the current stage from artifacts

Use artifacts, not memory:

- no approved `spec.md` means framing or specification is still incomplete
- draft or blocked `spec.md` with unresolved clarification gate status means specification is still incomplete
- approved `spec.md` but no approved design bundle means `technical design` is still incomplete
- approved design bundle but no approved expected `tasks.md` means planning is still incomplete
- approved `tasks.md` or an explicit `tasks.md` waiver still requires implementation readiness of `PASS`, eligible `CONCERNS`, or eligible `WAIVED` before the workflow can follow the recorded implementation routing
- implementation readiness of `FAIL` routes to the named earlier phase instead of implementation
- validation evidence plus updated `Outcome` means the task has reached validation or done

Use session control from the master file before doing any work:

- if the current phase points at a missing `workflow-plans/<phase>.md`, treat the phase workflow record as incomplete rather than reconstructing it from memory
- if `Session boundary reached: yes`, start a new session for the recorded next phase
- if `Ready for next session: no`, resume the same session-bounded phase instead of jumping forward
- if a reopen target points backward, reopen that earlier phase instead of continuing from the later artifact state

For exceptions, the skip rationale must explain skipped `workflow-plan.md`, `workflow-plans/`, `design/`, or `tasks.md`. Without required-artifact rationales, assume the artifact chain is incomplete rather than silently waived.

## 9. Direct-Path And Lightweight-Local Exceptions

Direct-path and lightweight-local work still exists; the workflow is not trying to force the full artifact bundle onto tiny fixes.

These shapes may:

- collapse workflow planning, research, synthesis, specification, `technical design`, and planning into one local pass,
- skip a separate `workflow-plan.md`, `workflow-plans/`, or `tasks.md` when tiny,
- skip `design/` only when the change is local, the behavior delta is obvious, and no ownership, data, or sequence ambiguity exists.

Same-session phase collapse still requires an upfront recorded waiver.

Never skip explicit planning-before-code, clear decision ownership, fresh validation evidence, or design-skip rationale when `design/` would otherwise be expected.

Example skip rationale: local validator change in one package; no persisted-state, contract, ownership, or runtime-sequence ambiguity; tasking inline because execution is one short reversible step.

If a supposedly small task uncovers a larger seam, escalate to the fuller artifact chain instead of pretending the original shortcut still fits.

## 10. Artifact-Focused Anti-Patterns

Avoid:

- **Stale master control:** Treating `workflow-plan.md` as a one-time pre-research note instead of the live master control artifact.
- **Competing phase files:** Letting `workflow-plans/<phase>.md` replace the master `workflow-plan.md` or grow into a competing design or execution artifact.
- **Phase-boundary drift:** Finishing one non-trivial phase and casually starting the next one in the same session without an upfront recorded waiver.
- **Planning from `spec.md` alone:** Planning non-trivial work from `spec.md` alone after the design-bundle stage exists.
- **Skipping clarification approval:** Marking non-trivial `spec.md` approved while the autonomous clarification gate is unresolved or blocked, instead of reconciling it, waiving it through an eligible direct/local exception, or leaving `spec.md` blocked with a reopen target.
- **Design bundle drift:** Letting `design/` turn into a second `spec.md` or a second task breakdown.
- **Task ledger drift:** Letting `tasks.md` become a second spec, second design bundle, competing plan, or a place to invent missing technical decisions.
- **Review-control drift:** Letting `workflow-plans/review-phase-N.md` become a raw transcript, hidden task ledger, second design bundle, or final decision record instead of an orchestrator-reconciled routing surface.
- **Coding control-file drift:** Creating a separate coding phase-control file instead of coding directly from approved `tasks.md`.
- **Premature outcome claims:** Writing `Outcome` as done or successful before fresh validation evidence exists.
- **Readiness bypass:** Treating implementation as ready when implementation readiness is missing or `FAIL`, using `CONCERNS` without named accepted risks and proof obligations, or treating `WAIVED` as the default for work that is not tiny, direct-path, or prototype-scoped.
- **New process artifacts after code starts:** Creating new workflow/process markdown during implementation or validation instead of reopening the correct earlier phase.
- **Pre-reading review skills:** Loading multiple subagent-internal review or domain skill bodies in the main flow because their descriptions match, instead of routing one skill name per read-only lane.
- **Just-in-case artifacts:** Creating `test-plan.md`, `rollout.md`, or conditional design files "just in case".
- **Missing skip rationale:** Forgetting to record the skip rationale when bypassing `design/`.
- **Architecture re-derivation:** Re-deriving repository architecture every session instead of loading [docs/repo-architecture.md](./repo-architecture.md).
- **Execution-order archaeology:** Forcing the coder to reconstruct execution order from technical prose when expected `tasks.md` should exist.
- **Stale artifact status:** Leaving artifact status stale so resume requires chat archaeology.

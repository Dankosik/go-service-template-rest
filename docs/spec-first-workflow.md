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

The `workflow-status` skill is a read-only status and next-action helper over those artifacts. It can summarize current phase, blockers, allowed writes, stop rule, implementation-readiness status, and implementation-start status, but it is not a session phase, gate, approval record, or replacement for `workflow-plan.md`, `workflow-plans/<phase>.md`, `spec.md`, `design/`, `tasks.md`, or an optional `plan.md` when one exists.

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
    implementation-phase-1.md   # conditional
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
  plan.md                  # optional; only for large/multi-checkpoint strategy notes
  test-plan.md             # conditional
  rollout.md               # conditional
```

### Artifact Purposes

- **`workflow-plan.md`:** Master routing and control artifact. It tracks cross-phase status: execution shape, current phase, artifact status, blockers, next session, links to phase workflow plans, and resume order. Required for non-trivial or agent-backed work.
- **`workflow-plans/<phase>.md`:** Phase-local workflow plan for one named phase only. It tracks that phase's local orchestration, order/parallelism, fan-in/challenge path when relevant, completion marker, stop rule, next action, and local blockers. Required for each named non-trivial phase that the task actually uses, including `workflow-planning` or `research` when those are dedicated sessions.
- **`spec.md`:** Canonical decision record: approved framing, scope, constraints, decisions, and accepted open questions. Always.
- **`design/`:** Task-local technical design bundle between `spec.md` and task breakdown. It explains how the approved change fits the repository architecture and what implementation must preserve. Required for non-trivial work unless explicitly skipped with rationale.
- **`tasks.md`:** Executable task ledger and final implementation handoff derived from approved `spec.md + design/` and any optional planning context. It owns markdown checkboxes with stable task IDs, phase/checkpoint labels when useful, optional `[P]` markers only for safe parallel work, concrete action and file/package surfaces, dependency markers when nontrivial, and proof expectations. Expected by default for non-trivial implementation work; direct-path or tiny work may skip it only with an explicit waiver.
- **`plan.md`:** Optional supplementary strategy note for unusually large, multi-checkpoint, or cross-session work when forcing all strategy into `tasks.md` would make the ledger noisy. It is not required before implementation, not produced by default, and never replaces `tasks.md` as the handoff artifact.
- **Implementation readiness:** Planning-phase exit gate recorded as `PASS`, `CONCERNS`, `FAIL`, or `WAIVED`. The status belongs in `workflow-plan.md`, the result and stop or handoff rule belong in `workflow-plans/planning.md`, and `tasks.md` may carry only a short reference when useful.
- **`research/*.md`:** Preserved evidence, comparisons, and validated research context. These files support decisions but do not own them. Create only when the task is long, ambiguous, or benefits from reusable research memory.
- **Artifact status vocabulary:** Use `approved`, `draft`, `missing`, `blocked`, `waived`, `not expected`, or `conditional` as the task needs. `waived` requires an eligible direct/local rationale and scope; `not expected` requires a short trigger-based reason when the artifact would otherwise be plausible; `conditional` means a later phase must decide the trigger instead of creating the artifact early.
- **`test-plan.md`:** Expanded validation strategy when test obligations are too large or too layered to fit cleanly inside `tasks.md`. Conditional.
- **`rollout.md`:** Operational rollout and migration notes when delivery order, compatibility, or recovery choreography matters. Conditional.

### Artifact Rules

- **Repository baseline:** [docs/repo-architecture.md](./repo-architecture.md) is the stable repository baseline, not a task-local design file.
- **Workflow control:** `workflow-plan.md` stays live through research, synthesis, `technical design`, planning, implementation, and validation; `workflow-plans/<phase>.md` stays phase-local and must not turn into a second `design/` bundle, `tasks.md`, or optional `plan.md`.
- **Pre-code phase plans:** Dedicated workflow-planning and research sessions use `workflow-plans/workflow-planning.md` and `workflow-plans/research.md`. Later pre-code phases normally get `workflow-plans/specification.md`, `workflow-plans/technical-design.md`, and `workflow-plans/planning.md`.
- **Post-code phase-control files:** Post-code phase-control files are optional control artifacts. Create `workflow-plans/implementation-phase-N.md`, `workflow-plans/review-phase-N.md`, or `workflow-plans/validation-phase-N.md` only when a named multi-session phase truly needs phase-local routing beyond `workflow-plan.md` and `tasks.md`; do not create them for already-small implementation tasks by default.
- **Artifact ownership:** `spec.md` stores decisions, `design/` stores task-specific technical context, and `tasks.md` stores executable task state and the implementation handoff. Optional `plan.md` stores only supplemental strategy when genuinely needed. Do not make any of them absorb another artifact's job.
- **Research shape:** `research/*.md` should be flexible and evidence-oriented; there is no mandatory universal template. When preserved, a research note should make the question or scope, findings with evidence and limits, conflicts or open points, and handoff implication visible enough that later synthesis does not need chat memory.
- **No duplicate authority:** Do not duplicate the same authority across artifacts. Link instead.

### Artifact-Producing Vs Artifact-Consuming Phases

#### Pre-code phases

Pre-code phases (`workflow planning`, `research`, `specification`, `technical design`, and `planning`) may create only the approved workflow/process artifact set:

- `workflow-plan.md`
- `workflow-plans/<phase>.md`
- `research/*.md`
- `spec.md`
- `design/`
- `tasks.md`
- optional `plan.md`
- optional `test-plan.md`
- optional `rollout.md`

If later implementation, review, or validation phase-control files are genuinely needed, planning records that need and creates only the named optional routing files before implementation starts.

#### Post-code phases

Post-code phases (`implementation`, `review`, `reconciliation`, and `validation`) consume that bundle.

Implementation and reconciliation may create approved codebase files such as code, tests, migrations, configs, generation inputs, and generated output when the task ledger requires them. Review remains read-only and advisory. Validation gathers proof rather than fixes. Post-code phases may update only existing control or closeout surfaces such as `workflow-plan.md`, the active `workflow-plans/<phase>.md` when one exists, checkpoint state in existing `tasks.md`, optional `plan.md` notes when one exists, and `spec.md` `Validation` or `Outcome`.

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
- Keep only a short task-breakdown or handoff link in `spec.md`; link an optional `plan.md` only when one exists for large-work strategy.
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

- `design/overview.md` — design entrypoint, chosen approach, artifact index, unresolved seams, and readiness summary.
- `design/component-map.md` — affected packages, modules, or components; what changes; what remains stable.
- `design/sequence.md` — call order, sync or async boundaries, failure points, side effects, and parallel versus sequential behavior.
- `design/ownership-map.md` — source-of-truth ownership, allowed dependency direction, and responsibility boundaries.

Conditional artifacts and trigger rules:

- `design/data-model.md` — create when the task changes persisted state, schema, cache contract, projections, replay behavior, or migration shape.
- `design/dependency-graph.md` — create when the task changes module or package dependency shape, generated-code dependency flow, or introduces a coupling risk that must be made explicit.
- `design/contracts/` — create when the task changes API contracts, event contracts, generated contracts, or material internal interfaces between subsystems. This folder is design-only context for the task, not an authoritative runtime contract source; canonical sources like `api/openapi/service.yaml`, generation inputs, and other repository-owned contract artifacts still win.
- `test-plan.md` — create when validation obligations are too large or multi-layered to fit cleanly inside `tasks.md`.
- `rollout.md` — create when the task needs migration sequencing, backfill and verify choreography, mixed-version compatibility, or explicit deploy and failback notes.

Design-bundle rules:

- `design/overview.md` is the entrypoint and link surface for the bundle.
- Create conditional artifacts only when their trigger is real.
- Keep technical design in `design/`; do not push it back into `spec.md`.
- Record design artifact status in `workflow-plans/technical-design.md` and master `workflow-plan.md`.
- Tiny or `direct path` work may skip the design bundle only with an explicit design-skip rationale.

## 6. Execution Loop

Typical path:

- `intake -> workflow planning -> research -> synthesis -> specification(clarification when non-trivial) -> technical design -> planning -> implementation -> validation -> done`

Path variations:

- **Idea-shaped work:** Adds `idea refinement` after `intake`.
- **Full orchestrated work:** Adds `synthesis(candidate -> challenge -> final)` and `specification(candidate -> clarification -> approved)`, and may add `review -> reconciliation` before validation.

Very small work may collapse several stages into one local pass, but stage names still drive artifact expectations and resume logic.

### Session-Bounded Phases

For non-trivial work, sessions are phase-scoped by default. Named phases map to `workflow-plans/workflow-planning.md`, `workflow-plans/research.md`, `workflow-plans/specification.md`, `workflow-plans/technical-design.md`, `workflow-plans/planning.md`, and, when used, `workflow-plans/implementation-phase-N.md`, `workflow-plans/review-phase-N.md`, or `workflow-plans/validation-phase-N.md`.

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
- whether later `design/`, `tasks.md`, optional `plan.md`, `test-plan.md`, or `rollout.md` artifacts are expected.

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

- The gate reviews `workflow-plan.md`, the active `workflow-plans/<phase>.md`, and any optional post-code phase-control files whose sufficiency affects handoff.
- The orchestrator invokes one read-only challenger lane with exactly one skill: `workflow-plan-adequacy-challenge`.
- The challenger checks task-specific sufficiency: routing, research mode, lane ownership, artifact expectations, blockers, stop rules, completion marker, next action, next-session handoff, and master/phase consistency.
- Findings must say what is insufficient, why it matters, what could fail, whether it blocks phase handoff or is recordable, and exactly what the orchestrator should add or clarify.
- The challenger must not edit artifacts, approve readiness, create a second `spec.md`, `design/`, `tasks.md`, or optional `plan.md`, or turn the pass into generic checklist coverage.
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

For non-trivial work, `planning-and-task-breakdown` should consume approved `spec.md + design/` and produce `tasks.md` as the executable task ledger and final implementation handoff. Create optional `plan.md` only when the work is unusually large, multi-checkpoint, or cross-session enough that a separate strategy note keeps `tasks.md` small and executable. For `direct path` work, the explicit plan may stay as 1-3 concise lines in the main flow.

Planning is the last artifact-producing phase before code:

- the workflow/design/planning bundle must exist or be explicitly waived before the first implementation session starts,
- `tasks.md` is created or repaired only here when expected; post-code phases may update existing checkbox/progress state but must reopen planning instead of inventing a missing required task ledger,
- any optional `workflow-plans/implementation-phase-N.md`, `workflow-plans/review-phase-N.md`, or `workflow-plans/validation-phase-N.md` must be explicitly justified by named multi-session routing before implementation starts,
- phased execution (`phase -> review/reconcile -> validate -> next phase`) is the default,
- single-pass big-bang implementation needs explicit rationale.

Minimum `tasks.md` content:

- markdown checkboxes,
- stable task IDs such as `T001`,
- phase or checkpoint label,
- optional `[P]` only when safe to parallelize,
- short action,
- exact file path when known or a narrow package/artifact surface when exact file choice is genuinely design-time unknown,
- dependency marker when nontrivial,
- proof or verification expectation.

Prefer vertical, reviewable slices and avoid generic tasks such as "implement feature." If exact tasking requires a missing design decision, reopen `technical design` instead of inventing the task.

Optional `plan.md` content, when justified:

- why a separate strategy note is needed,
- coarse phase or checkpoint strategy that would make `tasks.md` too noisy,
- cross-phase validation or rollback notes that do not fit cleanly in task items,
- links back to the task IDs that remain the executable handoff.

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
- Optional `plan.md`, if present, is consistent with `tasks.md` and does not carry hidden required tasking.
- Triggered `test-plan.md` and `rollout.md` exist, or are explicitly not expected.
- Optional implementation, review, or validation phase workflow files were created during planning only when named multi-session routing requires them.
- Material blockers are resolved, or explicitly accepted under `CONCERNS`.
- The validation and proof path is explicit.
- No unresolved high-impact open question remains that could change correctness, ownership, rollout, or validation.

Artifact placement:

- `workflow-plan.md` records the readiness status.
- `workflow-plans/planning.md` records the gate result and stop or handoff rule.
- `tasks.md` may carry a short readiness reference when useful.
- Optional `plan.md`, when present, may carry a compact summary only.

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

### 6.6 Implementation, Review, and Validation

Implementation happens in the main flow under orchestrator control and consumes approved `spec.md`, `design/`, existing `tasks.md` when expected, optional `plan.md` when present, optional `test-plan.md`, optional `rollout.md`, and any pre-created post-code phase workflow files.

Implementation rules:

- Create code, tests, migrations, configs, generation inputs, and generated artifacts only when the approved task ledger requires them.
- Update only existing control and checkpoint artifacts, including checkbox/progress state in existing `tasks.md` when the current implementation phase uses it.
- Do not create new workflow/process/planning/design/temp artifacts or ad hoc progress markdown.
- If coding exposes a real task-breakdown or design gap, or required `tasks.md` is missing, stop, record the reopen in existing control artifacts, and reopen the relevant earlier phase in a new session instead of silently drifting.

Review stays read-only and risk-driven. When planning created `workflow-plans/review-phase-N.md`, that file stays routing-focused: it records review scope, read-only lanes, finding status, orchestrator reconciliation status, accepted risks, blockers or reopen targets, validation implications, completion marker, and stop rule. It must not become a second review transcript, task ledger, design bundle, or decision record.

Reconciliation applies orchestrator-approved fixes or accepted-risk records from review inside the existing task ledger and control surfaces. If review exposes a missing spec, design, planning, or task-ledger decision, record the reopen target instead of letting review output become the new authority.

Validation is also artifact-consuming:

- use fresh evidence against the approved artifact bundle,
- update only existing closeout surfaces such as `workflow-plan.md`, the active `workflow-plans/validation-phase-N.md` when one was created before implementation, progress state in existing `tasks.md` when needed, optional `plan.md` notes when one exists, and `spec.md` `Validation` or `Outcome`,
- do not create new workflow/process/planning/design/temp artifacts,
- if proof exposes an upstream gap or expected control artifact is missing, reopen the right earlier phase instead of inventing a closeout artifact.

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

At minimum, it answers:

- current phase,
- phase status (`in_progress`, `blocked`, or `complete`),
- session-boundary state,
- next-session readiness and starting phase,
- artifact status (`approved`, `draft`, `missing`, `blocked`, `waived`, `not expected`, or `conditional` as applicable),
- blockers,
- phase workflow plan status,
- implementation-readiness status,
- workflow plan adequacy challenge status when required,
- default resume order.

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

It is phase-local routing, not a replacement for `spec.md`, `design/`, `tasks.md`, or an optional `plan.md`.

Recommended update cadence:

- After framing or workflow planning: update the master file with execution shape, current phase, blockers, next-session routing, phase-plan links/status, and artifact expectations; update the current phase file with local orchestration, lanes, completion marker, and stop rule; run or explicitly waive the workflow plan adequacy challenge before handoff.
- After synthesis/specification: update `spec.md` status in the master file, record clarification gate status, and record any blocker that prevents leaving `workflow-plans/specification.md`.
- After `technical design` or planning: record approved design artifacts, expected `tasks.md` status, optional `plan.md` status when one exists, and implementation-readiness status in the master file and current phase file; during planning, also create or repair `tasks.md` and create only those implementation/review/validation phase workflow files whose named multi-session routing is genuinely needed.
- After each implementation checkpoint: update only the existing current phase workflow plan plus the master file, and update existing `tasks.md` checkbox/progress state when the task ledger is in use. If a needed workflow/process artifact is missing, reopen the relevant earlier phase instead of creating it mid-implementation.
- After review or reconciliation: update only existing control/checkpoint artifacts with review scope, findings status, orchestrator reconciliation, accepted risks, blockers, reopen targets, and validation implications. Do not paste raw review transcripts or invent new tasks in review control files.
- After any phase-complete handoff: reconcile blocking workflow plan adequacy challenge findings, mark `Session boundary reached`, `Ready for next session`, and `Next session starts with` in the master file, and close the current phase workflow plan.
- After validation: record completion or remaining blockers in the master file and the active validation phase workflow plan when one already exists; update `spec.md` `Validation` and `Outcome` to match the actual proof; update existing `tasks.md` checkbox/progress state only when already in use. If an expected validation control file or required `tasks.md` is missing, reopen the relevant earlier phase instead of creating it during closeout.

Minimal split example:

```text
workflow-plan.md:
Current phase: technical-design; Session boundary reached: no; Ready for next session: no
Next session starts with: planning; Phase workflow plans: specification complete; technical-design active; planning pending
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
3. phase artifacts in the order the current phase needs them:
   - `spec.md`
   - [docs/repo-architecture.md](./repo-architecture.md) when the task depends on stable repository architecture context
   - `design/overview.md`
   - remaining required design artifacts plus any triggered conditional design files
   - `tasks.md` when present or expected
   - optional `plan.md` when present
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

For exceptions, the skip rationale must explain skipped `workflow-plan.md`, `workflow-plans/`, `design/`, or `tasks.md`; if optional `plan.md` is skipped, no rationale is needed because it is not required. Without required-artifact rationales, assume the artifact chain is incomplete rather than silently waived.

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
- **Premature outcome claims:** Writing `Outcome` as done or successful before fresh validation evidence exists.
- **Readiness bypass:** Treating implementation as ready when implementation readiness is missing or `FAIL`, using `CONCERNS` without named accepted risks and proof obligations, or treating `WAIVED` as the default for work that is not tiny, direct-path, or prototype-scoped.
- **New process artifacts after code starts:** Creating new workflow/process markdown during implementation or validation instead of reopening the correct earlier phase.
- **Pre-reading review skills:** Loading multiple subagent-internal review or domain skill bodies in the main flow because their descriptions match, instead of routing one skill name per read-only lane.
- **Just-in-case artifacts:** Creating `test-plan.md`, `rollout.md`, or conditional design files "just in case".
- **Missing skip rationale:** Forgetting to record the skip rationale when bypassing `design/`.
- **Architecture re-derivation:** Re-deriving repository architecture every session instead of loading [docs/repo-architecture.md](./repo-architecture.md).
- **Execution-order archaeology:** Forcing the coder to reconstruct execution order from technical prose when expected `tasks.md` should exist.
- **Stale artifact status:** Leaving artifact status stale so resume requires chat archaeology.

# Spec-First Workflow (Orchestrator/Subagent-First)

## 1. Purpose

This document is the detailed runtime companion to [AGENTS.md](/Users/daniil/Projects/Opensource/go-service-template-rest/AGENTS.md).
It explains the repository's artifact-driven workflow once a task needs more than a quick local fix: which artifacts exist, when they appear, how they relate, and how to resume later without guessing from chat history.

This file is intentionally narrower than `AGENTS.md`: `AGENTS.md` stays the authority for role ownership, invariants, subagent protocol, and stage gates.
If the two documents ever diverge, follow `AGENTS.md` and then repair this file.

Use this document when the task is non-trivial, agent-backed, needs preserved research or multi-session resume, or reaches `technical design` or later.

When the task depends on stable repository boundaries or runtime flows, load [docs/repo-architecture.md](/Users/daniil/Projects/Opensource/go-service-template-rest/docs/repo-architecture.md) before writing task-local design.

## 2. What This Workflow Does Not Change

The richer artifact model does not change ownership:
- The orchestrator still owns framing, routing, final decisions, implementation, validation, and artifact authority.
- Subagents still stay read-only and advisory.
- Skills are still optional tools rather than the top-level workflow.
- `spec.md` still remains the canonical decisions artifact.

The new artifacts add control and technical context around `spec.md`; they do not replace it and they do not create a second authority chain.

## 3. Artifact Model

Smallest valid task-local layout:

```text
specs/<feature-id>/
  spec.md
```

Repository-wide stable architecture baseline:

```text
docs/
  repo-architecture.md
```

Non-trivial task-local bundle:

```text
specs/<feature-id>/
  workflow-plan.md
  workflow-plans/
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
  plan.md
  test-plan.md             # conditional
  rollout.md               # conditional
```

Artifact purposes:

| Artifact | Purpose | When it matters |
| --- | --- | --- |
| `workflow-plan.md` | Master routing and control artifact. It tracks cross-phase status: execution shape, current phase, artifact status, blockers, next session, links to phase workflow plans, and resume order. | Required for non-trivial or agent-backed work. |
| `workflow-plans/<phase>.md` | Phase-local workflow plan for one named phase only. It tracks that phase's local orchestration, order/parallelism, fan-in/challenge path when relevant, completion marker, stop rule, next action, and local blockers. | Required for each named non-trivial phase that the task actually uses. |
| `spec.md` | Canonical decision record: approved framing, scope, constraints, decisions, and accepted open questions. | Always. |
| `design/` | Task-local technical design bundle between `spec.md` and `plan.md`. It explains how the approved change fits the repository architecture and what implementation must preserve. | Required for non-trivial work unless explicitly skipped with rationale. |
| `plan.md` | Coder-facing execution plan derived from approved `spec.md + design/`. It owns ordered implementation steps, checkpoints, and proof obligations. | Required when the work is non-trivial, long-running, parallelized, or handed to an implementation skill. |
| `research/*.md` | Preserved evidence, comparisons, and validated research context. These files support decisions but do not own them. | Create only when the task is long, ambiguous, or benefits from reusable research memory. |
| `test-plan.md` | Expanded validation strategy when test obligations are too large or too layered to fit cleanly inside `plan.md`. | Conditional. |
| `rollout.md` | Operational rollout and migration notes when delivery order, compatibility, or recovery choreography matters. | Conditional. |

Artifact rules:
- [docs/repo-architecture.md](/Users/daniil/Projects/Opensource/go-service-template-rest/docs/repo-architecture.md) is the stable repository baseline, not a task-local design file.
- `workflow-plan.md` stays live through research, synthesis, `technical design`, planning, implementation, and validation; `workflow-plans/<phase>.md` stays phase-local and must not turn into a second `design/` bundle or `plan.md`.
- Pre-code phases normally get `workflow-plans/specification.md`, `workflow-plans/technical-design.md`, and `workflow-plans/planning.md`.
- Post-code phase-control files are still pre-code artifacts: if the approved phase structure will use them, planning creates `workflow-plans/implementation-phase-N.md`, `workflow-plans/review-phase-N.md`, and `workflow-plans/validation-phase-N.md` before implementation starts, and post-code sessions only update them.
- `spec.md` stores decisions, `design/` stores task-specific technical context, and `plan.md` stores execution order. Do not make any of them absorb another artifact's job.
- `research/*.md` should be flexible and evidence-oriented; there is no mandatory universal template.
- Do not duplicate the same authority across artifacts. Link instead.

### Artifact-Producing Vs Artifact-Consuming Phases

Pre-code phases (`workflow planning`, `research`, `specification`, `technical design`, and `planning`) may create only the approved workflow/process artifact set: `workflow-plan.md`, `workflow-plans/<phase>.md`, `research/*.md`, `spec.md`, `design/`, `plan.md`, optional `test-plan.md`, and optional `rollout.md`. If later implementation, review, or validation phase-control files will be used, planning creates them before implementation starts.

Post-code phases (`implementation`, `review`, `reconciliation`, and `validation`) consume that bundle. They may still create approved codebase files such as code, tests, migrations, configs, generation inputs, and generated output when the plan requires them, but they may update only existing control or closeout surfaces such as `workflow-plan.md`, the active `workflow-plans/<phase>.md`, checkpoint state in existing `plan.md`, and `spec.md` `Validation` or `Outcome`. If required context is missing, stop, record the reopen in existing control artifacts, and reopen the appropriate earlier phase in a new session instead of inventing new artifacts here.

Direct-path exceptions may skip parts of the pre-code artifact bundle with explicit rationale, but they do not authorize creating new workflow/process artifacts mid-implementation or mid-validation.

## 4. Default `spec.md` Shape

The default sections are:
1. `Context`
2. `Scope / Non-goals`
3. `Constraints`
4. `Decisions`
5. `Open Questions / Assumptions`
6. `Plan Summary / Link`
7. `Validation`
8. `Outcome`

Rules:
- Merge sections when that makes the file clearer.
- Do not create empty headings for completeness.
- Keep final decisions in `Decisions`.
- Keep research evidence in `research/*.md` when it is worth preserving.
- Keep only the planning summary or plan link in `spec.md` when `plan.md` exists.

## 5. The Design Bundle Between `spec.md` And `plan.md`

For non-trivial work, `technical design` is now an explicit stage between `specification` and `planning`.
The design bundle carries task-specific technical context that `spec.md` should not absorb: how the approved change fits the repository structure, which components participate, what runtime sequence matters, where ownership and source-of-truth boundaries sit, what stays stable, and what creates correctness or rollout risk.

Load order for design work:
1. Read [docs/repo-architecture.md](/Users/daniil/Projects/Opensource/go-service-template-rest/docs/repo-architecture.md) when stable repository boundaries or flows matter.
2. Read the approved `spec.md`.
3. Produce the task-local `design/` bundle.
4. Plan from approved `spec.md + design/`.

Required core design artifacts for non-trivial work:
- `design/overview.md` — design entrypoint, chosen approach, artifact index, unresolved seams, and readiness summary.
- `design/component-map.md` — affected packages, modules, or components; what changes; what remains stable.
- `design/sequence.md` — call order, sync or async boundaries, failure points, side effects, and parallel versus sequential behavior.
- `design/ownership-map.md` — source-of-truth ownership, allowed dependency direction, and responsibility boundaries.

Conditional artifacts and trigger rules:
- `design/data-model.md` — create when the task changes persisted state, schema, cache contract, projections, replay behavior, or migration shape.
- `design/dependency-graph.md` — create when the task changes module or package dependency shape, generated-code dependency flow, or introduces a coupling risk that must be made explicit.
- `design/contracts/` — create when the task changes API contracts, event contracts, generated contracts, or material internal interfaces between subsystems. This folder is design-only context for the task, not an authoritative runtime contract source; canonical sources like `api/openapi/service.yaml`, generation inputs, and other repository-owned contract artifacts still win.
- `test-plan.md` — create when validation obligations are too large or multi-layered to fit cleanly inside `plan.md`.
- `rollout.md` — create when the task needs migration sequencing, backfill and verify choreography, mixed-version compatibility, or explicit deploy and failback notes.

Design-bundle rules:
- `design/overview.md` is the entrypoint and link surface for the bundle.
- Create conditional artifacts only when their trigger is real.
- Keep technical design in `design/`; do not push it back into `spec.md`.
- Record design artifact status in `workflow-plans/technical-design.md` and master `workflow-plan.md`.
- Tiny or `direct path` work may skip the design bundle only with an explicit design-skip rationale.

Concise repository-native example:

```text
Task: add a new generated admin API that writes durable state and publishes an async event

Required core:
- design/overview.md
- design/component-map.md
- design/sequence.md
- design/ownership-map.md

Also create:
- design/data-model.md   # persisted state or migration shape changes
- design/contracts/      # API or event contract changes

Skip:
- design/dependency-graph.md  # package dependency shape stays the same

Then `plan.md` turns the approved `spec.md + design/` bundle into phases such as contract + generation, schema or repository changes, app + transport + event wiring, and validation.
```

## 6. Execution Loop

Typical paths:
- Direct / lightweight local:
  `intake -> workflow planning -> research -> synthesis -> specification -> technical design -> planning -> implementation -> validation -> done`
- Idea-shaped work:
  `intake -> idea refinement -> workflow planning -> research -> synthesis -> specification -> technical design -> planning -> implementation -> validation -> done`
- Full orchestrated:
  `intake -> workflow planning -> research -> synthesis(candidate -> challenge -> final) -> specification -> technical design -> planning -> implementation -> review -> reconciliation -> validation -> done`

For very small work, several of these stages may collapse into one local pass.
The stage names still matter because artifact expectations and resume logic depend on them.

### Session-Bounded Phases

For non-trivial work, the repository treats a session as phase-scoped by default.
The default named phases are:
- `specification`
- `technical-design`
- `planning`
- `implementation-phase-N`
- optional `review-phase-N`
- optional `validation-phase-N`

These phases normally map to phase-local workflow plans:
- `workflow-plans/specification.md`
- `workflow-plans/technical-design.md`
- `workflow-plans/planning.md`
- `workflow-plans/implementation-phase-N.md` for each named implementation phase from `plan.md`
- `workflow-plans/review-phase-N.md` only when review is a dedicated post-code phase
- `workflow-plans/validation-phase-N.md` only when validation is a dedicated post-code phase

Rule:
- one session = one phase for non-trivial work unless an upfront `direct path` or `lightweight local` waiver was recorded before the boundary is crossed
- when that phase reaches its completion marker, update the owning artifact, the current `workflow-plans/<phase>.md`, and master `workflow-plan.md`, then stop
- begin the next phase in a new session

This is a control rule, not a second state machine: the workflow states stay the same, and the session-bounded phase tells later sessions where to resume and where not to drift.

### 6.1 Idea Refinement and Framing

If the request is still idea-shaped, refine it before treating it as implementation-ready.
If the request is already concrete, move directly into framing:
- what must change,
- what is in scope and out of scope,
- which constraints or risk hotspots already exist,
- which success checks matter,
- which facts are still missing.

### 6.2 Workflow Planning

Before any subagent call on non-trivial or agent-backed work, the orchestrator writes master `workflow-plan.md` plus the current phase workflow plan at `workflow-plans/<phase>.md`. The master file owns cross-phase control such as execution shape, current phase, artifact status, blockers, next-session routing, phase-plan links/status, and whether later `design/`, `plan.md`, `test-plan.md`, or `rollout.md` artifacts are expected. The phase file owns only local orchestration such as research mode, lanes, order or parallelism, fan-in or challenge path, status, completion marker, stop rule, next action, blockers, and parallelizable work. Section 7 defines the minimum shape in more detail.

For tiny local work, a brief explicit skip rationale in the main flow is enough instead of a full master `workflow-plan.md` plus `workflow-plans/`.

### 6.3 Research, Synthesis, and Specification

Research may stay local or fan out to read-only subagents, depending on the workflow plan.
After research:
- synthesize comparable claims,
- resolve or explicitly track key assumptions,
- run pre-spec challenge when task risk or ambiguity justifies it,
- stabilize final decisions in `spec.md`,
- preserve reusable evidence in `research/*.md` when worth keeping.

`spec.md` should be stable enough that `technical design` can derive task-local context without reopening core problem framing by default.

### 6.4 Technical Design

`technical design` begins after `spec.md` is stable enough to support design work.
This stage:
- uses [docs/repo-architecture.md](/Users/daniil/Projects/Opensource/go-service-template-rest/docs/repo-architecture.md) when repository baseline context matters,
- produces the required core `design/` artifacts,
- adds conditional design artifacts only when triggered,
- records approval state back in `workflow-plans/technical-design.md` and master `workflow-plan.md`.

If design work exposes a missing decision or unstable problem boundary, loop back to `spec.md` instead of silently letting design redefine the task.

### 6.5 Planning

Planning is separate from workflow planning.
Workflow planning decides orchestration and artifact expectations.
Implementation planning decides execution order after decisions and design are stable.

Planning entry rules:
- minimum viable framing is explicit,
- workflow planning already chose `local` or `fan-out`,
- `spec.md` is stable,
- required `design/` artifacts are approved, or an explicit design-skip rationale exists,
- for higher-risk work, pre-spec challenge is reconciled or explicitly waived.

Planning rules:
- for non-trivial work, `planning-and-task-breakdown` should consume approved `spec.md + design/` and produce the dedicated coder-facing `plan.md`,
- for `direct path` work, the explicit plan may stay as 1-3 concise lines in the main flow,
- for non-trivial work, planning is the last artifact-producing phase before code: the default phase order is `specification -> technical-design -> planning -> implementation-phase-N`, with optional `review-phase-N` and `validation-phase-N`, and the workflow/design/planning bundle that implementation will consume must already exist or be explicitly waived before the first implementation session starts,
- if the workflow will use `workflow-plans/implementation-phase-N.md`, `workflow-plans/review-phase-N.md`, or `workflow-plans/validation-phase-N.md`, create them during planning from the approved phase structure so post-code sessions only update them,
- phased execution is the default for non-trivial work: `phase -> review/reconcile -> validate -> next phase`,
- single-pass big-bang implementation needs explicit rationale.

Minimum `plan.md` content:
- ordered implementation steps,
- completion criteria for each meaningful step or phase,
- checkpoints and dependencies when relevant,
- validation expectations,
- rollback or mitigation notes only when the task actually needs them.

### Session-Boundary Gate

For non-trivial work, a session may advance only the `Current phase` recorded in master `workflow-plan.md` and the matching `workflow-plans/<phase>.md`.
When that phase's completion marker is satisfied, update the owning artifact, the current phase workflow plan, and master `workflow-plan.md`; mark `Session boundary reached: yes`; set `Ready for next session` appropriately; record `Next session starts with`; and stop instead of beginning the next phase in the same session.

If the phase cannot be finished honestly, end with the same phase still `in_progress` or `blocked`.
`Direct path` work and any upfront `lightweight local` waiver may collapse those boundaries only when the waiver is recorded before the boundary is crossed.

### 6.6 Implementation, Review, and Validation

Implementation happens in the main flow under orchestrator control.
Treat implementation as artifact-consuming:
- consume approved `spec.md`, `design/`, `plan.md`, optional `test-plan.md`, optional `rollout.md`, and any pre-created post-code phase workflow files,
- allow new code files, test files, migrations, configs, generation inputs, and generated artifacts only when the approved plan requires them,
- update only existing control and checkpoint artifacts such as the current `workflow-plan.md`, the active `workflow-plans/implementation-phase-N.md`, and progress state in existing `plan.md`,
- do not create new workflow/process/planning/design/temp artifacts or ad hoc progress markdown once implementation has started,
- if coding exposes a real plan or design gap, stop, record the reopen in the existing control artifacts, and reopen the relevant earlier phase in a new session instead of silently drifting.

Review stays read-only and risk-driven.
Validation is also artifact-consuming:
- use fresh evidence against the approved artifact bundle,
- update only existing closeout surfaces such as `workflow-plan.md`, the active `workflow-plans/validation-phase-N.md` when one was created before implementation, progress state in existing `plan.md` when needed, and `spec.md` `Validation` or `Outcome`,
- do not create new workflow/process/planning/design/temp artifacts during validation,
- if proof exposes a real upstream gap or an expected control artifact is missing, reopen the right earlier phase instead of inventing a new artifact during closeout.

Closeout is not complete until the artifacts reflect reality: `workflow-plan.md` shows the current phase or completion state, the current `workflow-plans/<phase>.md` shows the local handoff state, `plan.md` or `workflow-plan.md` reflects what phase completed or remains, `spec.md` records actual outcome and remaining open questions, and `Validation`/`Outcome` reflect what was actually proved.

## 7. Master `workflow-plan.md` And Phase Workflow Plans

The repository uses two workflow-control layers for non-trivial work:
- master `workflow-plan.md` for cross-phase control
- `workflow-plans/<phase>.md` for one phase only

Read the master file first, then the current phase workflow plan.

### 7.1 Master `workflow-plan.md`

`workflow-plan.md` owns runtime control across phases.
At minimum, it should always answer:
1. What phase is current right now?
2. Is that phase `in_progress`, `blocked`, or `complete`?
3. Has the session boundary been reached?
4. Is the task ready for the next session?
5. What phase does the next session start with?
6. Which artifacts are `approved`, `draft`, or `missing`?
7. What is blocked?
8. Which phase workflow plans exist, are active, or are still pending?
9. What is the default resume order?

### 7.2 `workflow-plans/<phase>.md`

`workflow-plans/<phase>.md` owns orchestration for one named phase only.
It should answer:
1. What local orchestration runs in this phase?
2. Which subagent lanes or local tracks run, and in what order or parallelism?
3. What completion marker ends this phase?
4. What is explicitly out of scope for this phase?
5. What is the stop rule for this phase?
6. What is the next action?
7. What can run in parallel?
8. What local blockers remain?

This file must not replace `spec.md`, `design/`, or `plan.md`.
It is phase-local routing, not a competing design or execution artifact.

Recommended update cadence:
- After framing or workflow planning: update the master file with execution shape, current phase, blockers, next-session routing, phase-plan links/status, and artifact expectations; update the current phase file with local orchestration, lanes, completion marker, and stop rule.
- After synthesis: update `spec.md` status in the master file and record any blocker that prevents leaving `workflow-plans/specification.md`.
- After `technical design` or planning: record approved design artifacts or `plan.md` status in the master file and current phase file; during planning, also create any implementation/review/validation phase workflow files that the approved phase structure will use.
- After each implementation checkpoint: update only the existing current phase workflow plan plus the master file. If a needed workflow/process artifact is missing, reopen the relevant earlier phase instead of creating it mid-implementation.
- After any phase-complete handoff: mark `Session boundary reached`, `Ready for next session`, and `Next session starts with` in the master file and close the current phase workflow plan.
- After validation: record completion or remaining blockers in the master file and the active validation phase workflow plan when one already exists; update `spec.md` `Validation` and `Outcome` to match the actual proof. If an expected validation control file is missing, reopen the relevant earlier phase instead of creating it during closeout.

Concise split example:

Master `workflow-plan.md`

```text
Current phase: technical-design
Session boundary reached: no
Ready for next session: no
Next session starts with: planning

Phase workflow plans:
- specification: complete
- technical-design: active
- planning: pending

Artifacts:
- spec.md: approved
- design/: draft
- plan.md: missing

Blockers:
- open cache invalidation decision from research/cache-contract.md
```

Current `workflow-plans/technical-design.md`

```text
Phase: technical-design
Phase status: in_progress
Completion marker:
- required design artifacts approved
- planning inputs stable

Next action:
- finish sequence and ownership mapping

Can run in parallel:
- draft design/component-map.md

Stop rule:
- do not begin planning in this session
```

## 8. Resume Order And Stage Inference

### Resume order

In a later session, read artifacts in this order:
1. `workflow-plan.md`
2. current `workflow-plans/<phase>.md`
3. phase artifacts in the order the current phase needs them:
   - `spec.md`
   - [docs/repo-architecture.md](/Users/daniil/Projects/Opensource/go-service-template-rest/docs/repo-architecture.md) when the task depends on stable repository architecture context
   - `design/overview.md`
   - remaining required design artifacts plus any triggered conditional design files
   - `plan.md`
   - optional `test-plan.md`, `rollout.md`, and selected `research/*.md`

If the task was intentionally small enough to skip some artifacts, read the recorded skip rationale before assuming the artifact is merely missing.

### How to infer the current stage from artifacts

Use artifacts, not memory:
- no approved `spec.md` means framing or specification is still incomplete
- approved `spec.md` but no approved design bundle means `technical design` is still incomplete
- approved design bundle but no approved `plan.md` means planning is still incomplete
- approved `plan.md` means the task is implementation-ready; `workflow-plan.md` then shows whether implementation is still ahead or already in progress
- validation evidence plus updated `Outcome` means the task has reached validation or done

Use session control from the master file before doing any work:
- if the current phase points at a missing `workflow-plans/<phase>.md`, treat the phase workflow record as incomplete rather than reconstructing it from memory
- if `Session boundary reached: yes`, start a new session for the recorded next phase
- if `Ready for next session: no`, resume the same session-bounded phase instead of jumping forward
- if a reopen target points backward, reopen that earlier phase instead of continuing from the later artifact state

Stage inference rules for exceptions:
- if the task is `direct path` or tiny and intentionally skips `workflow-plan.md`, `workflow-plans/`, or `design/`, the skip rationale should say why those artifacts are unnecessary
- if the task skips a separate `plan.md`, the main flow or master `workflow-plan.md` should still make the current execution step explicit
- if those rationales are absent, assume the artifact chain is incomplete rather than silently waived

## 9. Direct-Path And Lightweight-Local Exceptions

Direct-path and lightweight-local work still exists.
The workflow is not trying to force the full artifact bundle onto tiny fixes.

For these smaller execution shapes:
- workflow planning, research, synthesis, specification, `technical design`, and planning may collapse into one local pass,
- a separate master `workflow-plan.md` and `workflow-plans/` folder may be skipped for tiny work,
- a separate `plan.md` may be unnecessary,
- the design bundle may be skipped only when the change is local, the behavior delta is obvious, and no ownership, data, or sequence ambiguity exists,
- same-session phase collapse is allowed only when the waiver is recorded before the boundary is crossed.

What does not get skipped:
- explicit planning-before-code,
- clear decision ownership,
- fresh validation evidence,
- explicit rationale for bypassing `design/` when it would otherwise be expected.

Concise skip-rationale example:

```text
Design skip rationale:
- local validator change in one package
- no persisted-state change
- no contract change
- no ownership or runtime-sequence ambiguity
- plan kept inline because execution is one short reversible step
```

If a supposedly small task uncovers a larger seam, escalate to the fuller artifact chain instead of pretending the original shortcut still fits.

## 10. Artifact-Focused Anti-Patterns

Avoid:
- treating `workflow-plan.md` as a one-time pre-research note instead of the live master control artifact,
- letting `workflow-plans/<phase>.md` replace the master `workflow-plan.md` or grow into a competing design or execution artifact,
- finishing one non-trivial phase and casually starting the next one in the same session without an upfront recorded waiver,
- planning non-trivial work from `spec.md` alone after the design-bundle stage exists,
- letting `design/` turn into a second `spec.md` or a second `plan.md`,
- creating new workflow/process markdown during implementation or validation instead of reopening the correct earlier phase,
- creating `test-plan.md`, `rollout.md`, or conditional design files "just in case",
- forgetting to record the skip rationale when bypassing `design/`,
- re-deriving repository architecture every session instead of loading [docs/repo-architecture.md](/Users/daniil/Projects/Opensource/go-service-template-rest/docs/repo-architecture.md),
- forcing the coder to reconstruct execution order from technical prose when `plan.md` should exist,
- leaving artifact status stale so resume requires chat archaeology.

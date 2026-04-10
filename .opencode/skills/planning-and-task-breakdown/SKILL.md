---
name: planning-and-task-breakdown
description: "Turn approved `spec.md + design/` into phased, dependency-ordered, verifiable execution work for this repository. Use after `spec.md` is stable, required technical-design artifacts are approved or explicitly skipped, and pre-spec challenge is reconciled, whenever implementation should be driven from `plan.md` rather than improvised from the decision/design record. Reach for this when the work is large enough that execution order, checkpoints, or parallelism are not obvious. Skip unresolved architecture/API/data/security/reliability decisions and skip actual coding."
---

# Planning And Task Breakdown

## Purpose
Turn stable decisions plus approved technical design into a coder-facing execution plan that is small-slice, phase-aware, and honest about dependencies, checkpoints, and proof obligations.

## Scope
- convert approved decisions from `spec.md` and task-local technical context from `design/` into dependency-ordered phases and tasks
- make `plan.md` explicit when the work is non-trivial
- attach acceptance criteria, planned verification, checkpoints, and change-surface hints
- expose blockers, assumptions, and reopen conditions before coding starts
- preserve coder freedom on local code shape while removing ambiguity about execution order

## Boundaries
Do not:
- make new architecture, API, data, security, reliability, or rollout decisions
- reconstruct missing architecture, ownership, data, or sequence context from `spec.md` alone when `design/` should supply it
- write production code, tests, or migrations as the main deliverable
- dump raw research or repeat the whole spec in planning form
- treat `spec.md` as the place for full task breakdown by default
- hide blocked work behind optimistic task wording

## Escalate When
Escalate if:
- `spec.md` is not stable enough to derive tasks without reopening design
- non-trivial work is missing `design/overview.md`, the required core design artifacts, or an explicit design-skip rationale
- a conditional design artifact is clearly triggered but missing
- core behavior is still undecided across architecture, API, data, security, reliability, or domain semantics
- the right implementation order depends on a missing migration, compatibility, or ownership decision
- the change cannot be decomposed without inventing detail the spec does not actually approve

## Core Defaults
- `spec.md` is for decisions, `design/` is for technical context, and `plan.md` is for execution.
- For non-trivial work, plan from approved `spec.md + design/`, not from `spec.md` alone.
- Planning is the last artifact-producing phase before code. If later `workflow-plans/implementation-phase-N.md`, `workflow-plans/review-phase-N.md`, or `workflow-plans/validation-phase-N.md` are part of the approved execution shape, plan for them to exist before implementation starts instead of being invented later.
- Prefer phased execution over one giant task list.
- Prefer dependency-ordered vertical slices over horizontal subsystem dumps when possible.
- Keep tasks small enough to implement, verify, and review in one focused session.
- For non-trivial work, this pass ends the current session at approved `plan.md`; implementation begins in a new session unless an upfront `direct path` or `lightweight local` waiver was already recorded.
- Put risky or dependency-establishing work early.
- Use checkpoints to create real stop points, not ritual paperwork.
- Do not let `plan.md` become an architecture reconstruction document.

## Planning Workflow

### 1. Confirm Planning Readiness
- Read the stable `spec.md` and the relevant design bundle, not just the chat.
- Confirm that the main decisions, design constraints, ownership boundaries, and open questions are explicit.
- For non-trivial work, require `design/overview.md`, `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md` unless there is an explicit design-skip rationale.
- If the design or spec is not stable enough, stop and escalate instead of guessing.

### 2. Load Execution-Critical Design Context
- Use `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md` to understand what must land first and what may move in parallel.
- Load triggered conditional artifacts such as `design/data-model.md`, `design/dependency-graph.md`, `design/contracts/`, `test-plan.md`, or `rollout.md` when they affect sequencing.
- Identify what must exist first: schema or config changes, generated artifacts, interfaces, handlers, background workers, tests, docs, or migration controls.
- Make the ordering explicit when one task truly depends on another.
- Do not confuse implementation taste with real dependency.

### 3. Slice The Work
- Prefer one coherent reviewable increment per phase.
- When possible, use vertical slices that land observable behavior.
- If the work must start with enabling seams or migration groundwork, say so directly.
- If two tasks must land together to remain safe, explain the coupling.
- Use the design bundle's ownership and sequence constraints to decide where slices can and cannot be separated.

### 4. Write The Task Breakdown
- For each phase, list the concrete tasks.
- For each task, make acceptance criteria and planned verification explicit.
- Name likely change surfaces by package, layer, or artifact owner instead of rigid file lock-in unless exact files are a real constraint.

### 5. Add Checkpoints
- Add review and validation checkpoints at natural risk boundaries.
- Each checkpoint should say what must be true before the next phase starts.
- Keep checkpoints proportional; tiny work may need one final checkpoint only.

## Task Sizing
- `XS`: one tiny local step; prefer keeping it inline unless the surrounding work is already non-trivial
- `S`: one focused task, usually one behavior seam or one enabling change
- `M`: a small feature slice or tightly coupled implementation bundle
- `L+`: break it down further unless the coupling is unavoidable and explicitly named

Break a task down further when:
- it would take more than one focused coding session
- acceptance criteria cannot stay short and concrete
- it touches multiple independent subsystems
- the title needs `and` to stay accurate

## Preferred `plan.md` Shape
Return plan text that can drop directly into `plan.md` with minimal rewriting.

Use this structure by default:
- `Execution Context`
- `Phase Plan`
- `Cross-Phase Validation Plan`
- `Blockers / Assumptions`
- `Handoffs / Reopen Conditions` when relevant

For each phase, include:
- `Phase`
- `Objective`
- `Depends On` when nontrivial
- `Tasks`
- `Acceptance Criteria`
- `Change Surface`
- `Planned Verification`
- `Review / Checkpoint`
- `Exit Criteria`

## Planning Rules
- For direct-path work, a short inline plan may still be enough; do not force `plan.md` for a tiny change just to satisfy ceremony.
- For non-trivial work, default to `plan.md` and consume approved `spec.md + design/`.
- When later implementation, review, or validation phase-control files are part of the execution shape, planning should leave them ready to be created or linked before implementation begins; post-code phases should not need to invent new workflow/process artifacts.
- If required design artifacts are missing or inconsistent, reopen technical design instead of inferring the missing context locally.
- Keep planning aligned with repository realities: OpenAPI drift checks, `sqlc` regeneration, migrations, race tests, integration checks, or other real verification surfaces when they actually apply.
- If a phase is not independently mergeable or testable, name the coupling explicitly.
- Prefer sequential phases unless change surfaces are truly disjoint.
- Make the handoff explicit: the planning session stops at approved `plan.md`, and the first implementation phase starts in a new session unless a recorded waiver says otherwise.
- State what should trigger a reopen back into specification or technical design instead of letting coding discover it silently.

## Definition Of Done
The planning pass is complete when:
- the execution order is explicit
- each meaningful task has acceptance criteria and planned verification
- checkpoints exist where the risk actually changes
- blocked work is clearly separated from ready work
- the next implementation or validation session can start without creating new workflow/process artifacts to compensate for missing planning output
- the next session can start implementation without re-planning or guessing where this planning pass was supposed to stop
- the plan is specific enough for `go-coder` to execute without recreating the strategy or reverse-engineering missing design context

## Escalate Or Reject
- task breakdown derived from an unstable spec
- task breakdown that assumes missing `design/` context instead of escalating
- a phase list with no acceptance criteria or verification
- a generic task like `implement the feature`
- horizontal slicing that hides risk and postpones integration until the end
- planning output that duplicates the entire spec instead of turning it into execution work

---
name: planning-and-task-breakdown
description: "Turn an approved spec into phased, dependency-ordered, verifiable execution work for this repository. Use after `spec.md` is stable and pre-spec challenge is reconciled, whenever implementation should be driven from `plan.md` rather than improvised from the spec. Reach for this when the work is large enough that execution order, checkpoints, or parallelism are not obvious. Skip unresolved architecture/API/data/security/reliability decisions and skip actual coding."
---

# Planning And Task Breakdown

## Purpose
Turn a stable spec into a coder-facing execution plan that is small-slice, phase-aware, and honest about dependencies, checkpoints, and proof obligations.

## Scope
- convert approved decisions from `spec.md` into dependency-ordered phases and tasks
- make `plan.md` explicit when the work is non-trivial
- attach acceptance criteria, planned verification, checkpoints, and change-surface hints
- expose blockers, assumptions, and reopen conditions before coding starts
- preserve coder freedom on local code shape while removing ambiguity about execution order

## Boundaries
Do not:
- make new architecture, API, data, security, reliability, or rollout decisions
- write production code, tests, or migrations as the main deliverable
- dump raw research or repeat the whole spec in planning form
- treat `spec.md` as the place for full task breakdown by default
- hide blocked work behind optimistic task wording

## Escalate When
Escalate if:
- `spec.md` is not stable enough to derive tasks without reopening design
- core behavior is still undecided across architecture, API, data, security, reliability, or domain semantics
- the right implementation order depends on a missing migration, compatibility, or ownership decision
- the change cannot be decomposed without inventing detail the spec does not actually approve

## Core Defaults
- `spec.md` is for decisions; `plan.md` is for execution.
- Prefer phased execution over one giant task list.
- Prefer dependency-ordered vertical slices over horizontal subsystem dumps when possible.
- Keep tasks small enough to implement, verify, and review in one focused session.
- Put risky or dependency-establishing work early.
- Use checkpoints to create real stop points, not ritual paperwork.

## Planning Workflow

### 1. Confirm Planning Readiness
- Read the stable spec, not just the chat.
- Confirm that the main decisions and open questions are explicit.
- If the spec is not stable enough, stop and escalate instead of guessing.

### 2. Map The Dependency Graph
- Identify what must exist first: schema or config changes, generated artifacts, interfaces, handlers, background workers, tests, docs, or migration controls.
- Make the ordering explicit when one task truly depends on another.
- Do not confuse implementation taste with real dependency.

### 3. Slice The Work
- Prefer one coherent reviewable increment per phase.
- When possible, use vertical slices that land observable behavior.
- If the work must start with enabling seams or migration groundwork, say so directly.
- If two tasks must land together to remain safe, explain the coupling.

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
- For non-trivial work, default to `plan.md`.
- Keep planning aligned with repository realities: OpenAPI drift checks, `sqlc` regeneration, migrations, race tests, integration checks, or other real verification surfaces when they actually apply.
- If a phase is not independently mergeable or testable, name the coupling explicitly.
- Prefer sequential phases unless change surfaces are truly disjoint.
- State what should trigger a reopen back into specification instead of letting coding discover it silently.

## Definition Of Done
The planning pass is complete when:
- the execution order is explicit
- each meaningful task has acceptance criteria and planned verification
- checkpoints exist where the risk actually changes
- blocked work is clearly separated from ready work
- the plan is specific enough for `go-coder` to execute without recreating the strategy from scratch

## Escalate Or Reject
- task breakdown derived from an unstable spec
- a phase list with no acceptance criteria or verification
- a generic task like `implement the feature`
- horizontal slicing that hides risk and postpones integration until the end
- planning output that duplicates the entire spec instead of turning it into execution work

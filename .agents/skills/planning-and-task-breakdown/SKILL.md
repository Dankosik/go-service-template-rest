---
name: planning-and-task-breakdown
description: "Turn approved `spec.md + design/` into `plan.md` strategy plus a dependency-ordered, verifiable `tasks.md` ledger for this repository. Use after `spec.md` is stable, required technical-design artifacts are approved or explicitly skipped, and pre-spec challenge is reconciled, whenever implementation should be driven from planning artifacts rather than improvised from the decision/design record. Reach for this when the work is large enough that execution order, checkpoints, task ledger, or parallelism are not obvious. Skip unresolved architecture/API/data/security/reliability decisions and skip actual coding."
---

# Planning And Task Breakdown

## Purpose
Turn stable decisions plus approved technical design into `plan.md` execution strategy and a `tasks.md` executable task ledger that are small-slice, phase-aware, and honest about dependencies, checkpoints, and proof obligations.

## Scope
- convert approved decisions from `spec.md` and task-local technical context from `design/` into dependency-ordered phases and executable tasks
- make `plan.md` explicit when the work is non-trivial
- make `tasks.md` explicit by default when non-trivial work has `plan.md`
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
- let `tasks.md` become a second spec, second design bundle, or competing plan
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
- `spec.md` is for decisions, `design/` is for technical context, `plan.md` is for execution strategy, and `tasks.md` is for the executable task ledger.
- For non-trivial work, plan from approved `spec.md + design/`, not from `spec.md` alone.
- For non-trivial work with `plan.md`, default to creating `tasks.md`; direct-path or tiny work may skip a separate ledger only with an explicit waiver.
- Planning is the last artifact-producing phase before code. If later `workflow-plans/implementation-phase-N.md`, `workflow-plans/review-phase-N.md`, or `workflow-plans/validation-phase-N.md` are part of the approved execution shape, plan for them to exist before implementation starts instead of being invented later.
- When the planning pass generates or materially changes workflow-control files, expect a read-only `workflow-plan-adequacy-challenge` before handoff; do not treat `plan.md` detail as a substitute for adequate `workflow-plan.md` and `workflow-plans/<phase>.md` routing.
- Prefer phased execution over one giant task list.
- Prefer dependency-ordered vertical slices over horizontal subsystem dumps when possible.
- Keep tasks small enough to implement, verify, and review in one focused session.
- For non-trivial work, this pass ends the current session at approved `plan.md` and expected `tasks.md`; implementation begins in a new session unless an upfront `direct path` or `lightweight local` waiver was already recorded.
- Put risky or dependency-establishing work early.
- Use checkpoints to create real stop points, not ritual paperwork.
- Do not let `plan.md` become an architecture reconstruction document.
- Do not let `tasks.md` absorb phase strategy, design decisions, or speculative tasking that should reopen design.

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

### 4. Write The Plan And Task Ledger
- Use `plan.md` for phase strategy: objectives, dependencies, checkpoints, validation plan, risk notes, and reopen conditions.
- Use `tasks.md` for executable task checkboxes.
- For each executable task, make the action, dependency marker when nontrivial, change surface, and planned verification explicit.
- Name exact file paths when known. When exact file choice is genuinely design-time unknown, name a narrow package or artifact surface instead of vague subsystem labels.
- Do not add a task if tasking it requires inventing a missing design decision; reopen `technical design` instead.

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
- `Task Ledger Link / IDs`
- `Acceptance Criteria`
- `Change Surface`
- `Planned Verification`
- `Review / Checkpoint`
- `Exit Criteria`

Keep executable checkbox detail in `tasks.md` instead of expanding `plan.md` into a task ledger.

## Preferred `tasks.md` Shape
Return ledger text that can drop directly into `tasks.md` with minimal rewriting.

Use markdown checkboxes. Each task should include:
- stable task ID such as `T001`
- phase/checkpoint label
- optional `[P]` marker only when safe to parallelize
- short action
- exact file path when known, or a narrow package/artifact surface when exact file choice is genuinely design-time unknown
- dependency marker when nontrivial, such as `Depends on: T001`
- proof/verification expectation

Example:

```markdown
- [ ] T001 [Phase 1] Update `internal/http/handler.go` to preserve request ID echo behavior. Depends on: none. Proof: `go test ./internal/http`.
- [ ] T002 [Phase 1] [P] Add regression coverage in `internal/http/handler_test.go`. Depends on: T001. Proof: `go test ./internal/http`.
```

Prefer vertical, reviewable slices. Avoid generic tasks like `implement feature`.

## Planning Rules
- For direct-path work, a short inline plan may still be enough; do not force `plan.md` or `tasks.md` for a tiny change just to satisfy ceremony.
- For non-trivial work, default to `plan.md` plus `tasks.md` and consume approved `spec.md + design/`.
- When later implementation, review, or validation phase-control files are part of the execution shape, planning should leave them ready to be created or linked before implementation begins; post-code phases should not need to invent new workflow/process artifacts.
- The workflow-control handoff must be challenge-ready: master and phase-local plans should make phase status, blockers, stop rules, next-session start, `tasks.md` status, artifact expectations, and generated post-code phase files clear enough for an adequacy challenger to review without reconstructing intent from chat.
- If required design artifacts are missing or inconsistent, reopen technical design instead of inferring the missing context locally.
- Keep planning aligned with repository realities: OpenAPI drift checks, `sqlc` regeneration, migrations, race tests, integration checks, or other real verification surfaces when they actually apply.
- If a phase is not independently mergeable or testable, name the coupling explicitly.
- Prefer sequential phases unless change surfaces are truly disjoint.
- Make the handoff explicit: the planning session stops at approved `plan.md` and expected `tasks.md`, and the first implementation phase starts in a new session unless a recorded waiver says otherwise.
- State what should trigger a reopen back into specification or technical design instead of letting coding discover it silently.

## Definition Of Done
The planning pass is complete when:
- the execution order is explicit
- each meaningful task in `tasks.md` has concrete action, dependency/proof context, and planned verification
- checkpoints exist where the risk actually changes
- blocked work is clearly separated from ready work
- the next implementation or validation session can start without creating new workflow/process artifacts or missing `tasks.md` to compensate for incomplete planning output
- the workflow-control artifacts are ready for the read-only adequacy challenge, or the direct-path skip rationale is explicit
- the next session can start implementation without re-planning or guessing where this planning pass was supposed to stop
- the plan and task ledger are specific enough for `go-coder` to execute without recreating the strategy or reverse-engineering missing design context

## Escalate Or Reject
- task breakdown derived from an unstable spec
- task breakdown that assumes missing `design/` context instead of escalating
- a phase list with no acceptance criteria or verification
- a generic task like `implement the feature`
- horizontal slicing that hides risk and postpones integration until the end
- a `tasks.md` ledger that duplicates strategy from `plan.md` instead of listing executable, proof-bound work
- planning output that leaves workflow-control routing too vague for adequacy review before handoff
- planning output that duplicates the entire spec instead of turning it into execution work

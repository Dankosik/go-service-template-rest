---
name: planning-and-task-breakdown
description: "Turn approved `spec.md + design/` into a dependency-ordered, verifiable `tasks.md` ledger for this repository. Use after `spec.md` is stable, required technical-design artifacts are approved or explicitly skipped, and pre-spec challenge is reconciled, whenever implementation should be driven from planning artifacts rather than improvised from the decision/design record. Reach for this when executable task order, checkpoints, or parallelism are not obvious. Skip unresolved architecture/API/data/security/reliability decisions and skip actual coding."
---

# Planning And Task Breakdown

## Purpose
Turn stable decisions plus approved technical design into a `tasks.md` executable task ledger that is small-slice, phase-aware, and honest about dependencies, checkpoints, and proof obligations.

## Scope
- convert approved decisions from `spec.md` and task-local technical context from `design/` into dependency-ordered executable tasks
- make `tasks.md` explicit by default for non-trivial implementation work
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
- let `tasks.md` become a second spec, second design bundle, or bloated strategy memo
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
- `spec.md` is for decisions, `design/` is for technical context, and `tasks.md` is for the executable task ledger and final implementation handoff.
- For non-trivial work, plan from approved `spec.md + design/`, not from `spec.md` alone.
- For non-trivial work, default to creating `tasks.md`; direct-path or tiny work may skip a separate ledger only with an explicit waiver.
- Keep `tasks.md` task-local to the active spec-first bundle. Do not use a repository-root or historical ledger as the current implementation handoff unless workflow control explicitly reopens it and records the resume route.
- Planning is the last artifact-producing phase before code. If later `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` are truly needed for named multi-session routing, create only those files before implementation starts instead of inventing them later. Do not create a coding phase-control file.
- Planning must leave an implementation-readiness result for the handoff: `PASS`, `CONCERNS`, `FAIL`, or `WAIVED`. `CONCERNS` needs named accepted risks and proof obligations; `FAIL` names the earlier phase to reopen; `WAIVED` stays limited to explicit tiny/direct-path/prototype scope.
- When the planning pass generates or materially changes workflow-control files, expect a read-only `workflow-plan-adequacy-challenge` before handoff; do not treat `tasks.md` detail as a substitute for adequate `workflow-plan.md` and `workflow-plans/<phase>.md` routing.
- Prefer phased execution over one giant task list.
- Prefer dependency-ordered vertical slices over horizontal subsystem dumps when possible.
- Keep tasks small enough to implement, verify, and review in one focused session.
- For non-trivial work, this pass ends the current session at approved `tasks.md`; implementation begins in a new session unless an upfront `direct path` or `lightweight local` waiver was already recorded.
- Put risky or dependency-establishing work early.
- Use checkpoints to create real stop points, not ritual paperwork.
- Do not let `tasks.md` absorb phase strategy, design decisions, or speculative tasking that should reopen design.

## Lazily Loaded References
Keep this file as the operating contract. References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Load at most one reference by default; load multiple only when the task clearly spans independent decision pressures, such as dependency ordering plus implementation-readiness proof. Treat repository-local `AGENTS.md`, `docs/spec-first-workflow.md`, stable `spec.md`, approved `design/`, and existing task artifacts as higher authority than any example.

| Reference | Load For Symptom | Behavior Change |
| --- | --- | --- |
| `references/phase-strategy-examples.md` | phase boundaries, session stops, review/validation checkpoints, or single-pass versus phased execution are unclear | chooses one risk-bounded phase with a real handoff instead of a giant "implement everything" phase or a ceremony-only checkpoint |
| `references/dependency-ordered-task-ledgers.md` | task order, `[P]` markers, generated artifacts, migrations, or source-of-truth-first sequencing is unclear | derives dependencies from approved design artifacts and source-of-truth flow instead of marking everything parallel or starting with derived files |
| `references/task-sizing-and-slicing.md` | tasks are too large, too horizontal, too vague, hard to review, or hard to verify in one focused session | splits work into reviewable, proof-bound slices instead of hiding independent surfaces behind one broad task |
| `references/acceptance-criteria-and-proof-obligations.md` | acceptance criteria, proof commands, manual checks, or `CONCERNS` obligations are vague | states task-specific truths and matching proof commands instead of "looks good", "run tests", or optimistic readiness language |
| `references/checkpoints-and-reopen-conditions.md` | stop points, implementation-readiness handoff, blockers, reopen targets, or validation/reconciliation triggers need wording | names executable checkpoints and exact reopen targets instead of asking implementation to improvise or create missing workflow artifacts after coding starts |
| `references/planning-anti-patterns.md` | reviewing a draft plan or ledger for drift, invented decisions, duplicate authority, false parallelism, vague proof, or artifact misuse | challenges smell patterns as triage instead of treating a plausible-looking plan as ready by checklist momentum |

Reference snippets are patterns, not decisions. If an example would require an architecture, API, data, security, reliability, migration, rollout, or ownership choice not already approved in `spec.md + design/`, stop and reopen the right earlier phase instead of copying the snippet.

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

### 4. Write The Task Ledger
- Use `tasks.md` for executable task checkboxes and the final implementation handoff.
- For each executable task, make the action, dependency marker when nontrivial, change surface, and planned verification explicit.
- Name exact file paths when known. When exact file choice is genuinely design-time unknown, name a narrow package or artifact surface instead of vague subsystem labels.
- Do not add a task if tasking it requires inventing a missing design decision; reopen `technical design` instead.
- Add only a short readiness reference in `tasks.md` when useful.

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

## Preferred `tasks.md` Shape
Return ledger text that can drop directly into `tasks.md` with minimal rewriting.

Use markdown checkboxes. Each task should include:
- an optional compact `Implementation Handoff` header when it helps the next implementation session, limited to consumed artifacts, readiness status, first task or checkpoint, named `CONCERNS` proof obligations, and reopen target;
- stable task ID such as `T001`
- phase/checkpoint label
- optional `[P]` marker only when safe to parallelize
- short action
- exact file path when known, or a narrow package/artifact surface when exact file choice is genuinely design-time unknown
- dependency marker when nontrivial, such as `Depends on: T001`
- proof/verification expectation
- concise continuation lines when dependency, proof, accepted concern, or reopen detail would make a one-line checkbox hard to scan; continuation lines must support the same task item, not turn `tasks.md` into a design note or strategy memo

Example:

```markdown
## Implementation Handoff

Consumes: approved `spec.md`, `design/`, and this task ledger.
Implementation readiness: PASS.
First task: T001.
Accepted concerns: none.
Reopen target: planning if required artifact context is missing.

## Tasks

- [ ] T001 [Phase 1] Update `internal/http/handler.go` to preserve request ID echo behavior. Depends on: none. Proof: `go test ./internal/http`.
- [ ] T002 [Phase 1] [P] Add regression coverage in `internal/http/handler_test.go`. Depends on: T001. Proof: `go test ./internal/http`.
```

Prefer vertical, reviewable slices. Avoid generic tasks like `implement feature`. Keep the header short; if it starts carrying phase strategy or design rationale, trim it back or reopen `design/`. Use multi-line items for readability, not as permission to hide new decisions or broad subplans inside a checkbox.

## Planning Rules
- For direct-path work, a short inline plan may still be enough; do not force `tasks.md` for a tiny change just to satisfy ceremony.
- For non-trivial work, default to `tasks.md` and consume approved `spec.md + design/`.
- When later review or validation phase-control files are genuinely needed for named multi-session routing, planning should leave them ready to be created or linked before implementation begins; post-code work should not need to invent new workflow/process artifacts.
- The workflow-control handoff must be challenge-ready: master and phase-local plans should make phase status, blockers, stop rules, next-session start, `tasks.md` status, artifact expectations, and any named review or validation phase files clear enough for an adequacy challenger to review without reconstructing intent from chat.
- The implementation-readiness handoff must be explicit: `PASS` may proceed, `CONCERNS` may proceed only with named risks and proof obligations, `FAIL` must route to the named earlier phase, and `WAIVED` must remain a narrow tiny/direct-path/prototype exception.
- If required design artifacts are missing or inconsistent, reopen technical design instead of inferring the missing context locally.
- Keep planning aligned with repository realities: OpenAPI drift checks, `sqlc` regeneration, migrations, race tests, integration checks, or other real verification surfaces when they actually apply.
- If a phase is not independently mergeable or testable, name the coupling explicitly.
- Prefer sequential phases unless change surfaces are truly disjoint.
- Make the handoff explicit: the planning session stops at approved `tasks.md`, and the first implementation task starts in a new session unless a recorded waiver says otherwise.
- State what should trigger a reopen back into specification or technical design instead of letting coding discover it silently.

## Definition Of Done
The planning pass is complete when:
- the execution order is explicit
- each meaningful task in `tasks.md` has concrete action, dependency/proof context, and planned verification
- checkpoints exist where the risk actually changes
- blocked work is clearly separated from ready work
- the next implementation or validation session can start without creating new workflow/process artifacts or missing `tasks.md` to compensate for incomplete planning output
- implementation-readiness status is explicit and is not `FAIL` unless the planning result is honestly blocked or reopened
- the workflow-control artifacts are ready for the read-only adequacy challenge, or the direct-path skip rationale is explicit
- the next session can start implementation without re-planning or guessing where this planning pass was supposed to stop
- the task ledger is specific enough for `go-coder` to execute without recreating strategy or reverse-engineering missing design context

## Escalate Or Reject
- task breakdown derived from an unstable spec
- task breakdown that assumes missing `design/` context instead of escalating
- a phase list with no acceptance criteria or verification
- a generic task like `implement the feature`
- horizontal slicing that hides risk and postpones integration until the end
- a `tasks.md` ledger that turns into a strategy memo instead of listing executable, proof-bound work
- planning output that leaves workflow-control routing too vague for adequacy review before handoff
- planning output that duplicates the entire spec instead of turning it into execution work

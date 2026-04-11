---
name: planning-session
description: "Own a session dedicated only to implementation planning for this repository. Use when approved `spec.md + design/` are ready to turn into `plan.md` and `tasks.md`, plus optional `test-plan.md` or `rollout.md`, and when any later implementation/review/validation phase workflow files and the implementation-readiness gate must be completed before code starts, with task-local `workflow-plan.md` plus `workflow-plans/planning.md` updated without drifting into implementation. Skip tiny direct-path work and tasks whose spec or design are still unstable."
---

# Planning Session

## Purpose
Run only the planning checkpoint for one task-local session.
This wrapper makes implementation planning explicit and stoppable; it does not reopen `spec.md` or `design/`, and it does not start implementation.

## Use When
- the task already has approved workflow routing, stable `spec.md`, and planning-ready technical design
- the orchestrator must turn approved `spec.md + design/` into executable planning artifacts for a non-trivial change
- `plan.md` should become the phase-strategy artifact before any implementation session starts
- `tasks.md` should become the executable task ledger before any non-trivial implementation session starts
- implementation readiness must be checked and recorded before handoff to implementation
- `test-plan.md` or `rollout.md` may be needed because validation or rollout obligations are too large to fit cleanly inside `plan.md`
- master `workflow-plan.md` and `workflow-plans/planning.md` need the planning checkpoint completed or repaired before handoff into implementation

## Skip When
- the work is tiny enough that inline direct-path planning plus explicit rationale is sufficient and a dedicated planning session would be ceremony
- the task is still in `workflow planning`, `research`, `specification`, or `technical design`
- `spec.md` is unstable, required design artifacts are missing, or a triggered conditional design artifact has not been produced yet
- the request tries to combine planning with code changes, tests, migrations, or later implementation-phase execution in one session

## Required Inputs
Planning may begin only when the minimum planning-entry inputs exist:
- stable task-local `spec.md`
- approved `design/overview.md`
- approved `design/component-map.md`
- approved `design/sequence.md`
- approved `design/ownership-map.md`
- any triggered conditional design artifacts that affect sequencing, validation, or rollout, such as:
  - `design/data-model.md`
  - `design/dependency-graph.md`
  - `design/contracts/`
- existing task-local `workflow-plan.md`
- existing task-local `workflow-plans/planning.md`, if present
- explicit design-skip rationale only when the repository contract already allows it for tiny or direct-path work

If any required planning input is missing, stale, or contradictory, stop and route back to `technical design` or `specification` instead of guessing.

## Read First
Always read:
- `AGENTS.md`
- `docs/spec-first-workflow.md`

Then read current phase context in this order:
1. task-local `workflow-plan.md`, if present
2. task-local `workflow-plans/planning.md`, if present
3. task-local `spec.md`
4. `design/overview.md`
5. `design/component-map.md`
6. `design/sequence.md`
7. `design/ownership-map.md`
8. triggered conditional design artifacts and any existing `plan.md`, `tasks.md`, `test-plan.md`, or `rollout.md` that must be repaired rather than replaced

Rules:
- follow `AGENTS.md` if workflow guidance conflicts
- read the master `workflow-plan.md` before the phase-local planning file when both exist
- do not treat `spec.md` alone as sufficient for non-trivial planning
- do not broad-read unrelated repository surfaces when the design bundle already defines the sequencing and ownership constraints

## Lazily Loaded References
Keep this `SKILL.md` as the planning-session wrapper protocol. Load reference files only when their examples are directly useful for the current planning pass:
- `references/planning-session-readiness.md` - load when checking required planning inputs, good or bad planning-entry outcomes, and blocker routing before artifact writes.
- `references/allowed-writes-and-prohibited-actions.md` - load when the session needs concrete allowed-write boundaries or examples of planning-only prohibited actions.
- `references/implementation-readiness-gate.md` - load when assigning `PASS`, `CONCERNS`, `FAIL`, or `WAIVED` and recording the readiness result across planning artifacts.
- `references/workflow-plan-update-examples.md` - load when repairing master `workflow-plan.md` planning status, artifact status, adequacy challenge status, or next-session handoff notes.
- `references/phase-control-file-examples.md` - load when planning must create or repair `workflow-plans/planning.md` or pre-code implementation/review/validation phase-control files.
- `references/session-boundary-and-stop-rules.md` - load when closing the planning session and proving that the next phase will start in a later session.

Reference rules:
- `AGENTS.md` and `docs/spec-first-workflow.md` remain authoritative; examples are calibration only.
- Load the smallest relevant reference, not the full `references/` directory by default.
- Do not copy an example if it would combine planning with implementation, review, validation, or silent `spec.md`/`design/` edits.

## Allowed Writes
This session may write or update only:
- task-local `plan.md`
- task-local `tasks.md`
- task-local `test-plan.md` when validation obligations do not fit cleanly inside `plan.md`
- task-local `rollout.md` when migration or delivery choreography needs a dedicated artifact
- task-local `workflow-plans/implementation-phase-N.md` when the approved phase structure says those implementation checkpoints will be used
- task-local `workflow-plans/review-phase-N.md` when the approved phase structure says those review checkpoints will be used
- task-local `workflow-plans/validation-phase-N.md` when the approved phase structure says those validation checkpoints will be used
- task-local `workflow-plan.md`
- task-local `workflow-plans/planning.md`
- the `workflow-plans/` directory only when it must be created to hold the phase-local planning file

## Prohibited Actions
Do not:
- write production code, tests, migrations, generated artifacts, or runtime configuration changes
- write or finalize `spec.md`
- create or edit `design/`
- create surprise post-code phase files that the approved phase structure did not call for
- start implementation, review, validation, rollout execution, or closeout work
- reopen specification or technical design silently when planning exposes a missing decision or missing context
- make new architecture, API, data, security, reliability, or rollout decisions that belong in `spec.md` or `design/`
- use implementation skills or code edits as a backdoor to "test" the plan
- let `plan.md` become a second `spec.md` or a reconstructed design bundle
- let `tasks.md` become a second `spec.md`, second design bundle, or competing `plan.md`

## Core Defaults
- this is an orchestrator-facing wrapper, not the deeper planning method itself
- `AGENTS.md` owns the workflow contract; `docs/spec-first-workflow.md` owns the detailed artifact mechanics
- `planning-and-task-breakdown` remains the deeper planning method for dependency ordering, task sizing, acceptance criteria, checkpoints, and verification detail
- `plan.md` owns execution strategy; `tasks.md` owns the executable checkbox ledger derived from `spec.md + design/ + plan.md`
- implementation readiness is the planning-phase exit gate; it uses `PASS`, `CONCERNS`, `FAIL`, or `WAIVED` and is not a separate workflow phase
- this wrapper owns the planning-session boundary: required inputs, allowed outputs, workflow handoff updates, and the stop point before implementation
- before non-trivial handoff into implementation, run or record the read-only `workflow-plan-adequacy-challenge` over `workflow-plan.md`, `workflow-plans/planning.md`, `tasks.md` status, and any generated implementation/review/validation phase-control files
- for non-trivial work, the session ends at approved planning artifacts; implementation starts in a new session unless an upfront repository-approved waiver already exists

## Boundary With `planning-and-task-breakdown`
- use `planning-session` to control one planning-only session
- use `planning-and-task-breakdown` inside this session when detailed phase and task decomposition is needed
- keep the deeper skill responsible for execution slicing, acceptance criteria, verification shape, and checkpoint quality
- keep this wrapper responsible for phase readiness, allowed writes, master and phase-local workflow updates, adequacy-challenge reconciliation, and stopping before implementation
- do not duplicate the full planning method in this wrapper

## Required `tasks.md` Shape
For non-trivial work with `plan.md`, `tasks.md` should use markdown checkboxes and include, per task:
- stable task ID such as `T001`
- phase/checkpoint label
- optional `[P]` marker only when safe to parallelize
- short action
- exact file path when known, or a narrow package/artifact surface when exact file choice is genuinely design-time unknown
- dependency marker when nontrivial, such as `Depends on: T001`
- proof/verification expectation

Prefer vertical, reviewable slices. Avoid generic tasks such as "implement feature." If exact tasking requires a missing design decision, reopen `technical design` instead of inventing the task.

## Boundary With Future `implementation-phase-session`
- `planning-session` may write `plan.md`, `tasks.md`, optional `test-plan.md`, optional `rollout.md`, the later phase workflow files already required by the approved phase structure, `workflow-plan.md`, and `workflow-plans/planning.md`
- the future `implementation-phase-session` owns code changes, test changes, migrations, updates to pre-created implementation-phase workflow files, and phase-local validation evidence
- if planning is complete, record `Next session starts with` as the first named implementation phase or explicit implementation checkpoint, then stop instead of beginning it here

## Workflow

### 1. Confirm This Session Owns Planning Only
- check the current phase and active workflow artifacts first
- if the task is still earlier than planning, route back to the correct earlier session instead of forcing planning
- if the work is tiny enough for inline direct-path planning, say so directly and stop rather than forcing this wrapper
- if the task already moved into implementation or later, stop and point to the correct reopen point instead of reopening planning casually

### 2. Confirm Planning Readiness
- verify that `spec.md` is stable enough for task breakdown
- verify that the required core design artifacts exist unless an explicit design-skip rationale already covers the task
- verify that any triggered conditional design artifacts exist when they affect sequencing, validation, or rollout
- if planning exposes a missing spec or design input, route back explicitly; do not invent the missing context inside `plan.md` or `tasks.md`

### 3. Load Execution-Critical Context
- use the design bundle to identify dependency-establishing work, safe sequencing, coupling, validation obligations, and rollout risks
- read existing `plan.md`, `tasks.md`, `test-plan.md`, or `rollout.md` only when repairing or extending an existing planning pass
- keep the context narrow and planning-specific; this session does not need broad repository rediscovery when the approved design already carries the task-local technical context

### 4. Produce Or Repair Planning Artifacts
- apply `planning-and-task-breakdown` as the deeper method when the task needs phased execution breakdown
- write or update `plan.md` as the execution-strategy artifact
- write or update `tasks.md` as the executable task ledger by default for non-trivial work with `plan.md`
- create `test-plan.md` only when test obligations are too large or multi-layered for `plan.md`
- create `rollout.md` only when migration sequencing, backfill, compatibility, deploy order, or failback notes need a dedicated artifact
- create any implementation, review, or validation phase workflow files that the approved phase structure already names, so post-code sessions update existing control artifacts instead of inventing them mid-execution
- keep blocked work separate from ready work
- keep reopen conditions explicit when implementation must hand back to `specification` or `technical design`
- if exact tasking requires a missing design decision, route back to `technical design` instead of inventing executable tasks

### 5. Write Or Repair `workflow-plans/planning.md`
- record only the phase-local orchestration for this planning session
- include planning status, completion marker, stop rule, next action, blockers, artifact outputs, and what can run in parallel later
- record whether companion artifacts such as `tasks.md`, `test-plan.md`, `rollout.md`, or later implementation/review/validation phase workflow files were required, created, or explicitly not needed
- keep this file routing-only; do not turn it into `spec.md`, `design/`, `plan.md`, or `tasks.md`

### 6. Write Or Repair `workflow-plan.md`
- update the master file with current planning-phase status, blockers, handoff state, and artifact status
- make it explicit whether planning is complete, blocked, or reopened to an earlier phase
- record the next session start point without beginning that session here

### 7. Handoff Into Implementation
- if planning is complete, set `Next session starts with` to the first named implementation phase or explicit implementation checkpoint from `plan.md`
- record whether later implementation, review, or validation phase files are expected
- record whether `tasks.md` is approved, draft, missing, or explicitly waived for a tiny/direct-path exception
- run the implementation-readiness gate after `plan.md` and expected `tasks.md` are ready
- set readiness to `PASS` only when required decisions, design, planning artifacts, triggered `test-plan.md` or `rollout.md`, required phase workflow files, blockers, proof path, and high-impact open questions are all resolved for implementation
- set readiness to `CONCERNS` only when implementation may start with named accepted risks and explicit proof obligations
- set readiness to `FAIL` when implementation must not start, and name the earlier phase to reopen
- set readiness to `WAIVED` only for tiny, direct-path, or prototype work with explicit rationale and scope
- record readiness status in `workflow-plan.md`, the gate result and stop or handoff rule in `workflow-plans/planning.md`, a compact summary in `plan.md`, and a short reference in `tasks.md` when useful
- keep implementation entry prerequisites visible so the next session does not need to re-plan
- for non-trivial or agent-backed work, invoke one read-only challenger lane with exactly one skill: `workflow-plan-adequacy-challenge`
- pass the task frame, execution shape, master workflow plan, `workflow-plans/planning.md`, generated post-code phase-control files, planning artifact status, blockers, and proposed next-session handoff
- reconcile blocking findings before marking planning complete; leave planning blocked or reopened when the workflow-control artifacts are not sufficient for implementation handoff
- for tiny/direct-path work, record the explicit skip rationale instead of forcing the challenge

### 8. Stop At The Boundary
- once planning artifacts and workflow handoff are consistent, stop
- do not begin implementation, validation, or review work in the same session

## Required Master `workflow-plan.md` Updates
Every completed, blocked, or reopened planning pass must update the master file with:
- current phase set to this planning checkpoint and current phase status
- link or status for `workflow-plans/planning.md`
- status for `plan.md`
- status for `tasks.md` as `approved`, `draft`, `missing`, explicitly waived, or not expected only for an eligible tiny/direct-path exception
- status for `test-plan.md` and `rollout.md` as `approved`, `draft`, `missing`, or not expected
- whether later `workflow-plans/implementation-phase-N.md`, `workflow-plans/review-phase-N.md`, or `workflow-plans/validation-phase-N.md` were created now, are explicitly not expected, or still remain blocked on a reopen
- implementation-readiness status as `PASS`, `CONCERNS`, `FAIL`, or `WAIVED`
- named accepted risks and proof obligations when readiness is `CONCERNS`
- named earlier phase when readiness is `FAIL`
- waiver rationale and scope when readiness is `WAIVED`
- blockers, accepted assumptions, and reopen conditions that still affect implementation readiness
- workflow plan adequacy challenge status and resolution, or an explicit direct/local skip rationale
- `Session boundary reached`
- `Ready for next session`
- `Next session starts with`

Do not leave planning readiness or handoff state implicit in chat.

## Allowed Outputs
A finished planning session may produce only:
- updated or newly created `plan.md`
- updated or newly created `tasks.md`
- optional `test-plan.md`
- optional `rollout.md`
- optional `workflow-plans/implementation-phase-N.md`
- optional `workflow-plans/review-phase-N.md`
- optional `workflow-plans/validation-phase-N.md`
- updated or newly created `workflow-plan.md`
- updated or newly created `workflow-plans/planning.md`
- an honest `complete`, `blocked`, or `reopened` planning-phase state when the task cannot move cleanly into implementation yet

It does not produce code, tests, migrations, generated artifacts, or implementation execution output.

## Planning Completion Criteria
Planning is complete when:
- execution order is explicit enough for implementation to start without re-planning
- `tasks.md` exists for non-trivial work with `plan.md`, or an explicit tiny/direct-path waiver explains why it is not separate
- meaningful phases or tasks have acceptance criteria and planned verification
- blocked work is clearly separated from ready work
- `test-plan.md` and `rollout.md` exist only when their triggers are real, and their status is explicit when not needed
- any implementation, review, or validation phase workflow files that the approved phase structure requires were created before implementation begins, or their absence is recorded as a reopen blocker
- implementation-readiness gate is `PASS`, `CONCERNS` with named accepted risks and proof obligations, or eligible `WAIVED`; `FAIL` leaves planning blocked or reopened
- master and phase-local workflow artifacts agree on planning status, blockers, and the next session start point
- required workflow plan adequacy challenge findings are reconciled, or an eligible skip rationale is explicit
- the next session can begin the first implementation phase or explicit implementation checkpoint without silently reopening spec or design

## Stop Condition
The session is complete when the planning artifacts and workflow handoff are consistent enough that implementation can begin in the next session, implementation readiness is `PASS`, eligible `CONCERNS`, or eligible `WAIVED`, required adequacy-challenge findings are reconciled or explicitly waived, and no implementation work has started in the current one.

## Escalate When
Escalate instead of forcing output when:
- `spec.md` is unstable enough that planning would recreate missing decisions
- required core design artifacts are missing without an approved design-skip rationale
- a conditional design artifact is clearly triggered but missing
- rollout, compatibility, migration, or ownership questions remain unresolved and change the implementation order
- implementation readiness is `FAIL` or would require accepting unnamed risk
- the request tries to combine planning with implementation, validation, or review execution
- the work is so small that a dedicated planning session would be ceremony

## Anti-Patterns
- using this wrapper as a way to silently reopen `spec.md` or `design/`
- copying the whole spec into `plan.md` instead of turning it into execution work
- copying strategy or decisions into `tasks.md` instead of keeping it an executable task ledger
- creating generic tasks like "implement feature" instead of vertical, proof-bound slices
- forcing `test-plan.md` or `rollout.md` when their triggers are not real
- leaving later implementation/review/validation phase workflow files to be invented mid-implementation or mid-validation
- hiding blockers inside optimistic task wording
- marking implementation handoff ready while blocking workflow plan adequacy findings remain unresolved
- updating `workflow-plan.md` as if implementation already started
- writing "phase 1" and then immediately coding it in the same session

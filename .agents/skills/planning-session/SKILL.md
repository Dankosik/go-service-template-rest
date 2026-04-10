---
name: planning-session
description: "Own a session dedicated only to implementation planning for this repository. Use when approved `spec.md + design/` are ready to turn into `plan.md` plus optional `test-plan.md` or `rollout.md`, and the orchestrator must update task-local `workflow-plan.md` plus `workflow-plans/planning.md` without drifting into implementation. Skip tiny direct-path work and tasks whose spec or design are still unstable."
---

# Planning Session

## Purpose
Run only the planning checkpoint for one task-local session.
This wrapper makes implementation planning explicit and stoppable; it does not reopen `spec.md` or `design/`, and it does not start implementation.

## Use When
- the task already has approved workflow routing, stable `spec.md`, and planning-ready technical design
- the orchestrator must turn approved `spec.md + design/` into executable planning artifacts for a non-trivial change
- `plan.md` should become the coder-facing execution artifact before any implementation session starts
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
8. triggered conditional design artifacts and any existing `plan.md`, `test-plan.md`, or `rollout.md` that must be repaired rather than replaced

Rules:
- follow `AGENTS.md` if workflow guidance conflicts
- read the master `workflow-plan.md` before the phase-local planning file when both exist
- do not treat `spec.md` alone as sufficient for non-trivial planning
- do not broad-read unrelated repository surfaces when the design bundle already defines the sequencing and ownership constraints

## Allowed Writes
This session may write or update only:
- task-local `plan.md`
- task-local `test-plan.md` when validation obligations do not fit cleanly inside `plan.md`
- task-local `rollout.md` when migration or delivery choreography needs a dedicated artifact
- task-local `workflow-plan.md`
- task-local `workflow-plans/planning.md`
- the `workflow-plans/` directory only when it must be created to hold the phase-local planning file

## Prohibited Actions
Do not:
- write production code, tests, migrations, generated artifacts, or runtime configuration changes
- write or finalize `spec.md`
- create or edit `design/`
- create `workflow-plans/implementation-phase-N.md`, `workflow-plans/review-phase-N.md`, or `workflow-plans/validation-phase-N.md` in this session
- start implementation, review, validation, rollout execution, or closeout work
- reopen specification or technical design silently when planning exposes a missing decision or missing context
- make new architecture, API, data, security, reliability, or rollout decisions that belong in `spec.md` or `design/`
- use implementation skills or code edits as a backdoor to "test" the plan
- let `plan.md` become a second `spec.md` or a reconstructed design bundle

## Core Defaults
- this is an orchestrator-facing wrapper, not the deeper planning method itself
- `AGENTS.md` owns the workflow contract; `docs/spec-first-workflow.md` owns the detailed artifact mechanics
- `planning-and-task-breakdown` remains the deeper planning method for dependency ordering, task sizing, acceptance criteria, checkpoints, and verification detail
- this wrapper owns the planning-session boundary: required inputs, allowed outputs, workflow handoff updates, and the stop point before implementation
- for non-trivial work, the session ends at approved planning artifacts; implementation starts in a new session unless an upfront repository-approved waiver already exists

## Boundary With `planning-and-task-breakdown`
- use `planning-session` to control one planning-only session
- use `planning-and-task-breakdown` inside this session when detailed phase and task decomposition is needed
- keep the deeper skill responsible for execution slicing, acceptance criteria, verification shape, and checkpoint quality
- keep this wrapper responsible for phase readiness, allowed writes, master and phase-local workflow updates, and stopping before implementation
- do not duplicate the full planning method in this wrapper

## Boundary With Future `implementation-phase-session`
- `planning-session` may write `plan.md`, optional `test-plan.md`, optional `rollout.md`, `workflow-plan.md`, and `workflow-plans/planning.md`
- the future `implementation-phase-session` owns code changes, test changes, migrations, implementation-phase workflow files, and phase-local validation evidence
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
- if planning exposes a missing spec or design input, route back explicitly; do not invent the missing context inside `plan.md`

### 3. Load Execution-Critical Context
- use the design bundle to identify dependency-establishing work, safe sequencing, coupling, validation obligations, and rollout risks
- read existing `plan.md`, `test-plan.md`, or `rollout.md` only when repairing or extending an existing planning pass
- keep the context narrow and planning-specific; this session does not need broad repository rediscovery when the approved design already carries the task-local technical context

### 4. Produce Or Repair Planning Artifacts
- apply `planning-and-task-breakdown` as the deeper method when the task needs phased execution breakdown
- write or update `plan.md` as the coder-facing execution artifact
- create `test-plan.md` only when test obligations are too large or multi-layered for `plan.md`
- create `rollout.md` only when migration sequencing, backfill, compatibility, deploy order, or failback notes need a dedicated artifact
- keep blocked work separate from ready work
- keep reopen conditions explicit when implementation must hand back to `specification` or `technical design`

### 5. Write Or Repair `workflow-plans/planning.md`
- record only the phase-local orchestration for this planning session
- include planning status, completion marker, stop rule, next action, blockers, artifact outputs, and what can run in parallel later
- record whether companion artifacts such as `test-plan.md` or `rollout.md` were required or explicitly not needed
- keep this file routing-only; do not turn it into `spec.md`, `design/`, or `plan.md`

### 6. Write Or Repair `workflow-plan.md`
- update the master file with current planning-phase status, blockers, handoff state, and artifact status
- make it explicit whether planning is complete, blocked, or reopened to an earlier phase
- record the next session start point without beginning that session here

### 7. Handoff Into Implementation
- if planning is complete, set `Next session starts with` to the first named implementation phase or explicit implementation checkpoint from `plan.md`
- record whether later implementation, review, or validation phase files are expected
- keep implementation entry prerequisites visible so the next session does not need to re-plan

### 8. Stop At The Boundary
- once planning artifacts and workflow handoff are consistent, stop
- do not begin implementation, validation, or review work in the same session

## Required Master `workflow-plan.md` Updates
Every completed, blocked, or reopened planning pass must update the master file with:
- current phase set to this planning checkpoint and current phase status
- link or status for `workflow-plans/planning.md`
- status for `plan.md`
- status for `test-plan.md` and `rollout.md` as `approved`, `draft`, `missing`, or not expected
- whether later `workflow-plans/implementation-phase-N.md`, `workflow-plans/review-phase-N.md`, or `workflow-plans/validation-phase-N.md` are expected or still unknown
- blockers, accepted assumptions, and reopen conditions that still affect implementation readiness
- `Session boundary reached`
- `Ready for next session`
- `Next session starts with`

Do not leave planning readiness or handoff state implicit in chat.

## Allowed Outputs
A finished planning session may produce only:
- updated or newly created `plan.md`
- optional `test-plan.md`
- optional `rollout.md`
- updated or newly created `workflow-plan.md`
- updated or newly created `workflow-plans/planning.md`
- an honest `complete`, `blocked`, or `reopened` planning-phase state when the task cannot move cleanly into implementation yet

It does not produce code, tests, migrations, generated artifacts, or implementation-phase workflow files.

## Planning Completion Criteria
Planning is complete when:
- execution order is explicit enough for implementation to start without re-planning
- meaningful phases or tasks have acceptance criteria and planned verification
- blocked work is clearly separated from ready work
- `test-plan.md` and `rollout.md` exist only when their triggers are real, and their status is explicit when not needed
- master and phase-local workflow artifacts agree on planning status, blockers, and the next session start point
- the next session can begin the first implementation phase or explicit implementation checkpoint without silently reopening spec or design

## Stop Condition
The session is complete when the planning artifacts and workflow handoff are consistent enough that implementation can begin in the next session, and no implementation work has started in the current one.

## Escalate When
Escalate instead of forcing output when:
- `spec.md` is unstable enough that planning would recreate missing decisions
- required core design artifacts are missing without an approved design-skip rationale
- a conditional design artifact is clearly triggered but missing
- rollout, compatibility, migration, or ownership questions remain unresolved and change the implementation order
- the request tries to combine planning with implementation, validation, or review execution
- the work is so small that a dedicated planning session would be ceremony

## Anti-Patterns
- using this wrapper as a way to silently reopen `spec.md` or `design/`
- copying the whole spec into `plan.md` instead of turning it into execution work
- forcing `test-plan.md` or `rollout.md` when their triggers are not real
- hiding blockers inside optimistic task wording
- updating `workflow-plan.md` as if implementation already started
- writing "phase 1" and then immediately coding it in the same session

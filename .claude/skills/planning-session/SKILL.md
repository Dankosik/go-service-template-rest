---
name: planning-session
description: "Own a session dedicated only to task breakdown for this repository. Use when approved `spec.md + design/` are ready to turn into a `tasks.md` implementation handoff, plus `test-plan.md` or `rollout.md` only when truly triggered, and when the implementation-readiness gate must be completed before code starts, with task-local `workflow-plan.md` plus `workflow-plans/planning.md` updated without drifting into implementation. Skip tiny direct-path work and tasks whose spec or design are still unstable."
---

# Planning Session

## Purpose
Run only the planning checkpoint for one task-local session.
This wrapper makes task breakdown explicit and stoppable; it does not reopen `spec.md` or `design/`, and it does not start implementation.

## Outcome-First Operating Rules
- Start by naming the skill-specific outcome, success criteria, constraints, available evidence, and stop rule.
- Treat workflow steps as decision rules, not a ritual checklist. Follow exact order only when this skill or the repository contract makes the sequence an invariant.
- Use the minimum context, references, tools, and validation loops that can change the deliverable; stop expanding when the quality bar is met.
- Before acting, resolve prerequisite discovery, lookup, or artifact reads that the outcome depends on; parallelize only independent evidence gathering and synthesize before the next decision.
- Prefer bounded assumptions and local evidence over broad questioning; ask only when a missing fact would change correctness, ownership, safety, or scope.
- When evidence is missing or conflicting, retry once with a targeted strategy or label the assumption, blocker, or reopen target instead of treating absence as proof.
- Finish only when the requested deliverable is complete in the required shape and verification or a clearly named blocker/residual risk is recorded.

## Use When
- the task already has approved workflow routing, stable `spec.md`, and planning-ready technical design
- the orchestrator must turn approved `spec.md + design/` into executable planning artifacts for a non-trivial change
- `tasks.md` should become the executable task ledger and final implementation handoff before any non-trivial implementation session starts
- implementation readiness must be checked and recorded before handoff to implementation
- `test-plan.md` or `rollout.md` may be needed because validation or rollout obligations are too large to fit cleanly inside `tasks.md`
- master `workflow-plan.md` and `workflow-plans/planning.md` need the planning checkpoint completed or repaired before handoff into implementation

## Skip When
- the work is tiny enough that inline direct-path planning plus explicit rationale is sufficient and a dedicated planning session would be ceremony
- the task is still in `workflow planning`, `research`, `specification`, or `technical design`
- `spec.md` is unstable, required design artifacts are missing, or a triggered conditional design artifact has not been produced yet
- the request tries to combine planning with code changes, tests, migrations, or coding/execution in one session

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
8. triggered conditional design artifacts and any existing `tasks.md`, `test-plan.md`, or `rollout.md` that must be repaired rather than replaced

Rules:
- follow `AGENTS.md` if workflow guidance conflicts
- read the master `workflow-plan.md` before the phase-local planning file when both exist
- do not treat `spec.md` alone as sufficient for non-trivial planning
- do not broad-read unrelated repository surfaces when the design bundle already defines the sequencing and ownership constraints

## Lazily Loaded References
Keep this `SKILL.md` as the planning-session wrapper protocol. References are compact rubrics and example banks, not exhaustive checklists or documentation dumps.

Default loading rule:
- Load at most one reference by default.
- Load a second reference only when the task clearly spans multiple independent decision pressures, such as entry readiness plus later phase-control skeletons.
- Do not load the full `references/` directory by default.
- `AGENTS.md` and `docs/spec-first-workflow.md` remain authoritative; reference examples are calibration only.
- Do not copy an example if it would combine planning with implementation, review, validation, or silent `spec.md`/`design/` edits.

Routing table:

| Reference | Load When The Symptom Is | Behavior Change |
| --- | --- | --- |
| `references/planning-session-readiness.md` | Planning inputs are missing, stale, contradictory, or not yet checked before `tasks.md` writes. | Blocks or reopens upstream instead of planning from `spec.md` alone or inventing missing design context. |
| `references/allowed-writes-and-prohibited-actions.md` | The write boundary is contested, or the user asks to bundle planning with code, tests, `spec.md`, `design/`, review, or validation work. | Narrows the session to planning-only writes instead of editing downstream artifacts or creating just-in-case files. |
| `references/implementation-readiness-gate.md` | Assigning `PASS`, `CONCERNS`, `FAIL`, or `WAIVED`, especially when a handoff feels almost ready. | Chooses a gate status from concrete blockers and proof obligations instead of optimistic `PASS` or vague concern wording. |
| `references/workflow-plan-update-examples.md` | Updating master `workflow-plan.md` planning state, artifact status, adequacy challenge status, or next-session handoff. | Records cross-phase state in the master artifact instead of leaving it in chat or only in `workflow-plans/planning.md`. |
| `references/phase-control-file-examples.md` | Creating or repairing `workflow-plans/planning.md` or pre-code phase-control files for named review or validation phases. | Creates only named routing skeletons instead of just-in-case phase files or duplicate `tasks.md` content. |
| `references/session-boundary-and-stop-rules.md` | Closing the planning session or deciding whether the next action is implementation, a reopen target, or stop. | Stops at the planning boundary with a named next session instead of starting T001 or declaring completion with an incomplete handoff. |

## Allowed Writes
This session may write or update only:
- task-local `tasks.md`
- task-local `test-plan.md` when validation obligations do not fit cleanly inside `tasks.md`
- task-local `rollout.md` when migration or delivery choreography needs a dedicated artifact
- task-local `workflow-plans/review-phase-N.md` when named multi-session routing requires those review checkpoints
- task-local `workflow-plans/validation-phase-N.md` when named multi-session routing requires those validation checkpoints
- task-local `workflow-plan.md`
- task-local `workflow-plans/planning.md`
- the `workflow-plans/` directory only when it must be created to hold the phase-local planning file

## Prohibited Actions
Do not:
- write production code, tests, migrations, generated artifacts, or runtime configuration changes
- write or finalize `spec.md`
- create or edit `design/`
- create surprise review or validation phase files that named multi-session routing did not call for
- start implementation, review, validation, rollout execution, or closeout work
- reopen specification or technical design silently when planning exposes a missing decision or missing context
- make new architecture, API, data, security, reliability, or rollout decisions that belong in `spec.md` or `design/`
- use implementation skills or code edits as a backdoor to "test" the plan
- let `tasks.md` become a second `spec.md`, second design bundle, or bloated strategy memo

## Core Defaults
- this is an orchestrator-facing wrapper, not the deeper planning method itself
- `AGENTS.md` owns the workflow contract; `docs/spec-first-workflow.md` owns the detailed artifact mechanics
- `planning-and-task-breakdown` remains the deeper planning method for dependency ordering, task sizing, acceptance criteria, checkpoints, and verification detail
- `tasks.md` owns the executable checkbox ledger and final implementation handoff derived from `spec.md + design/`
- `tasks.md` must belong to the active task-local bundle. A repository-root or historical ledger is not the current handoff unless workflow control explicitly reopens it and records the resume route.
- implementation readiness is the planning-phase exit gate; it uses `PASS`, `CONCERNS`, `FAIL`, or `WAIVED` and is not a separate workflow phase
- this wrapper owns the planning-session boundary: required inputs, allowed outputs, workflow handoff updates, and the stop point before implementation
- before non-trivial handoff into implementation, run or record the read-only `workflow-plan-adequacy-challenge` over `workflow-plan.md`, `workflow-plans/planning.md`, `tasks.md` status, and any named review or validation phase-control files
- for non-trivial work, the session ends at approved planning artifacts; implementation starts in a new session unless an upfront repository-approved waiver already exists

## Boundary With `planning-and-task-breakdown`
- use `planning-session` to control one planning-only session
- use `planning-and-task-breakdown` inside this session when detailed phase and task decomposition is needed
- keep the deeper skill responsible for execution slicing, acceptance criteria, verification shape, and checkpoint quality
- keep this wrapper responsible for phase readiness, allowed writes, master and phase-local workflow updates, adequacy-challenge reconciliation, and stopping before implementation
- do not duplicate the full planning method in this wrapper

## Required `tasks.md` Shape
For non-trivial work, `tasks.md` should use markdown checkboxes and include, per task:
- stable task ID such as `T001`
- phase/checkpoint label
- optional `[P]` marker only when safe to parallelize
- short action
- exact file path when known, or a narrow package/artifact surface when exact file choice is genuinely design-time unknown
- dependency marker when nontrivial, such as `Depends on: T001`
- proof/verification expectation
- concise continuation lines when dependency, proof, accepted concern, or reopen detail would make a one-line checkbox hard to scan

Prefer vertical, reviewable slices. Avoid generic tasks such as "implement feature." Multi-line task items are allowed for readability, but they must remain executable ledger items instead of design notes or strategy memos. If exact tasking requires a missing design decision, reopen `technical design` instead of inventing the task.

## Boundary With Coding/Execution
- `planning-session` may write `tasks.md`, optional `test-plan.md`, optional `rollout.md`, review or validation phase workflow files already required by named multi-session routing, `workflow-plan.md`, and `workflow-plans/planning.md`
- coding/execution owns code changes, test changes, migrations, generated output, and task-level validation evidence
- if planning is complete, record `Next session starts with` as the first task ID or explicit implementation checkpoint from `tasks.md`, then stop instead of beginning it here

## Workflow

### 1. Confirm This Session Owns Planning Only
- check the current phase and active workflow artifacts first
- if the task is still earlier than planning, route back to the correct earlier session instead of forcing planning
- if the work is tiny enough for inline direct-path planning, say so directly and stop rather than forcing this wrapper
- if the task already moved into implementation or later, stop and point to the correct reopen point instead of reopening planning casually

### 2. Confirm Planning Readiness
- verify that `spec.md` is stable enough for task breakdown
- verify that the required core design artifacts exist unless an explicit design-skip rationale already covers the task
- accept concise approved design artifacts when they answer the current planning-critical questions explicitly; do not reopen design just because one required artifact is short or asymmetrical
- verify that any triggered conditional design artifacts exist when they affect sequencing, validation, or rollout
- if planning exposes a missing spec or design input, route back explicitly; do not invent the missing context inside `tasks.md`

### 3. Load Execution-Critical Context
- use the design bundle to identify dependency-establishing work, safe sequencing, coupling, validation obligations, and rollout risks
- read existing `tasks.md`, `test-plan.md`, or `rollout.md` only when repairing or extending an existing planning pass
- keep the context narrow and planning-specific; this session does not need broad repository rediscovery when the approved design already carries the task-local technical context
- keep the handoff focused on the first safe implementation slice; later-phase implications that do not change that slice should stay as explicit concerns, proof obligations, or follow-up notes instead of being expanded into new pre-code design work

### 4. Produce Or Repair Planning Artifacts
- apply `planning-and-task-breakdown` as the deeper method when the task needs phased execution breakdown
- write or update `tasks.md` as the executable task ledger by default for non-trivial work
- create `test-plan.md` only when test obligations are too large or multi-layered for `tasks.md` and the approved design already contains the needed validation context
- create `rollout.md` only when migration sequencing, backfill, compatibility, deploy order, or failback notes need a dedicated artifact and the approved design already contains the needed rollout context
- create review or validation phase workflow files only when named multi-session routing already requires them, so later sessions update existing control artifacts instead of inventing them mid-execution
- keep blocked work separate from ready work
- keep reopen conditions explicit when implementation must hand back to `specification` or `technical design`
- if exact tasking, `test-plan.md`, or `rollout.md` requires a missing design decision, route back to `technical design` instead of inventing executable tasks or companion-artifact context

### 5. Write Or Repair `workflow-plans/planning.md`
- record only the phase-local orchestration for this planning session
- include planning status, completion marker, stop rule, next action, blockers, artifact outputs, and what can run in parallel later
- record whether companion artifacts such as `tasks.md`, `test-plan.md`, `rollout.md`, or later review/validation phase workflow files were required, created, or explicitly not needed
- keep this file routing-only; do not turn it into `spec.md`, `design/`, or `tasks.md`

### 6. Write Or Repair `workflow-plan.md`
- update the master file with current planning-phase status, blockers, handoff state, and artifact status
- make the planning phase status explicit, and use a separate routing state when planning reopens an earlier phase
- record the next session start point without beginning that session here

### 7. Handoff Into Implementation
- if planning is complete, set `Next session starts with` to the first task ID or explicit implementation checkpoint from `tasks.md`
- record whether later review or validation phase files are expected
- record whether `tasks.md` is approved, draft, missing, or explicitly waived for a tiny/direct-path exception
- run the implementation-readiness gate after expected `tasks.md` is ready
- set readiness to `PASS` only when the next implementation slice can start without inventing hidden architecture, ownership, contract, sequencing, or rollout decisions; required artifacts and proof path must already support that slice
- set readiness to `CONCERNS` only when implementation may start with named accepted risks and explicit proof obligations that the next slice can satisfy without replanning
- set readiness to `FAIL` when implementation must not start, and name the earlier phase to reopen
- set readiness to `WAIVED` only for tiny, direct-path, or prototype work with explicit rationale and scope
- record readiness status in `workflow-plan.md`, the gate result and stop or handoff rule in `workflow-plans/planning.md`, and a short reference in `tasks.md` when useful
- keep implementation entry prerequisites visible so the next session does not need to re-plan
- for non-trivial or agent-backed work, invoke one read-only challenger lane with exactly one skill: `workflow-plan-adequacy-challenge`
- pass the task frame, execution shape, master workflow plan, `workflow-plans/planning.md`, any named review or validation phase-control files, planning artifact status, blockers, and proposed next-session handoff
- reconcile blocking findings before marking planning complete; leave planning blocked or reopened when the workflow-control artifacts are not sufficient for implementation handoff
- for tiny/direct-path work, record the explicit skip rationale instead of forcing the challenge

### 8. Stop At The Boundary
- once planning artifacts and workflow handoff are consistent, stop
- do not begin implementation, validation, or review work in the same session

## Required Master `workflow-plan.md` Updates
Every completed, blocked, or reopened planning pass must update the master file with:
- current phase set to this planning checkpoint and current phase status
- link or status for `workflow-plans/planning.md`
- status for `tasks.md` as `approved`, `draft`, `missing`, explicitly waived, or not expected only for an eligible tiny/direct-path exception
- status for `test-plan.md` and `rollout.md` as `approved`, `draft`, `missing`, `conditional`, `waived`, or not expected, with trigger rationale for `not expected`, `conditional`, or `waived`
- whether later `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` were created now because named multi-session routing needs them, are explicitly not expected with rationale, or still remain blocked on a reopen
- implementation-readiness status as `PASS`, `CONCERNS`, `FAIL`, or `WAIVED`
- named accepted risks and proof obligations when readiness is `CONCERNS`
- named earlier phase when readiness is `FAIL`
- waiver rationale and scope when readiness is `WAIVED`
- blockers, accepted assumptions, and reopen conditions that still affect implementation readiness
- workflow plan adequacy challenge status and resolution, or an explicit direct/local skip rationale
- `Session boundary reached`
- `Ready for next session`
- `Next session starts with`
- `Next session context bundle` as an always-present field: say default resume order is sufficient, or list exact artifact paths and one-line reasons for task-specific resume context

Do not leave planning readiness or handoff state implicit in chat.

## Allowed Outputs
A finished planning session may produce only:
- updated or newly created `tasks.md`
- optional `test-plan.md`
- optional `rollout.md`
- optional `workflow-plans/review-phase-N.md`
- optional `workflow-plans/validation-phase-N.md`
- updated or newly created `workflow-plan.md`
- updated or newly created `workflow-plans/planning.md`
- an honest planning phase status such as `complete` or `blocked`, plus a separate reopen routing state when the task cannot move cleanly into implementation yet

It does not produce code, tests, migrations, generated artifacts, or implementation execution output.

## Required Final Chat Handoff
When this session ends with `Session boundary reached: yes` and `Ready for next session: yes`, the final chat response must include a `Recommended next-session prompt` section with one copy-pastable fenced text block.

Derive that prompt from the recorded workflow handoff state, not memory:
- `Next session starts with`
- `Next session context bundle`
- this phase's stop rule
- blockers, accepted assumptions, accepted risks, or reopen conditions that still matter
- the expected artifact or output for the next session

Rules:
- keep the prompt chat-only; do not write it into workflow artifacts or create a new artifact for it
- target the recorded first task, implementation checkpoint, or reopen route exactly
- tell the next agent which files to read first, the immediate objective, important constraints, and expected outputs
- if there is no next session or `Ready for next session: no`, do not invent a prompt

## Planning Completion Criteria
Planning is complete when:
- execution order is explicit enough for implementation to start without re-planning
- `tasks.md` exists for non-trivial work, or an explicit tiny/direct-path waiver explains why it is not separate
- meaningful phases or tasks have acceptance criteria and planned verification
- blocked work is clearly separated from ready work
- `test-plan.md` and `rollout.md` exist only when their triggers are real, and their status is explicit when not needed
- any review or validation phase workflow files that named multi-session routing requires were created before implementation begins, or their absence is recorded as a reopen blocker
- implementation-readiness gate is `PASS`, `CONCERNS` with named accepted risks and proof obligations, or eligible `WAIVED`; `FAIL` leaves planning blocked or reopened
- master and phase-local workflow artifacts agree on planning status, blockers, and the next session start point
- required workflow plan adequacy challenge findings are reconciled, or an eligible skip rationale is explicit
- the next session can begin the first task or explicit implementation checkpoint without silently reopening spec or design
- visible later-phase implications that do not change the first safe slice are recorded explicitly instead of being forced into new planning blockers

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
- copying strategy or decisions into `tasks.md` instead of keeping it an executable task ledger
- creating generic tasks like "implement feature" instead of vertical, proof-bound slices
- forcing `test-plan.md` or `rollout.md` when their triggers are not real
- leaving required named review/validation phase workflow files to be invented mid-implementation or mid-validation
- hiding blockers inside optimistic task wording
- marking implementation handoff ready while blocking workflow plan adequacy findings remain unresolved
- updating `workflow-plan.md` as if implementation already started
- writing "phase 1" and then immediately coding it in the same session

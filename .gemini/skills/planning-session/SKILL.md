---
name: planning-session
description: "Own a session dedicated only to task breakdown for this repository when a distinct planning checkpoint is triggered. Use when specification-review-approved `spec.md` plus required compact or split design context, including triggered system/integration and Go code ownership design, any mandatory technical-design-review gate, and triggered test-design are ready to turn into a draft or repaired `tasks.md`, plus `rollout.md` only when truly triggered, then hand off to the separate task-review/readiness gate before code starts. Skip direct-path inline plans and tasks whose spec, specification review, design context, required technical design review, or triggered test design is still unstable."
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
- the task already has approved workflow routing when used, stable reviewed `spec.md`, and reviewed planning-ready compact or split technical design context
- mandatory specification review is complete for non-trivial task-local `spec.md`, or explicitly blocks planning
- any mandatory technical design review for separate design depth is complete or explicitly blocks planning
- the orchestrator must turn reviewed `spec.md` plus required design context into executable planning artifacts for a non-trivial change
- `tasks.md` should become the executable draft ledger that the separate task-review/readiness phase can approve before any non-trivial implementation session starts
- the completed `tasks.md` must be handed off for review against reviewed `spec.md`, specification-review obligations, required design context, technical-design-review obligations, approved test-design scenarios, and triggered validation or rollout obligations before implementation
- `rollout.md` may be needed because rollout obligations are too large to fit cleanly inside `tasks.md`; `test-plan.md` belongs to the earlier `test-design` phase when triggered
- master `workflow-plan.md` and `workflow-plans/planning.md` need the planning checkpoint completed or repaired before handoff into implementation

## Skip When
- the work is tiny enough that inline direct-path planning plus explicit rationale is sufficient and a dedicated planning session would be ceremony
- lean-local `tasks.md` can be produced in the same local pass without separate planning-phase routing and no multi-session state is needed
- the task is still in `workflow planning`, `research`, `specification`, unresolved `specification review`, `technical design`, or unresolved `technical design review`
- `spec.md` is unstable or unreviewed, required compact/split design context is missing, a triggered conditional design artifact has not been produced yet, mandatory technical design review is missing or blocking, or triggered test design is missing or blocking
- the request tries to combine planning with code changes, tests, migrations, or coding/execution in one session

## Required Inputs
Planning may begin only when the minimum planning-entry inputs exist:
- stable task-local `spec.md`
- completed specification review result for non-trivial `spec.md`, with `PASS` or `CONCERNS` plus named accepted spec risks and proof obligations
- compact `spec.md` design answers, one approved `design/overview.md`, or approved split design artifacts, according to the trigger decision
- approved `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md` only when split design is triggered
- completed design fan-out result when separate technical design depth was triggered, with `complete`, valid `scoped_down`, or eligible `local_only`
- completed technical design review result when separate `design/overview.md` or split `design/` was triggered, with `PASS` or `CONCERNS` plus named accepted design risks and proof obligations
- approved `test-plan.md` when test design was triggered, or explicit `test-design: not expected` rationale when proof can live directly in `tasks.md`
- any triggered conditional design artifacts that affect sequencing, validation, or rollout, such as:
  - `design/data-model.md`
  - `design/dependency-graph.md`
  - `design/pattern-fit.md`
  - `design/contracts/`
- approved dependency/OSS due-diligence outcome when implementation adds a dependency, integrates OSS, builds custom infrastructure, or introduces a material helper/abstraction
- approved Pattern Fit Diligence outcome when implementation relies on a non-trivial architecture, workflow, integration, resilience, consistency, data-flow, or abstraction pattern
- existing task-local `workflow-plan.md`
- existing task-local `workflow-plans/planning.md`, if present
- explicit design-skip or compact-design rationale only when the repository contract allows it for direct or lean-local work

If any required planning input is missing, stale, contradictory, blocked by design fan-out, or blocked by technical design review, stop and route back to `technical design review`, `technical design`, or `specification` instead of guessing.

If specification review is missing, stale after repair, `FAIL`, or has unresolved findings, stop and route back to `specification review`, `specification`, research, or user decision according to the review blocker.

## Read First
Always read:
- `AGENTS.md`
- `docs/spec-first-workflow.md`

Then read current phase context in this order:
1. task-local `workflow-plan.md`, if present
2. task-local `workflow-plans/planning.md`, if present
3. task-local `spec.md`
4. specification review result when non-trivial `spec.md` exists
5. compact design section in `spec.md`, `design/overview.md`, or split design artifacts according to the trigger decision
6. technical design review result when separate design depth was triggered
7. approved `test-plan.md` when test design was triggered, or the explicit no-test-plan rationale
8. triggered conditional design artifacts and any existing `tasks.md` or `rollout.md` that must be repaired rather than replaced

Rules:
- follow `AGENTS.md` if workflow guidance conflicts
- read the master `workflow-plan.md` before the phase-local planning file when both exist
- do not treat `spec.md` alone as sufficient for non-trivial planning unless it explicitly records lean-local compact design answers and the design-skip/merge rationale
- do not treat `spec.md` alone as sufficient for non-trivial planning unless specification review is `PASS` or `CONCERNS` with named obligations
- do not treat separate design artifacts as sufficient for planning unless the mandatory technical design review gate is recorded and reconciled
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
| `references/implementation-readiness-gate.md` | Preparing the completed `tasks.md` for the separate task-review/readiness phase, especially when a handoff feels almost ready. | Checks for likely readiness blockers and routes them to planning repair or the owning earlier phase instead of assigning the final gate status here. |
| `references/workflow-plan-update-examples.md` | Updating master `workflow-plan.md` planning state, artifact status, adequacy challenge status, or next-session handoff. | Records cross-phase state in the master artifact instead of leaving it in chat or only in `workflow-plans/planning.md`. |
| `references/phase-control-file-examples.md` | Creating or repairing `workflow-plans/planning.md` or pre-code phase-control files for named review or validation phases. | Creates only named routing skeletons instead of just-in-case phase files or duplicate `tasks.md` content. |
| `references/session-boundary-and-stop-rules.md` | Closing the planning session or deciding whether the next action is implementation, a reopen target, or stop. | Stops at the planning boundary with a named next session instead of starting T001 or declaring completion with an incomplete handoff. |

## Allowed Writes
This session may write or update only:
- task-local `tasks.md`
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
- reopen specification, specification review, technical design, or technical design review silently when planning exposes a missing decision, missing context, or missing review gate
- make new architecture, API, data, security, reliability, or rollout decisions that belong in `spec.md` or `design/`
- use implementation skills or code edits as a backdoor to "test" the plan
- let `tasks.md` become a second `spec.md`, second design bundle, or bloated strategy memo

## Core Defaults
- this is an orchestrator-facing wrapper, not the deeper planning method itself
- `AGENTS.md` owns the workflow contract; `docs/spec-first-workflow.md` is the router; `docs/spec-first-workflow/phases/planning.md` owns planning mechanics; `docs/spec-first-workflow/phases/task-review-readiness.md` owns the readiness verdict
- `planning-and-task-breakdown` remains the deeper planning method for dependency ordering, task sizing, acceptance criteria, checkpoints, and verification detail
- `tasks.md` owns the executable checkbox ledger and final implementation handoff derived from `spec.md` plus required compact or split design context
- `tasks.md` must belong to the active task-local bundle. A repository-root or unrelated ledger is not the current handoff unless workflow control explicitly reopens it and records the resume route.
- task-ledger review is the separate post-planning, pre-implementation stage owned by `docs/spec-first-workflow/phases/task-review-readiness.md`; planning prepares the ledger and handoff but does not assign `PASS`, `CONCERNS`, `FAIL`, or `WAIVED`
- Goal-ready `tasks.md` separates successful completion from blocked-stop behavior; a recorded blocker is a valid stop, not a successful completion claim
- broad or resumable ledgers should include selective read-before-coding context, task-specific read context, checkpoint gates, source traceability, approved test-design scenario IDs when present, proof-first tasking or an explicit `Proof-first waiver:`, structured evidence fields, a task-local implementation quality bar when needed, and a resume rule
- skipped, unavailable, stale, failing, or too-narrow proof cannot satisfy a task checkbox, checkpoint, or completion claim
- dependency/OSS due-diligence decisions from `spec.md` or design must be carried into `tasks.md` as dependency, integration, license/security, drift, and proof tasks when relevant; missing due diligence blocks planning handoff and routes back to specification or technical design
- Pattern Fit Diligence decisions from `spec.md` or design must be carried into `tasks.md` as design-preserving constraints, proof tasks, and reopen conditions when relevant; missing pattern comparison blocks planning handoff when implementation would otherwise choose or invent the pattern
- this wrapper owns the planning-session boundary: required inputs, allowed outputs, workflow handoff updates, and the stop point before implementation
- planning consumes the design authoring fan-out gate when separate technical design depth was triggered; missing, blocked, or ineligible `local_only` design fan-out reopens technical design
- technical design review is the required pre-planning gate for separate design depth; planning must not downgrade it into an optional note or infer it from the design author's own handoff
- planning consumes triggered test design; missing, blocked, stale, or ineligible `test-design` fan-out reopens test design before `tasks.md` is drafted
- before ending planning, leave task-ledger review fields as `pending_task_review` and route to `task-review/readiness`
- planning may record a suggested task-review fan-out candidate set, but the review phase owns the final lane decision, gate record, and implementation-readiness status
- before full-orchestrated, high-risk, complex workflow-control, or agent-backed handoff, planning may prepare the packet for `workflow-plan-adequacy-challenge`; the challenge or readiness gate still runs outside this planning wrapper
- for dedicated planning sessions, the session ends at draft or repaired planning artifacts; implementation starts only after the separate task-review/readiness gate passes

## Boundary With `planning-and-task-breakdown`
- use `planning-session` to control one planning-only session
- use `planning-and-task-breakdown` inside this session when detailed phase and task decomposition is needed
- keep the deeper skill responsible for execution slicing, acceptance criteria, verification shape, and checkpoint quality
- keep this wrapper responsible for planning-entry readiness, allowed writes, master and phase-local workflow updates, task-review handoff preparation, and stopping before implementation
- do not duplicate the full planning method in this wrapper

## Required `tasks.md` Shape
For non-trivial work, `tasks.md` should use markdown checkboxes and include, per task:
- a compact Goal Contract with one objective, one successful completion condition, and a separate blocked-stop condition
- selective read-before-coding context plus task-specific read context when the artifact list is long
- checkpoint gates when checkpoints represent real dependency or proof boundaries
- a task-local implementation quality bar when code quality, package ownership, generated-source discipline, dependency discipline, lifecycle behavior, observability, or proof-layer choices could otherwise be implicit
- a resume rule for long-running or resumable ledgers
- stable task ID such as `T001`
- phase/checkpoint label
- optional `[P]` marker only when safe to parallelize
- short action
- exact file path when known, or a narrow package/artifact surface when exact file choice is genuinely design-time unknown
- `Source:` anchors or equivalent traceability for material tasks whose requirements come from non-obvious spec, review, design, test-plan, or rollout artifacts
- dependency marker when nontrivial, such as `Depends on: T001`
- proof/verification expectation
- structured evidence fields that distinguish command/read, result, key output/ref, changed proof files when relevant, and residual blocker or narrower claim
- concise continuation lines when dependency, proof, accepted concern, or reopen detail would make a one-line checkbox hard to scan

Prefer vertical, reviewable slices with one reviewable diff story per task. If a task title needs "and" to be accurate, split it unless the approved design makes the coupling inseparable. Avoid generic tasks such as "implement feature." Multi-line task items are allowed for readability, but they must remain executable ledger items instead of design notes or strategy memos. If exact tasking requires a missing design decision or unresolved design-review finding, reopen `system-integration-design`, `go-code-ownership-design`, or `technical design review` instead of inventing the task.

## Boundary With Coding/Execution
- `planning-session` may write `tasks.md`, optional `rollout.md`, review or validation phase workflow files already required by named multi-session routing, `workflow-plan.md`, and `workflow-plans/planning.md`; it consumes `test-plan.md` from test design when triggered
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
- verify that mandatory specification review is complete for non-trivial `spec.md`
- verify that the required compact or split design context exists unless an explicit design-skip rationale already covers the task
- verify that mandatory technical design review is complete when separate design depth was triggered
- verify that dependency/OSS due-diligence decisions are present when custom infrastructure, a new dependency, or a material helper/abstraction is in scope
- verify that Pattern Fit Diligence decisions are present when architecture, workflow, integration, resilience, consistency, data-flow, or abstraction pattern choice is in scope
- accept concise approved design artifacts when they answer the current planning-critical questions explicitly; do not reopen design just because one required artifact is short or asymmetrical
- verify that any triggered conditional design artifacts exist when they affect sequencing, validation, or rollout
- if planning exposes a missing spec, specification-review, design, or technical-design-review input, route back explicitly; do not invent the missing context inside `tasks.md`

### 3. Load Execution-Critical Context
- use the compact design section or design bundle to identify dependency-establishing work, safe sequencing, coupling, validation obligations, and rollout risks
- read existing `tasks.md` or `rollout.md` only when repairing or extending an existing planning pass; read `test-plan.md` only as an approved test-design input
- keep the context narrow and planning-specific; this session does not need broad repository rediscovery when the approved design already carries the task-local technical context
- keep the handoff focused on the accepted target-state ledger; out-of-scope implications may stay as explicit concerns, proof obligations, or follow-up notes, but in-scope target-state cleanup belongs in `tasks.md` or in a reopened earlier phase

### 4. Produce Or Repair Planning Artifacts
- apply `planning-and-task-breakdown` as the deeper method when the task needs target-state execution breakdown, dependency ordering, or real checkpoints
- write or update `tasks.md` as the executable task ledger by default for non-trivial work
- create `rollout.md` only when migration sequencing, backfill, compatibility, deploy order, or failback notes need a dedicated artifact and the approved design already contains the needed rollout context
- create review or validation phase workflow files only when named multi-session routing already requires them, so later sessions update existing control artifacts instead of inventing them mid-execution
- keep blocked work separate from ready work
- keep reopen conditions explicit when implementation must hand back to `specification`, `specification review`, `technical design`, or `technical design review`
- if exact tasking or `rollout.md` requires a missing design decision or unresolved design-review finding, route back to `technical design` or `technical design review` instead of inventing executable tasks or companion-artifact context
- if exact tasking requires new scenario classes, proof levels, fail-before signals, or quality gates not present in approved `test-plan.md` or a no-test-plan rationale, route back to `test-design`

### 5. Write Or Repair `workflow-plans/planning.md`
- record only the phase-local orchestration for this planning session
- include planning status, completion marker, stop rule, next action, blockers, artifact outputs, and what can run in parallel later
- record whether companion artifacts such as `tasks.md`, approved `test-plan.md`, `rollout.md`, or later review/validation phase workflow files were required, created, or explicitly not needed
- record the technical-design-review gate result when separate design depth was triggered
- keep this file routing-only; do not turn it into `spec.md`, `design/`, or `tasks.md`

### 6. Write Or Repair `workflow-plan.md`
- update the master file with current planning-phase status, blockers, handoff state, and artifact status
- make the planning phase status explicit, and use a separate routing state when planning reopens an earlier phase
- record the next session start point without beginning that session here

### 7. Prepare The Task-Review / Readiness Handoff
- after expected `tasks.md` and any triggered companion planning artifacts are ready, leave `Task ledger review`, `Implementation readiness`, `Ledger-review fan-out`, and `Ledger-review fan-out rationale` as `pending_task_review`
- record whether later review or validation phase files are expected, created, or explicitly not expected
- record whether `tasks.md` is `draft_review_ready`, `blocked`, missing, or explicitly waived for an eligible tiny/direct-path exception
- keep implementation entry prerequisites visible so the task-review/readiness phase can judge them without re-planning
- set `Next session starts with` to `task-review/readiness` when a non-trivial ledger is ready for review, not to the first implementation task
- if planning self-checks expose coverage, order, proof, evidence-field, or workflow-control gaps that do not alter approved decisions, repair planning before handoff
- if planning self-checks expose missing or contradictory behavior, ownership, sequence, rollout, validation, or review-gate decisions, leave planning blocked and route to `specification`, `specification review`, `technical design`, or `technical design review` as the owning phase
- for full-orchestrated, high-risk, complex workflow-control, or agent-backed work, prepare the packet for `workflow-plan-adequacy-challenge` when triggered; do not treat that preparation as the challenge verdict
- for direct path or lean local with no formal trigger, record the explicit local self-check or skip rationale instead of forcing the challenge

### 8. Stop At The Boundary
- once planning artifacts and workflow handoff are consistent, stop
- do not begin implementation, validation, or review work in the same session

## Required Master `workflow-plan.md` Updates
Every completed, blocked, or reopened planning pass must update the master file with:
- current phase set to this planning checkpoint and current phase status
- link or status for `workflow-plans/planning.md`
- status for `tasks.md` as `draft_review_ready`, `blocked`, `missing`, explicitly waived, or not expected only for an eligible tiny/direct-path exception
- status for test design and `test-plan.md` as `approved`, `blocked`, `missing`, `conditional`, `waived`, or not expected, with trigger rationale for `not expected`, `conditional`, or `waived`
- status for `rollout.md` as `approved`, `draft`, `missing`, `conditional`, `waived`, or not expected, with trigger rationale for `not expected`, `conditional`, or `waived`
- specification-review status as `PASS`, `CONCERNS`, `FAIL`, or not expected with rationale
- accepted spec risks and review proof obligations carried forward when specification-review status is `CONCERNS`
- design fan-out status as `complete`, `scoped_down`, `local_only`, `blocked`, or not expected with rationale
- technical-design-review status as `PASS`, `CONCERNS`, `FAIL`, or not expected with rationale
- accepted design risks and review proof obligations carried forward when technical-design-review status is `CONCERNS`
- accepted test-design risks, scenario IDs, and proof obligations carried forward when `test-plan.md` exists
- whether later `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` were created now because named multi-session routing needs them, are explicitly not expected with rationale, or still remain blocked on a reopen
- implementation-readiness status as `pending_task_review` unless an eligible direct-path waiver means no ledger review is expected
- task-ledger review result as `pending_task_review` unless an eligible direct-path waiver means no ledger review is expected
- task-ledger review fan-out status and `Ledger-review fan-out rationale:` as `pending_task_review`
- subagent/readiness gates consumed by planning, plus unresolved blockers and reopen targets
- named accepted risks and proof obligations that task review must inspect
- named earlier phase when planning is blocked
- waiver rationale and scope when planning records an eligible tiny/direct-path waiver
- blockers, accepted assumptions, and reopen conditions that still affect task review
- workflow plan adequacy challenge packet status, or an explicit direct/lean local self-check or skip rationale
- `Session boundary reached`
- `Ready for next session`
- `Next session starts with`
- `Next session context bundle` as an always-present field: say default resume order is sufficient, or list exact artifact paths and one-line reasons for task-specific resume context

Do not leave planning readiness or handoff state implicit in chat.

## Allowed Outputs
A finished planning session may produce only:
- updated or newly created `tasks.md`
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

Assume the next session cannot see this chat. Make the prompt self-contained for the next phase but selective: include the recorded objective and current state, exact paths, phase names, task IDs, blocker names, accepted decisions, accepted assumptions or risks, proof obligations, and one-line reasons for non-obvious context files. Omit generic repo rules, resolved history, broad summaries, and artifact dumps that the next agent can read from the named files.

Rules:
- keep the prompt chat-only; do not write it into workflow artifacts or create a new artifact for it
- target the recorded first task, implementation checkpoint, or reopen route exactly
- tell the next agent which files to read first, the immediate objective, important constraints, and expected outputs
- for any non-trivial next phase, reopen target, implementation handoff, or gate that may use read-only lanes, include the exact `Subagent authorization:` line from `docs/spec-first-workflow/shared/subagents-and-handoff.md`
- missing explicit subagent authorization is not a valid `Ledger-review fan-out rationale:`; route the next session to repair authorization or keep the gate pending instead
- when the next session starts implementation from an approved `tasks.md`, render the prompt through `.agents/skills/codex-goal-prompt-composer/SKILL.md` and `docs/spec-first-workflow/shared/subagents-and-handoff.md`
- when the next phase depends on a subagent/review gate or local-only rationale, summarize `Subagent/readiness gates` from recorded artifacts and use the shared handoff doc for exact prompt shape
- if the `tasks.md` Goal Contract is missing, too vague, or conflates blocked-stop with successful completion, stop and reopen planning instead of inventing a broad objective in chat
- if there is no next session or `Ready for next session: no`, do not invent a prompt

## Planning Completion Criteria
Planning is complete when:
- execution order is explicit enough for implementation to start without re-planning
- `tasks.md` exists for lean-local or full-orchestrated non-trivial work, or an explicit tiny/direct-path waiver explains why it is not separate
- meaningful phases or tasks have acceptance criteria and planned verification
- blocked work is clearly separated from ready work
- `test-plan.md` exists only when test design triggered it, no-test-plan rationale is explicit when not needed, and `rollout.md` exists only when its trigger is real
- mandatory specification review is `PASS` or `CONCERNS` with named accepted spec risks and proof obligations for non-trivial `spec.md`
- mandatory technical design review is `PASS` or `CONCERNS` with named accepted design risks and proof obligations when separate design depth was triggered
- triggered test design is approved or explicitly not expected, and `tasks.md` traces proof-first/test work to scenario IDs when `test-plan.md` exists
- review findings classified as spec/design/planning blockers are resolved, explicitly rerouted, or still block planning rather than being hidden inside optimistic tasks
- selected design/system pattern constraints and proof obligations are explicit enough that implementation does not need to choose, reinterpret, or invent the pattern
- any review or validation phase workflow files that named multi-session routing requires were created before implementation begins, or their absence is recorded as a reopen blocker
- task-ledger review and implementation readiness are still `pending_task_review`, and the next session is routed to `task-review/readiness` unless an eligible direct-path waiver removes the review gate
- planning self-check confirms `tasks.md` is ready for the separate review to compare it against reviewed `spec.md`, specification-review obligations, required design context, technical-design-review obligations, and triggered validation or rollout obligations
- Goal-ready ledger fields separate completion from blocked-stop behavior and carry selective read context, checkpoint gates, traceability, evidence, resume, and task-local quality expectations when applicable
- master and phase-local workflow artifacts agree on planning status, blockers, and the next session start point
- required workflow plan adequacy challenge packet is prepared, or an eligible direct/lean skip rationale is explicit
- the next session can begin task-review/readiness without silently reopening spec or design
- out-of-scope implications are recorded explicitly instead of being forced into new planning blockers, while in-scope target-state work is included in the ledger or routed back to the owning earlier phase

## Stop Condition
The session is complete when the planning artifacts and workflow handoff are consistent enough for the separate task-review/readiness phase to run, readiness fields remain `pending_task_review`, required adequacy-challenge packet or skip rationale is recorded, and no implementation work has started in the current one.

## Escalate When
Escalate instead of forcing output when:
- `spec.md` is unstable, unreviewed, or failed specification review enough that planning would recreate missing decisions
- required compact or split design context is missing without an approved design-skip/merge rationale
- mandatory specification review is missing or `FAIL` for non-trivial `spec.md`
- mandatory technical design review is missing or `FAIL` after separate design depth was triggered
- triggered test design is missing, stale, blocked, or needed but not recorded
- a conditional design artifact is clearly triggered but missing
- rollout, compatibility, migration, or ownership questions remain unresolved and change the implementation order
- task-review/readiness would need to accept unnamed risk or invent missing context from planning
- the request tries to combine planning with implementation, validation, or review execution
- the work is so small that a dedicated planning session would be ceremony

## Anti-Patterns
- using this wrapper as a way to silently reopen `spec.md` or `design/`
- copying strategy or decisions into `tasks.md` instead of keeping it an executable task ledger
- creating generic tasks like "implement feature" instead of vertical, proof-bound slices
- forcing `test-plan.md` or `rollout.md` when their triggers are not real
- inventing test scenarios inside `tasks.md` instead of reopening `test-design`
- leaving required named review/validation phase workflow files to be invented mid-implementation or mid-validation
- hiding blockers inside optimistic task wording
- marking implementation handoff ready before the separate task-review/readiness gate has run
- updating `workflow-plan.md` as if implementation already started
- writing "phase 1" and then immediately coding it in the same session

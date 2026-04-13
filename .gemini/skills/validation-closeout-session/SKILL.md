---
name: validation-closeout-session
description: "Own a session dedicated only to final validation and closeout for this repository. Use when the orchestrator must prove a task is actually done with fresh evidence, update task-local `spec.md` `Validation` and `Outcome`, and update existing `workflow-plan.md`, existing `tasks.md` progress when used, plus any pre-created `workflow-plans/validation-phase-<n>.md` without drifting back into implementation. Skip tiny direct-path work and any task that still expects coding in the same session."
---

# Validation Closeout Session

## Purpose
Run only the final validation and closeout checkpoint for one task-local session.
This wrapper makes proof inputs, artifact updates, reopen handling, and stop conditions explicit; it does not implement code, repair failing behavior inline, or soften missing proof into completion language.

## Use When
- the task already completed its intended implementation and any planned review or reconciliation work
- the orchestrator needs one bounded session to run final validation with fresh evidence and close the task honestly
- task-local `spec.md` must have `Validation` and `Outcome` updated to reflect what was actually proved
- master `workflow-plan.md` must be closed or reopened explicitly
- existing `tasks.md` checkbox/progress state must be aligned with the proof when the task uses a ledger
- the task uses a dedicated post-code validation phase and its existing `workflow-plans/validation-phase-<n>.md` must be updated or repaired

## Skip When
- the work is tiny enough that inline validation plus an explicit note is sufficient and a dedicated closeout session would be ceremony
- implementation, review, reconciliation, or another earlier phase is still actively in progress
- the request tries to combine closeout with new coding, migration changes, or test authoring
- the task is not ready to state the exact claim being closed out

## Required Proof Inputs
Need the minimum closeout-ready inputs:
- the exact closeout claim or claims to prove, such as `ready for handoff`, `phase complete`, or `task done`
- current workflow routing and active phase context
- the implemented scope or planned phase that is being closed
- the proof obligations from task-local artifacts such as `spec.md`, existing `tasks.md`, optional `plan.md`, `test-plan.md`, `rollout.md`, or the current implementation or review phase file when present
- the current workspace state against which fresh commands can run
- existing `Validation`, `Outcome`, and validation-phase notes when this is a continuation or repair

Prior command output, agent reports, or chat summaries may inform the proof plan, but they are not sufficient proof for a positive closeout claim.

If a required claim, scope boundary, or proof obligation is unclear, narrow it first or reopen the right earlier phase instead of guessing.

## What Counts As Closeout-Ready Input
Treat the session as ready for closeout only when all of the following are true:
- the code or artifact changes intended for this task are already in the workspace
- the current closeout claim is explicit enough to bind to concrete proving commands
- the required proof obligations can be gathered from existing artifacts without inventing new acceptance criteria
- any expected validation-phase control artifact and required `tasks.md` already exist from pre-code planning, or the task explicitly does not use them
- any remaining uncertainty can be expressed honestly as a blocker or reopen condition rather than hidden under optimistic wording
- the next safe action, if proof fails, is to reopen an earlier phase instead of patching code here

If those conditions are not met, do not force closeout. Reopen the correct upstream phase.

## Read First
Always read:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.agents/skills/go-verification-before-completion/SKILL.md`

Then read current phase context in this order:
1. task-local `workflow-plan.md`, if present
2. task-local `workflow-plans/validation-phase-<n>.md`, if present
3. the most recent implementation or review phase workflow file that led into closeout, when present
4. task-local `spec.md`
5. existing task-local `tasks.md` when present or expected by the workflow
6. optional task-local `plan.md` when present
7. optional `test-plan.md`, `rollout.md`, or other task-local artifact only when it adds real proof obligations
8. only the smallest repository file set needed to bind proof commands to the claimed scope

Rules:
- follow `AGENTS.md` if other workflow guidance conflicts
- read the master `workflow-plan.md` before any phase-local file when both exist
- do not broad-read the repository once the closeout claim and proof scope are clear
- if phase context shows the task is not yet at validation or closeout, stop and point to the correct reopen point instead of validating by momentum

## Reference Loading
References are compact rubrics and example banks, not exhaustive checklists or replacement documentation. Load at most one reference by default. Load more than one only when the closeout task clearly spans multiple independent decision pressures, such as choosing proof scope and then updating a separate task ledger.

| Reference | Symptom | Behavior change |
|---|---|---|
| `references/closeout-readiness-examples.md` | unsure whether this session may proceed as validation closeout | choose proceed, skip, or reopen before running commands instead of validating by momentum |
| `references/claim-to-proof-closeout.md` | closeout claim is explicit but proof scope is uncertain | narrow the command set and success wording to the claim instead of treating one green check as task-wide proof |
| `references/spec-validation-outcome-updates.md` | updating `spec.md` `Validation` or `Outcome` | write a proof-shaped closeout record instead of vague "tests pass" or optimistic outcome prose |
| `references/tasks-progress-update-examples.md` | existing `tasks.md` needs checkbox or progress alignment | update only ledger items proved by fresh evidence instead of bulk-checking or creating tasks during validation |
| `references/workflow-plan-completion-vs-reopen.md` | existing workflow routing must record complete, blocked, or reopened state | make master and phase routing agree instead of leaving "mostly done" or `TBD` state |
| `references/failed-proof-and-reopen-handling.md` | required proof fails, is missing, stale, skipped, or too narrow | record the narrowest reopen target and stop instead of fixing code or softening failure during closeout |

Reference rules:
- do not bulk-load references for routine closeout
- do not let a reference override `AGENTS.md`, `docs/spec-first-workflow.md`, or `go-verification-before-completion`
- do not copy snippets blindly; bind them to the current task's artifacts, commands, and observed results
- prefer the narrowest matching reference; broad failure handling is a challenge rubric, not the default when a narrower positive update reference matches
- if an example would require implementing a fix, creating a missing phase file, creating missing `tasks.md`, or softening failed proof into completion language, reject the example and reopen instead

## Allowed Writes
This session may write or update only:
- task-local `spec.md`, limited to `Validation`, `Outcome`, and any minimal cross-reference needed to make reopen state honest
- existing task-local `workflow-plan.md`
- existing task-local `workflow-plans/validation-phase-<n>.md` when the task already uses a dedicated validation phase
- existing task-local `tasks.md`, limited to checkbox/progress state that the fresh proof actually supports

Do not create a phase-local validation file or missing `tasks.md` in this session. If either required artifact is missing, reopen planning or the relevant earlier phase instead of inventing it during closeout.

## Prohibited Actions
Do not:
- implement new code, tests, migrations, or configuration changes as part of closeout
- repair failing verification inline "just to finish"
- rewrite `Decisions`, `design/`, or optional `plan.md` instead of recording a reopen
- claim `done`, `complete`, `ready`, or equivalent success language without fresh proof that matches scope
- trust stale command output, delegated summaries, or yesterday's passing run as current proof
- create missing `workflow-plan.md` or `workflow-plans/validation-phase-<n>.md` during closeout
- create missing `tasks.md` or add new task entries during closeout
- turn `workflow-plans/validation-phase-<n>.md` into a second `spec.md`, a new plan, or a hidden implementation checklist
- silently continue into a new implementation phase when validation exposes a defect

## Core Defaults
- this is an orchestrator-facing wrapper, not a replacement for `go-verification-before-completion`
- `AGENTS.md` owns the workflow contract; `docs/spec-first-workflow.md` owns the artifact mechanics
- `go-verification-before-completion` owns claim-to-proof discipline, command sizing, and evidence wording
- validation is artifact-consuming: consume existing approved artifacts and fresh proof rather than authoring new workflow/process artifacts here
- this wrapper owns when a dedicated closeout session may run, what files may change, what `done` means for the session, how reopen conditions are recorded, and why the session must stop
- use the smallest sufficient command set, but never weaker than the closeout claim
- a finished closeout session ends at honest completion or an explicit reopen target; it does not drift back into implementation

## Boundary With `go-verification-before-completion`
- Reuse `go-verification-before-completion` for the actual proof pass: explicit claim, explicit scope, commands actually run, observed result, and proportional conclusion.
- Do not copy its claim-to-proof table into local folklore or weaken it for convenience.
- If its proof bar says the claim is not verified, this session must record a blocker or reopen. It may not "balance" the failure with optimistic closeout wording.
- This wrapper extends the verification gate only by adding artifact ownership:
  - update `spec.md` so `Validation` and `Outcome` reflect reality
  - update `workflow-plan.md` so completion or reopen routing is explicit
  - update existing `tasks.md` progress when the task uses a ledger
  - update an existing `workflow-plans/validation-phase-<n>.md` when a dedicated validation phase is active

## Workflow

### 1. Confirm This Session Owns Validation And Closeout Only
- check the master workflow plan and current phase context first
- if implementation, review, or reconciliation is still the active phase, stop and hand back the correct reopen point
- if the work is tiny enough for inline validation only, say so directly and stop rather than forcing this wrapper
- if the request asks for code changes during closeout, refuse that boundary crossing before doing anything else

### 2. Bind The Final Claim To The Right Scope
- name the exact closeout claim or claims
- bind each claim to the concrete changed surface, planned phase, or task boundary it covers
- separate proof required now from nice-to-have checks
- if the claim is broader than the available proof surface, narrow the wording or reopen earlier work

### 3. Gather Proof Inputs And Choose Commands
- derive proof obligations from `spec.md`, existing `tasks.md`, optional `plan.md`, `test-plan.md`, `rollout.md`, and current phase artifacts
- choose the smallest command set that honestly proves the current claim, following `go-verification-before-completion`
- keep the verification surface proportional: scoped claims may use scoped commands; repository-wide claims need repository-wide proof
- if a required command is unclear, stop and escalate instead of improvising a weaker check

### 4. Run Fresh Verification
- execute the proving commands against the current workspace state in this session
- capture the commands actually run and the key pass or fail signals
- treat stale output, agent summaries, or previous green runs as context only, never as positive proof
- if a command fails, record the failure and move to reopen handling instead of patching code

### 5. Update `spec.md` Validation And Outcome
- update `Validation` with the actual proof record from this session
- keep the verification report aligned with `go-verification-before-completion`:
  - `Claim`
  - `Scope`
  - `Verification Commands`
  - `Observed Result`
  - `Conclusion`
  - `Next Action`
- update `Outcome` to say only what the fresh evidence supports
- if proof is partial or failing, `Outcome` must say so directly instead of implying closure

### 6. Update Existing `tasks.md` Progress When Used
- only update `tasks.md` when it already exists and belongs to this task
- update checkbox/progress state only for tasks whose proof was actually run and observed in this session
- do not add, split, reorder, or rewrite tasks during closeout
- if expected `tasks.md` is missing, record a planning reopen target instead of creating it here

### 7. Record Reopen Conditions Instead Of Re-Implementing
- when proof fails, is missing, or reveals the wrong scope, record a reopen target instead of changing code
- choose the narrowest honest reopen target:
  - reopen `implementation-phase-<n>` when the behavior or tests are wrong
  - reopen `review-phase-<n>` when an unresolved review issue blocks honest closeout
  - reopen `planning`, `technical-design`, or `specification` when the proof gap exposes a real upstream contract or sequencing problem
- make each reopen item explicit:
  - failed or missing proof
  - why it blocks closeout now
  - which phase must reopen next
  - what the next session must resolve
- stop after recording the reopen; do not "just fix one thing" in this session

### 8. Write Or Repair `workflow-plans/validation-phase-<n>.md` When Used
- only update this file when planning created it before implementation started
- if the task should be using a dedicated validation phase file and it is missing, or if required `tasks.md` is missing, record a reopen target instead of creating it now
- record phase-local closeout routing only:
  - closeout claim or claims
  - proof inputs used
  - commands executed
  - phase status
  - completion marker
  - stop rule
  - next action
  - blockers or reopen target
- keep this file routing-only; do not turn it into a second `Validation` section, a second `tasks.md`, or an implementation scratchpad
- if the task is not using a dedicated validation phase file, do not invent one

### 9. Write Or Repair `workflow-plan.md`
- update master phase status, blockers, and next-session routing
- make it explicit whether closeout is complete, blocked, or reopened
- if the task is honestly done, close the workflow instead of leaving ambiguous "mostly done" language:
  - `Session boundary reached: yes`
  - `Ready for next session: no`
  - `Next session starts with: N/A` unless a later follow-up task is intentionally created
- if the task is not done, route the next session to the exact reopen target:
  - `Session boundary reached: yes`
  - `Ready for next session: yes`
  - `Next session starts with: <exact reopen target>`

### 10. Stop At The Boundary
- once `spec.md`, `workflow-plan.md`, existing `tasks.md` when used, and any active validation phase file agree on the result, stop
- do not begin code changes, new test authoring, or the next implementation phase in the same session

## What `Done` Means
Closeout is done only when all of the following are true:
- every positive closeout claim in scope has fresh evidence from this session
- `spec.md` `Validation` records the actual commands and observed results instead of intention or memory
- `spec.md` `Outcome` says only what the evidence proved, with no optimistic overreach
- `workflow-plan.md` makes the task state explicit as complete or done, with the session boundary closed
- existing `tasks.md`, when used, has checkbox/progress state aligned with the fresh proof and no invented tasks
- `workflow-plans/validation-phase-<n>.md`, when used, shows the phase is complete and why the session stopped
- no new implementation work was performed during closeout

If any of those fail, the task is not done yet. Record the reopen honestly.

## Required Master `workflow-plan.md` Updates
Every completed, blocked, or reopened pass must update the master file with:
- current phase set to this validation or closeout checkpoint and current phase status
- link or status for `workflow-plans/validation-phase-<n>.md` when a dedicated validation phase is active, or an explicit note that none is used
- status for `spec.md` closeout updates, including whether `Validation` and `Outcome` were refreshed this session
- status for existing `tasks.md` progress updates when the task uses a ledger, or the planning reopen target if required `tasks.md` is missing
- `Session boundary reached`
- `Ready for next session`
- `Next session starts with`
- blockers, failed proof, accepted limits, and reopen targets that still affect closure
- whether the task is honestly done or has reopened an earlier phase

Do not leave final task state implicit in chat.

## Expected Outputs
A finished validation-closeout session produces only closeout artifacts and routing:
- updated `spec.md` with fresh `Validation` evidence and honest `Outcome`
- updated `workflow-plan.md`
- updated `workflow-plans/validation-phase-<n>.md` only when a dedicated validation phase is actually in use and the file already exists
- updated existing `tasks.md` checkbox/progress state only when the task already uses it
- an honest `complete`, `blocked`, or `reopened` closeout state with the next session start point made explicit

It does not produce implementation output, design changes, new plans, or silent fixes.

## Stop Condition
The session is complete when:
- the closeout claim is explicit and bound to the right scope
- fresh proof was run or the proof gap was documented honestly
- `spec.md`, `workflow-plan.md`, existing `tasks.md` when used, and any active validation phase file agree on completion or reopen state
- the next session start point is explicit, including `N/A` for a truly closed task or the exact reopen target when not closed
- no implementation or other earlier-phase work started in this session

## Escalate When
Escalate instead of forcing output when:
- the claimed closeout scope is unclear or broader than the available proof surface
- the task is not actually at validation or closeout yet
- the request tries to combine closeout with new code changes
- required proving commands are unclear and a weaker substitute would be dishonest
- proof failures expose an upstream artifact problem that requires reopening specification, design, or planning
- the task is so small that a dedicated closeout session would be ceremony

## Anti-Patterns
- treating this wrapper as a permission slip to fix code during validation
- copying stale command output into `Validation` as if it were fresh evidence
- writing `Outcome` as a success summary when `Conclusion` is really `not verified`
- creating `workflow-plans/validation-phase-<n>.md` for tasks that never adopted a dedicated validation phase
- creating missing `tasks.md` or inventing new ledger items during closeout
- using closeout to rewrite `Decisions`, `design/`, or optional `plan.md` instead of naming a reopen target
- letting "almost green" become "done"

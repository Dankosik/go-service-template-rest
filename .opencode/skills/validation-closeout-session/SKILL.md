---
name: validation-closeout-session
description: "Own a session dedicated only to final validation and closeout for this repository when closeout needs artifact updates. Use when the orchestrator must prove a task is actually done with fresh evidence, update task-local `spec.md` `Validation` and `Outcome`, existing `tasks.md` progress when used, and any pre-created `workflow-plans/validation-phase-N.md` explicitly named by the approved ledger, without drifting back into implementation. Skip direct-path inline closeout and any task that still expects coding in the same session."
---

# Validation Closeout Session

## Purpose
Run only the final validation and closeout checkpoint for one task-local session.
This wrapper makes proof inputs, artifact updates, reopen handling, and stop conditions explicit; it does not implement code, repair failing behavior inline, or soften missing proof into completion language.

## Outcome-First Operating Rules
- Start by naming the skill-specific outcome, success criteria, constraints, available evidence, and stop rule.
- Treat workflow steps as decision rules, not a ritual checklist. Follow exact order only when this skill or the repository contract makes the sequence an invariant.
- Use the minimum context, references, tools, and validation loops that can change the deliverable; stop expanding when the quality bar is met.
- Before acting, resolve prerequisite discovery, lookup, or artifact reads that the outcome depends on; parallelize only independent evidence gathering and synthesize before the next decision.
- Prefer bounded assumptions and local evidence over broad questioning; ask only when a missing fact would change correctness, ownership, safety, or scope.
- When evidence is missing or conflicting, retry once with a targeted strategy or label the assumption, blocker, or reopen target instead of treating absence as proof.
- Finish only when the requested deliverable is complete in the required shape and verification or a clearly named blocker/residual risk is recorded.

## Use When
- the task already completed its intended implementation and any planned review or reconciliation work
- the orchestrator needs one bounded session to run final validation with fresh evidence and close the task honestly
- task-local `spec.md` must have `Validation` and `Outcome` updated to reflect what was actually proved
- existing `tasks.md` checkbox/progress state must be aligned with the proof when the task uses a ledger
- the approved `tasks.md` explicitly names a dedicated post-code validation phase and its existing `workflow-plans/validation-phase-<n>.md` must be updated or repaired
- lean-local `spec.md`/`tasks.md` closeout needs fresh proof recorded without creating extra phase-control artifacts

## Skip When
- the work is tiny enough that inline validation plus an explicit note is sufficient and a dedicated closeout session would be ceremony
- implementation, review, reconciliation, or another earlier phase is still actively in progress
- the request tries to combine closeout with new coding, migration changes, or test authoring
- the task is not ready to state the exact claim being closed out

## Required Proof Inputs
Need the minimum closeout-ready inputs:
- the exact closeout claim or claims to prove, such as `ready for handoff`, `phase complete`, or `task done`
- the approved `tasks.md` ledger, or current workflow routing only when no approved ledger exists yet
- the implemented scope or planned phase that is being closed
- the proof obligations from task-local artifacts such as `spec.md`, existing `tasks.md`, `test-plan.md`, `rollout.md`, or the current review phase file when present
- the cleanup classification required by the approved ledger: removed, refactored into active path, retained with owner/reason/proof/exit condition, or not applicable for each known old surface
- the current workspace state against which fresh commands can run
- existing `Validation`, `Outcome`, and validation-phase notes when this is a continuation or repair

Prior command output, agent reports, or chat summaries may inform the proof plan, but they are not sufficient proof for a positive closeout claim.

If a required claim, scope boundary, or proof obligation is unclear, narrow it first or reopen the right earlier phase instead of guessing.

## What Counts As Closeout-Ready Input
Treat the session as ready for closeout only when all of the following are true:
- the code or artifact changes intended for this task are already in the workspace
- the current closeout claim is explicit enough to bind to concrete proving commands
- the required proof obligations can be gathered from existing artifacts without inventing new acceptance criteria
- any expected validation-phase control artifact and required `tasks.md` already exist from pre-code planning, lean-local tasking, or the task explicitly does not use them
- any remaining uncertainty can be expressed honestly as a blocker or reopen condition rather than hidden under optimistic wording
- the next safe action, if proof fails, is to reopen an earlier phase instead of patching code here

If those conditions are not met, do not force closeout. Reopen the correct upstream phase.

## Read First
Always read:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.agents/skills/go-verification-before-completion/SKILL.md`

Then read current phase context in this order:
1. existing task-local `tasks.md` when present or expected by the workflow
2. task-local `spec.md`
3. `test-plan.md`, `rollout.md`, or other task-local artifact only when named by `tasks.md` or needed for real proof obligations
4. task-local `workflow-plans/validation-phase-<n>.md` only when `tasks.md` explicitly names that pre-created phase file
5. task-local `workflow-plan.md` only when no approved `tasks.md` exists yet
6. the smallest repository file set needed to bind proof commands to the claimed scope

Rules:
- follow `AGENTS.md` if other workflow guidance conflicts
- after approved `tasks.md` exists, treat it as the execution authority; do not let stale workflow routing override the ledger
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
| `references/workflow-plan-completion-vs-reopen.md` | the approved ledger explicitly names a pre-created validation phase file that must record done or reopen state | update only the named routing artifact instead of treating every existing `workflow-plan.md` as a closeout surface |
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
- existing task-local `tasks.md`, limited to checkbox/progress state that the fresh proof actually supports
- existing task-local `workflow-plans/validation-phase-<n>.md` only when the approved `tasks.md` explicitly names that pre-created phase file

Do not create a phase-local validation file or missing `tasks.md` in this session. If either required artifact is missing, reopen planning or the relevant earlier phase instead of inventing it during closeout.

## Prohibited Actions
Do not:
- implement new code, tests, migrations, or configuration changes as part of closeout
- repair failing verification inline "just to finish"
- rewrite `Decisions` or `design/` instead of recording a reopen
- claim `done`, `complete`, `ready`, or equivalent success language without fresh proof that matches scope
- trust stale command output, delegated summaries, or yesterday's passing run as current proof
- create missing `workflow-plan.md` or `workflow-plans/validation-phase-<n>.md` during closeout
- create missing `tasks.md` or add new task entries during closeout
- turn `workflow-plans/validation-phase-<n>.md` into a second `spec.md`, a new plan, or a hidden implementation checklist
- silently continue into coding fixes when validation exposes a defect

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
- Skipped, unavailable, stale, failing, or too-narrow proof cannot satisfy a task checkbox, checkpoint, or completion claim. Leave affected tasks unchecked and record `Blocked:` or the narrower claim.
- This wrapper extends the verification gate only by adding artifact ownership:
- update `spec.md` so `Validation` and `Outcome` reflect reality
- update existing `tasks.md` progress when the task uses a ledger
- record removal/refactor/retention evidence in existing `tasks.md` and `spec.md` closeout fields instead of leaving cleanup proof implicit in chat
- update an existing `workflow-plans/validation-phase-<n>.md` only when the approved ledger explicitly names that phase file
- for lean local, update `spec.md` and existing `tasks.md` directly; do not create a validation phase file just to mirror full-orchestrated closeout

## Workflow

### 1. Confirm This Session Owns Validation And Closeout Only
- check the approved `tasks.md` and claimed closeout scope first; use workflow routing only when no approved ledger exists yet
- if implementation, review, or reconciliation is still the active phase, stop and hand back the correct reopen point
- if the work is direct path and inline validation is enough, say so directly and stop rather than forcing this wrapper
- if the request asks for code changes during closeout, refuse that boundary crossing before doing anything else

### 2. Bind The Final Claim To The Right Scope
- name the exact closeout claim or claims
- bind each claim to the concrete changed surface, planned phase, or task boundary it covers
- separate proof required now from nice-to-have checks
- if the claim is broader than the available proof surface, narrow the wording or reopen earlier work

### 3. Gather Proof Inputs And Choose Commands
- derive proof obligations from `spec.md`, existing `tasks.md`, `test-plan.md`, `rollout.md`, and current phase artifacts
- include targeted old-surface negative checks, retained-surface proof, generated/mirror drift proof, and stale-proof rejection when the approved task includes cleanup or replacement work
- require named negative proof for each retired surface; do not accept generic searches such as `rg legacy` unless the retired surface is literally named `legacy`
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
- when `tasks.md` uses structured evidence fields, update the existing fields directly: command/read, result, key output or evidence ref, changed proof files when relevant, and residual blocker or narrower claim
- if the observed proof is skipped, unavailable, stale, failing, or narrower than the task, leave the checkbox unchecked and record `Blocked:` or the narrower claim in the existing evidence fields
- for cleanup tasks, record each known old surface as removed, refactored into the active path, retained with owner/reason/proof/exit condition, or not applicable based on fresh proof
- when a `Legacy cleanup audit` table exists, update it row by row from fresh evidence instead of replacing it with prose
- do not add, split, reorder, or rewrite tasks during closeout
- if expected `tasks.md` is missing, record a planning reopen target instead of creating it here

### 7. Record Reopen Conditions Instead Of Re-Implementing
- when proof fails, is missing, or reveals the wrong scope, record a reopen target instead of changing code
- choose the narrowest honest reopen target:
  - reopen implementation when the behavior or tests are wrong
  - reopen `review-phase-<n>` when an unresolved review issue blocks honest closeout
  - reopen `planning`, `technical-design`, or `specification` when the proof gap exposes a real upstream contract or sequencing problem
- make each reopen item explicit:
  - failed or missing proof
  - why it blocks closeout now
  - which phase must reopen next
  - what the next session must resolve
- stop after recording the reopen; do not "just fix one thing" in this session

### 8. Write Or Repair `workflow-plans/validation-phase-<n>.md` When Used
- only update this file when planning created it before implementation started and the approved `tasks.md` explicitly names it
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

### 9. Leave `workflow-plan.md` Alone After Ledger Approval
- do not update `workflow-plan.md` during closeout merely because it exists
- if fresh proof fails, record the reopen target in `spec.md` `Validation`/`Outcome`, existing `tasks.md` progress, and the explicitly named validation phase file when one is in use
- if there is no approved `tasks.md` yet and workflow routing still owns the phase, stop and reopen planning instead of using this closeout session to repair master workflow state

### 10. Stop At The Boundary
- once `spec.md`, existing `tasks.md` when used, and any explicitly named validation phase file agree on the result, stop
- do not begin code changes, new test authoring, or the next implementation task in the same session

## What `Done` Means
Closeout is done only when all of the following are true:
- every positive closeout claim in scope has fresh evidence from this session
- `spec.md` `Validation` records the actual commands and observed results instead of intention or memory
- `spec.md` `Outcome` says only what the evidence proved, with no optimistic overreach
- existing `tasks.md`, when used, has checkbox/progress state aligned with the fresh proof and no invented tasks
- no task, checkpoint, or completion claim is marked green using skipped, unavailable, stale, failing, or too-narrow proof
- cleanup-sensitive tasks record fresh remove/refactor/retain/not-applicable evidence, including retained-surface owner/reason/proof/exit condition when old artifacts remain
- `workflow-plans/validation-phase-<n>.md`, when explicitly named by `tasks.md`, shows the phase is complete and why the session stopped
- no new implementation work was performed during closeout

If any of those fail, the task is not done yet. Record the reopen honestly.

## No Required Master `workflow-plan.md` Updates After `tasks.md`
After approved `tasks.md` exists, master workflow-control updates are not part of validation closeout. Every completed, blocked, or reopened pass must instead update:
- `spec.md` closeout fields, including whether `Validation` and `Outcome` were refreshed this session
- existing `tasks.md` progress when the task uses a ledger, or a planning reopen target if required `tasks.md` is missing
- an explicitly named existing validation phase file, only when the approved ledger requires it

Do not leave final task state implicit in chat, but do not mirror it into `workflow-plan.md` just because that file exists.

## Expected Outputs
A finished validation-closeout session produces only closeout artifacts and routing:
- updated `spec.md` with fresh `Validation` evidence and honest `Outcome`
- updated existing `tasks.md` checkbox/progress state only when the task already uses it
- updated `workflow-plans/validation-phase-<n>.md` only when the approved ledger explicitly names it and the file already exists
- an honest closeout phase status such as `complete` or `blocked`, plus a separate task or routing state when reopened, with the next session start point made explicit

It does not produce implementation output, design changes, new plans, or silent fixes.

## Required Final Chat Handoff
When this session ends with `Session boundary reached: yes` and `Ready for next session: yes`, the final chat response must include a `Recommended next-session prompt` section with one copy-pastable fenced text block.

Derive that prompt from the recorded workflow handoff state, not memory:
- `Next session starts with`
- `Next session context bundle`
- this phase's stop rule
- blockers, failed proof, accepted limits, or reopen conditions that still matter
- the expected artifact or output for the next session

Assume the next session cannot see this chat. Make the prompt self-contained for the next phase but selective: include the recorded objective and current state, exact paths, phase names, task IDs, blocker names, accepted decisions, accepted limits or risks, proof obligations, and one-line reasons for non-obvious context files. Omit generic repo rules, resolved history, broad summaries, and artifact dumps that the next agent can read from the named files.

Rules:
- keep the prompt chat-only; do not write it into workflow artifacts or create a new artifact for it
- target the recorded reopen or follow-up route exactly
- tell the next agent which files to read first, the immediate objective, important constraints, and expected outputs
- if there is no next session or `Ready for next session: no`, do not invent a prompt; say there is no next-session prompt because the workflow is closed

## Stop Condition
The session is complete when:
- the closeout claim is explicit and bound to the right scope
- fresh proof was run or the proof gap was documented honestly
- `spec.md`, existing `tasks.md` when used, and any explicitly named validation phase file agree on completion or reopen state
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
- using closeout to rewrite `Decisions` or `design/` instead of naming a reopen target
- letting "almost green" become "done"
